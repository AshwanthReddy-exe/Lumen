package contract

import (
	"context"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
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
	s, err := host.NewWithRuntime(c, runtime{}, host.WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	for _, command := range []space.Command{
		{Type: space.CommandCreateSpace, SpaceID: "space", OwnerID: "owner", HostID: "host", RequestID: "create"},
		{Type: space.CommandAdvertiseCapability, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", NodeID: "host", CapabilityID: "agent.run/execute", Action: "run", RequestID: "advertise"},
		{Type: space.CommandSetGrant, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", NodeID: "host", CapabilityID: "agent.run/execute", Action: "run", Grant: space.GrantAllow, RequestID: "grant"},
	} {
		if tr, err := s.ApplyCommand(command); err != nil || tr.Rejection != "" {
			t.Fatalf("setup %s: %#v %v", command.Type, tr, err)
		}
	}
	tr, err := s.SubmitTask(context.Background(), host.ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "submit", TaskID: "task", OriginNodeID: "owner", TargetNodeID: "host", CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "digest"}, Runtime: hermes.CreateRunRequest{Input: "contract"}, RuntimeProfileDigest: "sha256:contract", ReconcileBy: 130})
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
