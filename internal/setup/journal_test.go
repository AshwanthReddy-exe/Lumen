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
	reopened, err := NewJournal(d)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Next() != Validated {
		t.Fatal(reopened.Next())
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
