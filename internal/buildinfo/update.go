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

// The two Downloader databases MisterZine's one entry, [misterzine], can
// follow: the free releases, and MisterZine Arcade, the members' beta,
// which the misterzine-arcade-betas repository serves (it holds built
// files only; the members' release in misterzine-arcade-private writes
// it). deploy/channel.py, deploy/downloader_misterzine.ini and
// tools/verify_package.py name the same two (urls_test.go).
const (
	FreeDBURL = "https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip"
	BetaDBURL = "https://raw.githubusercontent.com/matijaerceg/misterzine-arcade-betas/main/beta.json.zip"
)

// BetaCatalogueURL names the newest MisterZine Arcade beta. The members'
// release writes it beside the beta database, BetaDBURL, in the same
// commit, so a beta build announces exactly the update a member's
// misterzine entry brings.
const BetaCatalogueURL = "https://raw.githubusercontent.com/matijaerceg/misterzine-arcade-betas/main/catalogue.json"

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

// betaParts reads vX.Y.Z-beta.N, and vX.Y.Z as the release after all of
// its betas; anything else (a candidate, a -dev build) is not on the track.
func betaParts(s string) ([4]int, bool) {
	var v [4]int
	base, beta, isBeta := strings.Cut(s, "-beta.")
	stable, ok := stableParts(base)
	if !ok {
		return v, false
	}
	copy(v[:], stable[:])
	v[3] = int(^uint(0) >> 1)
	if isBeta {
		if beta == "" || strings.Trim(beta, "0123456789") != "" {
			return v, false
		}
		n, err := strconv.Atoi(beta)
		if err != nil {
			return v, false
		}
		v[3] = n
	}
	return v, true
}

// NewerBeta reports whether candidate, a beta, follows the running build
// on the beta track. A free release there, vX.Y.Z, follows its betas, so a
// beta build of that version has nothing newer until vX.Y.Z+1-beta.1.
func NewerBeta(candidate, current string) bool {
	if !strings.Contains(candidate, "-beta.") {
		return false
	}
	a, ok := betaParts(candidate)
	if !ok {
		return false
	}
	b, ok := betaParts(current)
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

// CheckBetaUpdate is CheckUpdate for a beta build: it reads the beta
// catalogue and returns its version when that is newer than current. A
// newer free release is not offered: the member's misterzine entry follows
// the beta database, which only a beta release moves, and every free
// release is followed by a beta built from it or later.
func CheckBetaUpdate(ctx context.Context, client *http.Client, url, current string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MisterZine")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("beta check: HTTP %d", resp.StatusCode)
	}
	var catalogue struct {
		Schema   int `json:"schema"`
		Releases struct {
			Beta struct {
				Version string `json:"version"`
			} `json:"beta"`
		} `json:"releases"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&catalogue); err != nil {
		return "", err
	}
	if catalogue.Schema != 1 {
		return "", fmt.Errorf("beta check: catalogue schema %d", catalogue.Schema)
	}
	if v := catalogue.Releases.Beta.Version; NewerBeta(v, current) {
		return v, nil
	}
	return "", nil
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
