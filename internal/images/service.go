package images

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
)

// Pic names one picture on the site: an arcade shot (Key = img stem, Slot
// title/snap/ingame) or a system photo (Key = core, Slot "system").
type Pic struct {
	Key, Slot string
}

type scaledKey struct {
	Pic
	W, H   int
	Native bool
}

type entry struct {
	img  *image.RGBA
	size int
	el   *list.Element
}

// Service implements app.Images over a store directory and the site.
type Service struct {
	dir    string
	client *fetch.Client // nil = never download
	lg     *log.Logger

	mu          sync.Mutex
	cache       map[scaledKey]*entry // scaled bitmaps, LRU by bytes
	lru         *list.List           // front = most recent
	bytes       int
	budget      int
	raw         map[Pic]*image.RGBA // decoded originals of the rows in view
	rawOrder    []Pic
	wanted      []scaledKey // priority order, replaced on every Want
	inflight    map[scaledKey]bool
	netBusy     map[Pic]bool
	missing     map[Pic]bool
	failed      map[Pic]bool      // decode failed this run
	retryAt     map[Pic]time.Time // download failed: not before this
	retries     map[Pic]int
	offline     bool
	prefetch    []Pic // background download list, in priority order
	prefetchOn  bool
	prefetchPos int
	prefetched  int
	kick        chan struct{}
	decodeKick  chan struct{}
	paused      bool // no decoding while the UI scrolls
	have        int  // prefetch pictures on disk (kept current, never rescanned per call)
	haveKnown   bool
	ready       chan struct{}
	stop        chan struct{}
	ctx         context.Context // cancelled on Close
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// New opens a service over dir (created on demand). client may be nil.
func New(dir string, client *fetch.Client, lg *log.Logger, budget int) *Service {
	if budget <= 0 {
		budget = 24 << 20
	}
	s := &Service{
		dir: dir, client: client, lg: lg,
		cache: map[scaledKey]*entry{}, lru: list.New(), budget: budget,
		raw: map[Pic]*image.RGBA{}, inflight: map[scaledKey]bool{}, netBusy: map[Pic]bool{},
		missing: map[Pic]bool{}, failed: map[Pic]bool{}, retryAt: map[Pic]time.Time{}, retries: map[Pic]int{},
		kick: make(chan struct{}, 1), decodeKick: make(chan struct{}, 1),
		ready: make(chan struct{}, 1), stop: make(chan struct{}),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	for _, sub := range []string{"title", "snap", "ingame", "systems"} {
		os.MkdirAll(filepath.Join(dir, sub), 0755)
	}
	s.loadMissing()
	s.wg.Add(3)
	go s.decodeLoop()
	go s.netLoop()
	go s.netLoop()
	return s
}

// Ready delivers a token whenever a picture became available; the UI loop
// repaints on it.
func (s *Service) Ready() <-chan struct{} { return s.ready }

func (s *Service) signal() {
	select {
	case s.ready <- struct{}{}:
	default:
	}
}

func (s *Service) poke(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// Close stops the workers, cutting any download short.
func (s *Service) Close() {
	close(s.stop)
	s.cancel()
	s.wg.Wait()
	s.saveMissing()
}

// SetOffline tells the service whether downloads can be attempted.
func (s *Service) SetOffline(off bool) {
	s.mu.Lock()
	changed := s.offline != off
	s.offline = off
	s.mu.Unlock()
	if changed {
		s.poke(s.kick)
		s.signal()
	}
}

// SetPrefetch installs the background download list (priority order) and
// switches prefetching on or off.
func (s *Service) SetPrefetch(pics []Pic, on bool) {
	s.mu.Lock()
	rescan := !s.haveKnown || len(pics) != len(s.prefetch)
	s.prefetch = pics
	s.prefetchOn = on
	s.prefetchPos = 0
	s.mu.Unlock()
	if rescan {
		go s.countHave(pics)
	}
	s.poke(s.kick)
}

// countHave counts the prefetch pictures already on disk, once, off the UI.
func (s *Service) countHave(pics []Pic) {
	n := 0
	for _, p := range pics {
		if s.exists(p) {
			n++
		}
	}
	s.mu.Lock()
	s.have, s.haveKnown = n, true
	s.mu.Unlock()
	s.signal()
}

// SetPaused stops the decoder (the UI is scrolling and wants both cores'
// worth of quiet); unpausing resumes with whatever is wanted now.
func (s *Service) SetPaused(p bool) {
	s.mu.Lock()
	s.paused = p
	s.mu.Unlock()
	if !p {
		s.poke(s.decodeKick)
		s.poke(s.kick)
	}
}

// Progress reports the prefetch state: files present, total in the list.
func (s *Service) Progress() (have, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.have, len(s.prefetch)
}

func (s *Service) path(p Pic) string {
	if p.Slot == "system" {
		return filepath.Join(s.dir, "systems", p.Key+".png")
	}
	return filepath.Join(s.dir, p.Slot, p.Key+".png")
}

func (s *Service) exists(p Pic) bool {
	_, err := os.Stat(s.path(p))
	return err == nil
}

// Get implements app.Images.
func (s *Service) Get(req app.ImageReq) (*image.RGBA, app.ImageState) {
	p := Pic{req.Key, req.Slot}
	k := scaledKey{p, req.W, req.H, req.Native}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.cache[k]; ok {
		s.lru.MoveToFront(e.el)
		return e.img, app.ImageReady
	}
	if s.missing[p] || s.failed[p] {
		return nil, app.ImageMissing
	}
	if s.raw[p] == nil && !s.exists(p) {
		if s.client == nil || s.offline {
			return nil, app.ImageOffline
		}
	}
	return nil, app.ImageLoading
}

// Want implements app.Images: the priority list of what the screen needs.
func (s *Service) Want(reqs []app.ImageReq) {
	s.mu.Lock()
	s.wanted = s.wanted[:0]
	for _, r := range reqs {
		s.wanted = append(s.wanted, scaledKey{Pic{r.Key, r.Slot}, r.W, r.H, r.Native})
	}
	s.mu.Unlock()
	s.poke(s.kick)
	s.poke(s.decodeKick)
}

// nextDecode picks the highest-priority wanted picture whose file exists
// and which is not cached or in flight.
func (s *Service) nextDecode() (scaledKey, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused {
		return scaledKey{}, false
	}
	for _, k := range s.wanted {
		if _, ok := s.cache[k]; ok || s.inflight[k] || s.missing[k.Pic] || s.failed[k.Pic] {
			continue
		}
		if s.raw[k.Pic] == nil && !s.exists(k.Pic) {
			continue
		}
		s.inflight[k] = true
		return k, true
	}
	return scaledKey{}, false
}

func (s *Service) decodeLoop() {
	defer s.wg.Done()
	for {
		k, ok := s.nextDecode()
		if !ok {
			select {
			case <-s.stop:
				return
			case <-s.decodeKick:
			}
			continue
		}
		s.decode(k)
	}
}

func (s *Service) decode(k scaledKey) {
	defer func() {
		s.mu.Lock()
		delete(s.inflight, k)
		s.mu.Unlock()
	}()
	s.mu.Lock()
	src := s.raw[k.Pic]
	s.mu.Unlock()
	if src == nil {
		b, err := os.ReadFile(s.path(k.Pic))
		if err != nil {
			return
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			s.lg.Printf("images: %s: %v (deleting)", s.path(k.Pic), err)
			os.Remove(s.path(k.Pic))
			s.mu.Lock()
			s.failed[k.Pic] = true
			s.mu.Unlock()
			s.signal()
			return
		}
		src = ToRGBA(img)
		s.mu.Lock()
		s.raw[k.Pic] = src
		s.rawOrder = append(s.rawOrder, k.Pic)
		for len(s.rawOrder) > 8 {
			old := s.rawOrder[0]
			s.rawOrder = s.rawOrder[1:]
			if old != k.Pic {
				delete(s.raw, old)
			}
		}
		s.mu.Unlock()
	}
	scaled := Scale(src, k.W, k.H, k.Native)
	s.mu.Lock()
	s.put(k, scaled)
	s.mu.Unlock()
	s.signal()
}

// put stores a scaled bitmap, evicting least recently used ones past budget.
func (s *Service) put(k scaledKey, img *image.RGBA) {
	size := len(img.Pix)
	if e, ok := s.cache[k]; ok {
		s.bytes -= e.size
		s.lru.Remove(e.el)
	}
	e := &entry{img: img, size: size}
	e.el = s.lru.PushFront(k)
	s.cache[k] = e
	s.bytes += size
	for s.bytes > s.budget && s.lru.Len() > 1 {
		back := s.lru.Back()
		old := back.Value.(scaledKey)
		if oe, ok := s.cache[old]; ok {
			s.bytes -= oe.size
			delete(s.cache, old)
		}
		s.lru.Remove(back)
	}
}

// nextDownload picks a wanted picture missing on disk, else a prefetch one.
func (s *Service) nextDownload() (Pic, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused {
		return Pic{}, false
	}
	if s.client == nil || s.offline {
		return Pic{}, false
	}
	now := time.Now()
	for _, k := range s.wanted {
		p := k.Pic
		if s.missing[p] || s.failed[p] || s.netBusy[p] || s.raw[p] != nil || s.exists(p) || now.Before(s.retryAt[p]) {
			continue
		}
		s.netBusy[p] = true
		return p, true
	}
	if s.prefetchOn {
		// at most one prefetch in flight while the screen still waits
		busy := len(s.netBusy)
		if busy > 0 {
			return Pic{}, false
		}
		for s.prefetchPos < len(s.prefetch) {
			p := s.prefetch[s.prefetchPos]
			s.prefetchPos++
			if s.missing[p] || s.netBusy[p] || s.exists(p) || now.Before(s.retryAt[p]) {
				continue
			}
			s.netBusy[p] = true
			return p, true
		}
	}
	return Pic{}, false
}

// backoff (mu held) schedules the next attempt at p: 15 s, then doubling
// up to 8 minutes, so a bad server or a full card is not hammered.
func (s *Service) backoff(p Pic) {
	n := s.retries[p]
	s.retries[p] = n + 1
	if n > 5 {
		n = 5
	}
	s.retryAt[p] = time.Now().Add(15 * time.Second << uint(n))
}

func (s *Service) netLoop() {
	defer s.wg.Done()
	for {
		p, ok := s.nextDownload()
		if !ok {
			select {
			case <-s.stop:
				return
			case <-s.kick:
			case <-time.After(30 * time.Second):
			}
			continue
		}
		s.download(p)
	}
}

func (s *Service) download(p Pic) {
	defer func() {
		s.mu.Lock()
		delete(s.netBusy, p)
		s.mu.Unlock()
		s.poke(s.kick)
	}()
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	b, err := s.client.Image(ctx, p.Slot, p.Key)
	if s.ctx.Err() != nil {
		return // closing: not a failure worth remembering
	}
	if err != nil {
		s.mu.Lock()
		if errors.Is(err, fetch.ErrNotFound) {
			s.missing[p] = true
		} else if errors.Is(err, fetch.ErrOffline) {
			s.offline = true
			s.lg.Printf("images: offline: %v", err)
		} else {
			s.backoff(p)
			s.lg.Printf("images: %s/%s: %v (retry in %v)", p.Slot, p.Key, err, time.Until(s.retryAt[p]).Round(time.Second))
		}
		s.mu.Unlock()
		s.signal()
		return
	}
	path := s.path(p)
	tmp := path + ".part"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		s.mu.Lock()
		s.backoff(p)
		s.mu.Unlock()
		s.lg.Printf("images: write %s: %v", tmp, err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		s.lg.Printf("images: rename %s: %v", tmp, err)
		return
	}
	s.mu.Lock()
	s.prefetched++
	s.mu.Unlock()
	s.mu.Lock()
	for _, q := range s.prefetch {
		if q == p {
			s.have++
			break
		}
	}
	s.mu.Unlock()
	s.poke(s.decodeKick)
	s.signal()
}

// missing.json remembers 404s so they are not retried every run.
func (s *Service) loadMissing() {
	b, err := os.ReadFile(filepath.Join(s.dir, "missing.json"))
	if err != nil {
		return
	}
	var keys []string
	if json.Unmarshal(b, &keys) != nil {
		return
	}
	for _, k := range keys {
		if i := indexByte(k, '/'); i > 0 {
			s.missing[Pic{Key: k[i+1:], Slot: k[:i]}] = true
		}
	}
}

func (s *Service) saveMissing() {
	s.mu.Lock()
	var keys []string
	for p := range s.missing {
		keys = append(keys, p.Slot+"/"+p.Key)
	}
	s.mu.Unlock()
	if len(keys) == 0 {
		return
	}
	b, _ := json.Marshal(keys)
	os.WriteFile(filepath.Join(s.dir, "missing.json"), b, 0644)
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ClearCache wipes the store and memory caches (settings action).
func (s *Service) ClearCache() {
	s.mu.Lock()
	s.cache = map[scaledKey]*entry{}
	s.lru.Init()
	s.bytes = 0
	s.raw = map[Pic]*image.RGBA{}
	s.rawOrder = nil
	s.missing = map[Pic]bool{}
	s.failed = map[Pic]bool{}
	s.prefetchPos = 0
	s.have = 0
	s.mu.Unlock()
	s.signal()
	for _, sub := range []string{"title", "snap", "ingame", "systems"} {
		os.RemoveAll(filepath.Join(s.dir, sub))
		os.MkdirAll(filepath.Join(s.dir, sub), 0755)
	}
	os.Remove(filepath.Join(s.dir, "missing.json"))
	s.poke(s.kick)
	s.signal()
}
