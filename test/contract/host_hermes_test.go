package contract

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

type runtime struct{}

func (runtime) Capabilities(context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{Object: "capabilities", Platform: "fake", Auth: hermes.CapabilityAuth{Type: "bearer", Required: true}, Features: map[string]bool{hermes.CapabilityRunSubmission: true, hermes.CapabilityRunStatus: true, hermes.CapabilityRunEvents: true, hermes.CapabilityRunApproval: true, hermes.CapabilityRunStop: true}}, nil
}
func (runtime) Health(context.Context) (hermes.Health, error) {
	return hermes.Health{Status: "ok"}, nil
}
func (runtime) CreateRun(context.Context, hermes.CreateRunRequest, string) (hermes.Run, error) {
	return hermes.Run{RunID: "contract-run", Status: "started"}, nil
}
func (runtime) RunStatus(context.Context, string) (hermes.Run, error) {
	return hermes.Run{RunID: "contract-run", Status: "completed"}, nil
}
func (runtime) Events(context.Context, string) ([]hermes.Event, error) {
	return []hermes.Event{{ID: "terminal", Type: "completed", Data: []byte(`{"status":"completed"}`)}}, hermes.ErrEventStreamDisconnected
}
func (runtime) ResolveApproval(context.Context, string, string) error    { return nil }
func (runtime) Steer(context.Context, string, hermes.SteerRequest) error { return nil }
func (runtime) Stop(context.Context, string) (hermes.Run, error) {
	return hermes.Run{RunID: "contract-run", Status: "cancelled"}, nil
}

func TestHostHermesDurableAllowRun(t *testing.T) {
	d := t.TempDir()
	c := host.Config{DataDir: d, SocketPath: d + "/host.sock", CredentialPath: d + "/operator"}
	if err := host.Initialize(c); err != nil {
		t.Fatal(err)
	}
	stateStore, err := store.Open(filepath.Join(d, "state.json"), filepath.Join(d, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := stateStore.Read()
	if closeErr := stateStore.Close(); err != nil {
		t.Fatal(err)
	} else if closeErr != nil {
		t.Fatal(closeErr)
	}
	s, err := host.NewWithRuntime(c, runtime{}, host.WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	if tr, err := s.ApplyCommand(space.Command{Type: space.CommandSetGrant, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, NodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", Grant: space.GrantAllow, RequestID: "contract:grant"}); err != nil || tr.Rejection != "" {
		t.Fatalf("setup grant: %#v %v", tr, err)
	}
	tr, err := s.SubmitTask(context.Background(), host.ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "submit", TaskID: "task", OriginNodeID: state.OwnerID, TargetNodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "digest"}, Runtime: hermes.CreateRunRequest{Input: "contract"}, RuntimeProfileDigest: "sha256:contract", ReconcileBy: 130})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		t.Fatalf("submit: %#v %v", tr, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		task, ok, readErr := s.Task("task")
		if readErr == nil && ok && task.Status == space.OutcomeCompleted {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("terminal evidence was not durably accepted")
}
