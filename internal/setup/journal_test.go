package setup

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestJournal(t *testing.T) *Journal {
	t.Helper()
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestJournalRejectsCorrupt(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "setup-journal.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewJournal(d); err == nil {
		t.Fatal("expected corruption error")
	}
}

func TestChangedInputSentinel(t *testing.T) {
	j := newTestJournal(t)
	if err := j.Record(StageEvidence{Stage: Detected, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err != nil {
		t.Fatal(err)
	}
	if err := j.Record(StageEvidence{Stage: Detected, InputDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}); !errors.Is(err, ErrInputChanged) {
		t.Fatalf("got %v", err)
	}
}

func TestJournalRejectsInvalidDigests(t *testing.T) {
	inputs := []string{"secret-value", "sha256:" + strings.Repeat("a", 63), "sha256:" + strings.Repeat("a", 65), "sha256:" + strings.Repeat("A", 64), "sha256:" + strings.Repeat("a", 63) + "\n", strings.Repeat("x", 4096)}
	for _, input := range inputs {
		d := t.TempDir()
		j, err := NewJournal(d)
		if err != nil {
			t.Fatal(err)
		}
		if !errors.Is(j.Record(StageEvidence{Stage: Detected, InputDigest: input}), ErrInvalidDigest) {
			t.Fatalf("accepted %q", input)
		}
		if _, err := os.Stat(filepath.Join(d, "setup-journal.json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("created journal")
		}
	}
}

func TestJournalParentSyncFailureIsDurabilityUncertain(t *testing.T) {
	d := t.TempDir()
	j, _ := NewJournal(d)
	valid := "sha256:" + strings.Repeat("a", 64)
	if err := j.Record(StageEvidence{Stage: Detected, InputDigest: valid}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(j.path)
	calls := 0
	j.syncParent = func(string) error { calls++; return errors.New("sync failed") }
	err := j.Record(StageEvidence{Stage: ArtifactsReady, InputDigest: valid})
	if !errors.Is(err, ErrDurabilityUncertain) || j.Next() != DirectoriesReady {
		t.Fatalf("err=%v next=%s", err, j.Next())
	}
	r, _ := NewJournal(d)
	if r.Next() != DirectoriesReady {
		t.Fatal(r.Next())
	}
	after, _ := os.ReadFile(j.path)
	if bytes.Equal(before, after) || calls != 1 {
		t.Fatal("candidate not committed")
	}
	j.syncParent = func(string) error { calls++; return errors.New("should not call") }
	if err := j.Record(StageEvidence{Stage: ArtifactsReady, InputDigest: valid}); err != nil || calls != 1 {
		t.Fatalf("retry err=%v calls=%d", err, calls)
	}
}

func TestJournalFullProgressionAndReopen(t *testing.T) {
	d := t.TempDir()
	j, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range stageOrder {
		if err := j.Record(StageEvidence{Stage: s, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", CompletedAt: int64(i + 1)}); err != nil {
			t.Fatal(err)
		}
	}
	if j.Next() != Validated {
		t.Fatal(j.Next())
	}
	if !j.IsValidated() {
		t.Fatal("terminal validation record was not recognized")
	}
	reopened, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Next() != Validated {
		t.Fatal(reopened.Next())
	}
	if !reopened.IsValidated() {
		t.Fatal("reopened terminal validation record was not recognized")
	}
}

func TestValidatedEvidenceMustMatchBindingToAuthorizeLaunch(t *testing.T) {
	dir := t.TempDir()
	journal, err := NewJournal(dir)
	if err != nil {
		t.Fatal(err)
	}
	planDigest := "sha256:" + strings.Repeat("b", 64)
	endpointDigest := digest("endpoint-a")
	binding := JournalBinding{Profile: Development, Topology: TopologyExternal, Supervisor: SupervisorLaunchd, PlanDigest: planDigest, EndpointOriginDigest: endpointDigest}
	if err := journal.Bind(binding); err != nil {
		t.Fatal(err)
	}
	for _, stage := range stageOrder {
		if err := journal.Record(StageEvidence{Stage: stage, InputDigest: "sha256:" + strings.Repeat("a", 64), Profile: Development, PlanDigest: planDigest}); err != nil {
			t.Fatal(err)
		}
	}
	if !journal.IsValidatedForBinding() {
		t.Fatal("matching terminal evidence did not authorize launch")
	}
	evidencePath := filepath.Join(dir, "setup-journal.json")
	evidenceBytes, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	legacyBytes := bytes.Replace(evidenceBytes, []byte(`,"bindingDigest":"`+bindingDigest(binding)+`"`), nil, 1)
	if bytes.Equal(legacyBytes, evidenceBytes) {
		t.Fatal("test did not remove the terminal binding digest")
	}
	if err := os.WriteFile(evidencePath, legacyBytes, 0600); err != nil {
		t.Fatal(err)
	}
	legacy, err := NewJournal(dir)
	if err != nil || legacy.IsValidatedForBinding() {
		t.Fatalf("legacy validation unexpectedly authorized: journal=%v err=%v", legacy != nil, err)
	}
	if err := legacy.BindValidatedEvidence(); err != nil || !legacy.IsValidatedForBinding() {
		t.Fatalf("verified legacy validation could not bind: authorized=%v err=%v", legacy.IsValidatedForBinding(), err)
	}

	bindingBytes, err := os.ReadFile(filepath.Join(dir, "setup-binding.json"))
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(bindingBytes, []byte(`"profile":"development"`), []byte(`"profile":"personal-alpha"`), 1)
	if bytes.Equal(tampered, bindingBytes) {
		t.Fatal("test did not alter the binding profile")
	}
	if err := os.WriteFile(filepath.Join(dir, "setup-binding.json"), tampered, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewJournal(dir); !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("mixed binding/evidence was accepted: %v", err)
	}
	tampered = bytes.Replace(bindingBytes, []byte(endpointDigest), []byte(digest("endpoint-b")), 1)
	if bytes.Equal(tampered, bindingBytes) {
		t.Fatal("test did not alter the endpoint binding")
	}
	if err := os.WriteFile(filepath.Join(dir, "setup-binding.json"), tampered, 0600); err != nil {
		t.Fatal(err)
	}
	changedEndpoint, err := NewJournal(dir)
	if err != nil || changedEndpoint.IsValidatedForBinding() {
		t.Fatalf("same-profile/plan endpoint substitution authorized launch: authorized=%v err=%v", changedEndpoint != nil && changedEndpoint.IsValidatedForBinding(), err)
	}
	if err := changedEndpoint.BindValidatedEvidence(); !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("tampered nonempty binding digest was rebound: %v", err)
	}
}

func TestUnboundValidatedJournalCannotAuthorizeLaunch(t *testing.T) {
	journal := newTestJournal(t)
	for _, stage := range stageOrder {
		if err := journal.Record(StageEvidence{Stage: stage, InputDigest: "sha256:" + strings.Repeat("a", 64)}); err != nil {
			t.Fatal(err)
		}
	}
	if !journal.IsValidated() || journal.IsValidatedForBinding() {
		t.Fatal("unbound journal was treated as launch authorization")
	}
}

func TestJournalRerunPreservesBytesAndCompletedAt(t *testing.T) {
	d := t.TempDir()
	j, _ := NewJournal(d)
	e := StageEvidence{Stage: Detected, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", CompletedAt: 42}
	if err := j.Record(e); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "setup-journal.json")
	before, _ := os.ReadFile(p)
	backup := filepath.Join(d, "committed-backup")
	if err := os.WriteFile(backup, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := j.Record(e); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("rerun rewrote journal")
	}
	r, _ := NewJournal(d)
	if r.evidence[0].CompletedAt != 42 {
		t.Fatal("changed completion time")
	}
}

func TestJournalInvalidFormsRejected(t *testing.T) {
	forms := []string{`{"stage":"bogus","inputDigest":"x"}`, `{"stage":"detected","inputDigest":""}`, `[{"stage":"detected","inputDigest":"x"},{"stage":"detected","inputDigest":"y"}]`, `[{"stage":"artifacts_ready","inputDigest":"x"}]`, `[`}
	for _, raw := range forms {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, "setup-journal.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewJournal(d); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestJournalRejectsUnknownAndTrailingJSON(t *testing.T) {
	valid := `[{"stage":"detected","inputDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]`
	for _, raw := range []string{valid[:len(valid)-1] + `,"digest":"secret"}]`, valid + ` {}`} {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, "setup-journal.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewJournal(d); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestJournalPrivatePermissionsAndLeftoverTemp(t *testing.T) {
	d := t.TempDir()
	j, _ := NewJournal(d)
	if err := j.Record(StageEvidence{Stage: Detected, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err != nil {
		t.Fatal(err)
	}
	if mode := mustStat(t, d).Mode().Perm(); mode != 0700 {
		t.Fatalf("dir mode %o", mode)
	}
	if mode := mustStat(t, filepath.Join(d, "setup-journal.json")).Mode().Perm(); mode != 0600 {
		t.Fatalf("file mode %o", mode)
	}
	if err := os.WriteFile(filepath.Join(d, ".journal-leftover"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := NewJournal(d)
	if err != nil || r.Next() != ArtifactsReady {
		t.Fatalf("reopen: %v", err)
	}
}

func TestJournalPersistenceFailurePreservesCommittedState(t *testing.T) {
	d := t.TempDir()
	j, _ := NewJournal(d)
	if err := j.Record(StageEvidence{Stage: Detected, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "setup-journal.json")
	before, _ := os.ReadFile(p)
	j.path = filepath.Join(d, "missing", "setup-journal.json")
	if err := j.Record(StageEvidence{Stage: ArtifactsReady, InputDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}); err == nil {
		t.Fatal("expected write failure")
	}
	if j.Next() != ArtifactsReady {
		t.Fatal(j.Next())
	}
	after, _ := os.ReadFile(filepath.Join(d, "setup-journal.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("committed bytes changed")
	}
}

func mustStat(t *testing.T, p string) os.FileInfo {
	t.Helper()
	s, e := os.Stat(p)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestCompletedEvidenceResumesAtNextStage(t *testing.T) {
	j := newTestJournal(t)
	if err := j.Record(StageEvidence{Stage: Detected, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err != nil {
		t.Fatal(err)
	}
	if got := j.Next(); got != ArtifactsReady {
		t.Fatalf("got %q", got)
	}
}

func TestJournalRejectsTamperedBindingBeforeMutation(t *testing.T) {
	for _, raw := range []string{
		`{"profile":"development","topology":"external","endpointIdentityDigest":"secret","planDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		`{"profile":"development","topology":"external","artifactDigests":{"lumen":"secret"},"planDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		`{"profile":"development","topology":"external","planDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {}`,
	} {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, "setup-binding.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewJournal(d); !errors.Is(err, ErrInvalidJournal) {
			t.Fatalf("accepted tampered binding: %v", err)
		}
		if _, err := os.Stat(filepath.Join(d, "setup-journal.json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("mutation occurred")
		}
	}
}

func TestJournalBindingReturnsDeepCopy(t *testing.T) {
	d := t.TempDir()
	j, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	binding := JournalBinding{
		Profile:         Development,
		Topology:        TopologyExternal,
		PlanDigest:      "sha256:" + strings.Repeat("a", 64),
		ArtifactDigests: map[string]string{"lumen": "sha256:" + strings.Repeat("b", 64)},
		ArtifactRefs:    map[string]string{"lumen": "ghcr.io/lumen/lumen@sha256:" + strings.Repeat("c", 64)},
	}
	if err := j.Bind(binding); err != nil {
		t.Fatal(err)
	}
	got, ok := j.Binding()
	if !ok {
		t.Fatal("expected binding")
	}
	got.ArtifactDigests["lumen"] = "sha256:" + strings.Repeat("c", 64)
	got.ArtifactDigests["new"] = "sha256:" + strings.Repeat("d", 64)
	got.ArtifactRefs["lumen"] = "ghcr.io/lumen/lumen@sha256:" + strings.Repeat("d", 64)
	got.ArtifactRefs["new"] = "ghcr.io/lumen/new@sha256:" + strings.Repeat("e", 64)
	again, ok := j.Binding()
	if !ok || again.ArtifactDigests["lumen"] != binding.ArtifactDigests["lumen"] || len(again.ArtifactDigests) != 1 || again.ArtifactRefs["lumen"] != binding.ArtifactRefs["lumen"] || len(again.ArtifactRefs) != 1 {
		t.Fatalf("binding was not copied: %#v", again)
	}
}

func TestJournalBindingPersistsSelectedSupervisor(t *testing.T) {
	d := t.TempDir()
	j, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	want := JournalBinding{
		Profile:    Development,
		Topology:   TopologyExternal,
		Supervisor: SupervisorSystemd,
		PlanDigest: "sha256:" + strings.Repeat("a", 64),
	}
	if err := j.Bind(want); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Binding()
	if !ok || got.Supervisor != want.Supervisor {
		t.Fatalf("binding supervisor = %#v, want %q", got, want.Supervisor)
	}
}

func TestJournalBindingPersistsDockerComposeProject(t *testing.T) {
	d := t.TempDir()
	j, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	want := JournalBinding{
		Profile:        Development,
		Topology:       TopologyCombined,
		Supervisor:     SupervisorDocker,
		ComposeProject: "lumen-fixed",
		PlanDigest:     "sha256:" + strings.Repeat("a", 64),
	}
	if err := j.Bind(want); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Binding()
	if !ok || got.ComposeProject != want.ComposeProject {
		t.Fatalf("binding Compose project = %#v, want %q", got, want.ComposeProject)
	}
}

func TestJournalBindingRequiresDockerComposeProject(t *testing.T) {
	j := newTestJournal(t)
	err := j.Bind(JournalBinding{Profile: Development, Topology: TopologyCombined, Supervisor: SupervisorDocker, PlanDigest: "sha256:" + strings.Repeat("a", 64)})
	if !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("got %v", err)
	}
}

func TestJournalBindingUpgradesLegacyDockerComposeProjectOnce(t *testing.T) {
	d := t.TempDir()
	legacy := `{"profile":"development","topology":"combined","supervisor":"docker","planDigest":"sha256:` + strings.Repeat("a", 64) + `"}`
	if err := os.WriteFile(filepath.Join(d, "setup-binding.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	j, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	next := JournalBinding{Profile: Development, Topology: TopologyCombined, Supervisor: SupervisorDocker, ComposeProject: "lumen-adopted", PlanDigest: "sha256:" + strings.Repeat("a", 64)}
	if err := j.Bind(next); err != nil {
		t.Fatalf("upgrade legacy binding: %v", err)
	}
	changed := next
	changed.ComposeProject = "lumen-substituted"
	if err := j.Bind(changed); !errors.Is(err, ErrInputChanged) {
		t.Fatalf("changed upgraded project: %v", err)
	}
}

func TestJournalBindingRejectsUnsupportedSupervisor(t *testing.T) {
	j := newTestJournal(t)
	err := j.Bind(JournalBinding{Profile: Development, Topology: TopologyExternal, Supervisor: Supervisor("other"), PlanDigest: "sha256:" + strings.Repeat("a", 64)})
	if !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("got %v", err)
	}
}
