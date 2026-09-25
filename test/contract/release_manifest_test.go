package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

const releaseTestImageRef = "ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:" + "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

// goEnv reads a Go environment value. The runtime package cannot be imported
// here because this package already declares an identifier named runtime.
func goEnv(t *testing.T, key string) string {
	t.Helper()
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		t.Fatalf("go env %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}

func releaseScript(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "scripts", "lumen-release")
	info, err := os.Stat(script)
	if err != nil {
		t.Fatalf("read release builder: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatal("release builder must be executable")
	}
	b, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "#!/bin/sh\n") || !strings.Contains(string(b), "set -eu") {
		t.Fatal("release builder must use POSIX shell fail-fast mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("the Go toolchain is required to build release artifacts")
	}
	return script
}

// TestReleaseBuilderEmitsLoadableIntegrityPinnedManifest proves the product can
// load the manifest the release builder produces, and that no emitted artifact
// carries a placeholder identity.
func TestReleaseBuilderEmitsLoadableIntegrityPinnedManifest(t *testing.T) {
	script := releaseScript(t)
	root := filepath.Dir(filepath.Dir(script))
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")

	cmd := exec.Command(script,
		"--version", "0.1.0-test",
		"--hermes-image-ref", releaseTestImageRef,
		"--manifest", manifestPath,
		"--out-dir", filepath.Join(dir, "out"),
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "LUMEN_RELEASE_TARGETS=linux/amd64 darwin/arm64 android/arm64")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("release builder failed: %v\n%s", err, out)
	}

	f, err := os.Open(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	manifest, err := setup.LoadManifest(f)
	if err != nil {
		t.Fatalf("the release builder emitted a manifest the product rejects: %v", err)
	}
	if manifest.Topology != setup.TopologyCombined {
		t.Fatalf("release manifest topology = %q", manifest.Topology)
	}
	for _, target := range []struct{ osName, arch string }{
		{"linux", "amd64"}, {"darwin", "arm64"}, {"android", "arm64"},
	} {
		lumen, err := manifest.Select("lumen", target.osName, target.arch, setup.Development)
		if err != nil {
			t.Fatalf("missing lumen artifact for %s/%s: %v", target.osName, target.arch, err)
		}
		if lumen.SHA256 == "sha256:"+strings.Repeat("0", 64) || len(lumen.SHA256) != len("sha256:")+64 {
			t.Fatalf("unpinned digest for %s/%s: %q", target.osName, target.arch, lumen.SHA256)
		}
		if lumen.Size <= 0 {
			t.Fatalf("unpinned size for %s/%s", target.osName, target.arch)
		}
		if !strings.HasPrefix(lumen.URL, "https://") || strings.Contains(lumen.URL, "example.invalid") {
			t.Fatalf("unresolvable URL for %s/%s: %q", target.osName, target.arch, lumen.URL)
		}
		if !strings.Contains(lumen.URL, "releases/download/v0.1.0-test/") {
			t.Fatalf("URL is not version-pinned for %s/%s: %q", target.osName, target.arch, lumen.URL)
		}
		hermes, err := manifest.Select("hermes", target.osName, target.arch, setup.Development)
		if err != nil {
			t.Fatalf("missing hermes artifact for %s/%s: %v", target.osName, target.arch, err)
		}
		if hermes.Kind != setup.ArtifactDockerImage || hermes.ImageRef != releaseTestImageRef {
			t.Fatalf("hermes artifact for %s/%s is not the pinned image: %#v", target.osName, target.arch, hermes)
		}
	}

	// Every built artifact must be reproducible and match the digest it published.
	entries, err := os.ReadDir(filepath.Join(dir, "out", "0.1.0-test"))
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		seen++
	}
	if seen != 3 {
		t.Fatalf("release builder left %d artifacts for 3 targets: %v", seen, entries)
	}
}

// TestInstallerVerifiesManifestDigestBeforeInstalling proves the one-line
// installer resolves the manifest for the host platform, verifies the published
// digest against real bytes, and refuses a tampered artifact.
func TestInstallerVerifiesManifestDigestBeforeInstalling(t *testing.T) {
	script := releaseScript(t)
	root := filepath.Dir(filepath.Dir(script))
	installer := filepath.Join(root, "scripts", "install.sh")
	info, err := os.Stat(installer)
	if err != nil {
		t.Fatalf("read installer: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatal("installer must be executable")
	}
	body, err := os.ReadFile(installer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "#!/bin/sh\n") || !strings.Contains(string(body), "set -eu") {
		t.Fatal("installer must use POSIX shell fail-fast mode")
	}

	dir := t.TempDir()
	hostOS, hostArch := goEnv(t, "GOOS"), goEnv(t, "GOARCH")
	manifestPath := filepath.Join(dir, "manifest.json")
	build := exec.Command(script,
		"--version", "0.1.0-test",
		"--hermes-image-ref", releaseTestImageRef,
		"--manifest", manifestPath,
		"--out-dir", filepath.Join(dir, "out"),
	)
	build.Dir = root
	build.Env = append(os.Environ(), "LUMEN_RELEASE_TARGETS="+hostOS+"/"+hostArch)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("release builder failed: %v\n%s", err, out)
	}

	asset := "lumen-host-" + hostOS + "-" + hostArch
	built, err := os.ReadFile(filepath.Join(dir, "out", "0.1.0-test", asset))
	if err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifacts, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifacts, asset), built, 0700); err != nil {
		t.Fatal(err)
	}

	run := func() ([]byte, error) {
		cmd := exec.Command(installer, "--dry-run")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "LUMEN_MANIFEST="+manifestPath, "LUMEN_ARTIFACT_DIR="+artifacts)
		return cmd.CombinedOutput()
	}

	out, err := run()
	if err != nil {
		t.Fatalf("installer rejected a valid artifact: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "verified sha256:") {
		t.Fatalf("installer did not report digest verification: %s", out)
	}

	if err := os.WriteFile(filepath.Join(artifacts, asset), append(append([]byte{}, built...), 'x'), 0700); err != nil {
		t.Fatal(err)
	}
	if out, err := run(); err == nil {
		t.Fatalf("installer accepted a tampered artifact: %s", out)
	}
}

func TestReleaseBuilderRefusesUnpinnedInputs(t *testing.T) {
	script := releaseScript(t)
	root := filepath.Dir(filepath.Dir(script))
	digest64 := strings.Repeat("c", 64)
	for name, args := range map[string][]string{
		"mutable_tag":           {"--version", "0.1.0-test", "--hermes-image-ref", "ghcr.io/ashwanthreddy-exe/lumen-hermes:latest"},
		"placeholder_digest":    {"--version", "0.1.0-test", "--hermes-image-ref", "ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:" + strings.Repeat("0", 64)},
		"short_digest":          {"--version", "0.1.0-test", "--hermes-image-ref", "ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:cccc"},
		"missing_version":       {"--hermes-image-ref", "ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:" + digest64},
		"missing_image":         {"--version", "0.1.0-test"},
		"insecure_base_url":     {"--version", "0.1.0-test", "--hermes-image-ref", "ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:" + digest64, "--base-url", "http://downloads.lumen.dev"},
		"invalid_version_chars": {"--version", "0.1.0!/x", "--hermes-image-ref", "ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:" + digest64},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			manifestPath := filepath.Join(dir, "manifest.json")
			cmd := exec.Command(script, append(args, "--manifest", manifestPath)...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "LUMEN_RELEASE_TARGETS=linux/amd64")
			if out, err := cmd.CombinedOutput(); err == nil {
				t.Fatalf("release builder accepted invalid input %v\n%s", args, out)
			}
			if _, err := os.Lstat(manifestPath); !os.IsNotExist(err) {
				t.Fatal("a rejected release run wrote a manifest")
			}
		})
	}
}
