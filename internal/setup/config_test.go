package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/host"
)

func TestGeneratedConfigHasNoManualPlaceholders(t *testing.T) {
	d := t.TempDir()
	p, err := WriteConfig(ConfigRequest{DataDir: filepath.Join(d, "lumen"), HermesDir: filepath.Join(d, "hermes"), Profile: Development, HermesBaseURL: "http://127.0.0.1:9090", HermesBearer: "token"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{p.Lumen, p.Hermes, p.Bearer} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "YOUR_") {
			t.Fatalf("placeholder in %q", path)
		}
	}
	if _, err := host.ConfigFromFile(p.Lumen); err != nil {
		t.Fatal(err)
	}
}

func TestWriteConfigRejectsSymlinkAndUncleanDirectories(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteConfig(ConfigRequest{DataDir: link, HermesDir: filepath.Join(d, "h"), Profile: Development, HermesBaseURL: "http://127.0.0.1"}); err == nil {
		t.Fatal("expected symlink rejection")
	}
	if _, err := WriteConfig(ConfigRequest{DataDir: d + "/x/../x", HermesDir: filepath.Join(d, "h2"), Profile: Development, HermesBaseURL: "http://127.0.0.1"}); err == nil {
		t.Fatal("expected unclean path rejection")
	}
}
