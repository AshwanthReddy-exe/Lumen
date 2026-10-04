package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

func TestSignedStatusAcrossTwoEndpoints(t *testing.T) {
	hostPublic, hostPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	nodePublic, nodePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1000, 0)
	allowed := true
	node := StatusHandler{SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", HostPublicKey: hostPublic, NodePrivateKey: nodePrivate, LocalAllowed: func() bool { return allowed }, Now: func() time.Time { return now }}
	server := httptest.NewTLSServer(node)
	defer server.Close()
	client := StatusClient{HTTPClient: server.Client(), URL: server.URL, HostPrivateKey: hostPrivate, NodePublicKey: nodePublic, Now: func() time.Time { return now }}
	request := StatusRequest{Version: 1, SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", TaskID: "task-1", CapabilityID: "node.status/read", Action: "read", ActionFingerprint: "status", IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}
	receipt, err := client.Read(context.Background(), request)
	if err != nil || receipt.TaskID != request.TaskID || receipt.NodeID != "desk" || receipt.Status != "online" {
		t.Fatalf("signed node status = %#v, %v", receipt, err)
	}

	for name, mutate := range map[string]func(*StatusRequest){
		"stale epoch":      func(r *StatusRequest) { r.HostEpoch-- },
		"wrong target":     func(r *StatusRequest) { r.NodeID = "other" },
		"wrong capability": func(r *StatusRequest) { r.CapabilityID = "files.read" },
		"expired":          func(r *StatusRequest) { r.ExpiresAt = now.Unix() },
	} {
		t.Run(name, func(t *testing.T) {
			bad := request
			mutate(&bad)
			if _, err := client.Read(context.Background(), bad); err == nil {
				t.Fatal("unauthorized node request succeeded")
			}
		})
	}
	allowed = false
	denied := httptest.NewTLSServer(node)
	defer denied.Close()
	client.URL, client.HTTPClient = denied.URL, denied.Client()
	if _, err := client.Read(context.Background(), request); err == nil {
		t.Fatal("local permission denial was ignored")
	}
}

type statusStore struct {
	state        space.State
	beforeUpdate func()
}

func (s *statusStore) Read() (space.State, error) { return s.state, nil }
func (s *statusStore) Update(fn func(space.State) space.Transition) (space.Transition, error) {
	if s.beforeUpdate != nil {
		s.beforeUpdate()
		s.beforeUpdate = nil
	}
	result := fn(s.state)
	if result.Rejection == "" {
		s.state = result.State
	}
	return result, nil
}

func TestStatusDispatchUsesCurrentHostGrantAndDurableTask(t *testing.T) {
	hostPublic, hostPrivate, _ := ed25519.GenerateKey(rand.Reader)
	nodePublic, nodePrivate, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Unix(1000, 0)
	allowed := true
	server := httptest.NewTLSServer(StatusHandler{SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", HostPublicKey: hostPublic, NodePrivateKey: nodePrivate, LocalAllowed: func() bool { return allowed }, Now: func() time.Time { return now }})
	defer server.Close()
	client := StatusClient{HTTPClient: server.Client(), URL: server.URL, HostPrivateKey: hostPrivate, NodePublicKey: nodePublic, Now: func() time.Time { return now }}
	state := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 3, Nodes: map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}, "desk": {ID: "desk", Status: "paired"}}, Advertisements: []space.CapabilityKey{{NodeID: "desk", CapabilityID: "node.status/read", Action: "read"}}, Grants: map[string]space.Grant{"desk|node.status/read|read": space.GrantAllow}, Tasks: map[string]space.Task{"task-1": {ID: "task-1", OriginNodeID: "owner", TargetNodeID: "desk", CapabilityID: "node.status/read", Action: "read", ActionFingerprint: "status", HostEpoch: 3, Status: space.OutcomeQueued}}}
	store := &statusStore{state: state}
	result, err := DispatchStatus(context.Background(), store, client, "task-1", now)
	if err != nil || result.Receipt.Outcome != space.OutcomeCompleted || store.state.Tasks["task-1"].Status != space.OutcomeCompleted {
		t.Fatalf("durable status dispatch = %#v, %v", result, err)
	}
	task := store.state.Tasks["task-1"]
	task.Status = space.OutcomeQueued
	store.state.Tasks["task-1"] = task
	store.state.Grants["desk|node.status/read|read"] = space.GrantDeny
	if _, err := DispatchStatus(context.Background(), store, client, "task-1", now); err == nil {
		t.Fatal("revoked Host grant still dispatched")
	}
	store.state.Grants["desk|node.status/read|read"] = space.GrantAllow
	allowed = false
	if _, err := DispatchStatus(context.Background(), store, client, "task-1", now); err == nil {
		t.Fatal("revoked local permission still completed")
	}
	allowed = true
	store.beforeUpdate = func() {
		changed := store.state.Tasks["task-1"]
		changed.ActionFingerprint = "different-action"
		store.state.Tasks["task-1"] = changed
	}
	if _, err := DispatchStatus(context.Background(), store, client, "task-1", now); err == nil {
		t.Fatal("receipt for old action completed a changed task")
	}
	if store.state.Tasks["task-1"].Status != space.OutcomeQueued {
		t.Fatal("changed task was completed")
	}
}

func TestStatusAcceptsBoundedClockSkew(t *testing.T) {
	hostPublic, hostPrivate, _ := ed25519.GenerateKey(rand.Reader)
	nodePublic, nodePrivate, _ := ed25519.GenerateKey(rand.Reader)
	hostNow := time.Unix(1000, 0)
	nodeNow := hostNow.Add(-10 * time.Second)
	server := httptest.NewTLSServer(StatusHandler{SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", HostPublicKey: hostPublic, NodePrivateKey: nodePrivate, LocalAllowed: func() bool { return true }, Now: func() time.Time { return nodeNow }})
	defer server.Close()
	client := StatusClient{HTTPClient: server.Client(), URL: server.URL, HostPrivateKey: hostPrivate, NodePublicKey: nodePublic, Now: func() time.Time { return hostNow }}
	request := StatusRequest{Version: 1, SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", TaskID: "task-1", CapabilityID: "node.status/read", Action: "read", ActionFingerprint: "status", IssuedAt: hostNow.Unix(), ExpiresAt: hostNow.Add(time.Minute).Unix()}
	if _, err := client.Read(context.Background(), request); err != nil {
		t.Fatalf("bounded clock skew rejected: %v", err)
	}
}

func TestStatusCompletionSurvivesEncryptedStoreReopen(t *testing.T) {
	hostPublic, hostPrivate, _ := ed25519.GenerateKey(rand.Reader)
	nodePublic, nodePrivate, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Unix(1000, 0)
	server := httptest.NewTLSServer(StatusHandler{SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", HostPublicKey: hostPublic, NodePrivateKey: nodePrivate, LocalAllowed: func() bool { return true }, Now: func() time.Time { return now }})
	defer server.Close()
	client := StatusClient{HTTPClient: server.Client(), URL: server.URL, HostPrivateKey: hostPrivate, NodePublicKey: nodePublic, Now: func() time.Time { return now }}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	statePath, keyPath := filepath.Join(dir, "state.json"), filepath.Join(dir, "state.key")
	ledger, err := store.New(statePath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	state := space.State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 3, Audit: []space.AuditEvent{}, Nodes: map[string]space.Node{"owner": {ID: "owner", Status: "paired"}, "desk": {ID: "desk", Status: "paired"}}, Advertisements: []space.CapabilityKey{{NodeID: "desk", CapabilityID: "node.status/read", Action: "read"}}, Grants: map[string]space.Grant{"desk|node.status/read|read": space.GrantAllow}, Tasks: map[string]space.Task{"task-1": {ID: "task-1", OriginNodeID: "owner", TargetNodeID: "desk", CapabilityID: "node.status/read", Action: "read", ActionFingerprint: "status", HostEpoch: 3, Status: space.OutcomeQueued}}}
	if err := ledger.Initialize(state); err != nil {
		t.Fatal(err)
	}
	if _, err := DispatchStatus(context.Background(), ledger, client, "task-1", now); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.New(statePath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	loaded, err := reopened.Read()
	if err != nil || loaded.Tasks["task-1"].Status != space.OutcomeCompleted {
		t.Fatalf("reopened node task = %#v, %v", loaded.Tasks["task-1"], err)
	}
}

func TestStatusRejectsTamperedSignatureAndUnknownFields(t *testing.T) {
	hostPublic, hostPrivate, _ := ed25519.GenerateKey(rand.Reader)
	_, nodePrivate, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Unix(1000, 0)
	node := StatusHandler{SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", HostPublicKey: hostPublic, NodePrivateKey: nodePrivate, LocalAllowed: func() bool { return true }, Now: func() time.Time { return now }}
	request := StatusRequest{Version: 1, SpaceID: "space", HostID: "host", HostEpoch: 3, NodeID: "desk", TaskID: "task-1", CapabilityID: "node.status/read", Action: "read", ActionFingerprint: "status", IssuedAt: 1000, ExpiresAt: 1060}
	signed, err := SignStatusRequest(request, hostPrivate)
	if err != nil {
		t.Fatal(err)
	}
	signed.Request.NodeID = "other"
	body, _ := json.Marshal(signed)
	response := httptest.NewRecorder()
	httpsRequest := httptest.NewRequest(http.MethodPost, "/v1/node/status", bytes.NewReader(body))
	httpsRequest.TLS = &tls.ConnectionState{}
	node.ServeHTTP(response, httpsRequest)
	if response.Code != http.StatusForbidden {
		t.Fatalf("tampered request status = %d", response.Code)
	}
	body = append(body[:len(body)-1], []byte(`,"unexpected":true}`)...)
	response = httptest.NewRecorder()
	httpsRequest = httptest.NewRequest(http.MethodPost, "/v1/node/status", bytes.NewReader(body))
	httpsRequest.TLS = &tls.ConnectionState{}
	node.ServeHTTP(response, httpsRequest)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d", response.Code)
	}
	tooLarge := append([]byte(`{"request":`), bytes.Repeat([]byte(" "), 16<<10)...)
	response = httptest.NewRecorder()
	httpsRequest = httptest.NewRequest(http.MethodPost, "/v1/node/status", bytes.NewReader(tooLarge))
	httpsRequest.TLS = &tls.ConnectionState{}
	node.ServeHTTP(response, httpsRequest)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized request status = %d", response.Code)
	}
	plain, _ := SignStatusRequest(request, hostPrivate)
	body, _ = json.Marshal(plain)
	response = httptest.NewRecorder()
	node.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/node/status", bytes.NewReader(body)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("plaintext signed request status = %d", response.Code)
	}
}
