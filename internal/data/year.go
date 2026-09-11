package data

import "strings"

// ReleaseYear deliberately leaves ranges and uncertain years in Unknown.
func ReleaseYear(raw string) string {
	y := strings.TrimSpace(raw)
	if len(y) != 4 || y[0] < '1' || y[0] > '9' {
		return ""
	}
	for _, c := range y {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return y
}

func Decade(year string) string {
	if y := ReleaseYear(year); y != "" {
		return y[:3] + "0s"
	}
	return ""
}
