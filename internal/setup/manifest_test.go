package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifestAndSelect(t *testing.T) {
	raw := `{"schemaVersion":1,"topology":"combined","artifacts":[{"name":"hermes","version":"1.2.3","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/hermes/1.2.3/linux-amd64","size":3,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"hermes"},{"name":"lumen","version":"1.2.3","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.2.3/linux-amd64","size":3,"sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","contractVersion":1,"executableMode":493,"ownership":"lumen"}]}`
	m, err := LoadManifest(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.Select("hermes", "linux", "amd64", Development)
	if err != nil || a.Name != "hermes" {
		t.Fatalf("select %#v, %v", a, err)
	}
	if a.Kind != ArtifactExecutable {
		t.Fatalf("legacy executable kind = %q", a.Kind)
	}
}

func TestCheckedInManifestRejectsUnreleasedFixture(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "deploy", "manifest-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	m, err := LoadManifest(f)
	if err == nil {
		t.Fatalf("accepted unreleased fixture: %#v", m)
	}
}

func TestManifestRejectsMalformedAndDuplicateEntries(t *testing.T) {
	base := `{"schemaVersion":1,"topology":"external","artifacts":[{"name":"lumen","version":"1.0.0","os":"darwin","architecture":"arm64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/darwin-arm64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":448,"ownership":"lumen"}]}`
	for _, raw := range []string{
		strings.Replace(base, `"schemaVersion":1`, `"schemaVersion":2`, 1),
		strings.Replace(base, `"url":"https://downloads.lumen.dev/lumen/1.0.0/darwin-arm64"`, `"url":"https://user:pass@downloads.lumen.dev/lumen?q=x#f"`, 1),
		strings.Replace(base, `}]}`, `},{"name":"lumen","version":"1.0.0","os":"darwin","architecture":"arm64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/darwin-arm64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":448,"ownership":"lumen"}]}`, 1),
	} {
		if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
			t.Fatalf("accepted invalid manifest: %s", raw)
		}
	}
}

func TestManifestRejectsUnsupportedTargetAndSelectHasNoFallback(t *testing.T) {
	raw := `{"schemaVersion":1,"topology":"external","artifacts":[{"name":"lumen","version":"1.0.0","os":"android","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/android-arm64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"lumen"}]}`
	if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
		t.Fatal("accepted unsupported target")
	}
	m := Manifest{Artifacts: []Artifact{{Name: "lumen", OS: "darwin", Architecture: "arm64", Profile: Development}}}
	if _, err := m.Select("untrusted", "untrusted", "untrusted", Profile("untrusted")); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestManifestRejectsPlaceholderDigest(t *testing.T) {
	raw := `{"schemaVersion":1,"topology":"external","artifacts":[{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:` + strings.Repeat("0", 64) + `","contractVersion":1,"executableMode":493,"ownership":"lumen"}]}`
	if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
		t.Fatal("accepted zero digest")
	}
}

func TestManifestRejectsMutableSource(t *testing.T) {
	raw := `{"schemaVersion":1,"topology":"combined","artifacts":[{"name":"hermes","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://github.com/NousResearch/hermes-agent/archive/main.tar.gz","size":1,"sha256":"sha256:` + strings.Repeat("a", 64) + `","contractVersion":1,"executableMode":493,"ownership":"hermes"},{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:` + strings.Repeat("b", 64) + `","contractVersion":1,"executableMode":493,"ownership":"lumen"}]}`
	if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
		t.Fatal("accepted mutable source")
	}
}

func TestManifestRequiresTopologyOwnership(t *testing.T) {
	raw := `{"schemaVersion":1,"topology":"external","artifacts":[{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"hermes"}]}`
	if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
		t.Fatal("accepted inconsistent ownership")
	}
}

func TestManifestRequiresTopologyArtifactSet(t *testing.T) {
	for _, raw := range []string{
		`{"schemaVersion":1,"artifacts":[]}`,
		`{"schemaVersion":1,"topology":"combined","artifacts":[{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"lumen"}]}`,
		`{"schemaVersion":1,"topology":"external","artifacts":[{"name":"hermes","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/hermes/1.0.0/linux-amd64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"hermes"}]}`,
	} {
		if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
			t.Fatalf("accepted invalid topology manifest: %s", raw)
		}
	}
}

func TestManifestAcceptsPinnedDockerImageAlongsideExecutable(t *testing.T) {
	image := "ghcr.io/nousresearch/hermes-agent@sha256:" + strings.Repeat("c", 64)
	raw := `{"schemaVersion":1,"topology":"combined","artifacts":[{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"lumen"},{"name":"hermes","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","kind":"docker-image","imageRef":"` + image + `","contractVersion":1,"ownership":"hermes"}]}`
	m, err := LoadManifest(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.Select("hermes", "linux", "amd64", Development)
	if err != nil {
		t.Fatal(err)
	}
	if a.Kind != ArtifactDockerImage || a.ImageRef != image {
		t.Fatalf("docker artifact = %#v", a)
	}
}

func TestManifestRejectsMutableDockerImageReference(t *testing.T) {
	for _, image := range []string{
		"ghcr.io/nousresearch/hermes-agent:latest",
		"ghcr.io/nousresearch/hermes-agent:v1.0.0",
		"ghcr.io/nousresearch/hermes-agent@sha256:short",
		"https://ghcr.io/nousresearch/hermes-agent@sha256:" + strings.Repeat("c", 64),
	} {
		raw := `{"schemaVersion":1,"topology":"combined","artifacts":[{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"lumen"},{"name":"hermes","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","kind":"docker-image","imageRef":"` + image + `","contractVersion":1,"ownership":"hermes"}]}`
		if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
			t.Fatalf("accepted mutable image reference %q", image)
		}
	}
}

func TestManifestRejectsDockerImageExecutableFields(t *testing.T) {
	image := "ghcr.io/nousresearch/hermes-agent@sha256:" + strings.Repeat("c", 64)
	raw := `{"schemaVersion":1,"topology":"combined","artifacts":[{"name":"lumen","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/linux-amd64","size":1,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contractVersion":1,"executableMode":493,"ownership":"lumen"},{"name":"hermes","version":"1.0.0","os":"linux","architecture":"amd64","profile":"development","kind":"docker-image","imageRef":"` + image + `","url":"https://downloads.lumen.dev/hermes/1.0.0/linux-amd64","size":1,"sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","contractVersion":1,"executableMode":493,"ownership":"hermes"}]}`
	if _, err := LoadManifest(strings.NewReader(raw)); err == nil {
		t.Fatal("accepted executable fields on docker image")
	}
}
