package contract

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

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

	d, err := os.MkdirTemp("/private/tmp", "lumen-setup-contract-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)
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
