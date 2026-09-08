// Package fetch talks to misterzine.fyi: the freshness check that mirrors
// the website (meta.json hash, then data.json keyed by that hash) and, later,
// the screenshot downloads. Every call has a bounded timeout; the UI never
// waits on any of it.
package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Site is the base URL of the release tracker.
const Site = "https://misterzine.fyi"

// ErrOffline wraps dial, DNS and TLS failures: the network is not there.
var ErrOffline = errors.New("offline")

// Client is a bounded HTTP client.
type Client struct {
	http *http.Client
	UA   string
}

// NewClient builds the client with the timeouts the plan fixes.
func NewClient(version string) *Client {
	tr := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxConnsPerHost:     2,
		IdleConnTimeout:     60 * time.Second,
		ForceAttemptHTTP2:   false,
	}
	return &Client{http: &http.Client{Transport: tr}, UA: "misterzine-on-device/" + version}
}

func (c *Client) get(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UA)
	req.Header.Set("Cache-Control", "no-store")
	resp, err := c.http.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) || errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrOffline, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, ErrNotFound
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
}

// ErrNotFound is a 404.
var ErrNotFound = errors.New("not found")

// Fresh is the result of a freshness check.
type Fresh struct {
	Changed bool
	Meta    data.Meta
	Rows    []data.Row
	RawData []byte // exact bytes of data.json, for the cache
	RawMeta []byte
}

// Check mirrors the site's check(): meta.json with a cache-buster, and when
// the hash differs from current, data.json keyed by that hash, verified by
// sha256 against the hash before it is accepted.
func (c *Client) Check(ctx context.Context, current string) (Fresh, error) {
	var out Fresh
	rawMeta, err := c.get(ctx, Site+"/releases/meta.json?t="+strconv.FormatInt(time.Now().UnixMilli(), 10), 6*time.Second)
	if err != nil {
		return out, err
	}
	meta, err := data.DecodeMeta(bytes.NewReader(rawMeta))
	if err != nil {
		return out, err
	}
	out.Meta = meta
	out.RawMeta = rawMeta
	if meta.Hash == "" || meta.Hash == current {
		return out, nil
	}
	raw, err := c.get(ctx, Site+"/releases/data.json?v="+meta.Hash, 20*time.Second)
	if err != nil {
		return out, err
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != meta.Hash {
		return out, fmt.Errorf("data.json hash %s does not match meta %s", got[:8], meta.Hash[:8])
	}
	rows, err := data.DecodeRows(bytes.NewReader(raw))
	if err != nil {
		return out, err
	}
	out.Changed = true
	out.Rows = rows
	out.RawData = raw
	return out, nil
}

// Image downloads one picture: images/<slot>/<key>.png or the system photo.
func (c *Client) Image(ctx context.Context, slot, key string) ([]byte, error) {
	url := Site + "/images/" + slot + "/" + key + ".png"
	if slot == "system" {
		url = Site + "/images/systems/" + key + ".png"
	}
	return c.get(ctx, url, 8*time.Second)
}
