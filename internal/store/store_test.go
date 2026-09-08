package store

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

func testStore(t *testing.T) (*Store, string, string) {
	t.Helper()
	d := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(d, "state.json"), filepath.Join(d, "state.key")
	s, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, state, key
}

func testState() space.State {
	return space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Audit: []space.AuditEvent{}}
}

func TestInitializeIsCreateOnlyAndRoundTrips(t *testing.T) {
	s, statePath, keyPath := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(testState()); err == nil {
		t.Fatal("expected create-only initialization")
	}
	got, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !stateEqual(got, testState()) {
		t.Fatalf("round trip mismatch: %#v", got)
	}
	for _, path := range []string{statePath, keyPath} {
		if st, err := os.Stat(path); err != nil || st.Mode().Perm() != 0600 {
			t.Fatalf("%s mode: %v %v", path, st, err)
		}
	}
	key, err := os.ReadFile(keyPath)
	if err != nil || len(key) != 32 {
		t.Fatalf("key: %d bytes, %v", len(key), err)
	}
}

func TestUnsupportedStateSchemaIsRejectedWithoutMutation(t *testing.T) {
	s, statePath, _ := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, statePath)
	if err := s.Initialize(space.State{SchemaVersion: 2}); err == nil {
		t.Fatal("future schema initialized")
	}
	if got := mustRead(t, statePath); string(got) != string(before) {
		t.Fatal("failed initialization changed state")
	}
	if _, err := s.Update(func(space.State) space.Transition {
		return space.Transition{State: space.State{SchemaVersion: 2}}
	}); err == nil {
		t.Fatal("future schema update committed")
	}
	if got, err := s.Read(); err != nil || !stateEqual(got, testState()) {
		t.Fatalf("committed state not preserved: %#v %v", got, err)
	}
}

func TestUpdatePersistsTransitionAndUsesFreshNonce(t *testing.T) {
	s, statePath, _ := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	tr, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "updated"
		return space.Transition{State: st, Receipt: space.Receipt{RequestID: "r"}}
	})
	if err != nil || tr.State.SpaceID != "updated" {
		t.Fatalf("update: %#v %v", tr, err)
	}
	first, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	tr, err = s.Update(func(st space.State) space.Transition {
		st.OwnerID = "changed"
		return space.Transition{State: st}
	})
	if err != nil || tr.State.OwnerID != "changed" {
		t.Fatalf("second update: %#v %v", tr, err)
	}
	second, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var a, b Envelope
	if err := json.Unmarshal(first, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &b); err != nil {
		t.Fatal(err)
	}
	if a.Nonce == b.Nonce {
		t.Fatal("nonce was reused")
	}
	if _, err := base64.RawStdEncoding.DecodeString(a.Nonce); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataAndCiphertextAreAuthenticated(t *testing.T) {
	s, statePath, _ := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Envelope){
		func(e *Envelope) { e.Metadata["tampered"] = "true" },
		func(e *Envelope) { e.CipherSuite = "AES-128-GCM" },
		func(e *Envelope) { e.KeyID = "wrong" },
	} {
		var e Envelope
		if err := json.Unmarshal(b, &e); err != nil {
			t.Fatal(err)
		}
		mutate(&e)
		changed, _ := json.Marshal(e)
		if err := os.WriteFile(statePath, changed, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Read(); err == nil {
			t.Fatal("tampered envelope was accepted")
		}
		if err := os.WriteFile(statePath, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadRejectsMissingWrongKeyAndInvalidEnvelope(t *testing.T) {
	s, statePath, keyPath := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	key, _ := os.ReadFile(keyPath)
	wrong := append([]byte(nil), key...)
	wrong[0] ^= 1
	if err := os.WriteFile(keyPath, wrong, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatal("wrong key accepted")
	}
	if err := os.Remove(keyPath); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatal("missing key accepted")
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, key, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(keyPath, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatalf("permissive key: %v", err)
	}
	if err := os.WriteFile(keyPath, make([]byte, 32), 0600); err != nil {
		t.Fatal(err)
	}
	for _, content := range [][]byte{nil, []byte("{}"), []byte(`{"formatVersion":2}`), append(mustRead(t, statePath), []byte("x")...)} {
		if err := os.WriteFile(statePath, content, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Read(); err == nil {
			t.Fatal("invalid envelope accepted")
		}
	}
}

func TestLockContentionIsNonBlocking(t *testing.T) {
	d := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(d, "state.json"), filepath.Join(d, "state.key")
	first, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := New(state, key)
	if err == nil || second != nil {
		t.Fatal("expected lock contention")
	}
}

func TestFailedCommitPreservesLastCommittedState(t *testing.T) {
	d := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(d, "state.json"), filepath.Join(d, "state.key")
	s, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	s.hooks.Write = func(*os.File, []byte) error { return errors.New("injected write failure") }
	if _, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "lost"
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected write failure")
	}
	if got, _ := os.ReadFile(state); string(got) != string(original) {
		t.Fatal("write failure changed committed state")
	}

	s.hooks.Write = nil
	s.hooks.Sync = func(*os.File) error { return errors.New("injected sync failure") }
	if _, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "lost"
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected sync failure")
	}
	if got, _ := os.ReadFile(state); string(got) != string(original) {
		t.Fatal("sync failure changed committed state")
	}

	s.hooks.Sync = nil
	s.hooks.Rename = func(string, string) error { return errors.New("injected rename failure") }
	if _, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "lost"
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected rename failure")
	}
	if got, _ := os.ReadFile(state); string(got) != string(original) {
		t.Fatal("rename failure changed committed state")
	}

	s.hooks.Rename = nil
	syncCalls := 0
	s.hooks.DirSync = func(f *os.File) error {
		syncCalls++
		if syncCalls == 1 {
			return errors.New("injected directory sync failure")
		}
		return f.Sync()
	}
	if _, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "lost"
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected directory sync failure")
	}
	if got, err := s.Read(); err != nil || string(mustRead(t, state)) != string(original) || !stateEqual(got, testState()) {
		t.Fatalf("directory sync failure did not preserve state: %#v %v", got, err)
	}
	if entries, err := os.ReadDir(d); err != nil {
		t.Fatal(err)
	} else {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".state.tmp-") {
				t.Fatalf("temporary file remains: %s", entry.Name())
			}
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got, err := reopened.Read(); err != nil || !stateEqual(got, testState()) {
		t.Fatalf("reopened state not preserved: %#v %v", got, err)
	}
}

func TestPersistentRollbackSyncFailureRetainsRecoveryAndReopens(t *testing.T) {
	s, statePath, keyPath := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	s.hooks.DirSync = func(*os.File) error { return errors.New("persistent directory sync failure") }
	if _, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "must-not-commit"
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected persistent directory sync failure")
	}
	if got, err := s.Read(); err != nil || !stateEqual(got, testState()) {
		t.Fatalf("active state changed after rollback failure: %#v %v", got, err)
	}
	recovery := false
	for _, entry := range mustReadDir(t, filepath.Dir(statePath)) {
		if strings.HasSuffix(entry.Name(), ".recovery") {
			recovery = true
		}
	}
	if !recovery {
		t.Fatal("persistent rollback failure discarded recovery material")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(statePath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got, err := reopened.Read(); err != nil || !stateEqual(got, testState()) {
		t.Fatalf("reopened state changed after rollback failure: %#v %v", got, err)
	}
}

func TestCleanupFailureDoesNotReportCommittedUpdateAsFailure(t *testing.T) {
	s, statePath, keyPath := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	s.hooks.Remove = func(string) error { return errors.New("injected cleanup failure") }
	tr, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "committed"
		return space.Transition{State: st}
	})
	if err != nil || tr.State.SpaceID != "committed" {
		t.Fatalf("committed update reported failure: %#v %v", tr, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(statePath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got, err := reopened.Read(); err != nil || got.SpaceID != "committed" {
		t.Fatalf("committed state unavailable after cleanup failure: %#v %v", got, err)
	}
}

func TestReadRejectsTrailingDataBeyondLimit(t *testing.T) {
	s, statePath, _ := testStore(t)
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	b := append(mustRead(t, statePath), bytes.Repeat([]byte{' '}, 8<<20+1)...)
	if err := os.WriteFile(statePath, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatal("oversized trailing data accepted")
	}
}

func TestCloseSerializesWithUpdateAndRejectsAfterClose(t *testing.T) {
	d := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(d, "state.json"), filepath.Join(d, "state.key")
	started, release := make(chan struct{}), make(chan struct{})
	s, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	s.hooks.Write = func(f *os.File, b []byte) error {
		close(started)
		<-release
		_, err := f.Write(b)
		return err
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.Update(func(st space.State) space.Transition {
			st.SpaceID = "blocked"
			return space.Transition{State: st}
		})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("update did not enter write hook")
	}
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	select {
	case <-closed:
		t.Fatal("close released lock during update")
	case <-time.After(50 * time.Millisecond):
	}
	if contender, err := New(state, key); err == nil || contender != nil {
		t.Fatal("second store acquired lock during update")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatal("read succeeded after close")
	}
	if contender, err := New(state, key); err != nil {
		t.Fatal(err)
	} else {
		contender.Close()
	}
}

func TestStoreRejectsSymlinkAndPermissivePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	actual := filepath.Join(root, "actual")
	if err := os.Mkdir(actual, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(actual, link); err != nil {
		t.Fatal(err)
	}
	if s, err := New(filepath.Join(link, "state.json"), filepath.Join(link, "state.key")); err == nil || s != nil {
		t.Fatal("symlink parent accepted")
	}
	if err := os.Chmod(actual, 0750); err != nil {
		t.Fatal(err)
	}
	if s, err := New(filepath.Join(actual, "state.json"), filepath.Join(actual, "state.key")); err == nil || s != nil {
		t.Fatal("permissive parent accepted")
	}
}

func TestOperationsStayOnOpenedParentAfterPathReplacement(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "data")
	if err := os.Mkdir(data, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(data, "state.json"), filepath.Join(data, "state.key")
	s, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "moved")
	if err := os.Rename(data, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(data, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(func(st space.State) space.Transition {
		st.SpaceID = "opened-parent"
		return space.Transition{State: st}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("replacement parent was modified: %v", err)
	}
	if got, err := s.Read(); err != nil || got.SpaceID != "opened-parent" {
		t.Fatalf("opened parent state unavailable: %#v %v", got, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLockRemainsBoundToOpenedParentAfterRename(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "data")
	if err := os.Mkdir(data, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(data, "state.json"), filepath.Join(data, "state.key")
	first, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if err := first.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "moved")
	if err := os.Rename(data, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(data, 0700); err != nil {
		t.Fatal(err)
	}
	second, err := New(filepath.Join(data, "state.json"), filepath.Join(data, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	second.Close()
	if _, err := first.Read(); err != nil {
		t.Fatalf("opened parent lock/data became unusable after rename: %v", err)
	}
}

func TestUnrelatedRelativeIOIsNotRedirectedDuringStoreOperation(t *testing.T) {
	working := t.TempDir()
	data := filepath.Join(working, "data")
	if err := os.Mkdir(data, 0700); err != nil {
		t.Fatal(err)
	}
	state, key := filepath.Join(data, "state.json"), filepath.Join(data, "state.key")
	s, err := New(state, key)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Initialize(testState()); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	s.hooks.Write = func(f *os.File, b []byte) error {
		close(started)
		<-release
		_, err := f.Write(b)
		return err
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(working); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(original)
	updateDone := make(chan error, 1)
	go func() {
		_, err := s.Update(func(st space.State) space.Transition {
			st.SpaceID = "relative-safe"
			return space.Transition{State: st}
		})
		updateDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("write hook was not entered")
	}
	if err := os.WriteFile("unrelated.txt", []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-updateDone; err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(working, "unrelated.txt")); err != nil || string(got) != "outside" {
		t.Fatalf("relative I/O was redirected: %q %v", got, err)
	}
}

func stateEqual(a, b space.State) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustReadDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
