package setup

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestDownloadAndInstallWritesExactBytes(t *testing.T) {
	body := "exact artifact bytes"
	h := sha256.Sum256([]byte(body))
	a := Artifact{Name: "hermes", Version: "1.0.0", OS: "linux", Architecture: "amd64", Profile: Development, URL: "https://example.invalid/a", Size: int64(len(body)), SHA256: fmt.Sprintf("sha256:%x", h[:]), ContractVersion: 1, ExecutableMode: 0700}
	c := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}
	dir := t.TempDir()
	if err := DownloadAndInstall(context.Background(), a, Downloader{Client: c}, Installer{StageDir: dir, InstallDir: dir + "/installed"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dir + "/installed/hermes")
	if err != nil || string(got) != body {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestDownloadAndInstallRejectsDockerImageWithoutNetwork(t *testing.T) {
	calls := 0
	c := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("unexpected registry request")
	})}
	a := Artifact{Name: "hermes", Version: "1.0.0", OS: "linux", Architecture: "amd64", Profile: Development, Kind: ArtifactDockerImage, ImageRef: "ghcr.io/nousresearch/hermes-agent@sha256:" + strings.Repeat("c", 64), URL: "https://example.com/not-a-file", ContractVersion: 1, Ownership: "hermes"}
	if err := DownloadAndInstall(context.Background(), a, Downloader{Client: c}, Installer{StageDir: t.TempDir(), InstallDir: t.TempDir()}); !errors.Is(err, ErrInvalidArtifact) {
		t.Fatalf("docker image download error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("docker image triggered %d network calls", calls)
	}
}

func TestDownloaderRejectsUnsafeURLs(t *testing.T) {
	for _, u := range []string{"http://example.com/a", "https://user:pass@example.com/a", "https://example.com/a?q=secret", "https://example.com/a#x"} {
		if _, err := (Downloader{}).Open(context.Background(), u, 1); err != ErrDownloadURL {
			t.Errorf("%s: %v", u, err)
		}
	}
}

func TestDownloaderRejectsRedirectAndRedactsHTTPError(t *testing.T) {
	c := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusFound, Body: io.NopCloser(strings.NewReader("secret body")), Header: make(http.Header), Request: r}, nil
	})}
	_, err := (Downloader{Client: c}).Open(context.Background(), "https://example.com/a", 10)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("bad error: %v", err)
	}
}

func TestDownloaderRejectsOversizedStream(t *testing.T) {
	c := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("123456")), Header: make(http.Header), Request: r}, nil
	})}
	b, err := (Downloader{Client: c}).Open(context.Background(), "https://example.com/a", 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(b)
	b.Close()
	if err != ErrDownloadTooLarge {
		t.Fatalf("got %v", err)
	}
}

func TestDownloaderCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Downloader{Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })}}).Open(ctx, "https://example.com/a", 1)
	if err == nil {
		t.Fatal("expected cancellation")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
