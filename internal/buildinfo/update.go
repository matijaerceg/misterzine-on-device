package buildinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const LatestReleaseURL = "https://api.github.com/repos/matijaerceg/misterzine-on-device/releases/latest"

// stableParts rejects development/prerelease tags and malformed remote data.
func stableParts(s string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		if p == "" {
			return v, false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return v, false
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func NewerStable(candidate, current string) bool {
	a, ok := stableParts(candidate)
	if !ok {
		return false
	}
	// Development builds deliberately do not advertise a stable downgrade.
	b, ok := stableParts(current)
	if !ok {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// CheckUpdate only reads release metadata; it never installs anything.
func CheckUpdate(ctx context.Context, client *http.Client, url, current string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MisterZine")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release check: HTTP %d", resp.StatusCode)
	}
	var release struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&release); err != nil {
		return "", err
	}
	if !release.Draft && !release.Prerelease && NewerStable(release.Tag, current) {
		return release.Tag, nil
	}
	return "", nil
}
