package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

func resolvedContractTempDir(t *testing.T, pattern string) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	d, err := os.MkdirTemp(root, pattern)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

func TestSetupCreatedDeploymentCompletesBoundedRun(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	runtime := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/capabilities":
			_, _ = w.Write([]byte(`{"object":"hermes.api_server.capabilities","platform":"test","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval_response":true,"run_stop":true}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs":
			_ = json.NewEncoder(w).Encode(hermes.Run{RunID: "setup-run", Status: "started"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/runs/setup-run":
			_ = json.NewEncoder(w).Encode(hermes.Run{RunID: "setup-run", Status: "completed", Output: "synthetic marker"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/runs/setup-run/events":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("id: terminal\nevent: completed\ndata: {\"status\":\"completed\",\"output\":\"synthetic marker\"}\n\n"))
		default:
			http.NotFound(w, r)
		}
	})}
	go runtime.Serve(listener)
	defer runtime.Close()
	runtimeURL := "http://" + listener.Addr().String()

	d := resolvedContractTempDir(t, "lumen-setup-contract-")
	paths, err := setup.WriteConfig(setup.ConfigRequest{
		DataDir: d, HermesDir: filepath.Join(d, "hermes"), Profile: setup.Development,
		HermesBaseURL: runtimeURL, HermesBearer: "setup-test-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := host.ConfigFromFile(paths.Lumen)
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Initialize(c); err != nil {
		t.Fatal(err)
	}
	stateStore, err := store.Open(filepath.Join(d, "state.json"), filepath.Join(d, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := stateStore.Read()
	_ = stateStore.Close()
	if err != nil {
		t.Fatal(err)
	}
	s, err := host.New(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()

	if tr, err := s.ApplyCommand(space.Command{Type: space.CommandSetGrant, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, NodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", Grant: space.GrantAllow, RequestID: "setup:grant"}); err != nil || tr.Rejection != "" {
		t.Fatalf("grant: %#v %v", tr, err)
	}
	got, err := s.SubmitTask(context.Background(), host.ExecuteRequest{
		Submit:  space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "setup:submit", TaskID: "setup:task", OriginNodeID: state.OwnerID, TargetNodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "marker"},
		Runtime: hermes.CreateRunRequest{Input: "return the synthetic marker"}, RuntimeProfileDigest: "sha256:setup", ReconcileBy: time.Now().Add(time.Second).Unix(),
	})
	if err != nil || got.Rejection != "" {
		t.Fatalf("submit: %#v %v", got, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		task, ok, readErr := s.Task("setup:task")
		if readErr == nil && ok && task.Status == space.OutcomeCompleted {
			return
		}
		time.Sleep(time.Millisecond)
	}
	task, _, _ := s.Task("setup:task")
	t.Fatalf("got %#v, want completed", task)
}

type configuredRunReply struct {
	status int
	body   string
}

type configuredRunServer struct {
	mu             sync.Mutex
	server         *http.Server
	listener       net.Listener
	events         string
	eventsGate     <-chan struct{}
	statusReplies  []configuredRunReply
	statusDefault  configuredRunReply
	stop           hermes.Run
	createCalls    int
	eventCalls     int
	approvalChoice []string
	createBody     string
	stopHook       func()
	stopCalls      int
	unauthorized   int
}

func newConfiguredRunServer(t *testing.T, events string, statusReplies []configuredRunReply, statusDefault configuredRunReply) *configuredRunServer {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	r := &configuredRunServer{events: events, statusReplies: statusReplies, statusDefault: statusDefault, stop: hermes.Run{RunID: "configured-run", Status: "cancelled"}, listener: listener}
	r.server = &http.Server{Handler: http.HandlerFunc(r.handle)}
	go func() { _ = r.server.Serve(listener) }()
	t.Cleanup(func() {
		_ = r.server.Close()
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.unauthorized != 0 {
			t.Errorf("Hermes received %d unauthenticated requests", r.unauthorized)
		}
	})
	return r
}

func (r *configuredRunServer) handle(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Authorization") != "Bearer setup-test-token" {
		r.mu.Lock()
		r.unauthorized++
		r.mu.Unlock()
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch {
	case req.Method == http.MethodGet && req.URL.Path == "/health":
		r.writeJSON(w, http.StatusOK, `{"status":"ok","version":"1.0.0"}`)
	case req.Method == http.MethodGet && req.URL.Path == "/v1/capabilities":
		r.writeJSON(w, http.StatusOK, `{"object":"hermes.api_server.capabilities","platform":"test","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval_response":true,"run_stop":true,"browser_extension_control":true,"run_steer":true}}`)
	case req.Method == http.MethodPost && req.URL.Path == "/v1/runs":
		body, _ := io.ReadAll(req.Body)
		r.mu.Lock()
		r.createCalls++
		r.createBody = string(body)
		r.mu.Unlock()
		r.writeJSON(w, http.StatusOK, `{"run_id":"configured-run","status":"started"}`)
	case req.Method == http.MethodGet && req.URL.Path == "/v1/runs/configured-run":
		r.mu.Lock()
		reply := r.statusDefault
		if len(r.statusReplies) > 0 {
			reply, r.statusReplies = r.statusReplies[0], r.statusReplies[1:]
		}
		r.mu.Unlock()
		r.writeJSON(w, reply.status, reply.body)
	case req.Method == http.MethodGet && req.URL.Path == "/v1/runs/configured-run/events":
		r.mu.Lock()
		r.eventCalls++
		gate := r.eventsGate
		events := r.events
		r.mu.Unlock()
		if gate != nil {
			<-gate
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(events))
	case req.Method == http.MethodPost && req.URL.Path == "/v1/runs/configured-run/approval":
		body, _ := io.ReadAll(req.Body)
		var choice struct {
			Choice string `json:"choice"`
		}
		if json.Unmarshal(body, &choice) != nil || (choice.Choice != "once" && choice.Choice != "deny") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.mu.Lock()
		r.approvalChoice = append(r.approvalChoice, choice.Choice)
		r.mu.Unlock()
		r.writeJSON(w, http.StatusOK, `{"object":"hermes.run.approval_response","run_id":"configured-run","choice":"`+choice.Choice+`","resolved":1}`)
	case req.Method == http.MethodPost && req.URL.Path == "/v1/runs/configured-run/stop":
		if r.stopHook != nil {
			r.stopHook()
		}
		r.mu.Lock()
		r.stopCalls++
		stop := r.stop
		r.mu.Unlock()
		r.writeJSON(w, http.StatusOK, mustJSON(stop))
	default:
		http.NotFound(w, req)
	}
}

func (r *configuredRunServer) writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func configuredFixture(t *testing.T, runtime *configuredRunServer) (*host.Service, space.State, host.Config) {
	t.Helper()
	d := resolvedContractTempDir(t, "lumen-hermes-contract-")
	paths, err := setup.WriteConfig(setup.ConfigRequest{DataDir: d, HermesDir: filepath.Join(d, "hermes"), Profile: setup.Development, HermesBaseURL: "http://" + runtime.listener.Addr().String(), HermesBearer: "setup-test-token"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := host.ConfigFromFile(paths.Lumen)
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Initialize(c); err != nil {
		t.Fatal(err)
	}
	stateStore, err := store.Open(filepath.Join(d, "state.json"), filepath.Join(d, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := stateStore.Read()
	closeErr := stateStore.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("read bootstrap state: %v %v", err, closeErr)
	}
	s, err := host.New(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Shutdown)
	return s, state, c
}

func allowConfiguredRun(t *testing.T, s *host.Service, state space.State, requestID string) {
	t.Helper()
	if tr, err := s.ApplyCommand(space.Command{Type: space.CommandSetGrant, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, NodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", Grant: space.GrantAllow, RequestID: requestID}); err != nil || tr.Rejection != "" {
		t.Fatalf("grant: %#v %v", tr, err)
	}
}

func submitConfiguredRun(t *testing.T, s *host.Service, state space.State, requestID, taskID string, reconcileBy int64) {
	t.Helper()
	tr, err := s.SubmitTask(context.Background(), host.ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: requestID, TaskID: taskID, OriginNodeID: state.OwnerID, TargetNodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "bounded"}, Runtime: hermes.CreateRunRequest{Input: "bounded task input"}, RuntimeProfileDigest: "sha256:configured", ReconcileBy: reconcileBy})
	if err != nil || tr.Rejection != "" {
		t.Fatalf("submit: %#v %v", tr, err)
	}
}

func waitConfiguredTask(t *testing.T, s *host.Service, taskID string, want space.Outcome) space.Task {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		task, ok, err := s.Task(taskID)
		if err == nil && ok && task.Status == want {
			return task
		}
		time.Sleep(time.Millisecond)
	}
	task, _, _ := s.Task(taskID)
	t.Fatalf("task %s did not reach %s: %#v", taskID, want, task)
	return space.Task{}
}

func waitConfiguredApproval(t *testing.T, s *host.Service, taskID string) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if task, ok, err := s.Task(taskID); err == nil && ok && task.Status == space.OutcomeAwaitingPermission {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("runtime approval was not durably recorded")
}

func waitConfiguredEvents(t *testing.T, runtime *configuredRunServer, want int) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		runtime.mu.Lock()
		got := runtime.eventCalls
		runtime.mu.Unlock()
		if got >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("events calls did not reach %d", want)
}

func resolveConfiguredApproval(t *testing.T, s *host.Service, state space.State, taskID, approvalID, decision, requestID string) {
	t.Helper()
	tr, err := s.ResolveRuntimeApproval(context.Background(), host.RuntimeApprovalRequest{Command: space.Command{
		Type: space.CommandResolveRuntimeApproval, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch,
		ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, RuntimeRunID: "configured-run", RuntimeProfileDigest: "sha256:configured",
		RuntimeApprovalID: approvalID, TargetNodeID: state.HostID, ActionFingerprint: "bounded", Decision: decision, ObservedAt: time.Now().Unix(),
	}})
	if err != nil || tr.Rejection != "" {
		t.Fatalf("resolve %s: %#v %v", decision, tr, err)
	}
}

func cancelConfiguredRun(t *testing.T, s *host.Service, state space.State, taskID, requestID string) space.Transition {
	t.Helper()
	tr, err := s.CancelTask(context.Background(), host.CancelRequest{Command: space.Command{
		Type: space.CommandRequestHostRunCancellation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch,
		ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, RuntimeRunID: "configured-run", RuntimeProfileDigest: "sha256:configured", ObservedAt: time.Now().Unix(),
	}})
	if err != nil {
		t.Fatalf("cancel: %#v %v", tr, err)
	}
	return tr
}

func readConfiguredState(t *testing.T, c host.Config) space.State {
	t.Helper()
	stateStore, err := store.Open(filepath.Join(c.DataDir, "state.json"), filepath.Join(c.DataDir, "state.key"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := stateStore.Read()
	closeErr := stateStore.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("read state: %v %v", err, closeErr)
	}
	return state
}

func approvalEvent(id, target string, expiresAt int64) string {
	return fmt.Sprintf("id: %s\nevent: approval.requested\ndata: {\"status\":\"awaiting_approval\",\"approval_id\":\"%s\",\"target_node_id\":\"%s\",\"action_fingerprint\":\"bounded\",\"expires_at\":%d}\n\n", id, id, target, expiresAt)
}

func TestConfiguredHermesRunsCertification(t *testing.T) {
	completed := configuredRunReply{status: http.StatusOK, body: `{"run_id":"configured-run","status":"completed","output":"done"}`}
	running := configuredRunReply{status: http.StatusOK, body: `{"run_id":"configured-run","status":"running"}`}
	unavailable := configuredRunReply{status: http.StatusServiceUnavailable, body: `{}`}

	t.Run("approval_once", func(t *testing.T) {
		by := time.Now().Add(5 * time.Second).Unix()
		runtime := newConfiguredRunServer(t, "", nil, completed)
		s, state, _ := configuredFixture(t, runtime)
		runtime.events = approvalEvent("runtime-approval-once", state.HostID, by-1)
		allowConfiguredRun(t, s, state, "approval-once:grant")
		submitConfiguredRun(t, s, state, "approval-once:submit", "approval-once:task", by)
		waitConfiguredApproval(t, s, "approval-once:task")
		resolveConfiguredApproval(t, s, state, "approval-once:task", "runtime-approval-once", "once", "approval-once:resolve")
		waitConfiguredTask(t, s, "approval-once:task", space.OutcomeCompleted)
		runtime.mu.Lock()
		choices, creates := append([]string(nil), runtime.approvalChoice...), runtime.createCalls
		runtime.mu.Unlock()
		if len(choices) != 1 || choices[0] != "once" || creates != 1 {
			t.Fatalf("approval=%v creates=%d, want one once and one create", choices, creates)
		}
	})

	t.Run("approval_deny", func(t *testing.T) {
		by := time.Now().Add(5 * time.Second).Unix()
		failed := configuredRunReply{status: http.StatusOK, body: `{"run_id":"configured-run","status":"failed","output":"denied"}`}
		runtime := newConfiguredRunServer(t, "", nil, failed)
		s, state, _ := configuredFixture(t, runtime)
		runtime.events = approvalEvent("runtime-approval-deny", state.HostID, by-1)
		allowConfiguredRun(t, s, state, "approval-deny:grant")
		submitConfiguredRun(t, s, state, "approval-deny:submit", "approval-deny:task", by)
		waitConfiguredApproval(t, s, "approval-deny:task")
		resolveConfiguredApproval(t, s, state, "approval-deny:task", "runtime-approval-deny", "deny", "approval-deny:resolve")
		waitConfiguredTask(t, s, "approval-deny:task", space.OutcomeFailed)
		runtime.mu.Lock()
		choices, creates := append([]string(nil), runtime.approvalChoice...), runtime.createCalls
		runtime.mu.Unlock()
		if len(choices) != 1 || choices[0] != "deny" || creates != 1 {
			t.Fatalf("approval=%v creates=%d, want one deny and one create", choices, creates)
		}
	})

	t.Run("cancel_terminal_ordering", func(t *testing.T) {
		gate := make(chan struct{})
		runtime := newConfiguredRunServer(t, "", nil, running)
		runtime.eventsGate = gate
		s, state, _ := configuredFixture(t, runtime)
		var sawCancelling bool
		runtime.stopHook = func() {
			task, ok, err := s.Task("cancel:task")
			sawCancelling = err == nil && ok && task.Status == space.OutcomeCancelling
		}
		allowConfiguredRun(t, s, state, "cancel:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "cancel:submit", "cancel:task", by)
		waitConfiguredEvents(t, runtime, 1)
		cancelConfiguredRun(t, s, state, "cancel:task", "cancel:request")
		if !sawCancelling {
			t.Fatal("Hermes stop was requested before durable cancelling state")
		}
		close(gate)
		waitConfiguredTask(t, s, "cancel:task", space.OutcomeCancelled)
	})

	t.Run("terminal_wins_over_late_cancel", func(t *testing.T) {
		runtime := newConfiguredRunServer(t, "id: terminal\nevent: completed\ndata: {\"status\":\"completed\"}\n\n", nil, completed)
		s, state, _ := configuredFixture(t, runtime)
		allowConfiguredRun(t, s, state, "terminal:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "terminal:submit", "terminal:task", by)
		waitConfiguredTask(t, s, "terminal:task", space.OutcomeCompleted)
		tr := cancelConfiguredRun(t, s, state, "terminal:task", "terminal:cancel")
		if tr.Rejection == "" {
			t.Fatal("late cancel was accepted after terminal completion")
		}
		runtime.mu.Lock()
		stops := runtime.stopCalls
		runtime.mu.Unlock()
		if stops != 0 {
			t.Fatalf("stop calls=%d, want 0", stops)
		}
	})

	t.Run("duplicate_reordered_events", func(t *testing.T) {
		events := "id: 2\nevent: running\ndata: {\"status\":\"running\"}\n\nid: 1\nevent: queued\ndata: {\"status\":\"queued\"}\n\nid: 2\nevent: running\ndata: {\"status\":\"running\"}\n\nid: 3\nevent: completed\ndata: {\"status\":\"completed\"}\n\nid: 4\nevent: failed\ndata: {\"status\":\"failed\"}\n\n"
		runtime := newConfiguredRunServer(t, events, nil, completed)
		s, state, _ := configuredFixture(t, runtime)
		allowConfiguredRun(t, s, state, "events:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "events:submit", "events:task", by)
		waitConfiguredTask(t, s, "events:task", space.OutcomeCompleted)
		runtime.mu.Lock()
		calls, creates := runtime.eventCalls, runtime.createCalls
		runtime.mu.Unlock()
		if calls != 1 || creates != 1 {
			t.Fatalf("events calls=%d creates=%d, want one stream and one create", calls, creates)
		}
	})

	t.Run("lost_stream_status_recovery", func(t *testing.T) {
		runtime := newConfiguredRunServer(t, "id: progress\nevent: running\ndata: {\"status\":\"running\"}\n\n", nil, completed)
		s, state, _ := configuredFixture(t, runtime)
		allowConfiguredRun(t, s, state, "recovery:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "recovery:submit", "recovery:task", by)
		waitConfiguredTask(t, s, "recovery:task", space.OutcomeCompleted)
		runtime.mu.Lock()
		creates := runtime.createCalls
		runtime.mu.Unlock()
		if creates != 1 {
			t.Fatalf("create calls=%d, want one after stream loss", creates)
		}
	})

	t.Run("lost_stream_deadline_unknown_outcome", func(t *testing.T) {
		runtime := newConfiguredRunServer(t, "id: progress\nevent: running\ndata: {\"status\":\"running\"}\n\n", nil, unavailable)
		s, state, _ := configuredFixture(t, runtime)
		allowConfiguredRun(t, s, state, "unknown:grant")
		by := time.Now().Add(2 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "unknown:submit", "unknown:task", by)
		waitConfiguredTask(t, s, "unknown:task", space.OutcomeUnknown)
		runtime.mu.Lock()
		creates := runtime.createCalls
		runtime.mu.Unlock()
		if creates != 1 {
			t.Fatalf("create calls=%d, want one before unknown outcome", creates)
		}
	})

	t.Run("host_restart_without_second_create", func(t *testing.T) {
		gate := make(chan struct{})
		runtime := newConfiguredRunServer(t, "", nil, completed)
		runtime.eventsGate = gate
		first, state, c := configuredFixture(t, runtime)
		allowConfiguredRun(t, first, state, "restart:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, first, state, "restart:submit", "restart:task", by)
		waitConfiguredEvents(t, runtime, 1)
		first.Shutdown()
		close(gate)
		second, err := host.New(c)
		if err != nil {
			t.Fatal(err)
		}
		defer second.Shutdown()
		waitConfiguredTask(t, second, "restart:task", space.OutcomeCompleted)
		runtime.mu.Lock()
		creates := runtime.createCalls
		runtime.mu.Unlock()
		if creates != 1 {
			t.Fatalf("create calls=%d, want one across Host restart", creates)
		}
	})

	t.Run("hermes_outage_recovery_without_redispatch", func(t *testing.T) {
		runtime := newConfiguredRunServer(t, "", []configuredRunReply{unavailable, completed}, completed)
		s, state, _ := configuredFixture(t, runtime)
		allowConfiguredRun(t, s, state, "outage:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "outage:submit", "outage:task", by)
		waitConfiguredTask(t, s, "outage:task", space.OutcomeCompleted)
		runtime.mu.Lock()
		creates := runtime.createCalls
		runtime.mu.Unlock()
		if creates != 1 {
			t.Fatalf("create calls=%d, want no redispatch after outage", creates)
		}
	})

	t.Run("discovery_no_grant", func(t *testing.T) {
		runtime := newConfiguredRunServer(t, "", nil, completed)
		s, state, c := configuredFixture(t, runtime)
		s.Shutdown()
		durable := readConfiguredState(t, c)
		grantKey := state.HostID + "|agent.run/execute|run"
		if len(durable.Grants) != 1 || durable.Grants[grantKey] != space.GrantAsk {
			t.Fatalf("grants after capability discovery: %#v", durable.Grants)
		}
		if len(durable.Advertisements) != 1 {
			t.Fatalf("advertisements after capability discovery: %#v", durable.Advertisements)
		}
	})

	t.Run("task_scoped_request_isolation", func(t *testing.T) {
		runtime := newConfiguredRunServer(t, "id: terminal\nevent: completed\ndata: {\"status\":\"completed\"}\n\n", nil, completed)
		s, state, _ := configuredFixture(t, runtime)
		allowConfiguredRun(t, s, state, "isolation:grant")
		by := time.Now().Add(5 * time.Second).Unix()
		submitConfiguredRun(t, s, state, "isolation:submit", "isolation:task", by)
		waitConfiguredTask(t, s, "isolation:task", space.OutcomeCompleted)
		runtime.mu.Lock()
		body := runtime.createBody
		runtime.mu.Unlock()
		var payload map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			t.Fatalf("create payload: %v", err)
		}
		if string(payload["input"]) != `"bounded task input"` {
			t.Fatalf("input payload=%s", payload["input"])
		}
		lower := strings.ToLower(body)
		for _, forbidden := range []string{"state", "grant", "credential", "operator", "snapshot"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("task request leaked %q: %s", forbidden, body)
			}
		}
	})
}
