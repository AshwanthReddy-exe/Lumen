package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifestAndSelect(t *testing.T) {
	raw := `{"schemaVersion":1,"artifacts":[{"name":"hermes","version":"1.2.3","os":"linux","architecture":"amd64","profile":"development","url":"https://example.invalid/hermes","size":3,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493}]}`
	m, err := LoadManifest(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.Select("hermes", "linux", "amd64", Development)
	if err != nil || a.Name != "hermes" {
		t.Fatalf("select %#v, %v", a, err)
	}
}

func TestCheckedInManifestSelectsEveryDeclaredTarget(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "deploy", "manifest-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	m, err := LoadManifest(f)
	if err != nil {
		t.Fatal(err)
	}
	type target struct {
		os, arch string
		profile  Profile
	}
	groups := map[target]map[string]bool{}
	for _, want := range m.Artifacts {
		key := target{want.OS, want.Architecture, want.Profile}
		if groups[key] == nil {
			groups[key] = map[string]bool{}
		}
		groups[key][want.Name] = true
	}
	for key, names := range groups {
		if len(names) != 2 || !names["lumen"] || !names["hermes"] {
			t.Fatalf("target %v has names %#v", key, names)
		}
		for _, name := range []string{"lumen", "hermes"} {
			got, err := m.Select(name, key.os, key.arch, key.profile)
			if err != nil || got.ContractVersion != 1 {
				t.Fatalf("select %s/%v: %#v, %v", name, key, got, err)
			}
		}
	}
}

func TestManifestRejectsMalformedAndDuplicateEntries(t *testing.T) {
	base := `{"schemaVersion":1,"artifacts":[{"name":"lumen","version":"1.0.0","os":"darwin","architecture":"arm64","profile":"development","url":"https://example.invalid/lumen","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":448}]}`
	for _, raw := range []string{
		strings.Replace(base, `"schemaVersion":1`, `"schemaVersion":2`, 1),
		strings.Replace(base, `"url":"https://example.invalid/lumen"`, `"url":"https://user:pass@example.invalid/lumen?q=x#f"`, 1),
		strings.Replace(base, `}]}`, `},{"name":"lumen","version":"1.0.0","os":"darwin","architecture":"arm64","profile":"development","url":"https://example.invalid/lumen2","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":448}]}`, 1),
	} {
		if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
			t.Fatalf("accepted invalid manifest: %s", raw)
		}
	}
}

func TestManifestRejectsUnsupportedTargetAndSelectHasNoFallback(t *testing.T) {
	raw := `{"schemaVersion":1,"artifacts":[{"name":"lumen","version":"1.0.0","os":"android","architecture":"amd64","profile":"development","url":"https://example.invalid/lumen","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493}]}`
	if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
		t.Fatal("accepted unsupported target")
	}
	m := Manifest{Artifacts: []Artifact{{Name: "lumen", OS: "darwin", Architecture: "arm64", Profile: Development}}}
	if _, err := m.Select("untrusted", "untrusted", "untrusted", Profile("untrusted")); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("got %v", err)
	}
}
