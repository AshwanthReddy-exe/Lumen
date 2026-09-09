package host

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

func TestConfigFromEnvironment(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SOCKET_PATH", filepath.Join(d, "host.sock"))
	t.Setenv("LUMEN_OPERATOR_CREDENTIAL_FILE", filepath.Join(d, "operator"))
	t.Setenv("LUMEN_HERMES_BASE_URL", "https://hermes.example.test")
	t.Setenv("LUMEN_HERMES_PROFILE", "hardened")
	t.Setenv("LUMEN_HERMES_BEARER_FILE", filepath.Join(d, "hermes.token"))
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != d {
		t.Fatalf("data dir %q", c.DataDir)
	}
	if c.HermesProfile != "hardened" || c.HermesBaseURL != "https://hermes.example.test" || c.HermesBearerPath == "" {
		t.Fatalf("Hermes config not loaded: %#v", c)
	}
}

func TestProductionNewBuildsNonNilRuntimeAdapter(t *testing.T) {
	d := t.TempDir()
	tokenPath := filepath.Join(d, "hermes.token")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator"), HermesBaseURL: "http://127.0.0.1:1", HermesProfile: "development", HermesBearerPath: tokenPath}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	s, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	if s.executor == nil || s.executor.runtime == nil {
		t.Fatal("production constructor left Hermes runtime nil")
	}
}

func TestProductionNewFailsReadinessForInvalidHermesConfiguration(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator"), HermesProfile: hermes.ProfileHardened}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	if _, err := New(c); err == nil {
		t.Fatal("expected invalid Hermes configuration to fail readiness")
	}
}

func TestHermesSecretReadRejectsSymlink(t *testing.T) {
	d := t.TempDir()
	realPath := filepath.Join(d, "real.token")
	linkPath := filepath.Join(d, "link.token")
	if err := os.WriteFile(realPath, []byte("token"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatal(err)
	}
	_, err := buildHermesClient(Config{HermesBaseURL: "http://127.0.0.1:1", HermesProfile: hermes.ProfileDevelopment, HermesBearerPath: linkPath})
	if err == nil {
		t.Fatal("expected symlinked Hermes secret to be rejected")
	}
}

func TestInitIsCreateOnly(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err == nil {
		t.Fatal("expected create-only failure")
	}
	if _, err := os.Stat(c.CredentialPath); err != nil {
		t.Fatal(err)
	}
}

func TestInitBootstrapsDurableSpaceAndFreshTaskFlow(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	stateStore, err := store.Open(filepath.Join(d, "state.json"), filepath.Join(d, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := stateStore.Read()
	stateStore.Close()
	if err != nil {
		t.Fatal(err)
	}
	if state.SpaceID == "" || state.OwnerID == "" || state.HostID == "" || state.OwnerID == state.HostID || state.Epoch != 1 {
		t.Fatalf("bootstrap identities: %#v", state)
	}
	if len(state.Identities) != 2 || state.Nodes[state.OwnerID].Status != "paired" || state.Nodes[state.HostID].Status != "paired" {
		t.Fatalf("bootstrap pairing: %#v", state)
	}
	if len(state.Advertisements) != 1 || state.Advertisements[0] != (space.CapabilityKey{NodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run"}) {
		t.Fatalf("bootstrap advertisement: %#v", state.Advertisements)
	}
	if state.Grants[state.HostID+"|agent.run/execute|run"] != space.GrantAsk {
		t.Fatalf("bootstrap grant: %#v", state.Grants)
	}
	for _, requestID := range []string{"bootstrap:create-space", "bootstrap:advertise-execute", "bootstrap:grant-execute"} {
		if _, ok := state.Commands[requestID]; !ok {
			t.Fatalf("bootstrap command missing: %s", requestID)
		}
	}
	if len(state.Audit) < 3 {
		t.Fatalf("bootstrap audit incomplete: %#v", state.Audit)
	}
	runtime := &fakeRuntime{}
	s, err := NewWithRuntime(c, runtime, WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	tr, err := s.SubmitTask(context.Background(), ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: 1, ActorID: state.HostID, RequestID: "fresh-submit", TaskID: "fresh-task", OriginNodeID: state.OwnerID, TargetNodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "digest"}, RuntimeProfileDigest: "profile", ReconcileBy: 130})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeAwaitingPermission {
		t.Fatalf("fresh task flow: %#v %v", tr, err)
	}
}

func TestInitResumesCompleteBootstrapWithoutMarker(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(c.CredentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(d, "initialized")); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	after, err := os.ReadFile(c.CredentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("resume replaced the operator credential")
	}
	if _, err := os.Stat(filepath.Join(d, "initialized")); err != nil {
		t.Fatal(err)
	}
}

func TestInitResumesCompleteBootstrapWithMissingCredential(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(d, "initialized")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(c.CredentialPath); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err != nil {
		t.Fatalf("resume with missing credential failed: %v", err)
	}
	credential, err := control.ReadCredential(c.CredentialPath)
	if err != nil || len(credential) != control.CredentialSize {
		t.Fatalf("recreated credential: %v", err)
	}
}

func TestInitRejectsPartialBootstrapWithoutMutation(t *testing.T) {
	d := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	stateStore, err := store.New(filepath.Join(d, "state.json"), filepath.Join(d, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	if err := stateStore.Initialize(space.State{SchemaVersion: 1, Audit: []space.AuditEvent{}}); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err == nil {
		t.Fatal("expected incomplete bootstrap to fail closed")
	}
	if exists(filepath.Join(d, "initialized")) || exists(c.CredentialPath) {
		t.Fatal("partial bootstrap was finalized")
	}
	stateStore, err = store.Open(filepath.Join(d, "state.json"), filepath.Join(d, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := stateStore.Read()
	stateStore.Close()
	if err != nil {
		t.Fatal(err)
	}
	if state.SpaceID != "" || len(state.Commands) != 0 {
		t.Fatalf("partial state was mutated: %#v", state)
	}
}

func TestInitCanRetryAfterPostStoreFailure(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	if err := control.WriteCredential(c.CredentialPath, make([]byte, control.CredentialSize)); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err == nil {
		t.Fatal("expected credential creation failure")
	}
	if _, err := os.Stat(filepath.Join(c.DataDir, "initialized")); !os.IsNotExist(err) {
		t.Fatalf("marker exists after failed init: %v", err)
	}
	if err := os.Remove(c.CredentialPath); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(c); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
}

func TestInitRollsBackWhenMarkerDurabilityFails(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: filepath.Join(d, "host.sock"), CredentialPath: filepath.Join(d, "operator")}
	originalFile, originalParent := markerSyncFile, markerSyncParent
	defer func() { markerSyncFile, markerSyncParent = originalFile, originalParent }()
	markerSyncFile = func(*os.File) error { return errors.New("injected marker file sync failure") }
	if err := Initialize(c); err == nil {
		t.Fatal("expected marker file sync failure")
	}
	for _, path := range []string{"state.json", "state.key", "initialized", "operator"} {
		if _, err := os.Lstat(filepath.Join(d, path)); !os.IsNotExist(err) {
			t.Fatalf("artifact survived marker file sync failure: %s (%v)", path, err)
		}
	}
	markerSyncFile = originalFile
	markerSyncParent = func(string) error { return errors.New("injected marker parent sync failure") }
	if err := Initialize(c); err == nil {
		t.Fatal("expected marker parent sync failure")
	}
	for _, path := range []string{"state.json", "state.key", "initialized", "operator"} {
		if _, err := os.Lstat(filepath.Join(d, path)); !os.IsNotExist(err) {
			t.Fatalf("artifact survived marker parent sync failure: %s (%v)", path, err)
		}
	}
	markerSyncParent = originalParent
	if err := Initialize(c); err != nil {
		t.Fatalf("retry after durability failures failed: %v", err)
	}
}
