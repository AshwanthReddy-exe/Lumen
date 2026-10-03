package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

func TestPlaintextCanaryAbsentFromStoreAndRecoveryArtifacts(t *testing.T) {
	s, statePath, _ := testStore(t)
	canary := "e4-plaintext-canary-9b8b3e67"
	state := testState()
	state.SpaceID = canary
	if err := s.Initialize(state); err != nil {
		t.Fatal(err)
	}

	// Exercise a partially written temporary envelope and confirm cleanup.
	s.hooks.Write = func(f *os.File, data []byte) error {
		if _, err := f.Write(data); err != nil {
			return err
		}
		return errors.New("injected write failure after full write")
	}
	if _, err := s.Update(func(st space.State) space.Transition {
		st.Epoch++
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected injected write failure")
	}

	// Force a rollback recovery link so the scan covers retained recovery data.
	s.hooks.Write = nil
	dirSyncCalls := 0
	s.hooks.DirSync = func(f *os.File) error {
		dirSyncCalls++
		if dirSyncCalls > 1 {
			return errors.New("injected directory sync failure")
		}
		return f.Sync()
	}
	if _, err := s.Update(func(st space.State) space.Transition {
		st.Epoch++
		return space.Transition{State: st}
	}); err == nil {
		t.Fatal("expected injected directory sync failure")
	}

	entries, err := os.ReadDir(filepath.Dir(statePath))
	if err != nil {
		t.Fatal(err)
	}
	foundRecovery := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".recovery" {
			foundRecovery = true
		}
		data, err := os.ReadFile(filepath.Join(filepath.Dir(statePath), entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(canary)) {
			t.Fatalf("plaintext canary found in store artifact %q", entry.Name())
		}
	}
	if !foundRecovery {
		t.Fatal("test did not produce a retained recovery artifact")
	}
	if got, err := s.Read(); err != nil || got.SpaceID != canary {
		t.Fatalf("rollback did not preserve readable canary state: %#v %v", got, err)
	}
}
