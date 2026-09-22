package scan

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ROMResult is the verdict on one MRA's ROM files. Text is empty when
// nothing is wrong. Block says MiSTer cannot load the game as it stands;
// without it Text is a warning and Start still hands the game over.
type ROMResult struct {
	Text  string
	Block bool
}

// ROMCheck looks inside the ROM archives an MRA names, the way MiSTer's
// Main loads them (support/arcade/mra_loader.cpp, file_io.cpp): each
// <part> is looked for in its zips in order, by CRC first and then by
// name, and the first zip that yields it wins. It cannot know whether the
// assembled ROM runs; it only reports what Main would fail to find, and a
// same-named file whose CRC differs.
//
// Details asks without fresh: a result under three seconds old is reused,
// and an older one is returned while a worker checks again, so the drawing
// loop never waits on the card for long. Ready fires when a worker's answer
// is worth a repaint. Start asks with fresh and waits for the answer.
type ROMCheck struct {
	card  string
	ready chan struct{}
	wait  time.Duration              // how long a first Details check waits
	run   func(rel string) ROMResult // the check itself; a test may replace it

	mu      sync.Mutex // guards results and pending
	results map[string]romEntry
	pending map[string]*romPending

	work sync.Mutex // serialises checks and guards the caches below
	mras map[string]mraEntry
	zips map[string]*zipIndex
}

type romEntry struct {
	at  time.Time
	res ROMResult
}

type romPending struct {
	done chan struct{}
	late bool // Details gave up waiting and drew no line
}

// NewROMCheck returns the checker for one card.
func NewROMCheck(card string) *ROMCheck {
	c := &ROMCheck{card: card, ready: make(chan struct{}, 1), wait: 8 * time.Millisecond,
		results: map[string]romEntry{}, pending: map[string]*romPending{},
		mras: map[string]mraEntry{}, zips: map[string]*zipIndex{}}
	c.run = c.check
	return c
}

// Ready delivers a signal when a background check changed what Details
// should show. A nil checker never signals.
func (c *ROMCheck) Ready() <-chan struct{} {
	if c == nil {
		return nil
	}
	return c.ready
}

// Check reports on the MRA at rel (card-relative). Files other than MRAs
// have no ROM requirements.
func (c *ROMCheck) Check(rel string, fresh bool) ROMResult {
	if !strings.EqualFold(filepath.Ext(rel), ".mra") {
		return ROMResult{}
	}
	if fresh {
		c.work.Lock()
		res := c.run(rel)
		c.work.Unlock()
		c.mu.Lock()
		c.results[rel] = romEntry{time.Now(), res}
		c.mu.Unlock()
		return res
	}
	c.mu.Lock()
	e, known := c.results[rel]
	if known && time.Since(e.at) < 3*time.Second {
		c.mu.Unlock()
		return e.res
	}
	p := c.pending[rel]
	if p == nil {
		p = &romPending{done: make(chan struct{})}
		c.pending[rel] = p
		go c.refresh(rel, p)
	}
	c.mu.Unlock()
	if known {
		return e.res
	}
	select {
	case <-p.done:
	case <-time.After(c.wait):
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.results[rel]; ok {
		return e.res
	}
	p.late = true
	return ROMResult{}
}

func (c *ROMCheck) refresh(rel string, p *romPending) {
	c.work.Lock()
	res := c.run(rel)
	c.work.Unlock()
	c.mu.Lock()
	prev, known := c.results[rel]
	c.results[rel] = romEntry{time.Now(), res}
	delete(c.pending, rel)
	close(p.done)
	changed := known && prev.res != res || !known && p.late && res != ROMResult{}
	c.mu.Unlock()
	if changed {
		select {
		case c.ready <- struct{}{}:
		default:
		}
	}
}

// romPart is one named <part>: its file name inside the archive, the CRC
// the MRA gives it (0 for none) and the zips to try, in order.
type romPart struct {
	name string
	crc  uint32
	zips []string
}

type romSection struct {
	index string
	// realMD5: Main checks the assembled ROM against an md5 and discards it
	// on a mismatch. With md5 None, none at all, or no zip on the <rom>,
	// it sends whatever it assembled, so a wrong file may still play.
	realMD5 bool
	parts   []romPart
}

type mraEntry struct {
	size     int64
	mtime    time.Time
	sections []romSection
	issue    string
}

func (c *ROMCheck) check(rel string) ROMResult {
	sections, issue := c.requirements(rel)
	if issue != "" {
		return ROMResult{issue, true}
	}
	root := arcadeROMRoot(c.card)
	type found struct {
		pos int
		ROMResult
	}
	var issues []found
	// Multiple index-0 sections are alternative ROM layouts in Main. One
	// that resolves cleanly suffices, then one with only a warning;
	// otherwise the first layout's problem stands.
	zeroPos, zeroClean := -1, false
	var zeroWarn, zeroBlock *ROMResult
	for i, s := range sections {
		res := c.sectionIssue(root, s)
		if s.index != "0" {
			if res.Text != "" {
				issues = append(issues, found{i, res})
			}
			continue
		}
		if zeroPos < 0 {
			zeroPos = i
		}
		switch {
		case res.Text == "":
			zeroClean = true
		case !res.Block && zeroWarn == nil:
			zeroWarn = &res
		case res.Block && zeroBlock == nil:
			zeroBlock = &res
		}
	}
	if !zeroClean {
		if zeroWarn != nil {
			issues = append(issues, found{zeroPos, *zeroWarn})
		} else if zeroBlock != nil {
			issues = append(issues, found{zeroPos, *zeroBlock})
		}
	}
	// what stops the game first, then warnings, each in document order
	var pick *found
	for i := range issues {
		f := &issues[i]
		if pick == nil || f.Block && !pick.Block || f.Block == pick.Block && f.pos < pick.pos {
			pick = f
		}
	}
	if pick == nil {
		return ROMResult{}
	}
	return pick.ROMResult
}

// sectionIssue returns the first part of s that stops the game, or else
// the first that only warns.
func (c *ROMCheck) sectionIssue(root string, s romSection) ROMResult {
	var warn ROMResult
	for _, p := range s.parts {
		res := c.partIssue(root, p, s.realMD5)
		if res.Block {
			return res
		}
		if res.Text != "" && warn.Text == "" {
			warn = res
		}
	}
	return warn
}

// partIssue walks the part's zips as Main does and words what went wrong.
// Main builds root/mame/<zip>/<part> (root/<zip>/<part> for a zip starting
// with "/") without trimming the zip, cuts that at the first ".zip", and
// looks the rest up inside the archive. A zip that is absent or unreadable
// is passed over; the first that has the part's CRC, or failing that its
// name, is the one Main loads from.
func (c *ROMCheck) partIssue(root string, p romPart, realMD5 bool) ROMResult {
	var absent, unreadable, tried []string
	wrong := ""
	for _, z := range p.zips {
		label := strings.TrimSpace(z)
		tried = append(tried, label)
		path := z + "/" + p.name
		cut := strings.Index(asciiLower(path), ".zip")
		if cut < 0 {
			unreadable = append(unreadable, label) // Main only opens zip archives
			continue
		}
		archive := filepath.Join(root, "mame", filepath.FromSlash(path[:cut+4]))
		if strings.HasPrefix(z, "/") {
			archive = filepath.Join(root, filepath.FromSlash(path[:cut+4]))
		}
		member := ""
		if cut+5 < len(path) {
			member = path[cut+5:]
		}
		ix := c.zipIndex(archive)
		if ix.absent {
			absent = append(absent, label)
			continue
		}
		if ix.unreadable {
			unreadable = append(unreadable, label)
			continue
		}
		if p.crc != 0 {
			if ok, hit := ix.crcs[p.crc]; hit {
				if ok {
					return ROMResult{}
				}
				// Main does not fall back to the name when it cannot
				// extract the entry its CRC found
				unreadable = append(unreadable, label)
				continue
			}
		}
		if ok, hit := ix.names[asciiLower(member)]; hit {
			if !ok {
				unreadable = append(unreadable, label)
				continue
			}
			if p.crc == 0 {
				return ROMResult{}
			}
			wrong = label
			break
		}
	}
	res := ROMResult{Block: wrong == "" || realMD5}
	switch {
	case len(absent) > 0:
		res.Text = "Missing game ROM: " + strings.Join(absent, " or ")
	case len(unreadable) > 0:
		res.Text = "Unreadable ROM: " + strings.Join(unreadable, " or ")
	case wrong != "":
		res.Text = "Wrong ROM version: " + wrong + " (" + p.name + ")"
	default:
		res.Text = "Incomplete ROM: " + strings.Join(tried, " or ") + " (no " + p.name + ")"
	}
	return res
}

// requirements returns the MRA's ROM sections, parsed again only when the
// file changes: some MRAs carry megabytes of inline data.
func (c *ROMCheck) requirements(rel string) ([]romSection, string) {
	p := filepath.Join(c.card, filepath.FromSlash(rel))
	st, err := os.Stat(p)
	if err != nil {
		return nil, "Cannot read game menu file"
	}
	if e, ok := c.mras[rel]; ok && e.size == st.Size() && e.mtime.Equal(st.ModTime()) {
		return e.sections, e.issue
	}
	sections, issue := parseROMSections(p)
	c.mras[rel] = mraEntry{st.Size(), st.ModTime(), sections, issue}
	return sections, issue
}

func parseROMSections(p string) ([]romSection, string) {
	f, err := os.Open(p)
	if err != nil {
		return nil, "Cannot read game menu file"
	}
	defer f.Close()
	dec := xml.NewDecoder(&commentStripper{br: bufio.NewReader(f)})
	dec.Strict = false
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	var sections []romSection
	current := -1
	zipList := ""
	attr := func(el xml.StartElement, key string) (string, bool) {
		for _, a := range el.Attr {
			if strings.EqualFold(a.Name.Local, key) {
				return a.Value, true
			}
		}
		return "", false
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "Cannot read game ROM requirements"
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch strings.ToLower(el.Name.Local) {
			case "rom":
				index, _ := attr(el, "index")
				zipList, _ = attr(el, "zip")
				md5, _ := attr(el, "md5")
				real := zipList != "" && md5 != "" && !strings.EqualFold(md5, "none")
				sections = append(sections, romSection{index: index, realMD5: real})
				current = len(sections) - 1
			case "part":
				name, _ := attr(el, "name")
				if current < 0 || name == "" {
					continue
				}
				// a part's own zip replaces the <rom>'s list
				z, ok := attr(el, "zip")
				if !ok || z == "" {
					z = zipList
				}
				var zips []string
				for _, t := range strings.Split(z, "|") {
					// an empty entry never yields a file in Main either
					if strings.TrimSpace(t) != "" {
						zips = append(zips, t)
					}
				}
				// A named part with no zip at all cannot be found by Main;
				// such MRAs are left alone as before rather than blocked.
				if len(zips) == 0 {
					continue
				}
				crc, _ := attr(el, "crc")
				sections[current].parts = append(sections[current].parts, romPart{name: name, crc: parseCRC(crc), zips: zips})
			}
		case xml.EndElement:
			if strings.EqualFold(el.Name.Local, "rom") {
				current = -1
			}
		}
	}
	return sections, ""
}

// parseCRC reads a crc attribute the way Main does, with strtoul(v, NULL,
// 16) into a 32-bit value: leading space, a sign and a 0x prefix allowed,
// digits up to the first character that is not one, 0 when there are none.
func parseCRC(v string) uint32 {
	s := strings.TrimLeft(v, " \t\n\v\f\r")
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') && hexDigit(s[2]) >= 0 {
		s = s[2:]
	}
	var n uint64
	for i := 0; i < len(s) && hexDigit(s[i]) >= 0; i++ {
		n = n*16 + uint64(hexDigit(s[i]))
		if n > 0xffffffff {
			return 0xffffffff // ULONG_MAX on the boards
		}
	}
	if neg {
		return uint32(-n)
	}
	return uint32(n)
}

func hexDigit(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	}
	return -1
}

// asciiLower folds case as miniz does when Main looks a name up in a zip:
// A-Z only.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// zipIndex is what Main can see of one archive: for each CRC, whether the
// first entry carrying it can be extracted, and the same for each name
// (folded to lower case). Main extracts stored and deflated entries only,
// and none that are encrypted or patched.
type zipIndex struct {
	size       int64
	mtime      time.Time
	absent     bool
	unreadable bool
	crcs       map[uint32]bool
	names      map[string]bool
}

// zipIndex returns the archive's index, read again only when its size or
// time changes. Only the zip's directory is read.
func (c *ROMCheck) zipIndex(path string) *zipIndex {
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &zipIndex{absent: true}
		}
		return &zipIndex{unreadable: true}
	}
	if ix, ok := c.zips[path]; ok && ix.size == st.Size() && ix.mtime.Equal(st.ModTime()) {
		return ix
	}
	ix := &zipIndex{size: st.Size(), mtime: st.ModTime()}
	c.zips[path] = ix
	if !st.Mode().IsRegular() {
		ix.unreadable = true // Main cannot read a folder in place of an archive
		return ix
	}
	r, err := zip.OpenReader(path)
	if err != nil && !(errors.Is(err, zip.ErrInsecurePath) && r != nil) {
		ix.unreadable = true
		return ix
	}
	defer r.Close()
	ix.crcs = make(map[uint32]bool, len(r.File))
	ix.names = make(map[string]bool, len(r.File))
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		ok := f.Flags&(0x1|0x20|0x40) == 0 && (f.Method == zip.Store || f.Method == zip.Deflate)
		if _, seen := ix.crcs[f.CRC32]; !seen {
			ix.crcs[f.CRC32] = ok
		}
		name := asciiLower(f.Name)
		if _, seen := ix.names[name]; !seen {
			ix.names[name] = ok
		}
	}
	return ix
}

// arcadeROMRoot follows Main's findGamesDir precedence. Main selects one
// directory; it does not combine archives from different storage devices.
func arcadeROMRoot(card string) string {
	var roots []string
	for i := 0; i < 6; i++ {
		roots = append(roots, filepath.Join(filepath.Dir(card), fmt.Sprintf("usb%d", i)))
	}
	roots = append(roots, filepath.Join(filepath.Dir(card), "network"), filepath.Join(card, "cifs"), card)
	for _, root := range roots {
		for _, p := range []string{root, filepath.Join(root, "games")} {
			if st, err := os.Stat(filepath.Join(p, "mame")); err == nil && st.IsDir() {
				return p
			}
		}
	}
	return filepath.Join(card, "_Arcade")
}
