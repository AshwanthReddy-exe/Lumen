package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
)

func TestConfigFromEnvironment(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SOCKET_PATH", filepath.Join(d, "host.sock"))
	t.Setenv("LUMEN_OPERATOR_CREDENTIAL_FILE", filepath.Join(d, "operator"))
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != d {
		t.Fatalf("data dir %q", c.DataDir)
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
