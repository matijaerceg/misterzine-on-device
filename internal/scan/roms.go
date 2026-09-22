package scan

import (
	"archive/zip"
	"bufio"
	"crypto/md5"
	"encoding/hex"
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
// same-named file whose CRC differs, which stops the game only when the
// ROM's md5, rebuilt as Main builds it, no longer fits.
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
	md5s map[string]bool // md5Fits answers by section and zip stamps
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
		mras: map[string]mraEntry{}, zips: map[string]*zipIndex{}, md5s: map[string]bool{}}
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
		defer c.work.Unlock()
		// A zip replaced by another of the same size and time is rare, but
		// Start's answer has to hold, and its directory is quick to read
		// again. The parsed MRAs stay: some take a second to read.
		clear(c.zips)
		clear(c.md5s)
		res := c.run(rel)
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
	// published before Start may run, so an older answer never replaces
	// the one a Start check has just stored
	c.work.Lock()
	res := c.run(rel)
	c.mu.Lock()
	prev, known := c.results[rel]
	c.results[rel] = romEntry{time.Now(), res}
	delete(c.pending, rel)
	close(p.done)
	changed := known && prev.res != res || !known && p.late && res != ROMResult{}
	c.mu.Unlock()
	c.work.Unlock()
	if changed {
		select {
		case c.ready <- struct{}{}:
		default:
		}
	}
}

// romPart is one <part>: its file name inside the archive, the CRC the MRA
// gives it (0 for none) and the zips to try, in order, with the offset,
// length and repeat Main reads it with. A part without a name carries its
// bytes inline (data), read only when an md5 is rebuilt.
type romPart struct {
	name                   string
	crc                    uint32
	zips                   []string
	offset, length, repeat int
	data                   []byte
}

type romSection struct {
	index int // as Main reads it, with atoi: 0 when absent
	// realMD5: Main checks the assembled ROM against md5 and discards it on
	// a mismatch. With md5 None, none at all, or no zip on the <rom>, it
	// sends whatever it assembled, so a wrong file may still play.
	realMD5 bool
	md5     string
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
	root, rootMame := arcadeROMRoot(c.card)
	if rootMame {
		// Main takes a mame folder at the card's root before games/mame,
		// then loses the folder's path and looks for every zip under /mame
		// at the root of the filesystem: no zip is found, whatever the
		// folder holds (tried on a DE10). Say so once, not per zip.
		for _, s := range sections {
			for _, p := range s.parts {
				if p.name != "" {
					return ROMResult{"MiSTer can't load ROMs while " + filepath.ToSlash(filepath.Join(c.card, "mame")) + " exists", true}
				}
			}
		}
		return ROMResult{}
	}
	type found struct {
		pos int
		ROMResult
	}
	var issues []found
	// Main sends the index-0 sections in turn until one passes a real md5
	// check, which ends the list: one that fails its md5 is discarded, one
	// without an md5 is sent whatever it holds, and the core keeps the last
	// one sent. That one's problem is the game's; with none sent, the
	// first discarded one's.
	var last, discarded *found
	zeroDone := false
	for i, s := range sections {
		if s.index != 0 {
			if res := c.sectionIssue(root, rel, i, s); res.Text != "" {
				issues = append(issues, found{i, res})
			}
			continue
		}
		if zeroDone {
			continue
		}
		res := c.sectionIssue(root, rel, i, s)
		switch {
		case !s.realMD5:
			last = &found{i, res}
		case !res.Block:
			last, zeroDone = &found{i, res}, true
		case discarded == nil:
			discarded = &found{i, res}
		}
	}
	if last == nil {
		last = discarded
	}
	if last != nil && last.Text != "" {
		issues = append(issues, *last)
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
// the first that only warns. A file of the right name with another CRC
// blocks only when the section's md5 says the assembled ROM is not the one
// the MRA was made for: Main discards it then, and otherwise sends it. An
// MRA whose part CRCs went stale while its md5 still fits the files, as
// Galaxian (New Invasion) from HBMame does, loads and plays.
func (c *ROMCheck) sectionIssue(root, rel string, i int, s romSection) ROMResult {
	var warn ROMResult
	for _, p := range s.parts {
		if p.name == "" {
			continue
		}
		res, wrong := c.partIssue(root, p)
		if res.Text != "" && !wrong {
			return res
		}
		if wrong && warn.Text == "" {
			warn = res
		}
	}
	if warn.Text == "" || !s.realMD5 {
		return warn
	}
	if c.md5Fits(root, rel, i, s) {
		return ROMResult{}
	}
	warn.Block = true
	return warn
}

// zipPath finds where Main looks for part name in zip list entry z: it
// builds root/mame/<zip>/<part> (root/<zip>/<part> for a zip starting with
// "/") without trimming the entry, cuts that at the first ".zip" and looks
// the rest up inside the archive. ok is false when there is no ".zip" to
// cut at, which Main cannot open.
func zipPath(root, z, name string) (archive, member string, ok bool) {
	path := z + "/" + name
	cut := strings.Index(asciiLower(path), ".zip")
	if cut < 0 {
		return "", "", false
	}
	archive = filepath.Join(root, "mame", filepath.FromSlash(path[:cut+4]))
	if strings.HasPrefix(z, "/") {
		archive = filepath.Join(root, filepath.FromSlash(path[:cut+4]))
	}
	if cut+5 < len(path) {
		member = path[cut+5:]
	}
	return archive, member, true
}

// partIssue walks the part's zips as Main does and words what went wrong.
// A zip that is absent or unreadable is passed over; the first that has
// the part's CRC, or failing that its name, is the one Main loads from.
// wrong reports that the part was found only by a name whose CRC differs:
// Main loads it, and whether that stops the game is the section's md5 to
// say. Anything else with Text set means Main cannot load the part.
func (c *ROMCheck) partIssue(root string, p romPart) (res ROMResult, wrong bool) {
	var absent, unreadable, tried []string
	at := ""
	for _, z := range p.zips {
		label := strings.TrimSpace(z)
		tried = append(tried, label)
		archive, member, ok := zipPath(root, z, p.name)
		if !ok {
			unreadable = append(unreadable, label) // Main only opens zip archives
			continue
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
					return ROMResult{}, false
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
				return ROMResult{}, false
			}
			at = label
			break
		}
	}
	res = ROMResult{Block: at == ""}
	switch {
	case len(absent) > 0:
		res.Text = "Missing game ROM: " + strings.Join(absent, " or ")
	case len(unreadable) > 0:
		res.Text = "Unreadable ROM: " + strings.Join(unreadable, " or ")
	case at != "":
		res.Text = "Wrong ROM version: " + at + " (" + p.name + ")"
	default:
		res.Text = "Incomplete ROM: " + strings.Join(tried, " or ") + " (no " + p.name + ")"
	}
	return res, at != ""
}

// md5Fits rebuilds the md5 Main computes over a <rom>: every part's bytes
// in document order, as read (from offset, cut to length, repeat times),
// before interleaving or patches. Only a section with a stale part CRC
// gets here, so reading its files, and the MRA again for its inline bytes,
// is rare; the answer is kept while the MRA and the zips stay the same.
func (c *ROMCheck) md5Fits(root, rel string, i int, s romSection) bool {
	var stamps strings.Builder
	e := c.mras[rel]
	fmt.Fprintf(&stamps, "%s#%d:%d:%d", rel, i, e.size, e.mtime.UnixNano())
	for _, p := range s.parts {
		for _, z := range p.zips {
			if archive, _, ok := zipPath(root, z, p.name); ok {
				ix := c.zipIndex(archive)
				fmt.Fprintf(&stamps, "|%s:%d:%d", archive, ix.size, ix.mtime.UnixNano())
			}
		}
	}
	if fits, ok := c.md5s[stamps.String()]; ok {
		return fits
	}
	full, issue := parseROMSections(filepath.Join(c.card, filepath.FromSlash(rel)), true)
	if issue != "" || i >= len(full) {
		return false
	}
	h := md5.New()
	open := map[string]*zip.ReadCloser{}
	defer func() {
		for _, r := range open {
			r.Close()
		}
	}()
	for _, p := range full[i].parts {
		if p.name == "" {
			for range p.repeat {
				h.Write(p.data)
			}
			continue
		}
		f := c.partEntry(root, p, open)
		if f == nil {
			continue
		}
		for range p.repeat {
			if !hashRange(h, f, p.offset, p.length) {
				break
			}
		}
	}
	fits := strings.EqualFold(s.md5, hex.EncodeToString(h.Sum(nil)))
	c.md5s[stamps.String()] = fits
	return fits
}

// hashRange feeds the part's bytes to h as Main's rom_file reads them:
// from offset, at most length bytes when length is set, streamed rather
// than held, since a member can be many megabytes.
func hashRange(h io.Writer, f *zip.File, offset, length int) bool {
	rc, err := f.Open()
	if err != nil {
		return false
	}
	defer rc.Close()
	if offset > 0 {
		if _, err := io.CopyN(io.Discard, rc, int64(offset)); err != nil {
			return true // past the end: nothing left to read
		}
	}
	if length > 0 {
		io.CopyN(h, rc, int64(length))
	} else {
		io.Copy(h, rc)
	}
	return true
}

// partEntry returns the entry Main would load p from: in the first zip
// that has it, the first entry with its CRC, else its name. Nil when no
// zip has it; the part is reported missing then anyway.
func (c *ROMCheck) partEntry(root string, p romPart, open map[string]*zip.ReadCloser) *zip.File {
	for _, z := range p.zips {
		archive, member, ok := zipPath(root, z, p.name)
		if !ok {
			continue
		}
		r := open[archive]
		if r == nil {
			var err error
			if r, err = zip.OpenReader(archive); err != nil && !(errors.Is(err, zip.ErrInsecurePath) && r != nil) {
				continue
			}
			open[archive] = r
		}
		var entry *zip.File
		for _, f := range r.File {
			if p.crc != 0 && f.CRC32 == p.crc && !strings.HasSuffix(f.Name, "/") {
				entry = f
				break
			}
		}
		if entry == nil {
			for _, f := range r.File {
				if asciiLower(f.Name) == asciiLower(member) {
					entry = f
					break
				}
			}
		}
		if entry == nil || entry.Flags&(0x1|0x20|0x40) != 0 || entry.Method != zip.Store && entry.Method != zip.Deflate {
			continue
		}
		return entry
	}
	return nil
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
	sections, issue := parseROMSections(p, false)
	c.mras[rel] = mraEntry{st.Size(), st.ModTime(), sections, issue}
	return sections, issue
}

// parseROMSections reads an MRA's <rom> sections. Unnamed parts, whose
// inline bytes can run to megabytes, are kept only with inline, and only
// in a section whose md5 Main checks.
func parseROMSections(p string, inline bool) ([]romSection, string) {
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
	filling := -1 // the unnamed part whose hex text is being read
	var text strings.Builder
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
				v, _ := attr(el, "index")
				index := mainAtoi(v)
				zipList, _ = attr(el, "zip")
				sum, _ := attr(el, "md5")
				real := zipList != "" && sum != "" && !strings.EqualFold(sum, "none")
				sections = append(sections, romSection{index: index, realMD5: real, md5: sum})
				current = len(sections) - 1
			case "part":
				if current < 0 {
					continue
				}
				name, _ := attr(el, "name")
				v, _ := attr(el, "offset")
				offset := parseCUL(v)
				v, _ = attr(el, "length")
				length := parseCUL(v)
				repeat := 1
				if v, ok := attr(el, "repeat"); ok {
					repeat = parseCUL(v)
				}
				if name == "" {
					if inline && sections[current].realMD5 {
						sections[current].parts = append(sections[current].parts, romPart{repeat: repeat})
						filling = len(sections[current].parts) - 1
						text.Reset()
					}
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
				sections[current].parts = append(sections[current].parts, romPart{name: name, crc: parseCRC(crc), zips: zips,
					offset: offset, length: length, repeat: repeat})
			}
		case xml.CharData:
			if filling >= 0 {
				text.Write(el)
			}
		case xml.EndElement:
			switch strings.ToLower(el.Name.Local) {
			case "part":
				if filling >= 0 && current >= 0 {
					sections[current].parts[filling].data = mainHex(text.String())
				}
				filling = -1
			case "rom":
				current, filling = -1, -1
			}
		}
	}
	return sections, ""
}

// mainHex turns a part's inline text into bytes as Main's hexstr_to_char
// does: newlines, spaces, commas and tabs between pairs are skipped, any
// other character counts as a digit through (c%32+9)%25, and a lone last
// digit is a byte of its own.
func mainHex(s string) []byte {
	digit := func(c byte) int { return (int(c)%32 + 9) % 25 }
	var out []byte
	for i := 0; i < len(s); {
		for i < len(s) && (s[i] == '\n' || s[i] == '\r' || s[i] == ' ' || s[i] == ',' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		hi := digit(s[i])
		i++
		if i >= len(s) {
			out = append(out, byte(hi))
			break
		}
		out = append(out, byte(hi*16+digit(s[i])))
		i++
	}
	return out
}

// mainAtoi reads a <rom> index as Main does, with atoi: leading space, a
// sign, decimal digits up to the first that is not one, 0 for none, so
// "00" and a missing index are both 0.
func mainAtoi(v string) int {
	s := strings.TrimLeft(v, " \t\n\v\f\r")
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	n := 0
	for i := 0; i < len(s) && s[i] >= '0' && s[i] <= '9' && n < 1<<20; i++ {
		n = n*10 + int(s[i]-'0')
	}
	if neg {
		return -n
	}
	return n
}

// parseCUL reads offset, length and repeat as Main does, with strtoul(v,
// NULL, 0) into an int: 0x for hex, a leading 0 for octal, digits up to
// the first that is not one.
func parseCUL(v string) int {
	s := strings.TrimLeft(v, " \t\n\v\f\r")
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	base := 10
	switch {
	case len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') && hexDigit(s[2]) >= 0:
		base, s = 16, s[2:]
	case s != "" && s[0] == '0':
		base = 8
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		d := hexDigit(s[i])
		if d < 0 || d >= base {
			break
		}
		n = n*uint64(base) + uint64(d)
		if n > 0xffffffff {
			n = 0xffffffff
			break
		}
	}
	if neg {
		n = uint64(uint32(-n))
	}
	return int(int32(uint32(n)))
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
// rootMame reports a mame folder at the card's root, which Main takes and
// then cannot read from (see check).
func arcadeROMRoot(card string) (root string, rootMame bool) {
	var roots []string
	for i := 0; i < 6; i++ {
		roots = append(roots, filepath.Join(filepath.Dir(card), fmt.Sprintf("usb%d", i)))
	}
	roots = append(roots, filepath.Join(filepath.Dir(card), "network"), filepath.Join(card, "cifs"), card)
	for _, root := range roots {
		for _, p := range []string{root, filepath.Join(root, "games")} {
			if st, err := os.Stat(filepath.Join(p, "mame")); err == nil && st.IsDir() {
				return p, p == card
			}
		}
	}
	return filepath.Join(card, "_Arcade"), false
}
