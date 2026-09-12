package data

import (
	"strings"
	"unicode"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
)

// CoreLabel mirrors the site's coreLabel(core): the CORE_NAMES entry, else
// the one title the core serves (sole is built by Ingest), else the rbf name
// with camelCase split and the first letter capitalised.
func CoreLabel(core string, sole map[string]string) string {
	if core == "" {
		return ""
	}
	if n, ok := gen.CoreNames[core]; ok {
		return n
	}
	if t, ok := sole[core]; ok {
		return t
	}
	var b strings.Builder
	rs := []rune(core)
	for i, r := range rs {
		if i > 0 && unicode.IsUpper(r) {
			p := rs[i-1]
			if (p >= 'a' && p <= 'z') || (p >= '0' && p <= '9') {
				b.WriteByte(' ')
			}
		}
		if i == 0 {
			r = unicode.ToUpper(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// SoleTitles mirrors the site's ingest(): core -> title for cores whose rows
// all share ONE title (a core with two distinct titles gets no entry).
func SoleTitles(rows []Row) map[string]string {
	seen := map[string]*string{}
	for i := range rows {
		r := &rows[i]
		if r.Core == "" {
			continue
		}
		if cur, ok := seen[r.Core]; !ok {
			t := r.Title
			seen[r.Core] = &t
		} else if cur != nil && *cur != r.Title {
			seen[r.Core] = nil
		}
	}
	out := map[string]string{}
	for c, t := range seen {
		if t != nil {
			out[c] = *t
		}
	}
	return out
}

// TypeLabel mirrors typeLabel(d): "Arcade" or "<base> core (deprecated)".
func TypeLabel(r *Row) string {
	if r.Base == "Arcade" {
		return "Arcade"
	}
	s := r.Base + " core"
	if r.Deprecated {
		s += " (deprecated)"
	}
	return s
}

// SrcFull is the site's display name for a source id, the raw id when unknown.
func SrcFull(src string) string {
	if n, ok := gen.SrcNames[src]; ok {
		return n
	}
	return src
}

// srcShort are the device's short source chips (the list is 53 columns wide).
var srcShort = map[string]string{
	"distribution_mister": "MiSTer",
	"jtbindb":             "Jotego",
	"coinop":              "Coin-Op",
	"meathax":             "Meathax",
	"rmcores":             "rmCores",
}

// SrcShort is the short source chip, falling back to the full name.
func SrcShort(src string) string {
	if s, ok := srcShort[src]; ok {
		return s
	}
	return SrcFull(src)
}

// ASCII transliterates a string to the printable ASCII the bitmap font has:
// accents folded, the pipeline's middot separator becomes " / ", anything
// else unrepresentable becomes "?". Copy rule: nothing fancier than ASCII.
func ASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '·' || r == '•': // middot, bullet
			b.WriteString(" / ")
		case r == '’' || r == '‘': // curly single quotes
			b.WriteByte('\'')
		case r == '“' || r == '”': // curly double quotes
			b.WriteByte('"')
		case r == '–' || r == '—': // en/em dash
			b.WriteByte('-')
		case r == '…':
			b.WriteString("...")
		case r == '°':
			b.WriteString(" deg")
		case r == '™':
			b.WriteString("(tm)")
		case r == '©':
			b.WriteString("(c)")
		case r == '®':
			b.WriteString("(r)")
		case r >= 0x20 && r < 0x7f:
			b.WriteRune(r)
		case unicode.IsLetter(r):
			f := fold(r)
			if f != "" && f[0] < 0x80 {
				if unicode.IsUpper(r) && len(f) == 1 {
					b.WriteString(strings.ToUpper(f))
				} else {
					b.WriteString(f)
				}
			} else {
				b.WriteByte('?')
			}
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		default:
			b.WriteByte('?')
		}
	}
	// collapse the double spaces a " / " replacement can leave behind
	out := b.String()
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}
