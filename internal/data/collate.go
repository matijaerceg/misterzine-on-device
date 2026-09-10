package data

import (
	"strings"
	"unicode"
)

// The site sorts with String.prototype.localeCompare(b, undefined,
// {numeric: true, sensitivity: 'base'}): case- and accent-insensitive, digit
// runs compared by value. Browsers implement that with ICU root collation,
// whose primary order is spaces, then punctuation, then symbols, then digits,
// then letters. This file reproduces that order for the characters that occur
// in release titles; the golden test in sort_test.go checks it against the
// browser's own result for every title in the data.

type elemClass uint8

const (
	clsSpace elemClass = iota
	clsPunct
	clsSymbol
	clsDigit
	clsLetter
)

// elem is one collation element: a class and, within it, an ordering value.
// Digit runs carry their value as a string without leading zeros.
type elem struct {
	cls elemClass
	val rune
	num string
}

// punctOrder is the ICU root order of ASCII punctuation, then symbols.
// Characters absent here fall into clsSymbol ordered by code point after
// everything listed, which keeps the sort total even for exotic titles.
const punctOrder = "_-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$"

var punctRank = func() map[rune]rune {
	m := map[rune]rune{}
	for i, c := range punctOrder {
		m[c] = rune(i)
	}
	return m
}()

// fold maps a rune to its base letters, lower-cased, ASCII where possible.
// Returns "" for characters that carry no primary weight on their own.
func fold(r rune) string {
	if r < 0x80 {
		if r >= 'A' && r <= 'Z' {
			return string(r + 32)
		}
		return string(r)
	}
	switch r {
	case 'À', 'Á', 'Â', 'Ã', 'Ä', 'Å', 'à', 'á', 'â', 'ã', 'ä', 'å', 'Ā', 'ā':
		return "a"
	case 'Æ', 'æ':
		return "ae"
	case 'Ç', 'ç':
		return "c"
	case 'È', 'É', 'Ê', 'Ë', 'è', 'é', 'ê', 'ë', 'Ē', 'ē':
		return "e"
	case 'Ì', 'Í', 'Î', 'Ï', 'ì', 'í', 'î', 'ï', 'Ī', 'ī':
		return "i"
	case 'Ñ', 'ñ':
		return "n"
	case 'Ò', 'Ó', 'Ô', 'Õ', 'Ö', 'Ø', 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'Ō', 'ō':
		return "o"
	case 'Œ', 'œ':
		return "oe"
	case 'Ù', 'Ú', 'Û', 'Ü', 'ù', 'ú', 'û', 'ü', 'Ū', 'ū':
		return "u"
	case 'Ý', 'ý', 'ÿ', 'Ÿ':
		return "y"
	case 'ß':
		return "ss"
	case 'Ð', 'ð':
		return "d"
	case 'Þ', 'þ':
		return "th"
	case 'Š', 'š':
		return "s"
	case 'Ž', 'ž':
		return "z"
	case 'Ł', 'ł':
		return "l"
	}
	return string(unicode.ToLower(r))
}

// Key builds the collation elements of s.
func Key(s string) []elem {
	var out []elem
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r >= 0xFF10 && r <= 0xFF19 { // fullwidth digits
			r = '0' + (r - 0xFF10)
			rs[i] = r
		}
		if unicode.Is(unicode.Mn, r) { // combining marks carry no primary weight
			continue
		}
		switch {
		case r >= '0' && r <= '9':
			j := i
			for j < len(rs) && rs[j] >= '0' && rs[j] <= '9' {
				j++
			}
			num := strings.TrimLeft(string(rs[i:j]), "0")
			if num == "" {
				num = "0"
			}
			out = append(out, elem{cls: clsDigit, num: num})
			i = j - 1
		case unicode.IsSpace(r):
			out = append(out, elem{cls: clsSpace})
		case r < 0x80 && !unicode.IsLetter(r):
			if rank, ok := punctRank[r]; ok {
				out = append(out, elem{cls: clsPunct, val: rank})
			} else {
				out = append(out, elem{cls: clsSymbol, val: r})
			}
		case unicode.IsLetter(r):
			for _, f := range fold(r) {
				out = append(out, elem{cls: clsLetter, val: f})
			}
		case unicode.IsDigit(r):
			out = append(out, elem{cls: clsDigit, num: string(r)})
		default:
			out = append(out, elem{cls: clsSymbol, val: r})
		}
	}
	return out
}

// CompareKeys orders two precomputed keys.
func CompareKeys(a, b []elem) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		x, y := a[i], b[i]
		if x.cls != y.cls {
			if x.cls < y.cls {
				return -1
			}
			return 1
		}
		if x.cls == clsDigit {
			if len(x.num) != len(y.num) {
				if len(x.num) < len(y.num) {
					return -1
				}
				return 1
			}
			if x.num != y.num {
				if x.num < y.num {
					return -1
				}
				return 1
			}
			continue
		}
		if x.val != y.val {
			if x.val < y.val {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// Compare orders two strings the way the site's localeCompare does.
func Compare(a, b string) int { return CompareKeys(Key(a), Key(b)) }

// TitleInitial groups titles using the same leading element as alphabetical
// sorting. Numbers, punctuation and empty titles share the initial # group.
func (d *Derived) TitleInitial() rune {
	if len(d.titleKey) > 0 && d.titleKey[0].cls == clsLetter {
		return d.titleKey[0].val
	}
	return '#'
}
