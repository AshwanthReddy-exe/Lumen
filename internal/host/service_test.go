package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
)

func TestConfigFromEnvironment(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SOCKET_PATH", filepath.Join(d, "host.sock"))
	t.Setenv("LUMEN_OPERATOR_CREDENTIAL_FILE", filepath.Join(d, "operator"))
	t.Setenv("LUMEN_HERMES_BASE_URL", "https://hermes.example.test")
	t.Setenv("LUMEN_HERMES_PROFILE", "hardened")
	t.Setenv("LUMEN_HERMES_BEARER_FILE", filepath.Join(d, "hermes.token"))
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != d {
		t.Fatalf("data dir %q", c.DataDir)
	}
	if c.HermesProfile != "hardened" || c.HermesBaseURL != "https://hermes.example.test" || c.HermesBearerPath == "" {
		t.Fatalf("Hermes config not loaded: %#v", c)
	}
}

func TestProductionNewBuildsNonNilRuntimeAdapter(t *testing.T) {
	d := t.TempDir()
	tokenPath := filepath.Join(d, "hermes.token")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator"), HermesBaseURL: "http://127.0.0.1:1", HermesProfile: "development", HermesBearerPath: tokenPath}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	s, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	if s.executor == nil || s.executor.runtime == nil {
		t.Fatal("production constructor left Hermes runtime nil")
	}
}

func TestProductionNewFailsReadinessForInvalidHermesConfiguration(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator"), HermesProfile: hermes.ProfileHardened}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	if _, err := New(c); err == nil {
		t.Fatal("expected invalid Hermes configuration to fail readiness")
	}
}

func TestHermesSecretReadRejectsSymlink(t *testing.T) {
	d := t.TempDir()
	realPath := filepath.Join(d, "real.token")
	linkPath := filepath.Join(d, "link.token")
	if err := os.WriteFile(realPath, []byte("token"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatal(err)
	}
	_, err := buildHermesClient(Config{HermesBaseURL: "http://127.0.0.1:1", HermesProfile: hermes.ProfileDevelopment, HermesBearerPath: linkPath})
	if err == nil {
		t.Fatal("expected symlinked Hermes secret to be rejected")
	}
}

func TestInitIsCreateOnly(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err == nil {
		t.Fatal("expected create-only failure")
	}
	if _, err := os.Stat(c.CredentialPath); err != nil {
		t.Fatal(err)
	}
}

func TestInitCanRetryAfterPostStoreFailure(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := control.WriteCredential(c.CredentialPath, make([]byte, control.CredentialSize)); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err == nil {
		t.Fatal("expected credential creation failure")
	}
	if _, err := os.Stat(filepath.Join(c.DataDir, "initialized")); !os.IsNotExist(err) {
		t.Fatalf("marker exists after failed init: %v", err)
	}
	if err := os.Remove(c.CredentialPath); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
}

func TestInitRollsBackWhenMarkerDurabilityFails(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	originalFile, originalParent := markerSyncFile, markerSyncParent
	defer func() { markerSyncFile, markerSyncParent = originalFile, originalParent }()
	markerSyncFile = func(*os.File) error { return errors.New("injected marker file sync failure") }
	if err := Initialize(c); err == nil {
		t.Fatal("expected marker file sync failure")
	}
	for _, path := range []string{"state.json", "state.key", "initialized", "operator"} {
		if _, err := os.Lstat(filepath.Join(d, path)); !os.IsNotExist(err) {
			t.Fatalf("artifact survived marker file sync failure: %s (%v)", path, err)
		}
	}
	markerSyncFile = originalFile
	markerSyncParent = func(string) error { return errors.New("injected marker parent sync failure") }
	if err := Initialize(c); err == nil {
		t.Fatal("expected marker parent sync failure")
	}
	for _, path := range []string{"state.json", "state.key", "initialized", "operator"} {
		if _, err := os.Lstat(filepath.Join(d, path)); !os.IsNotExist(err) {
			t.Fatalf("artifact survived marker parent sync failure: %s (%v)", path, err)
		}
	}
	markerSyncParent = originalParent
	if err := Initialize(c); err != nil {
		t.Fatalf("retry after durability failures failed: %v", err)
	}
}
