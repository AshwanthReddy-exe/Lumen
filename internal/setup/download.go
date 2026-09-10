package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrDownloadURL      = errors.New("download URL is not allowed")
	ErrDownloadHTTP     = errors.New("download request failed")
	ErrDownloadTooLarge = errors.New("download exceeds declared size")
)

// Downloader performs a bounded, HTTPS-only artifact fetch. The caller owns
// closing the returned body and remains responsible for digest verification.
type Downloader struct {
	Client   *http.Client
	MaxBytes int64
}

func DownloadAndInstall(ctx context.Context, a Artifact, d Downloader, i Installer) error {
	b, err := d.Open(ctx, a.URL, a.Size)
	if err != nil {
		return err
	}
	defer b.Close()
	return i.Install(ctx, a, b)
}

func (d Downloader) Open(ctx context.Context, rawURL string, size int64) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, ErrDownloadURL
	}
	if size <= 0 || (d.MaxBytes > 0 && size > d.MaxBytes) {
		return nil, ErrDownloadTooLarge
	}
	c := d.Client
	if c == nil {
		c = http.DefaultClient
	}
	clone := *c
	clone.CheckRedirect = func(*http.Request, []*http.Request) error { return ErrDownloadHTTP }
	c = &clone
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ErrDownloadURL
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request unavailable", ErrDownloadHTTP)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: HTTP %d", ErrDownloadHTTP, resp.StatusCode)
	}
	if resp.ContentLength > size {
		resp.Body.Close()
		return nil, ErrDownloadTooLarge
	}
	return &structReadCloser{Reader: io.LimitReader(resp.Body, size+1), close: resp.Body.Close, size: size}, nil
}

type structReadCloser struct {
	io.Reader
	close func() error
	size  int64
	n     int64
}

func (r *structReadCloser) Close() error { return r.close() }
func (r *structReadCloser) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.n += int64(n)
	if r.n > r.size {
		return n, ErrDownloadTooLarge
	}
	return n, err
}

func validHTTPS(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && strings.EqualFold(u.Scheme, "https") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
