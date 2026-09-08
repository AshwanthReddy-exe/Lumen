package host

import (
	"os"
	"path/filepath"
	"testing"
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
