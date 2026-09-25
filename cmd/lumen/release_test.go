package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

// rewriteManifestExecutable repoints one executable manifest entry at new
// release bytes, mirroring a published release.
func rewriteManifestExecutable(t *testing.T, name string, body []byte) {
	t.Helper()
	path := os.Getenv("LUMEN_MANIFEST")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	artifacts, _ := manifest["artifacts"].([]any)
	found := false
	for _, entry := range artifacts {
		item, ok := entry.(map[string]any)
		if !ok || item["name"] != name {
			continue
		}
		item["version"] = "2.0.0"
		item["url"] = "https://downloads.lumen.dev/" + name + "/2.0.0/" + runtime.GOOS + "-" + runtime.GOARCH
		item["size"] = float64(len(body))
		item["sha256"] = fixtureDigest(body)
		found = true
	}
	if !found {
		t.Fatalf("manifest has no %s artifact", name)
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		t.Fatal(err)
	}
}

func installedArtifactPath(t *testing.T, name string) string {
	t.Helper()
	path := os.Getenv("LUMEN_" + strings.ToUpper(name) + "_ARTIFACT")
	if path == "" {
		t.Fatalf("%s artifact path is unset", name)
	}
	return path
}

func TestUpdateAdvancesGenerationAndRollbackRestoresIt(t *testing.T) {
	newSetupJourneyFixture(t, setup.TopologyCombined)
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("setup report = %#v", report)
	}
	installed := installedArtifactPath(t, "lumen")
	original, err := os.ReadFile(installed)
	if err != nil {
		t.Fatal(err)
	}
	replacement := []byte("lumen-release-two")
	source := filepath.Join(filepath.Dir(installed), "lumen-release-two")
	if err := os.WriteFile(source, replacement, 0700); err != nil {
		t.Fatal(err)
	}
	rewriteManifestExecutable(t, "lumen", replacement)
	t.Setenv("LUMEN_LUMEN_ARTIFACT", source)

	if report := runForTest([]string{"update"}); report.Outcome != setup.Ready {
		t.Fatalf("update report = %#v", report)
	}
	if live, err := os.ReadFile(installed); err != nil || string(live) != string(replacement) {
		t.Fatalf("updated artifact = %q, %v", live, err)
	}
	// The advanced generation must still satisfy the bound deployment, which
	// proves the release record, not the stale binding digest, is authoritative.
	if report := runForTest([]string{"service", "status"}); report.Outcome != setup.Ready {
		t.Fatalf("service status after update = %#v", report)
	}
	if report := runForTest([]string{"rollback"}); report.Outcome != setup.Ready {
		t.Fatalf("rollback report = %#v", report)
	}
	if live, err := os.ReadFile(installed); err != nil || string(live) != string(original) {
		t.Fatalf("rolled back artifact = %q, %v", live, err)
	}
	if report := runForTest([]string{"service", "status"}); report.Outcome != setup.Ready {
		t.Fatalf("service status after rollback = %#v", report)
	}
}

func TestFailedUpdatePreservesInstalledArtifact(t *testing.T) {
	newSetupJourneyFixture(t, setup.TopologyCombined)
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("setup report = %#v", report)
	}
	installed := installedArtifactPath(t, "lumen")
	original, err := os.ReadFile(installed)
	if err != nil {
		t.Fatal(err)
	}
	// The manifest advertises new release bytes, but the available source does
	// not match the published digest.
	replacement := []byte("lumen-release-two")
	rewriteManifestExecutable(t, "lumen", replacement)
	tampered := filepath.Join(filepath.Dir(installed), "tampered-lumen")
	if err := os.WriteFile(tampered, []byte("not-the-published-release"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUMEN_LUMEN_ARTIFACT", tampered)

	if report := runForTest([]string{"update"}); report.Outcome != setup.ActionRequired {
		t.Fatalf("tampered update report = %#v", report)
	}
	if live, err := os.ReadFile(installed); err != nil || string(live) != string(original) {
		t.Fatalf("failed update changed the live artifact: %q, %v", live, err)
	}
	// No generation was recorded, so there is nothing to roll back to.
	if report := runForTest([]string{"rollback"}); report.Outcome != setup.ActionRequired {
		t.Fatalf("rollback after a failed update = %#v", report)
	}
}

func TestReleaseCommandsRequireAValidatedDeployment(t *testing.T) {
	t.Setenv("LUMEN_DATA_DIR", "")
	for _, command := range []string{"update", "rollback"} {
		if report := runForTest([]string{command}); report.Outcome != setup.ActionRequired {
			t.Fatalf("%s without a data directory = %#v", command, report)
		}
	}
	newSetupJourneyFixture(t, setup.TopologyCombined)
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("setup report = %#v", report)
	}
	report := runForTest([]string{"rollback"})
	if report.Outcome != setup.ActionRequired || len(report.Actions) == 0 || report.Actions[0].Code != "no_rollback_generation" {
		t.Fatalf("rollback without a recorded generation = %#v", report)
	}
}

func TestUpdateWithoutAChangedArtifactIsNotAdvanceable(t *testing.T) {
	newSetupJourneyFixture(t, setup.TopologyCombined)
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("setup report = %#v", report)
	}
	report := runForTest([]string{"update"})
	if report.Outcome != setup.ActionRequired || len(report.Actions) == 0 || report.Actions[0].Code != "release_not_advanceable" {
		t.Fatalf("no-op update report = %#v", report)
	}
	if _, err := os.Lstat(releaseRecordPath(os.Getenv("LUMEN_DATA_DIR"))); !os.IsNotExist(err) {
		t.Fatalf("a no-op update recorded a generation: %v", err)
	}
}

func TestUpdateRefusesAnImageGeneration(t *testing.T) {
	newSetupJourneyFixture(t, setup.TopologyCombined)
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("setup report = %#v", report)
	}
	// A changed Hermes image cannot be replaced by an in-place file swap, so it
	// must be reported honestly rather than faked.
	rewriteHermesManifestArtifact(t, "ghcr.io/lumen/hermes@sha256:"+strings.Repeat("b", 64))
	t.Setenv("LUMEN_HERMES_ARTIFACT", "")
	previousInspect := dockerImageInspect
	dockerImageInspect = func(string) error { return nil }
	t.Cleanup(func() { dockerImageInspect = previousInspect })

	report := runForTest([]string{"update"})
	if report.Outcome != setup.ActionRequired || len(report.Actions) == 0 || report.Actions[0].Code != "image_update_requires_deployment" {
		t.Fatalf("image update report = %#v", report)
	}
}
