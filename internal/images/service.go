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
	"slices"
	"sync"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/media"
)

// Pic names one picture on the site: an arcade shot (Key = img stem, Slot
// title/snap/ingame) or a system photo (Key = core, Slot "system").
type Pic struct {
	Key, Slot string
}

type scaledKey struct {
	Pic
	Frame    int
	W, H     int
	Native   bool
	Stretch  bool
	Pixelate bool
}

func keyFor(r app.ImageReq) scaledKey {
	frame := 0
	if r.Slot == fetch.SlotVideo {
		frame = r.Frame
	}
	return scaledKey{Pic: Pic{r.Key, r.Slot}, Frame: frame, W: r.W, H: r.H, Native: r.Native, Stretch: r.Stretch, Pixelate: r.Pixelate}
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

	mu             sync.Mutex
	videoTiming    videoTiming
	cache          map[scaledKey]*entry // scaled bitmaps, LRU by bytes
	lru            *list.List           // front = most recent
	bytes          int
	budget         int
	raw            map[Pic]*image.RGBA // decoded originals of the rows in view
	rawOrder       []Pic
	clips          map[Pic]*media.Clip // compressed previews; decoded frames never live here or in raw
	clipOrder      []Pic               // most recently used last; at most two clips
	wanted         []scaledKey         // priority order, replaced on every Want
	inflight       map[scaledKey]bool
	futureBlocked  map[scaledKey]bool // capacity failures, retried only after demand changes
	waitingVideo   scaledKey
	waitingVideoOn bool
	netBusy        map[Pic]bool
	missing        map[Pic]time.Time // 404 seen at; forgotten after missingTTL
	missingDirty   bool
	missingSaved   time.Time
	failed         map[Pic]bool      // decode failed this run
	retryAt        map[Pic]time.Time // download failed: not before this
	retries        map[Pic]int
	offline        bool
	prefetch       []Pic // background download list, in priority order
	prefetchOn     bool
	prefetchPos    int
	prefetched     int
	kick           chan struct{}
	decodeKick     chan struct{}
	paused         bool // no decoding while the UI scrolls
	have           int  // prefetch pictures on disk (kept current, never rescanned per call)
	haveKnown      bool
	haveGen        int          // bumped per new prefetch list; a stale count is dropped
	got            map[Pic]bool // downloaded this run, so a count racing a download keeps it
	progressReady  chan struct{}
	ready          chan struct{}
	stop           chan struct{}
	ctx            context.Context // cancelled on Close
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// New opens a service over dir (created on demand). client may be nil.
func New(dir string, client *fetch.Client, lg *log.Logger, budget int) *Service {
	if budget <= 0 {
		budget = 24 << 20
	}
	s := &Service{
		dir: dir, client: client, lg: lg,
		cache: map[scaledKey]*entry{}, lru: list.New(), budget: budget,
		raw: map[Pic]*image.RGBA{}, clips: map[Pic]*media.Clip{}, inflight: map[scaledKey]bool{}, futureBlocked: map[scaledKey]bool{}, netBusy: map[Pic]bool{},
		missing: map[Pic]time.Time{}, failed: map[Pic]bool{}, retryAt: map[Pic]time.Time{}, retries: map[Pic]int{},
		got:  map[Pic]bool{},
		kick: make(chan struct{}, 1), decodeKick: make(chan struct{}, 1),
		progressReady: make(chan struct{}, 1), ready: make(chan struct{}, 1), stop: make(chan struct{}),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	for _, sub := range slotDirs {
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

// ProgressReady updates the Options tally without repainting unrelated screens.
func (s *Service) ProgressReady() <-chan struct{} { return s.progressReady }

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
	if changed && !off {
		s.prefetchPos = 0
	}
	s.mu.Unlock()
	if changed {
		s.poke(s.kick)
		s.signal()
	}
}

// SetPrefetch installs the background download list (priority order) and
// switches prefetching on or off. A picture two rows share is listed once,
// or the tally's total would count it twice while its download counts once.
func (s *Service) SetPrefetch(pics []Pic, on bool) {
	pics = uniquePics(pics)
	s.mu.Lock()
	rescan := !s.haveKnown || !samePics(pics, s.prefetch)
	s.prefetch = pics
	s.prefetchOn = on
	s.prefetchPos = 0
	gen := s.haveGen
	if rescan {
		s.haveGen++
		gen = s.haveGen
	}
	s.mu.Unlock()
	if rescan {
		go s.countHave(pics, gen)
	}
	s.poke(s.kick)
}

func uniquePics(pics []Pic) []Pic {
	seen := make(map[Pic]bool, len(pics))
	out := make([]Pic, 0, len(pics))
	for _, p := range pics {
		if validPic(p) && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func samePics(a, b []Pic) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// countHave counts the prefetch pictures already on disk, off the UI. A
// download that lands while it walks the list is counted from got, and the
// count for a list that has since been replaced is dropped.
func (s *Service) countHave(pics []Pic, gen int) {
	on := make([]bool, len(pics))
	for i, p := range pics {
		on[i] = s.exists(p)
	}
	s.mu.Lock()
	if gen != s.haveGen {
		s.mu.Unlock()
		return
	}
	n := 0
	for i, p := range pics {
		if on[i] || s.got[p] {
			n++
		}
	}
	s.have, s.haveKnown = n, true
	s.mu.Unlock()
	s.poke(s.progressReady)
}

// SetPaused stops the decoder (the UI is scrolling and wants both cores'
// worth of quiet); unpausing resumes with whatever is wanted now.
func (s *Service) SetPaused(p bool) {
	s.mu.Lock()
	changed := s.paused != p
	s.paused = p
	if changed {
		s.futureBlocked = map[scaledKey]bool{}
	}
	s.mu.Unlock()
	if !p || changed {
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
	if p.Slot == fetch.SlotVideo {
		return filepath.Join(s.dir, fetch.SlotVideo, p.Key+".mzv")
	}
	return filepath.Join(s.dir, p.Slot, p.Key+".png")
}

func (s *Service) exists(p Pic) bool {
	if !validPic(p) {
		return false
	}
	_, err := os.Stat(s.path(p))
	return err == nil
}

// Get implements app.Images.
func (s *Service) Get(req app.ImageReq) (*image.RGBA, app.ImageState) {
	k := keyFor(req)
	if !validPic(k.Pic) {
		return nil, app.ImageMissing
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k = s.normalizeKey(k)
	p := k.Pic
	if s.isMissing(p) || s.failed[p] {
		if p.Slot == fetch.SlotVideo {
			s.clearWaitingVideoLocked(k)
		}
		return nil, app.ImageMissing
	}
	if e, ok := s.cache[k]; ok {
		s.lru.MoveToFront(e.el)
		if p.Slot == fetch.SlotVideo {
			s.clearWaitingVideoLocked(k)
		}
		return e.img, app.ImageReady
	}
	if s.raw[p] == nil && !s.exists(p) {
		if s.client == nil || s.offline {
			if p.Slot == fetch.SlotVideo {
				s.clearWaitingVideoLocked(k)
			}
			return nil, app.ImageOffline
		}
	}
	if p.Slot == fetch.SlotVideo {
		s.waitingVideo, s.waitingVideoOn = k, true
	}
	return nil, app.ImageLoading
}

// Want implements app.Images: the priority list of what the screen needs.
func (s *Service) Want(reqs []app.ImageReq) {
	wanted := make([]scaledKey, 0, len(reqs))
	for _, r := range reqs {
		if k := keyFor(r); validPic(k.Pic) {
			wanted = append(wanted, k)
		}
	}
	s.mu.Lock()
	changed := !slices.Equal(s.wanted, wanted)
	if changed {
		s.futureBlocked = map[scaledKey]bool{}
	}
	s.wanted = wanted
	ready := false
	if changed && s.waitingVideoOn {
		waiting := s.normalizeKey(s.waitingVideo)
		if s.cache[waiting] != nil && s.isWantedLocked(waiting) {
			ready = true
		}
		s.waitingVideoOn = false
	}
	s.mu.Unlock()
	if ready {
		s.signal()
	}
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
	ordered := s.decodeOrderLocked()
	for _, k := range ordered {
		k = s.normalizeKey(k)
		if s.futureBlocked[k] {
			continue
		}
		if _, ok := s.cache[k]; ok || s.inflight[k] || s.isMissing(k.Pic) || s.failed[k.Pic] {
			continue
		}
		if s.raw[k.Pic] == nil && !s.exists(k.Pic) {
			continue
		}
		if k.Slot == fetch.SlotVideo && !s.isWantedLocked(k) && !s.canHoldVideoWindowLocked() {
			s.blockVideoWindowLocked()
			continue
		}
		s.inflight[k] = true
		return k, true
	}
	return scaledKey{}, false
}

// decodeOrderLocked prioritizes the visible video and its finite lookahead
// window, then visits all other screen requests in their original order.
func (s *Service) decodeOrderLocked() []scaledKey {
	current, found := s.firstVideoLocked()
	if !found {
		return append([]scaledKey(nil), s.wanted...)
	}
	ordered := make([]scaledKey, 0, len(s.wanted)+videoAheadFrames)
	ordered = append(ordered, current)
	ordered = append(ordered, s.videoWindowLocked()...)
	for _, k := range s.wanted {
		if s.normalizeKey(k) == current {
			continue
		}
		ordered = append(ordered, k)
	}
	return ordered
}

func (s *Service) firstVideoLocked() (scaledKey, bool) {
	for _, k := range s.wanted {
		if k.Slot == fetch.SlotVideo {
			return s.normalizeKey(k), true
		}
	}
	return scaledKey{}, false
}

// videoWindowLocked returns the next distinct frames of the visible clip.
// It waits for the visible frame and clip to be decoded before opening a window.
func (s *Service) videoWindowLocked() []scaledKey {
	current, ok := s.firstVideoLocked()
	if !ok || s.paused || s.clips[current.Pic] == nil || s.cache[current] == nil {
		return nil
	}
	clip := s.clips[current.Pic]
	seen := map[int]bool{current.Frame: true}
	frames := make([]scaledKey, 0, videoAheadFrames)
	for d := 1; d <= clip.Len() && len(frames) < videoAheadFrames; d++ {
		k := current
		k.Frame = clip.Normalize(current.Frame + d)
		if seen[k.Frame] {
			continue
		}
		seen[k.Frame] = true
		frames = append(frames, k)
	}
	return frames
}

func (s *Service) isVideoAheadLocked(k scaledKey) bool {
	for _, future := range s.videoWindowLocked() {
		if future == k {
			return true
		}
	}
	return false
}

func (s *Service) canHoldVideoWindowLocked() bool {
	current, ok := s.firstVideoLocked()
	if !ok {
		return false
	}
	frame := s.cache[current]
	if frame == nil {
		return false
	}
	needed := s.protectedBytesLocked()
	for range s.videoWindowLocked() {
		needed += frame.size
	}
	return needed <= s.budget
}

func (s *Service) blockVideoWindowLocked() {
	for _, k := range s.videoWindowLocked() {
		s.futureBlocked[k] = true
	}
}

func (s *Service) decodeLoop() {
	defer s.wg.Done()
	for {
		// Failed reads can leave work queued; stopping must not depend on
		// the decoder becoming idle.
		select {
		case <-s.stop:
			return
		default:
		}
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
	s.mu.Lock()
	eligible := s.decodeEligibleLocked(k)
	s.mu.Unlock()
	inflightKey := k
	defer func() {
		s.mu.Lock()
		delete(s.inflight, inflightKey)
		s.mu.Unlock()
	}()
	if !eligible {
		return
	}
	if k.Slot == fetch.SlotVideo {
		s.decodeVideo(k)
		return
	}
	s.mu.Lock()
	src := s.raw[k.Pic]
	s.mu.Unlock()
	if src == nil {
		var img image.Image
		if k.Slot == fetch.SlotFlyer {
			f, err := os.Open(s.path(k.Pic))
			if err != nil {
				s.failRead(k.Pic, err)
				return
			}
			img, err = media.ReadFlyer(f)
			f.Close()
			if err != nil {
				s.failCorrupt(k.Pic, err)
				return
			}
		} else {
			b, err := os.ReadFile(s.path(k.Pic))
			if err != nil {
				s.failRead(k.Pic, err)
				return
			}
			img, err = png.Decode(bytes.NewReader(b))
			if err != nil {
				s.failCorrupt(k.Pic, err)
				return
			}
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
	scaled := scaleFor(src, k)
	s.mu.Lock()
	s.put(k, scaled)
	s.mu.Unlock()
	s.signal()
}

func (s *Service) decodeVideo(k scaledKey) {
	s.mu.Lock()
	eligible := s.decodeEligibleLocked(k)
	clip := s.clips[k.Pic]
	if clip != nil {
		s.touchClip(k.Pic)
	}
	s.mu.Unlock()
	if !eligible {
		return
	}
	if clip == nil {
		f, err := os.Open(s.path(k.Pic))
		if err != nil {
			s.failRead(k.Pic, err)
			return
		}
		clip, err = media.Read(f)
		f.Close()
		if err != nil {
			s.failCorrupt(k.Pic, err)
			return
		}
		s.mu.Lock()
		s.rememberClip(k.Pic, clip)
		s.mu.Unlock()
	}
	k.Frame = clip.Normalize(k.Frame)
	timing := s.videoTimingEnabled()
	// Recheck immediately before JPEG decoding because the UI may have
	// replaced its wanted list while the clip was being read.
	s.mu.Lock()
	eligible = s.decodeEligibleLocked(k)
	_, cached := s.cache[k]
	s.mu.Unlock()
	if !eligible || cached {
		return
	}
	var decodeTime, convertTime, scaleTime time.Duration
	var started time.Time
	if timing {
		started = time.Now()
	}
	img, err := clip.Decode(k.Frame)
	if timing {
		decodeTime = time.Since(started)
	}
	if err != nil {
		s.failCorrupt(k.Pic, err)
		return
	}
	if timing {
		started = time.Now()
	}
	rgba := ToRGBA(img)
	if timing {
		convertTime = time.Since(started)
	}
	if timing {
		started = time.Now()
	}
	scaled := scaleFor(rgba, k)
	if timing {
		scaleTime = time.Since(started)
	}
	s.mu.Lock()
	current := s.isWantedLocked(k)
	future := !current && s.decodeEligibleLocked(k)
	if current {
		s.put(k, scaled)
	} else if future && s.canHoldVideoWindowLocked() && s.putFutureLocked(k, scaled) {
		// The request remains speculative until Want names this frame.
	} else {
		if future {
			s.blockVideoWindowLocked()
		}
		s.mu.Unlock()
		return
	}
	wakeWaiting := s.waitingVideoOn && s.normalizeKey(s.waitingVideo) == k
	s.mu.Unlock()
	if timing {
		s.recordVideoTiming(decodeTime, convertTime, scaleTime)
	}
	if current || wakeWaiting {
		s.signal()
	}
}

// putFutureLocked makes room only by evicting cache entries the current screen
// does not need. The caller bounds the full visible window against budget first.
func (s *Service) putFutureLocked(k scaledKey, img *image.RGBA) bool {
	need := s.bytes + len(img.Pix) - s.budget
	if need > 0 {
		var evict []*list.Element
		freed := 0
		for el := s.lru.Back(); el != nil && freed < need; el = el.Prev() {
			old := el.Value.(scaledKey)
			if old == k || s.isPinnedLocked(old) {
				continue
			}
			evict = append(evict, el)
			if e := s.cache[old]; e != nil {
				freed += e.size
			}
		}
		if freed < need {
			return false
		}
		for _, el := range evict {
			old := el.Value.(scaledKey)
			if e := s.cache[old]; e != nil {
				s.bytes -= e.size
				delete(s.cache, old)
			}
			s.lru.Remove(el)
		}
	}
	s.put(k, img)
	return true
}

func (s *Service) protectedBytesLocked() int {
	seen := map[scaledKey]bool{}
	total := 0
	for _, wanted := range s.wanted {
		wanted = s.normalizeKey(wanted)
		if seen[wanted] {
			continue
		}
		seen[wanted] = true
		if e := s.cache[wanted]; e != nil {
			total += e.size
		}
	}
	return total
}

func (s *Service) isWantedLocked(k scaledKey) bool {
	for _, w := range s.wanted {
		if s.normalizeKey(w) == k {
			return true
		}
	}
	return false
}

func (s *Service) clearWaitingVideoLocked(k scaledKey) {
	if s.waitingVideoOn && s.normalizeKey(s.waitingVideo) == k {
		s.waitingVideoOn = false
	}
}

func (s *Service) isPinnedLocked(k scaledKey) bool {
	if s.isWantedLocked(k) {
		return true
	}
	for _, future := range s.videoWindowLocked() {
		if future == k {
			return true
		}
	}
	return false
}

func (s *Service) decodeEligibleLocked(k scaledKey) bool {
	if s.paused {
		return false
	}
	if s.isWantedLocked(k) {
		return true
	}
	return k.Slot == fetch.SlotVideo && !s.futureBlocked[k] && s.isVideoAheadLocked(k) && s.canHoldVideoWindowLocked()
}

func (s *Service) failRead(p Pic, err error) {
	s.lg.Printf("images: read %s: %v (skipping this session)", s.path(p), err)
	s.mu.Lock()
	s.failed[p] = true
	s.mu.Unlock()
	s.signal()
}

func (s *Service) failCorrupt(p Pic, err error) {
	s.lg.Printf("images: %s: %v (deleting)", s.path(p), err)
	os.Remove(s.path(p))
	s.mu.Lock()
	delete(s.clips, p)
	for i, q := range s.clipOrder {
		if q == p {
			s.clipOrder = append(s.clipOrder[:i], s.clipOrder[i+1:]...)
			break
		}
	}
	s.failed[p] = true
	s.mu.Unlock()
	s.signal()
}

// normalizeKey is called with mu held. A clip's count is unavailable before
// its first read, so that first request remains unnormalized until decode.
func (s *Service) normalizeKey(k scaledKey) scaledKey {
	if clip := s.clips[k.Pic]; clip != nil {
		k.Frame = clip.Normalize(k.Frame)
		s.touchClip(k.Pic)
	}
	return k
}

func (s *Service) touchClip(p Pic) {
	for i, q := range s.clipOrder {
		if q == p {
			copy(s.clipOrder[i:], s.clipOrder[i+1:])
			s.clipOrder[len(s.clipOrder)-1] = p
			return
		}
	}
	s.clipOrder = append(s.clipOrder, p)
}

func (s *Service) rememberClip(p Pic, clip *media.Clip) {
	s.clips[p] = clip
	s.touchClip(p)
	for len(s.clipOrder) > 2 {
		old := s.clipOrder[0]
		s.clipOrder = s.clipOrder[1:]
		delete(s.clips, old)
	}
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
		if s.isMissing(p) || s.failed[p] || s.netBusy[p] || s.raw[p] != nil || s.exists(p) || now.Before(s.retryAt[p]) {
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
			if s.isMissing(p) || s.failed[p] || s.netBusy[p] || s.exists(p) || now.Before(s.retryAt[p]) {
				continue
			}
			s.netBusy[p] = true
			return p, true
		}
		// The first pass advances past failures. Revisit only delayed entries,
		// without stat-ing every successfully cached file on each idle wake.
		for _, p := range s.prefetch {
			at, retry := s.retryAt[p]
			if !retry || now.Before(at) || s.isMissing(p) || s.failed[p] || s.netBusy[p] {
				continue
			}
			if s.exists(p) {
				delete(s.retryAt, p)
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
		// A cancelled download remains wanted. Exit before selecting it
		// again, rather than waiting for the queue to become empty.
		select {
		case <-s.stop:
			return
		default:
		}
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
	if !validPic(p) {
		return
	}
	timeout := 10 * time.Second
	if p.Slot == fetch.SlotVideo {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()
	b, err := s.client.Image(ctx, p.Slot, p.Key)
	if s.ctx.Err() != nil {
		return // closing: not a failure worth remembering
	}
	if err != nil {
		s.mu.Lock()
		if errors.Is(err, fetch.ErrNotFound) {
			s.missing[p] = time.Now()
			s.missingDirty = true
			save := time.Since(s.missingSaved) > time.Minute
			if save {
				s.missingSaved = time.Now()
			}
			s.mu.Unlock()
			if save {
				s.saveMissing() // not only on Close: a crash must not forget a run's 404s
			}
			s.signal()
			return
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
		s.mu.Lock()
		s.backoff(p)
		s.mu.Unlock()
		s.lg.Printf("images: rename %s: %v", tmp, err)
		return
	}
	s.mu.Lock()
	delete(s.retryAt, p)
	delete(s.retries, p)
	s.prefetched++
	s.got[p] = true
	for _, q := range s.prefetch {
		if q == p {
			s.have++
			break
		}
	}
	s.mu.Unlock()
	s.poke(s.decodeKick)
	s.poke(s.progressReady)
}

// isMissing reports a remembered 404 (the caller holds mu).
func (s *Service) isMissing(p Pic) bool {
	_, ok := s.missing[p]
	return ok
}

func validPic(p Pic) bool {
	if p.Slot == fetch.SlotVideo || p.Slot == fetch.SlotFlyer {
		return media.ValidKey(p.Key)
	}
	return true
}

// slotDirs are the picture folders under the store directory.
var slotDirs = []string{"title", "snap", "ingame", "systems", fetch.SlotLocalSnap, fetch.SlotVideo, fetch.SlotFlyer}

// missingTTL is how long a 404 is remembered. The site's catalogue pictures
// rarely appear later, but the screenshot service fills in over time, so a
// miss is asked again after a week rather than never.
const missingTTL = 7 * 24 * time.Hour

const videoAheadFrames = 2

// missingFile is missing.json: version 2 keeps when each 404 was seen.
type missingFile struct {
	V       int              `json:"v"`
	Missing map[string]int64 `json:"missing"` // "slot/key" -> unix seconds
}

// missing.json remembers 404s so they are not retried every run. The first
// format was a flat array of "slot/key"; those entries are dated now, so
// they expire a week after the upgrade like any other.
func (s *Service) loadMissing() {
	b, err := os.ReadFile(filepath.Join(s.dir, "missing.json"))
	if err != nil {
		return
	}
	now := time.Now()
	entries := map[string]time.Time{}
	var keys []string
	var mf missingFile
	if json.Unmarshal(b, &keys) == nil {
		for _, k := range keys {
			entries[k] = now
		}
		s.missingDirty = true // rewrite in the dated format
	} else if json.Unmarshal(b, &mf) == nil && mf.V == 2 {
		for k, at := range mf.Missing {
			entries[k] = time.Unix(at, 0)
		}
	} else {
		return
	}
	for k, at := range entries {
		if now.Sub(at) > missingTTL {
			s.missingDirty = true // expired: the file shrinks on the next save
			continue
		}
		if i := indexByte(k, '/'); i > 0 {
			s.missing[Pic{Key: k[i+1:], Slot: k[:i]}] = at
		}
	}
}

func (s *Service) saveMissing() {
	s.mu.Lock()
	if !s.missingDirty {
		s.mu.Unlock()
		return
	}
	s.missingDirty = false
	mf := missingFile{V: 2, Missing: map[string]int64{}}
	for p, at := range s.missing {
		mf.Missing[p.Slot+"/"+p.Key] = at.Unix()
	}
	s.mu.Unlock()
	path := filepath.Join(s.dir, "missing.json")
	if len(mf.Missing) == 0 {
		os.Remove(path)
		return
	}
	b, _ := json.Marshal(mf)
	os.WriteFile(path, b, 0644)
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
	s.clips = map[Pic]*media.Clip{}
	s.clipOrder = nil
	s.missing = map[Pic]time.Time{}
	s.missingDirty = false
	s.failed = map[Pic]bool{}
	s.prefetchPos = 0
	s.have = 0
	s.got = map[Pic]bool{}
	s.mu.Unlock()
	s.signal()
	for _, sub := range slotDirs {
		os.RemoveAll(filepath.Join(s.dir, sub))
		os.MkdirAll(filepath.Join(s.dir, sub), 0755)
	}
	os.Remove(filepath.Join(s.dir, "missing.json"))
	s.poke(s.kick)
	s.signal()
}
