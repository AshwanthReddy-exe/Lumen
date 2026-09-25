package setup

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func releaseExecutable(t *testing.T, path, body string) ReleaseArtifact {
	t.Helper()
	h := sha256.Sum256([]byte(body))
	return ReleaseArtifact{Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + fmtHex(h[:]), Path: path}
}

func releaseRecord(topology Topology, artifacts map[string]ReleaseArtifact) ReleaseRecord {
	return ReleaseRecord{Generation: 1, Profile: Development, Topology: topology, Current: artifacts}
}

// privateTempDir resolves symlinks so the record lives under a directory chain
// that satisfies the owner-only private-path contract (macOS /var is a symlink).
func privateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNextReleaseAdvancesGenerationAndKeepsIdentity(t *testing.T) {
	dir := t.TempDir()
	current := releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen":  {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: filepath.Join(dir, "lumen")},
		"hermes": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("b", 64), Path: filepath.Join(dir, "hermes")},
	})
	current.Profile = Hardened
	next, err := NextRelease(current, map[string]ReleaseArtifact{
		"lumen":  {Kind: ArtifactExecutable, Version: "1.1.0", Digest: "sha256:" + strings.Repeat("c", 64), Path: filepath.Join(dir, "lumen")},
		"hermes": current.Current["hermes"],
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.Generation != 2 || next.Profile != Hardened || next.Topology != TopologyCombined {
		t.Fatalf("identity drifted: %#v", next)
	}
	if next.Current["lumen"].Version != "1.1.0" {
		t.Fatalf("live artifact = %#v", next.Current["lumen"])
	}
	if next.Rollback["lumen"].Digest != current.Current["lumen"].Digest || next.Rollback["hermes"].Digest != current.Current["hermes"].Digest {
		t.Fatalf("rollback generation not recorded: %#v", next.Rollback)
	}
	back, err := RollbackRelease(next)
	if err != nil {
		t.Fatal(err)
	}
	if back.Generation != 3 || back.Current["lumen"].Digest != current.Current["lumen"].Digest || back.Rollback["lumen"].Digest != next.Current["lumen"].Digest {
		t.Fatalf("rollback round trip = %#v", back)
	}
	if back.Profile != next.Profile || back.Topology != next.Topology {
		t.Fatalf("rollback changed identity: %#v", back)
	}
}

func TestNextReleaseRejectsChangedArtifactSet(t *testing.T) {
	dir := t.TempDir()
	current := releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen":  {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: filepath.Join(dir, "lumen")},
		"hermes": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("b", 64), Path: filepath.Join(dir, "hermes")},
	})
	newer := ReleaseArtifact{Kind: ArtifactExecutable, Version: "1.1.0", Digest: "sha256:" + strings.Repeat("c", 64), Path: filepath.Join(dir, "lumen")}
	if _, err := NextRelease(current, map[string]ReleaseArtifact{"lumen": newer}); !errors.Is(err, ErrReleaseNotAdvanceable) {
		t.Fatalf("removed artifact accepted: %v", err)
	}
	if _, err := NextRelease(current, map[string]ReleaseArtifact{"lumen": newer, "hermes": current.Current["hermes"], "extra": newer}); !errors.Is(err, ErrReleaseNotAdvanceable) {
		t.Fatalf("added artifact accepted: %v", err)
	}
	if _, err := NextRelease(current, current.Current); !errors.Is(err, ErrReleaseNotAdvanceable) {
		t.Fatalf("no-op update accepted: %v", err)
	}
}

func TestReleaseRecordRejectsArtifactOutsideTopology(t *testing.T) {
	r := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{
		"hermes": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("b", 64), Path: "/opt/lumen/bin/hermes"},
	})
	if !errors.Is(validReleaseRecord(r), ErrReleaseRecordInvalid) {
		t.Fatal("accepted a Hermes artifact in an external-topology record")
	}
	r = releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: "/opt/lumen/bin/lumen"},
	})
	if !errors.Is(validReleaseRecord(r), ErrReleaseRecordInvalid) {
		t.Fatal("accepted a combined record without Hermes")
	}
}

func TestReleaseArtifactForRejectsPlaceholderIdentity(t *testing.T) {
	base := Artifact{Name: "lumen", Version: "1.0.0", OS: "linux", Architecture: "amd64", Profile: Development, URL: "https://downloads.lumen.dev/lumen/1.0.0/linux-amd64", Size: 1, ContractVersion: 1, ExecutableMode: 0700, Ownership: "lumen"}
	placeholder := base
	placeholder.SHA256 = "sha256:" + strings.Repeat("0", 64)
	if _, err := ReleaseArtifactFor(placeholder); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("accepted placeholder digest: %v", err)
	}
	real := base
	real.SHA256 = "sha256:" + strings.Repeat("a", 64)
	got, err := ReleaseArtifactFor(real)
	if err != nil || got.Digest != real.SHA256 || got.Kind != ArtifactExecutable || got.Path != "" {
		t.Fatalf("executable identity = %#v, %v", got, err)
	}
	image := Artifact{Name: "hermes", Version: "1.0.0", OS: "linux", Architecture: "amd64", Profile: Development, Kind: ArtifactDockerImage, ImageRef: "ghcr.io/nousresearch/hermes-agent@sha256:" + strings.Repeat("c", 64), ContractVersion: 1, Ownership: "hermes"}
	gotImage, err := ReleaseArtifactFor(image)
	if err != nil || gotImage.Ref != image.ImageRef || gotImage.Digest != "sha256:"+strings.Repeat("c", 64) {
		t.Fatalf("image identity = %#v, %v", gotImage, err)
	}
}

func TestReleaseRecordAcceptsPinnedDockerImageAndRejectsDigestDrift(t *testing.T) {
	ref := "ghcr.io/nousresearch/hermes-agent@sha256:" + strings.Repeat("c", 64)
	r := releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen":  {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: "/opt/lumen/bin/lumen"},
		"hermes": {Kind: ArtifactDockerImage, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("c", 64), Ref: ref},
	})
	if err := validReleaseRecord(r); err != nil {
		t.Fatal(err)
	}
	drifted := releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen":  r.Current["lumen"],
		"hermes": {Kind: ArtifactDockerImage, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("d", 64), Ref: ref},
	})
	if !errors.Is(validReleaseRecord(drifted), ErrReleaseRecordInvalid) {
		t.Fatal("accepted a docker-image digest that does not match its pinned reference")
	}
	mutable := releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen":  r.Current["lumen"],
		"hermes": {Kind: ArtifactDockerImage, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("c", 64), Ref: "ghcr.io/nousresearch/hermes-agent:latest"},
	})
	if !errors.Is(validReleaseRecord(mutable), ErrReleaseRecordInvalid) {
		t.Fatal("accepted a mutable docker image reference")
	}
}

func TestReleaseMatchesBindingRejectsDrift(t *testing.T) {
	r := releaseRecord(TopologyCombined, map[string]ReleaseArtifact{
		"lumen":  {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: "/opt/lumen/bin/lumen"},
		"hermes": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("b", 64), Path: "/opt/lumen/bin/hermes"},
	})
	binding := JournalBinding{
		Profile:       Development,
		Topology:      TopologyCombined,
		PlanDigest:    "sha256:" + strings.Repeat("e", 64),
		ArtifactPaths: map[string]string{"lumen": "/opt/lumen/bin/lumen", "hermes": "/opt/lumen/bin/hermes"},
		ArtifactDigests: map[string]string{
			"lumen":  "sha256:" + strings.Repeat("1", 64),
			"hermes": "sha256:" + strings.Repeat("2", 64),
		},
	}
	// An advanced digest is the point of an update; paths still must match.
	if err := ReleaseMatchesBinding(r, binding); err != nil {
		t.Fatal(err)
	}
	driftedProfile := binding
	driftedProfile.Profile = Hardened
	if !errors.Is(ReleaseMatchesBinding(r, driftedProfile), ErrReleaseBindingDrift) {
		t.Fatal("accepted a profile drift against the binding")
	}
	driftedTopology := binding
	driftedTopology.Topology = TopologyExternal
	if !errors.Is(ReleaseMatchesBinding(r, driftedTopology), ErrReleaseBindingDrift) {
		t.Fatal("accepted a topology drift against the binding")
	}
	redirectedPath := binding
	redirectedPath.ArtifactPaths = map[string]string{"lumen": "/tmp/attacker/lumen", "hermes": "/opt/lumen/bin/hermes"}
	if !errors.Is(ReleaseMatchesBinding(r, redirectedPath), ErrReleaseBindingDrift) {
		t.Fatal("accepted a record that redirects the verified install path")
	}
	pathAsImage := binding
	pathAsImage.ArtifactRefs = map[string]string{"lumen": "ghcr.io/lumen/lumen@sha256:" + strings.Repeat("a", 64)}
	if !errors.Is(ReleaseMatchesBinding(r, pathAsImage), ErrReleaseBindingDrift) {
		t.Fatal("accepted a record that changes an executable into an image")
	}
}

func TestRollbackReleaseWithoutGenerationFails(t *testing.T) {
	r := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{
		"lumen": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: "/opt/lumen/bin/lumen"},
	})
	if _, err := RollbackRelease(r); !errors.Is(err, ErrReleaseNoRollback) {
		t.Fatalf("rollback without a generation = %v", err)
	}
}

func TestReleaseRecordDurabilityAndStrictDecoding(t *testing.T) {
	dir := privateTempDir(t)
	path := filepath.Join(dir, "release-record.json")
	record := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{
		"lumen": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Path: "/opt/lumen/bin/lumen"},
	})
	if err := SaveReleaseRecord(path, record); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadReleaseRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Generation != record.Generation || loaded.Current["lumen"] != record.Current["lumen"] {
		t.Fatalf("round trip = %#v", loaded)
	}
	if st, err := os.Stat(path); err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("record mode = %v, %v", st.Mode(), err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	fields["unexpected"] = true
	tampered, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, tampered, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseRecord(path); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("accepted an unknown field: %v", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseRecord(path); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("accepted a group-readable record: %v", err)
	}
}

func TestSaveReleaseRecordRejectsPlaceholderRecord(t *testing.T) {
	dir := privateTempDir(t)
	path := filepath.Join(dir, "release-record.json")
	placeholder := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{
		"lumen": {Kind: ArtifactExecutable, Version: "1.0.0", Digest: "sha256:" + strings.Repeat("0", 64), Path: "/opt/lumen/bin/lumen"},
	})
	if err := SaveReleaseRecord(path, placeholder); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("persisted a placeholder record: %v", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("placeholder record was written")
	}
	empty := ReleaseRecord{Generation: 1, Profile: Development, Topology: TopologyExternal}
	if err := SaveReleaseRecord(path, empty); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("persisted an empty record: %v", err)
	}
	// A well-formed record must still save, proving the rejections above came
	// from the record contract rather than from an unusable temp directory.
	good := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{"lumen": releaseExecutable(t, "/opt/lumen/bin/lumen", "body")})
	if err := SaveReleaseRecord(path, good); err != nil {
		t.Fatalf("rejected a valid record: %v", err)
	}
}

func TestRetainAndRestoreGenerationWithoutNetwork(t *testing.T) {
	dir := privateTempDir(t)
	live := filepath.Join(dir, "lumen")
	if err := os.WriteFile(live, []byte("generation-one"), 0700); err != nil {
		t.Fatal(err)
	}
	gen1 := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{"lumen": releaseExecutable(t, live, "generation-one")})
	store := ReleaseStoreDir(dir)
	if err := RetainExecutableGeneration(store, gen1); err != nil {
		t.Fatal(err)
	}
	if err := RetainExecutableGeneration(store, gen1); err != nil {
		t.Fatalf("retention is not idempotent: %v", err)
	}
	retained := retainedArtifactPath(store, "lumen", gen1.Current["lumen"].Digest)
	if err := privateRetainedDigestMatches(retained, gen1.Current["lumen"].Digest); err != nil {
		t.Fatalf("retained bytes are not private and verified: %v", err)
	}
	// A later update replaces the live bytes; rollback must restore them from
	// the store alone.
	if err := os.WriteFile(live, []byte("generation-two"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := RestoreRetainedGeneration(store, gen1, 0700); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(live)
	if err != nil || string(b) != "generation-one" {
		t.Fatalf("restored bytes = %q, %v", b, err)
	}
	if err := VerifyReleaseGeneration(gen1); err != nil {
		t.Fatalf("restored generation does not verify: %v", err)
	}
	if st, err := os.Stat(live); err != nil || st.Mode().Perm() != 0700 {
		t.Fatalf("restored mode = %v, %v", st.Mode(), err)
	}
}

func TestRetainRefusesTamperedLiveArtifact(t *testing.T) {
	dir := privateTempDir(t)
	live := filepath.Join(dir, "lumen")
	if err := os.WriteFile(live, []byte("original"), 0700); err != nil {
		t.Fatal(err)
	}
	r := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{"lumen": releaseExecutable(t, live, "original")})
	if err := os.WriteFile(live, []byte("tampered"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := RetainExecutableGeneration(ReleaseStoreDir(dir), r); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("retained a tampered live artifact: %v", err)
	}
	if _, err := os.Lstat(retainedArtifactPath(ReleaseStoreDir(dir), "lumen", r.Current["lumen"].Digest)); !os.IsNotExist(err) {
		t.Fatal("a tampered artifact was retained")
	}
}

func TestRestoreRetainedGenerationRequiresVerifiedBytes(t *testing.T) {
	dir := privateTempDir(t)
	live := filepath.Join(dir, "lumen")
	if err := os.WriteFile(live, []byte("present"), 0700); err != nil {
		t.Fatal(err)
	}
	missing := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{"lumen": releaseExecutable(t, live, "absent")})
	if err := RestoreRetainedGeneration(ReleaseStoreDir(dir), missing, 0700); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("restored without retained bytes: %v", err)
	}
	// A corrupt retained file must be rejected rather than installed.
	good := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{"lumen": releaseExecutable(t, live, "present")})
	retained := retainedArtifactPath(ReleaseStoreDir(dir), "lumen", good.Current["lumen"].Digest)
	if err := RetainExecutableGeneration(ReleaseStoreDir(dir), good); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retained, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("live-two"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := RestoreRetainedGeneration(ReleaseStoreDir(dir), good, 0700); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("installed corrupt retained bytes: %v", err)
	}
	b, err := os.ReadFile(live)
	if err != nil || string(b) != "live-two" {
		t.Fatalf("failed restore changed the live artifact: %q, %v", b, err)
	}
	if err := RestoreRetainedGeneration(ReleaseStoreDir(dir), good, 0644); !errors.Is(err, ErrReleaseRecordInvalid) {
		t.Fatalf("accepted an unsupported install mode: %v", err)
	}
}

func TestVerifyReleaseGenerationDetectsModifiedArtifact(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "lumen")
	if err := os.WriteFile(bin, []byte("original"), 0755); err != nil {
		t.Fatal(err)
	}
	r := releaseRecord(TopologyExternal, map[string]ReleaseArtifact{"lumen": releaseExecutable(t, bin, "original")})
	if err := VerifyReleaseGeneration(r); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("tampered"), 0755); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(VerifyReleaseGeneration(r), ErrDigestMismatch) {
		t.Fatal("accepted a modified live artifact")
	}
	if err := os.Remove(bin); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(VerifyReleaseGeneration(r), ErrReleaseRecordInvalid) {
		t.Fatal("accepted a missing live artifact")
	}
}
