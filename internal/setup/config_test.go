package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/host"
)

func TestGeneratedConfigHasNoManualPlaceholders(t *testing.T) {
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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

func TestGeneratedPersonalAlphaConfigLoads(t *testing.T) {
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := WriteConfig(ConfigRequest{DataDir: filepath.Join(d, "lumen"), HermesDir: filepath.Join(d, "hermes"), Profile: PersonalAlpha, HermesBaseURL: "http://127.0.0.1", HermesBearer: "token"})
	if err != nil {
		t.Fatal(err)
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

func TestWriteConfigRejectsSymlinkedIntermediateExistingFinal(t *testing.T) {
	d, _ := filepath.EvalSymlinks(t.TempDir())
	real := filepath.Join(d, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(link, "final")
	if err := os.Mkdir(final, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteConfig(ConfigRequest{DataDir: final, HermesDir: filepath.Join(d, "h"), Profile: Development, HermesBaseURL: "http://127.0.0.1"}); err == nil {
		t.Fatal("expected intermediate symlink rejection")
	}
}

func TestExternalWriteConfigDoesNotWriteHermesOwnedConfigOrSecrets(t *testing.T) {
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cred := filepath.Join(d, "external-token")
	if err := os.WriteFile(cred, []byte("remote-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := WriteConfig(ConfigRequest{DataDir: filepath.Join(d, "lumen"), HermesDir: filepath.Join(d, "hermes"), Profile: Development, Topology: TopologyExternal, HermesBaseURL: "https://hermes.example", HermesCredentialFile: cred, HermesCAFile: filepath.Join(d, "ca"), HermesClientCertFile: filepath.Join(d, "cert"), HermesClientKeyFile: filepath.Join(d, "key")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.Hermes); !os.IsNotExist(err) {
		t.Fatalf("external topology wrote Hermes config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(d, "hermes", "hermes.token")); !os.IsNotExist(err) {
		t.Fatalf("external topology wrote secret: %v", err)
	}
	b, err := os.ReadFile(p.Lumen)
	if err != nil || strings.Contains(string(b), "remote-secret") {
		t.Fatalf("secret leaked: %s (%v)", b, err)
	}
}

func TestExternalWriteConfigRejectsWrongOwnerReference(t *testing.T) {
	d, _ := filepath.EvalSymlinks(t.TempDir())
	cred := filepath.Join(d, "token")
	if err := os.WriteFile(cred, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(cred, os.Getuid()+1, os.Getgid()); err != nil {
		t.Skip("chown unavailable")
	}
	_, err := WriteConfig(ConfigRequest{DataDir: filepath.Join(d, "lumen"), HermesDir: filepath.Join(d, "hermes"), Profile: Development, Topology: TopologyExternal, HermesBaseURL: "https://hermes.example", HermesCredentialFile: cred})
	if err == nil {
		t.Fatal("wrong-owner reference accepted")
	}
}
