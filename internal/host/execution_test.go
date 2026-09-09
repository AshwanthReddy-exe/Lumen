package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

type fakeRuntime struct {
	mu                sync.Mutex
	create            hermes.Run
	createErr         error
	status            []hermes.Run
	statusDefault     hermes.Run
	statusErr         error
	events            []hermes.Event
	eventsErr         error
	stop              hermes.Run
	stopErr           error
	capErr            error
	healthErr         error
	created           int
	eventCalls        int
	statusCalls       int
	stopCalls         int
	beforeCreate      func()
	approvalDecisions []string
	approvalErr       error
	approvalErrors    []error
	beforeApproval    func()
	eventsBlock       <-chan struct{}
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
	if f.beforeCreate != nil {
		f.beforeCreate()
	}
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
		if f.statusDefault.RunID != "" {
			return f.statusDefault, f.statusErr
		}
		return hermes.Run{}, f.statusErr
	}
	r := f.status[0]
	f.status = f.status[1:]
	return r, f.statusErr
}
func (f *fakeRuntime) Events(ctx context.Context, _ string) ([]hermes.Event, error) {
	f.mu.Lock()
	f.eventCalls++
	f.mu.Unlock()
	if f.eventsBlock != nil {
		select {
		case <-f.eventsBlock:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hermes.Event(nil), f.events...), f.eventsErr
}
func (f *fakeRuntime) ResolveApproval(_ context.Context, _ string, decision string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.approvalDecisions = append(f.approvalDecisions, decision)
	if f.beforeApproval != nil {
		before := f.beforeApproval
		f.beforeApproval = nil
		before()
	}
	if len(f.approvalErrors) > 0 {
		err := f.approvalErrors[0]
		f.approvalErrors = f.approvalErrors[1:]
		return err
	}
	return f.approvalErr
}
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

func TestCreateIntentIsDurableBeforeRuntimePOST(t *testing.T) {
	var service *Service
	r := &fakeRuntime{create: hermes.Run{RunID: "run-intent", Status: "started"}}
	r.beforeCreate = func() {
		state, err := service.state.Read()
		if err != nil {
			t.Errorf("read create intent: %v", err)
			return
		}
		intent, ok := state.HostCreates["task-intent"]
		if !ok || intent.IdempotencyKey != "submit-intent" || state.Tasks["task-intent"].Status != space.OutcomeCreating {
			t.Errorf("create intent not durable before POST: %#v task=%#v", intent, state.Tasks["task-intent"])
		}
	}
	service = executionService(t, r)
	req := submitRequest("submit-intent", "task-intent")
	if _, err := service.SubmitTask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func TestCreateIntentRecoversToUnknownWithoutRetryAfterRestart(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: d + "/host.sock", CredentialPath: d + "/operator"}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	r := &fakeRuntime{create: hermes.Run{RunID: "must-not-be-created"}}
	s, err := NewWithRuntime(c, r, WithExecutionTiming(5*time.Millisecond, 35*time.Millisecond), WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	state := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1,
		Nodes:          map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}},
		Advertisements: []space.CapabilityKey{{NodeID: "host", CapabilityID: "agent.run/execute", Action: "run"}},
		Grants:         map[string]space.Grant{"host|agent.run/execute|run": space.GrantAllow}, Audit: []space.AuditEvent{}}
	if _, err := s.state.Update(func(space.State) space.Transition { return space.Transition{State: state} }); err != nil {
		s.Shutdown()
		t.Fatal(err)
	}
	req := submitRequest("submit-crash", "task-crash")
	if tr, err := s.ApplyCommand(req.Submit); err != nil || tr.Rejection != "" {
		s.Shutdown()
		t.Fatalf("submit: %#v %v", tr, err)
	}
	if tr, err := s.ApplyCommand(space.Command{Type: space.CommandCreateHostRun, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "create:submit-crash", TaskID: "task-crash", RuntimeIdempotencyKey: "submit-crash", RuntimeProfileDigest: "sha256:profile", DispatchedAt: 100, ReconcileBy: 130}); err != nil || tr.Rejection != "" {
		s.Shutdown()
		t.Fatalf("create intent: %#v %v", tr, err)
	}
	s.Shutdown()

	recovered, err := NewWithRuntime(c, r, WithExecutionTiming(5*time.Millisecond, 35*time.Millisecond), WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Shutdown()
	task, ok, err := recovered.Task("task-crash")
	if err != nil || !ok {
		t.Fatalf("recovered task: %#v %v %v", task, ok, err)
	}
	if task.Status != space.OutcomeUnknown || task.TerminalReason != "host_restarted" {
		t.Fatalf("create intent recovery = %#v", task)
	}
	if r.created != 0 {
		t.Fatalf("restarted service retried create %d times", r.created)
	}
}

func TestCreateFailureIsDurablyFailedBeforeReply(t *testing.T) {
	r := &fakeRuntime{createErr: fmt.Errorf("%w: invalid request", hermes.ErrCreateRejected)}
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

func TestAmbiguousCreateIsDurablyUnknown(t *testing.T) {
	r := &fakeRuntime{createErr: errors.New("connection reset after send")}
	s := executionService(t, r)
	trn, err := s.SubmitTask(context.Background(), submitRequest("submit-ambiguous", "task-ambiguous"))
	if err != nil || trn.Rejection != "" || trn.Receipt.Outcome != space.OutcomeUnknown {
		t.Fatalf("ambiguous create: %#v %v", trn, err)
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

func TestConcurrentSubmitReplayCreatesOneRun(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-concurrent", Status: "started"}, events: []hermes.Event{{ID: "done", Data: []byte(`{"status":"completed"}`)}}}
	s := executionService(t, r)
	req := submitRequest("submit-concurrent", "task-concurrent")
	results := make(chan error, 2)
	go func() { _, err := s.SubmitTask(context.Background(), req); results <- err }()
	go func() { _, err := s.SubmitTask(context.Background(), req); results <- err }()
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if r.created != 1 {
		t.Fatalf("concurrent replay created %d runs", r.created)
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
	good.ApprovedAt = 100
	good.ObservedAt = 0
	var observed atomic.Int64
	observed.Store(100)
	s.executor.options.now = func() time.Time { return time.Unix(observed.Load(), 0) }
	direct := good
	direct.RequestID = "direct-bypass"
	direct.ApprovedAt = 1
	direct.ExpiresAt = 50
	direct.ObservedAt = 1
	if directResult, err := s.ApplyCommand(direct); err != nil || directResult.Rejection != "approval_expired" {
		t.Fatalf("direct authority bypass: %#v %v", directResult, err)
	}
	tr, err = s.ResolveApproval(context.Background(), ApprovalRequest{Command: good, Runtime: req.Runtime, RuntimeProfileDigest: req.RuntimeProfileDigest, ReconcileBy: req.ReconcileBy})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		t.Fatalf("approval: %#v %v", tr, err)
	}
	observed.Store(110)
	goodRetry := good
	goodRetry.ObservedAt = 0
	if replay, err := s.ResolveApproval(context.Background(), ApprovalRequest{Command: goodRetry, Runtime: req.Runtime, RuntimeProfileDigest: req.RuntimeProfileDigest, ReconcileBy: req.ReconcileBy}); err != nil || !replay.Replayed {
		t.Fatalf("approval retry did not replay stable command: %#v %v", replay, err)
	}
	waitForTask(t, s, "task-ask", space.OutcomeCompleted)
	if r.created != 1 {
		t.Fatalf("created %d runs", r.created)
	}
}

func TestConcurrentFirstApprovalsUseOneStableObservedAt(t *testing.T) {
	r := &fakeRuntime{}
	s := executionService(t, r)
	if _, err := s.state.Update(func(state space.State) space.Transition {
		state.Grants["host|agent.run/execute|run"] = space.GrantAsk
		return space.Transition{State: state}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SubmitTask(context.Background(), submitRequest("submit-concurrent-approval", "task-concurrent-approval")); err != nil {
		t.Fatal(err)
	}
	var clock atomic.Int64
	clock.Store(100)
	s.executor.options.now = func() time.Time { return time.Unix(clock.Add(1), 0) }
	command := space.Command{Type: space.CommandApprove, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "concurrent-approval", TaskID: "task-concurrent-approval", TargetNodeID: "host", ActionFingerprint: "digest", ApprovalID: "concurrent-approval-id", ApprovedAt: 100, ExpiresAt: 200, ObservedAt: 1}
	start := make(chan struct{})
	results := make(chan space.Transition, 2)
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			tr, err := s.ApplyCommand(command)
			results <- tr
			errs <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if first.Replayed == second.Replayed {
		t.Fatalf("expected one first application and one replay: first=%#v second=%#v", first, second)
	}
	state, err := s.state.Read()
	if err != nil {
		t.Fatal(err)
	}
	if state.Commands[command.RequestID].Content == "" || !strings.Contains(state.Commands[command.RequestID].Content, `"observedAt":101`) {
		t.Fatalf("approval did not retain one authority-stamped observation: %s", state.Commands[command.RequestID].Content)
	}
}

func TestInitialApprovalDenyRequiresExactPendingBinding(t *testing.T) {
	r := &fakeRuntime{}
	s := executionService(t, r)
	if _, err := s.state.Update(func(state space.State) space.Transition {
		state.Grants["host|agent.run/execute|run"] = space.GrantAsk
		return space.Transition{State: state}
	}); err != nil {
		t.Fatal(err)
	}
	req := submitRequest("submit-deny", "task-deny")
	if _, err := s.SubmitTask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	bad := req.Submit
	bad.Type = space.CommandApprove
	bad.RequestID = "deny-bad"
	bad.ActorID = "owner"
	bad.ApprovalID = "approval"
	bad.ApprovedAt = 101
	bad.ExpiresAt = 120
	bad.Decision = "deny"
	bad.ActionFingerprint = "wrong"
	expired := bad
	expired.RequestID = "deny-expired"
	expired.ExpiresAt = 99
	expired.ActionFingerprint = "digest"
	if tr, err := s.ResolveApproval(context.Background(), ApprovalRequest{Command: expired}); err != nil || tr.Rejection != "approval_expired" {
		t.Fatalf("expired approval: %#v %v", tr, err)
	}
	if tr, err := s.ResolveApproval(context.Background(), ApprovalRequest{Command: bad}); err != nil || tr.Rejection != "approval_mismatch" {
		t.Fatalf("unbound deny: %#v %v", tr, err)
	}
	good := bad
	good.RequestID = "deny-good"
	good.ActionFingerprint = "digest"
	good.ApprovedAt = 100
	tr, err := s.ResolveApproval(context.Background(), ApprovalRequest{Command: good})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeFailed {
		t.Fatalf("bound deny: %#v %v", tr, err)
	}
	if r.created != 0 {
		t.Fatalf("deny created runtime run %d", r.created)
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
	if replay, err := s.CancelTask(context.Background(), CancelRequest{Command: space.Command{Type: space.CommandRequestHostRunCancellation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "cancel", TaskID: "task-cancel", RuntimeRunID: "run-cancel", RuntimeProfileDigest: "sha256:profile", ObservedAt: 100}}); err != nil || !replay.Replayed || r.stopCalls != 1 {
		t.Fatalf("cancel replay: %#v err=%v stopCalls=%d", replay, err, r.stopCalls)
	}
}

func TestStopFailureReconcilesToUnknownAtBoundedDeadline(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-stop-fail", Status: "started"}, eventsErr: hermes.ErrEventStreamDisconnected, statusErr: errors.New("status unavailable"), stopErr: errors.New("stop unavailable")}
	s := executionService(t, r)
	s.executor.options.now = time.Now
	by := time.Now().Unix() + 2
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

func TestRuntimeApprovalIsBoundAndForwardsOnceOrDeny(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-approval", Status: "started"}, events: []hermes.Event{{ID: "approval", Type: "approval.requested", Data: []byte(`{"status":"awaiting_approval","approval_id":"runtime-1","target_node_id":"host","action_fingerprint":"digest","expires_at":120}`)}}}
	s := executionService(t, r)
	if _, err := s.SubmitTask(context.Background(), submitRequest("submit-runtime-approval", "task-runtime-approval")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if task, ok, _ := s.Task("task-runtime-approval"); ok && task.Status == space.OutcomeAwaitingPermission {
			break
		}
		time.Sleep(time.Millisecond)
	}
	state, _ := s.state.Read()
	if state.Tasks["task-runtime-approval"].Status != space.OutcomeAwaitingPermission {
		t.Fatalf("runtime approval state=%#v audit=%#v commands=%#v", state.Tasks["task-runtime-approval"], state.Audit, state.Commands)
	}
	tr, err := s.ResolveRuntimeApproval(context.Background(), RuntimeApprovalRequest{Command: space.Command{Type: space.CommandResolveRuntimeApproval, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "resolve-runtime-approval", TaskID: "task-runtime-approval", RuntimeRunID: "run-approval", RuntimeProfileDigest: "sha256:profile", RuntimeApprovalID: "runtime-1", TargetNodeID: "host", ActionFingerprint: "digest", Decision: "once", ObservedAt: 110}})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeRunning {
		t.Fatalf("runtime approval: %#v %v", tr, err)
	}
	if len(r.approvalDecisions) != 1 || r.approvalDecisions[0] != "once" {
		t.Fatalf("forwarded approvals=%#v", r.approvalDecisions)
	}
	if replay, err := s.ResolveRuntimeApproval(context.Background(), RuntimeApprovalRequest{Command: space.Command{Type: space.CommandResolveRuntimeApproval, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "resolve-runtime-approval", TaskID: "task-runtime-approval", RuntimeRunID: "run-approval", RuntimeProfileDigest: "sha256:profile", RuntimeApprovalID: "runtime-1", TargetNodeID: "host", ActionFingerprint: "digest", Decision: "once", ObservedAt: 110}}); err != nil || !replay.Replayed || len(r.approvalDecisions) != 1 {
		t.Fatalf("approval replay duplicated delivery: %#v err=%v calls=%d", replay, err, len(r.approvalDecisions))
	}
}

func TestRuntimeApprovalDeliveryFailureRemainsPendingAndRetries(t *testing.T) {
	r := &fakeRuntime{create: hermes.Run{RunID: "run-approval-retry", Status: "started"}, statusDefault: hermes.Run{RunID: "run-approval-retry", Status: "awaiting_approval"}, approvalErrors: []error{errors.New("approval delivery uncertain")}, events: []hermes.Event{{ID: "approval", Type: "approval.requested", Data: []byte(`{"status":"awaiting_approval","approval_id":"runtime-retry","target_node_id":"host","action_fingerprint":"digest","expires_at":120}`)}}}
	s := executionService(t, r)
	req := submitRequest("submit-runtime-retry", "task-runtime-retry")
	if _, err := s.SubmitTask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	waitForTask(t, s, "task-runtime-retry", space.OutcomeAwaitingPermission)
	command := space.Command{Type: space.CommandResolveRuntimeApproval, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "resolve-runtime-retry", TaskID: "task-runtime-retry", RuntimeRunID: "run-approval-retry", RuntimeProfileDigest: "sha256:profile", RuntimeApprovalID: "runtime-retry", TargetNodeID: "host", ActionFingerprint: "digest", Decision: "once", ObservedAt: 110}
	if _, err := s.ResolveRuntimeApproval(context.Background(), RuntimeApprovalRequest{Command: command}); err == nil {
		t.Fatal("expected uncertain approval delivery error")
	}
	if task, _, _ := s.Task("task-runtime-retry"); task.Status != space.OutcomeAwaitingPermission {
		t.Fatalf("uncertain delivery changed task: %#v", task)
	}
	tr, err := s.ResolveRuntimeApproval(context.Background(), RuntimeApprovalRequest{Command: command})
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeRunning {
		t.Fatalf("approval retry: %#v %v", tr, err)
	}
	if len(r.approvalDecisions) != 2 {
		t.Fatalf("approval forward attempts=%d", len(r.approvalDecisions))
	}
}

func TestRuntimeApprovalRetriesAfterDeliveryReceiptPersistenceFailure(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: d + "/host.sock", CredentialPath: d + "/operator"}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	base, err := store.Open(d+"/state.json", d+"/state.key")
	if err != nil {
		t.Fatal(err)
	}
	initial := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1,
		Nodes:            map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}},
		Tasks:            map[string]space.Task{"task": {ID: "task", TargetNodeID: "host", ActionFingerprint: "digest", Status: space.OutcomeAwaitingPermission}},
		HostRuns:         map[string]space.HostRun{"task": {TaskID: "task", RuntimeRunID: "run", RuntimeProfileDigest: "profile", HostEpoch: 1, DispatchedAt: 100, ReconcileBy: 130}},
		RuntimeApprovals: map[string]space.RuntimeApproval{"approval": {ID: "approval", TaskID: "task", RuntimeRunID: "run", TargetNodeID: "host", ActionFingerprint: "digest", ExpiresAt: 120, DeliveryState: "pending"}},
		Audit:            []space.AuditEvent{}}
	if _, err := base.Update(func(space.State) space.Transition { return space.Transition{State: initial} }); err != nil {
		base.Close()
		t.Fatal(err)
	}
	if err := base.Close(); err != nil {
		t.Fatal(err)
	}
	failNext := false
	failing, err := store.NewWithHooks(d+"/state.json", store.Hooks{Sync: func(*os.File) error {
		if failNext {
			failNext = false
			return errors.New("injected approval receipt persistence failure")
		}
		return nil
	}}, d+"/state.key")
	if err != nil {
		t.Fatal(err)
	}
	r := &fakeRuntime{statusDefault: hermes.Run{RunID: "run", Status: "completed"}}
	r.beforeApproval = func() { failNext = true }
	s := &Service{state: failing, ready: make(chan struct{}), stop: make(chan struct{})}
	s.executor = newExecutor(s, r, executionOptions{now: func() time.Time { return time.Unix(110, 0) }, pollInterval: time.Millisecond, reconcileWindow: time.Second})
	t.Cleanup(s.Shutdown)
	command := space.Command{Type: space.CommandResolveRuntimeApproval, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "resolve-receipt-failure", TaskID: "task", RuntimeRunID: "run", RuntimeProfileDigest: "profile", RuntimeApprovalID: "approval", TargetNodeID: "host", ActionFingerprint: "digest", Decision: "once", ObservedAt: 110}
	if _, err := s.ResolveRuntimeApproval(context.Background(), RuntimeApprovalRequest{Command: command}); err == nil {
		t.Fatal("expected delivery receipt persistence failure")
	}
	state, err := failing.Read()
	if err != nil {
		t.Fatal(err)
	}
	if state.RuntimeApprovals["approval"].DeliveryState != "sending" {
		t.Fatalf("failed receipt changed durable delivery state: %#v", state.RuntimeApprovals["approval"])
	}
	tr, err := s.ResolveRuntimeApproval(context.Background(), RuntimeApprovalRequest{Command: command})
	if err != nil || tr.Rejection != "" {
		t.Fatalf("retry after receipt failure: %#v %v", tr, err)
	}
	if len(r.approvalDecisions) != 1 {
		t.Fatalf("approval was blindly resent after receipt failure: %#v", r.approvalDecisions)
	}
	state, err = failing.Read()
	if err != nil {
		t.Fatal(err)
	}
	if state.RuntimeApprovals["approval"].DeliveryState != "delivered" || state.Tasks["task"].Status != space.OutcomeRunning {
		t.Fatalf("retry did not durably resolve approval: %#v task=%#v", state.RuntimeApprovals["approval"], state.Tasks["task"])
	}
}

func TestPendingRuntimeApprovalReattachesConsumerAfterRestart(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: d + "/host.sock", CredentialPath: d + "/operator"}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	r := &fakeRuntime{status: []hermes.Run{{RunID: "run", Status: "awaiting_approval"}}, eventsErr: errors.New("approval stream unavailable"), statusErr: errors.New("status unavailable")}
	s, err := NewWithRuntime(c, r, WithExecutionTiming(time.Millisecond, 50*time.Millisecond), WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	state := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1,
		Nodes:            map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}},
		Tasks:            map[string]space.Task{"task": {ID: "task", TargetNodeID: "host", HostEpoch: 1, Status: space.OutcomeAwaitingPermission}},
		HostRuns:         map[string]space.HostRun{"task": {TaskID: "task", RuntimeRunID: "run", RuntimeProfileDigest: "profile", HostEpoch: 1, DispatchedAt: 100, ReconcileBy: 130}},
		RuntimeApprovals: map[string]space.RuntimeApproval{"approval": {ID: "approval", TaskID: "task", RuntimeRunID: "run", TargetNodeID: "host", ActionFingerprint: "digest", ExpiresAt: 120, Decision: "once", DeliveryState: "uncertain"}}, Audit: []space.AuditEvent{}}
	if _, err := s.state.Update(func(space.State) space.Transition { return space.Transition{State: state} }); err != nil {
		s.Shutdown()
		t.Fatal(err)
	}
	s.Shutdown()
	recovered, err := NewWithRuntime(c, r, WithExecutionTiming(time.Millisecond, 50*time.Millisecond), WithClock(func() time.Time { return time.Unix(100, 0) }))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.approvalDecisions) != 1 {
		t.Fatalf("restart did not retry uncertain approval delivery: %#v", r.approvalDecisions)
	}
	defer recovered.Shutdown()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		calls := r.eventCalls
		r.mu.Unlock()
		if calls > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("pending approval consumer was not reattached")
}

func TestExpiredRunSkipsSSEAndPersistsUnknown(t *testing.T) {
	r := &fakeRuntime{events: []hermes.Event{{ID: "terminal", Data: []byte(`{"status":"completed"}`)}}}
	s := executionService(t, r)
	if _, err := s.state.Update(func(state space.State) space.Transition {
		state.Tasks = map[string]space.Task{}
		state.Tasks["expired"] = space.Task{ID: "expired", TargetNodeID: "host", HostEpoch: 1, Status: space.OutcomeDispatched}
		state.HostRuns = map[string]space.HostRun{"expired": {TaskID: "expired", RuntimeRunID: "run-expired", RuntimeProfileDigest: "profile", HostEpoch: 1, DispatchedAt: 90, ReconcileBy: 99}}
		return space.Transition{State: state}
	}); err != nil {
		t.Fatal(err)
	}
	s.executor.startConsumer("expired", space.HostRun{TaskID: "expired", RuntimeRunID: "run-expired", RuntimeProfileDigest: "profile", HostEpoch: 1, DispatchedAt: 90, ReconcileBy: 99})
	waitForTask(t, s, "expired", space.OutcomeUnknown)
	if r.eventCalls != 0 {
		t.Fatalf("expired run opened SSE %d times", r.eventCalls)
	}
}

func TestSubsecondReconcileDeadlineIsNotRoundedUp(t *testing.T) {
	r := &fakeRuntime{eventsBlock: make(chan struct{})}
	s := executionService(t, r)
	s.executor.options.now = func() time.Time { return time.Unix(100, 900*int64(time.Millisecond)) }
	if _, err := s.state.Update(func(state space.State) space.Transition {
		state.Tasks = map[string]space.Task{"subsecond": {ID: "subsecond", TargetNodeID: "host", HostEpoch: 1, Status: space.OutcomeDispatched}}
		state.HostRuns = map[string]space.HostRun{"subsecond": {TaskID: "subsecond", RuntimeRunID: "run-subsecond", RuntimeProfileDigest: "profile", HostEpoch: 1, DispatchedAt: 100, ReconcileBy: 101}}
		return space.Transition{State: state}
	}); err != nil {
		t.Fatal(err)
	}
	s.executor.startConsumer("subsecond", space.HostRun{TaskID: "subsecond", RuntimeRunID: "run-subsecond", RuntimeProfileDigest: "profile", HostEpoch: 1, DispatchedAt: 100, ReconcileBy: 101})
	waitForTask(t, s, "subsecond", space.OutcomeUnknown)
}

func TestEvidencePersistenceFailureIsNotConsumedOrTerminal(t *testing.T) {
	d := t.TempDir()
	c := Config{DataDir: d, SocketPath: d + "/host.sock", CredentialPath: d + "/operator"}
	if err := Initialize(c); err != nil {
		t.Fatal(err)
	}
	state, err := store.Open(d+"/state.json", d+"/state.key")
	if err != nil {
		t.Fatal(err)
	}
	initial := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1, Nodes: map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}}, Tasks: map[string]space.Task{"task": {ID: "task", TargetNodeID: "host", HostEpoch: 1, Status: space.OutcomeDispatched}}, HostRuns: map[string]space.HostRun{"task": {TaskID: "task", RuntimeRunID: "run", RuntimeProfileDigest: "sha256:p", HostEpoch: 1, DispatchedAt: 100, ReconcileBy: 200}}, Audit: []space.AuditEvent{}}
	if _, err := state.Update(func(space.State) space.Transition { return space.Transition{State: initial} }); err != nil {
		t.Fatal(err)
	}
	_ = state.Close()
	failing, err := store.NewWithHooks(d+"/state.json", store.Hooks{Write: func(*os.File, []byte) error { return errors.New("injected persistence failure") }}, d+"/state.key")
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{state: failing, ready: make(chan struct{}), stop: make(chan struct{})}
	s.executor = newExecutor(s, nil, executionOptions{now: func() time.Time { return time.Unix(110, 0) }, pollInterval: time.Millisecond, reconcileWindow: time.Second})
	t.Cleanup(s.Shutdown)
	if err := s.persistEvidence("task", initial.HostRuns["task"], space.EvidenceCompleted, "failure"); err == nil {
		t.Fatal("expected persistence failure")
	}
	got, err := failing.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got.Tasks["task"].Status != space.OutcomeDispatched {
		t.Fatalf("evidence became terminal despite failed persistence: %#v", got.Tasks["task"])
	}
}

func TestShutdownCancelsAndJoinsConsumer(t *testing.T) {
	block := make(chan struct{})
	r := &fakeRuntime{create: hermes.Run{RunID: "run-shutdown", Status: "started"}, eventsBlock: block}
	s := executionService(t, r)
	if _, err := s.SubmitTask(context.Background(), submitRequest("submit-shutdown", "task-shutdown")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		calls := r.eventCalls
		r.mu.Unlock()
		if calls > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	done := make(chan struct{})
	go func() { s.Shutdown(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not join event consumer")
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
