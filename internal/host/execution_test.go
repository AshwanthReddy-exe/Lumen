package host

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

type fakeRuntime struct {
	mu          sync.Mutex
	create      hermes.Run
	createErr   error
	status      []hermes.Run
	statusErr   error
	events      []hermes.Event
	eventsErr   error
	stop        hermes.Run
	stopErr     error
	capErr      error
	healthErr   error
	created     int
	eventCalls  int
	statusCalls int
	stopCalls   int
}

func (f *fakeRuntime) Capabilities(context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{Object: "capabilities", Platform: "fake", Auth: hermes.CapabilityAuth{Type: "bearer", Required: true}, Features: map[string]bool{
		hermes.CapabilityRunSubmission: true, hermes.CapabilityRunStatus: true, hermes.CapabilityRunEvents: true, hermes.CapabilityRunApproval: true, hermes.CapabilityRunStop: true,
	}}, f.capErr
}
func (f *fakeRuntime) Health(context.Context) (hermes.Health, error) {
	return hermes.Health{Status: "ok"}, f.healthErr
}
func (f *fakeRuntime) CreateRun(context.Context, hermes.CreateRunRequest, string) (hermes.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created++
	return f.create, f.createErr
}
func (f *fakeRuntime) RunStatus(context.Context, string) (hermes.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusCalls++
	if len(f.status) == 0 {
		return hermes.Run{}, f.statusErr
	}
	r := f.status[0]
	f.status = f.status[1:]
	return r, f.statusErr
}
func (f *fakeRuntime) Events(context.Context, string) ([]hermes.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eventCalls++
	return append([]hermes.Event(nil), f.events...), f.eventsErr
}
func (f *fakeRuntime) ResolveApproval(context.Context, string, string) error    { return nil }
func (f *fakeRuntime) Steer(context.Context, string, hermes.SteerRequest) error { return nil }
func (f *fakeRuntime) Stop(context.Context, string) (hermes.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	return f.stop, f.stopErr
}

func executionService(t *testing.T, runtime hermes.Adapter) *Service {
	t.Helper()
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: d + "/host.sock", CredentialPath: d + "/operator"}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	s, err := NewWithRuntime(c, runtime, WithExecutionTiming(5*time.Millisecond, 35*time.Millisecond), WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Shutdown() })
	state := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1,
		Nodes:          map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}},
		Advertisements: []space.CapabilityKey{{NodeID: "host", CapabilityID: "agent.run/execute", Action: "run"}},
		Grants:         map[string]space.Grant{"host|agent.run/execute|run": space.GrantAllow}, Audit: []space.AuditEvent{}}
	if _, err := s.state.Update(func(space.State) space.Transition { return space.Transition{State: state} }); err != nil {
		t.Fatal(err)
	}
	return s
}

func submitRequest(id, task string) ExecuteRequest {
	return ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: id, TaskID: task, OriginNodeID: "owner", TargetNodeID: "host", CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "digest"}, Runtime: hermes.CreateRunRequest{Input: "hello"}, RuntimeProfileDigest: "sha256:profile", ReconcileBy: 130}
}

func TestDurableExecutionPersistsDispatchBeforeSSEAndFirstTerminalWins(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-1", Status: "started"}, events: []hermes.Event{
		{ID: "1", Type: "progress", Data: []byte(`{"status":"running"}`)},
		{ID: "2", Type: "completed", Data: []byte(`{"status":"completed"}`)},
		{ID: "2", Type: "failed", Data: []byte(`{"status":"failed"}`)},
	}, eventsErr: hermes.ErrEventStreamDisconnected}
	s := executionService(t, r)
	tr, err := s.SubmitTask(context.Background(), submitRequest("submit-1", "task-1"))
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		t.Fatalf("submit: %#v %v", tr, err)
	}
	waitForTask(t, s, "task-1", space.OutcomeCompleted)
	state, err := s.state.Read()
	if err != nil {
		t.Fatal(err)
	}
	if state.HostRuns["task-1"].RuntimeRunID != "run-1" || r.eventCalls != 1 {
		t.Fatalf("dispatch/event ownership: state=%#v calls=%d", state.HostRuns, r.eventCalls)
	}
	if got := state.Tasks["task-1"].Status; got != space.OutcomeCompleted {
		t.Fatalf("first terminal outcome = %q", got)
	}
}

func TestCreateFailureIsDurablyFailedBeforeReply(t *testing.T) {
	r := &fakeRuntime{createErr: errors.New("runtime unavailable")}
	s := executionService(t, r)
	trn, err := s.SubmitTask(context.Background(), submitRequest("submit-2", "task-2"))
	if err != nil || trn.Rejection != "" || trn.Receipt.Outcome != space.OutcomeFailed {
		t.Fatalf("create failure: %#v %v", trn, err)
	}
	state, err := s.state.Read()
	if err != nil {
		t.Fatal(err)
	}
	if state.Tasks["task-2"].Status != space.OutcomeFailed || r.created != 1 {
		t.Fatalf("failure was not durable: %#v", state.Tasks["task-2"])
	}
}

func TestDuplicateSubmitReturnsDurableReceiptWithoutCreatingAnotherRun(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-duplicate", Status: "started"}, events: []hermes.Event{{ID: "done", Data: []byte(`{"status":"completed"}`)}}}
	s := executionService(t, r)
	req := submitRequest("submit-duplicate", "task-duplicate")
	first, err := s.SubmitTask(context.Background(), req)
	if err != nil || first.Receipt.Outcome != space.OutcomeQueued {
		t.Fatalf("first submit: %#v %v", first, err)
	}
	second, err := s.SubmitTask(context.Background(), req)
	if err != nil || second.Receipt != first.Receipt {
		t.Fatalf("replay: %#v %#v %v", first, second, err)
	}
	waitForTask(t, s, "task-duplicate", space.OutcomeCompleted)
	if r.created != 1 {
		t.Fatalf("duplicate created %d runs", r.created)
	}
}

func TestAskRequiresExactApprovalAndStartsOnlyOnce(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-ask", Status: "started"}, events: []hermes.Event{{ID: "done", Data: []byte(`{"status":"completed"}`)}}}
	s := executionService(t, r)
	if _, err := s.state.Update(func(state space.State) space.Transition {
		state.Grants["host|agent.run/execute|run"] = space.GrantAsk
		return space.Transition{State: state}
	}); err != nil {
		t.Fatal(err)
	}
	req := submitRequest("submit-ask", "task-ask")
	tr, err := s.SubmitTask(context.Background(), req)
	if err != nil || tr.Receipt.Outcome != space.OutcomeAwaitingPermission || r.created != 0 {
		t.Fatalf("ask submit: %#v %v creates=%d", tr, err, r.created)
	}
	bad := space.Command{Type: space.CommandApprove, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "approve-bad", TaskID: "task-ask", TargetNodeID: "host", ActionFingerprint: "wrong", ApprovalID: "approval", ApprovedAt: 101, ExpiresAt: 120}
	if got, err := s.ResolveApproval(context.Background(), ApprovalRequest{Command: bad}); err != nil || got.Rejection != "approval_mismatch" {
		t.Fatalf("stale approval: %#v %v", got, err)
	}
	good := bad
	good.RequestID = "approve-good"
	good.ActionFingerprint = "digest"
	tr, err = s.ResolveApproval(context.Background(), ApprovalRequest{Command: good, Runtime: req.Runtime, RuntimeProfileDigest: req.RuntimeProfileDigest, ReconcileBy: req.ReconcileBy})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		t.Fatalf("approval: %#v %v", tr, err)
	}
	waitForTask(t, s, "task-ask", space.OutcomeCompleted)
	if r.created != 1 {
		t.Fatalf("created %d runs", r.created)
	}
}

func TestCancellationPersistsBeforeStopAndTerminalRaceWins(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-cancel", Status: "started"}, events: []hermes.Event{{ID: "done", Data: []byte(`{"status":"completed"}`)}}, stop: hermes.Run{RunID: "run-cancel", Status: "cancelled"}}
	s := executionService(t, r)
	if _, err := s.SubmitTask(context.Background(), submitRequest("submit-cancel", "task-cancel")); err != nil {
		t.Fatal(err)
	}
	tr, err := s.CancelTask(context.Background(), CancelRequest{Command: space.Command{Type: space.CommandRequestHostRunCancellation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "cancel", TaskID: "task-cancel", RuntimeRunID: "run-cancel", RuntimeProfileDigest: "sha256:profile", ObservedAt: 100}})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeCancelling {
		t.Fatalf("cancel: %#v %v", tr, err)
	}
	waitForTerminal(t, s, "task-cancel")
	state, err := s.state.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Tasks["task-cancel"].Status; got != space.OutcomeCompleted && got != space.OutcomeCancelled {
		t.Fatalf("unexpected race result %q", got)
	}
	if r.stopCalls != 1 {
		t.Fatalf("stop calls=%d", r.stopCalls)
	}
}

func TestStopFailureReconcilesToUnknownAtBoundedDeadline(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-stop-fail", Status: "started"}, eventsErr: hermes.ErrEventStreamDisconnected, statusErr: errors.New("status unavailable"), stopErr: errors.New("stop unavailable")}
	s := executionService(t, r)
	s.executor.options.now = time.Now
	by := time.Now().Unix() + 1
	if _, err := s.SubmitTask(context.Background(), func() ExecuteRequest {
		request := submitRequest("submit-stop-fail", "task-stop-fail")
		request.ReconcileBy = by
		return request
	}()); err != nil {
		t.Fatal(err)
	}
	tr, err := s.CancelTask(context.Background(), CancelRequest{Command: space.Command{Type: space.CommandRequestHostRunCancellation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "cancel-stop-fail", TaskID: "task-stop-fail", RuntimeRunID: "run-stop-fail", RuntimeProfileDigest: "sha256:profile", ObservedAt: time.Now().Unix()}})
	if err != nil || tr.Rejection != "" {
		t.Fatalf("cancel with stop failure: %#v %v", tr, err)
	}
	waitForTask(t, s, "task-stop-fail", space.OutcomeUnknown)
	if r.stopCalls != 1 {
		t.Fatalf("stop calls=%d", r.stopCalls)
	}
}

func TestDisconnectReconcilesThenUnknownAtDeadline(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-disconnect", Status: "started"}, eventsErr: hermes.ErrEventStreamDisconnected,
		status: []hermes.Run{{RunID: "run-disconnect", Status: "running"}, {RunID: "run-disconnect", Status: "failed"}}}
	s := executionService(t, r)
	if _, err := s.SubmitTask(context.Background(), submitRequest("submit-disconnect", "task-disconnect")); err != nil {
		t.Fatal(err)
	}
	waitForTask(t, s, "task-disconnect", space.OutcomeFailed)
	if r.statusCalls < 2 {
		t.Fatalf("status reconciliation calls=%d", r.statusCalls)
	}

	r2 := &fakeRuntime{create: hermes.Run{RunID: "run-unknown", Status: "started"}, eventsErr: hermes.ErrEventStreamDisconnected, statusErr: errors.New("still unavailable")}
	s2 := executionService(t, r2)
	var tick int64 = 100
	s2.executor.options.now = func() time.Time { tick++; return time.Unix(tick, 0) }
	req := submitRequest("submit-unknown", "task-unknown")
	req.ReconcileBy = 101
	if _, err := s2.SubmitTask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	waitForTask(t, s2, "task-unknown", space.OutcomeUnknown)
}

func waitForTask(t *testing.T, s *Service, task string, want space.Outcome) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, err := s.state.Read()
		if err == nil && state.Tasks[task].Status == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	state, _ := s.state.Read()
	t.Fatalf("task %s did not reach %s: %#v", task, want, state.Tasks[task])
}

func waitForTerminal(t *testing.T, s *Service, task string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, err := s.state.Read()
		if err == nil {
			switch state.Tasks[task].Status {
			case space.OutcomeCompleted, space.OutcomeFailed, space.OutcomeCancelled, space.OutcomeUnknown:
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("task did not reach terminal state")
}
