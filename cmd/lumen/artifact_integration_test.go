package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

const fixtureHermesImageRef = "ghcr.io/lumen/hermes@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCombinedSetupUsesPinnedDockerImageWithoutExecutablePath(t *testing.T) {
	h := newSetupJourneyFixture(t, setup.TopologyCombined)
	rewriteHermesManifestArtifact(t, fixtureHermesImageRef)
	t.Setenv("LUMEN_HERMES_ARTIFACT", "")

	var inspected []string
	previousInspect := dockerImageInspect
	dockerImageInspect = func(ref string) error {
		inspected = append(inspected, ref)
		if ref != fixtureHermesImageRef {
			return errors.New("unexpected image reference")
		}
		return nil
	}
	t.Cleanup(func() { dockerImageInspect = previousInspect })

	report := runForTest([]string{"setup"})
	if report.Outcome != setup.Ready {
		t.Fatalf("combined image setup report = %#v", report)
	}
	if len(inspected) == 0 || inspected[0] != fixtureHermesImageRef {
		t.Fatalf("docker image was not inspected: %v", inspected)
	}

	j, err := setup.NewJournal(filepath.Join(h.dataDir, "setup"))
	if err != nil {
		t.Fatal(err)
	}
	binding, ok := j.Binding()
	if !ok {
		t.Fatal("setup did not persist an artifact binding")
	}
	if _, ok := binding.ArtifactPaths["hermes"]; ok {
		t.Fatalf("image artifact persisted as executable path: %#v", binding.ArtifactPaths)
	}
	if binding.ArtifactRefs["hermes"] != fixtureHermesImageRef {
		t.Fatalf("persisted image reference = %q", binding.ArtifactRefs["hermes"])
	}
	if binding.ArtifactDigests["hermes"] != "sha256:"+strings.Repeat("a", 64) {
		t.Fatalf("persisted image digest = %q", binding.ArtifactDigests["hermes"])
	}
	if !verifyDurableArtifacts(binding, binding.ArtifactDigests) {
		t.Fatal("durable image binding was not accepted")
	}
}

func TestCombinedSetupRejectsMissingOrSubstitutedDockerImageBeforeMutation(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  string
	}{
		{name: "missing", ref: fixtureHermesImageRef},
		{name: "substituted", ref: "ghcr.io/lumen/hermes@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newSetupJourneyFixture(t, setup.TopologyCombined)
			rewriteHermesManifestArtifact(t, tc.ref)
			t.Setenv("LUMEN_HERMES_ARTIFACT", "")
			if err := os.RemoveAll(h.dataDir); err != nil {
				t.Fatal(err)
			}

			var inspected []string
			previousInspect := dockerImageInspect
			dockerImageInspect = func(ref string) error {
				inspected = append(inspected, ref)
				return errors.New("docker image identity unavailable")
			}
			t.Cleanup(func() { dockerImageInspect = previousInspect })

			report := runForTest([]string{"setup"})
			if report.Outcome != setup.ActionRequired {
				t.Fatalf("rejected image setup report = %#v", report)
			}
			if len(inspected) != 1 || inspected[0] != tc.ref {
				t.Fatalf("image inspection = %v, want one inspection of %q", inspected, tc.ref)
			}
			if _, err := os.Stat(h.dataDir); !os.IsNotExist(err) {
				t.Fatalf("rejected image setup mutated state directory: %v", err)
			}
		})
	}
}

func rewriteHermesManifestArtifact(t *testing.T, imageRef string) {
	t.Helper()
	path := os.Getenv("LUMEN_MANIFEST")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	start := strings.Index(raw, `{"name":"hermes"`)
	if start < 0 {
		t.Fatalf("manifest has no Hermes artifact: %s", raw)
	}
	end := strings.IndexByte(raw[start:], '}')
	if end < 0 {
		t.Fatalf("manifest Hermes artifact is malformed: %s", raw)
	}
	replacement := `{"name":"hermes","version":"1.0.0","os":"` + runtime.GOOS + `","architecture":"` + runtime.GOARCH + `","profile":"development","kind":"docker-image","imageRef":"` + imageRef + `","contractVersion":1,"ownership":"hermes"}`
	raw = raw[:start] + replacement + raw[start+end+1:]
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
}
