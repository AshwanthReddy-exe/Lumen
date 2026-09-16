package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

type memoryStore struct{ state space.State }

func (m *memoryStore) Read() (space.State, error) { return m.state, nil }
func (m *memoryStore) Update(fn func(space.State) space.Transition) (space.Transition, error) {
	tr := fn(m.state)
	if tr.Rejection == "" {
		m.state = tr.State
	}
	return tr, nil
}

type fakeRuntime struct {
	created int
	before  func()
	run     hermes.Run
	req     hermes.CreateRunRequest
}

func (f *fakeRuntime) Capabilities(context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{Features: map[string]bool{hermes.CapabilityRunSubmission: true}}, nil
}
func (f *fakeRuntime) Health(context.Context) (hermes.Health, error) {
	return hermes.Health{Status: "ok"}, nil
}
func (f *fakeRuntime) CreateRun(_ context.Context, req hermes.CreateRunRequest, _ string) (hermes.Run, error) {
	if f.before != nil {
		f.before()
	}
	f.created++
	f.req = req
	return f.run, nil
}
func (f *fakeRuntime) RunStatus(context.Context, string) (hermes.Run, error)    { return f.run, nil }
func (f *fakeRuntime) Events(context.Context, string) ([]hermes.Event, error)   { return nil, nil }
func (f *fakeRuntime) ResolveApproval(context.Context, string, string) error    { return nil }
func (f *fakeRuntime) Steer(context.Context, string, hermes.SteerRequest) error { return nil }
func (f *fakeRuntime) Stop(context.Context, string) (hermes.Run, error)         { return f.run, nil }

type fakeCertifier struct {
	cert space.RuntimeCertification
	err  error
}

func (f fakeCertifier) Certify(context.Context, space.State, space.RuntimeProfile) (space.RuntimeCertification, error) {
	return f.cert, f.err
}

func TestPublicConversationRequestsRejectRuntimeKnobs(t *testing.T) {
	svc := NewService(&memoryStore{state: conversationState()}, &fakeRuntime{}, nil)
	for name, req := range map[string]SendRequest{
		"instructions": {RequestID: "r1", ConversationID: "c1", SurfaceID: "web", Input: "hello", Instructions: "override"},
		"session":      {RequestID: "r2", ConversationID: "c1", SurfaceID: "web", Input: "hello", SessionID: "session-x"},
		"provider":     {RequestID: "r3", ConversationID: "c1", SurfaceID: "web", Input: "hello", Provider: "other"},
		"model":        {RequestID: "r4", ConversationID: "c1", SurfaceID: "web", Input: "hello", Model: "other"},
		"profile":      {RequestID: "r5", ConversationID: "c1", SurfaceID: "web", Input: "hello", RuntimeProfileDigest: "sha256:caller"},
	} {
		_, err := svc.Send(context.Background(), req)
		if !errors.Is(err, ErrPublicRuntimeOverride) {
			t.Fatalf("%s: err=%v, want %v", name, err, ErrPublicRuntimeOverride)
		}
	}
}

func TestProjectionIsBoundedDeterministicAndSelectsAcceptedPreferences(t *testing.T) {
	s := conversationState()
	s.ContextRecords["accepted"] = space.ContextRecord{ID: "accepted", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "owner", Classification: "private", AcceptedAt: 1, Digest: space.DigestText(`{"preferred_name":"Ada"}`), Payload: json.RawMessage(`{"preferred_name":"Ada"}`)}
	s.ContextRecords["unaccepted"] = space.ContextRecord{ID: "unaccepted", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "runtime", Classification: "private", AcceptedAt: 0, Digest: space.DigestText(`{"name":"Mallory"}`), Payload: json.RawMessage(`{"name":"Mallory"}`)}
	for i := 0; i < space.MaxProjectedMessages+3; i++ {
		task := "task-" + string(rune('a'+i))
		tr := space.Apply(s, space.Command{Type: space.CommandSendConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "c1", SurfaceID: "web", TaskID: task, Content: "message-" + task, CreatedAt: int64(i + 1), ReconcileBy: int64(i + 100), RequestID: task})
		if tr.Rejection != "" {
			break
		}
		s = tr.State
		tr = space.Apply(s, space.Command{Type: space.CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: task, Outcome: space.OutcomeCompleted, Output: "answer-" + task, CreatedAt: int64(i + 1), RequestID: "done-" + task})
		s = tr.State
	}
	one, err := Project(s, "c1", space.DefaultPersona(), space.DefaultRuntimeProfile())
	if err != nil {
		t.Fatal(err)
	}
	two, err := Project(s, "c1", space.DefaultPersona(), space.DefaultRuntimeProfile())
	if err != nil || !reflect.DeepEqual(one, two) {
		t.Fatalf("projection not deterministic: one=%#v two=%#v err=%v", one, two, err)
	}
	if len(one.Messages) > space.MaxProjectedMessages || len(one.Context) > space.MaxProjectedContextBytes {
		t.Fatalf("projection bounds exceeded: messages=%d context=%d", len(one.Messages), len(one.Context))
	}
	if strings.Contains(one.Context, "Mallory") || !strings.Contains(one.Context, "Ada") {
		t.Fatalf("preference selection leaked or omitted values: %q", one.Context)
	}
}

func TestProjectionContextNeverExceedsByteBudget(t *testing.T) {
	records := map[string]space.ContextRecord{
		"a": {ID: "a", Namespace: "user.preferences/v1", Provenance: "owner", AcceptedAt: 1, Payload: json.RawMessage(`{"locale":"en"}`)},
		"b": {ID: "b", Namespace: "user.preferences/v1", Provenance: "owner", AcceptedAt: 1, Payload: json.RawMessage(`{"locale":"en"}`)},
	}
	const budget = 74
	if contextText := projectedPreferences(records, budget); len(contextText) > budget {
		t.Fatalf("projected context bytes=%d, want at most %d: %q", len(contextText), budget, contextText)
	}
}

func TestSendPersistsBeforeIOAndFailsClosedWithoutExactCertification(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "hello"}}
	runtime.before = func() {
		if len(store.state.Messages["c1"]) != 1 || store.state.Tasks["task-1"].Status != space.OutcomeQueued {
			t.Fatal("runtime called before canonical user intent was durable")
		}
	}
	svc := NewService(store, runtime, fakeCertifier{err: ErrRuntimeProfileUnverified}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if !errors.Is(err, ErrRuntimeProfileUnverified) || runtime.created != 0 {
		t.Fatalf("uncertified send: err=%v creates=%d", err, runtime.created)
	}
	if got := store.state.Tasks["task-1"].Status; got != space.OutcomeFailed {
		t.Fatalf("uncertified task status=%q, want failed", got)
	}

	store = &memoryStore{state: conversationState()}
	runtime = &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "hello"}}
	cert := exactCertification()
	svc = NewService(store, runtime, fakeCertifier{cert: cert}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err = svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err != nil {
		t.Fatal(err)
	}
	if runtime.created != 1 || len(store.state.Messages["c1"]) != 2 || store.state.Messages["c1"][1].Content != "hello" {
		t.Fatalf("certified completion not reconciled: creates=%d messages=%#v", runtime.created, store.state.Messages["c1"])
	}
}

func TestPreferenceSetAcceptsOnlyTypedOwnerPreference(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	svc := NewService(store, nil, nil)
	if _, err := svc.SetPreference(context.Background(), PreferenceRequest{RequestID: "pref-1", Name: "preferred_name", Value: "Ada"}); err != nil {
		t.Fatal(err)
	}
	record := store.state.ContextRecords["user.preference.preferred_name"]
	if record.Namespace != "user.preferences/v1" || record.Provenance != "owner" || !strings.Contains(string(record.Payload), "Ada") {
		t.Fatalf("preference record=%#v", record)
	}
}

func conversationState() space.State {
	persona := space.DefaultPersona()
	profile := space.DefaultRuntimeProfile()
	return space.State{SchemaVersion: space.StateSchemaVersionV2, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1, Audit: []space.AuditEvent{}, Conversations: map[string]space.Conversation{"c1": {ID: "c1", OwnerID: "owner", SurfaceID: "web", PersonaID: persona.ID, Status: space.ConversationActive, CreatedAt: 1, UpdatedAt: 1, NextSequence: 1}}, Messages: map[string][]space.Message{"c1": {}}, Personas: map[string]space.Persona{persona.ID: persona}, Surfaces: map[string]space.Surface{"web": {ID: "web", SchemaVersion: 1, Type: "web"}}, ContextRecords: map[string]space.ContextRecord{}, RuntimeSessions: map[string]space.RuntimeSessionMapping{}, RuntimeProfiles: map[string]space.RuntimeProfile{profile.ID: profile}, RuntimeCertifications: map[string]space.RuntimeCertification{}, Tasks: map[string]space.Task{}, HostCreates: map[string]space.HostCreate{}, HostRuns: map[string]space.HostRun{}, Commands: map[string]space.RecordedCommand{}}
}

func exactCertification() space.RuntimeCertification {
	p := space.DefaultRuntimeProfile()
	return space.RuntimeCertification{ID: "cert-1", RuntimeIdentity: "hermes:test", EndpointIdentity: "endpoint:test", ProfileDigest: p.Digest, EffectiveToolsets: []string{}, MemoryRead: false, MemoryWrite: false, Limits: space.RuntimeProfileLimits{Version: 1, MaxTurns: p.MaxTurns, MaxMessages: p.MaxMessages, MaxContextBytes: p.MaxContextBytes, MaxInputTokens: p.MaxInputTokens, MaxOutputTokens: p.MaxOutputTokens, MaxTotalTokens: p.MaxTotalTokens, DeadlineSeconds: p.DeadlineSeconds}, ExpiresAt: 1000}
}
