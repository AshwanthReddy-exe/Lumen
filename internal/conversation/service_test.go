package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
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
	created      int
	endpoint     string
	stopped      int
	before       func()
	beforeEvents func()
	beforeStop   func()
	run          hermes.Run
	events       []hermes.Event
	eventsErr    error
	req          hermes.CreateRunRequest
}

func (f *fakeRuntime) VerifiedEndpointIdentity(context.Context) (string, error) {
	if f.endpoint != "" {
		return f.endpoint, nil
	}
	return "endpoint:test", nil
}

func (f *fakeRuntime) VerifiedConversationEndpointIdentity(ctx context.Context) (string, error) {
	return f.VerifiedEndpointIdentity(ctx)
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
func (f *fakeRuntime) RunStatus(context.Context, string) (hermes.Run, error) { return f.run, nil }
func (f *fakeRuntime) Events(context.Context, string) ([]hermes.Event, error) {
	if f.beforeEvents != nil {
		f.beforeEvents()
	}
	return f.events, f.eventsErr
}
func (f *fakeRuntime) ResolveApproval(context.Context, string, string) error    { return nil }
func (f *fakeRuntime) Steer(context.Context, string, hermes.SteerRequest) error { return nil }
func (f *fakeRuntime) Stop(context.Context, string) (hermes.Run, error) {
	if f.beforeStop != nil {
		f.beforeStop()
	}
	f.stopped++
	return f.run, nil
}

type fakeCertifier struct {
	cert   space.RuntimeCertification
	err    error
	before func()
}

type changingProcessCertifier struct{ calls int }

func (c *changingProcessCertifier) Certify(context.Context, space.State, space.RuntimeProfile) (space.RuntimeCertification, error) {
	c.calls++
	cert := exactCertification()
	if c.calls > 1 {
		cert.ProcessIdentity = "restarted-process"
	}
	return cert, nil
}

type deadlineCheckingCertifier struct{ t *testing.T }

func (d deadlineCheckingCertifier) Certify(ctx context.Context, _ space.State, _ space.RuntimeProfile) (space.RuntimeCertification, error) {
	if _, ok := ctx.Deadline(); !ok {
		d.t.Fatal("recovery certification has no deadline")
	}
	return space.RuntimeCertification{}, ErrRuntimeProfileUnverified
}

func (f fakeCertifier) Certify(context.Context, space.State, space.RuntimeProfile) (space.RuntimeCertification, error) {
	if f.before != nil {
		f.before()
	}
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

func TestConversationLogsShowStageWithoutLoggingPrivateContent(t *testing.T) {
	var output strings.Builder
	state := &memoryStore{state: conversationState()}
	svc := NewService(state, &fakeRuntime{}, nil, WithLogger(slog.New(slog.NewJSONHandler(&output, nil))), WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "manual-test", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "private live log sentinel", CreatedAt: 100, ReconcileBy: 200})
	if !errors.Is(err, ErrRuntimeProfileUnverified) {
		t.Fatalf("unverified runtime was not blocked: %v", err)
	}
	logs := output.String()
	if !strings.Contains(logs, `"event":"intent_persisted"`) || !strings.Contains(logs, `"event":"runtime_blocked"`) {
		t.Fatalf("missing conversation stages: %s", logs)
	}
	if strings.Contains(logs, "private live log sentinel") {
		t.Fatal("conversation content appeared in logs")
	}
}

func TestConversationLogsTraceHermesRoundTripWithoutContent(t *testing.T) {
	var output strings.Builder
	state := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "private response sentinel"}, events: []hermes.Event{{Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"private response sentinel"}`)}}}
	svc := NewService(state, runtime, fakeCertifier{cert: exactCertification()}, WithLogger(slog.New(slog.NewJSONHandler(&output, nil))), WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "round-trip", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "private request sentinel", CreatedAt: 100, ReconcileBy: 200})
	if err != nil {
		t.Fatal(err)
	}
	logs := output.String()
	for _, event := range []string{"intent_persisted", "certification_started", "context_projected", "hermes_request_started", "hermes_request_accepted", "hermes_evidence_received", "terminal_persisted"} {
		if !strings.Contains(logs, `"event":"`+event+`"`) {
			t.Errorf("missing %s stage: %s", event, logs)
		}
	}
	if strings.Contains(logs, "private request sentinel") || strings.Contains(logs, "private response sentinel") {
		t.Fatal("private conversation text appeared in logs")
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
	one, err := Project(s, "c1", space.DefaultPersona(), space.DefaultRuntimeProfile(), 100)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Project(s, "c1", space.DefaultPersona(), space.DefaultRuntimeProfile(), 100)
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

func TestProjectionUsesOnlyCurrentOwnerMemory(t *testing.T) {
	s := conversationState()
	saved := space.Apply(s, space.Command{Type: space.CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "remember", MemoryID: "fact-1", Content: "My project is Lumen", CreatedAt: 10})
	if saved.Rejection != "" {
		t.Fatal(saved.Rejection)
	}
	s = saved.State
	s.ContextRecords["runtime-memory"] = space.ContextRecord{ID: "runtime-memory", Namespace: "user.memory/v1", SchemaVersion: 1, Provenance: "runtime", Classification: "private", AcceptedAt: 10, Payload: json.RawMessage(`{"text":"unauthorized"}`), Digest: space.DigestText(`{"text":"unauthorized"}`)}
	projection, err := Project(s, "c1", space.DefaultPersona(), space.DefaultRuntimeProfile(), 20)
	if err != nil || !strings.Contains(projection.Context, "My project is Lumen") || strings.Contains(projection.Context, "unauthorized") {
		t.Fatalf("memory projection: %#v %v", projection, err)
	}
	deleted := space.Apply(s, space.Command{Type: space.CommandDeleteMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "forget", MemoryID: "fact-1"})
	if deleted.Rejection != "" {
		t.Fatal(deleted.Rejection)
	}
	projection, err = Project(deleted.State, "c1", space.DefaultPersona(), space.DefaultRuntimeProfile(), 20)
	if err != nil || strings.Contains(projection.Context, "My project is Lumen") {
		t.Fatalf("deleted memory projected: %#v %v", projection, err)
	}
}

func TestProjectionPicksNewestMemoryWithinBudget(t *testing.T) {
	s := conversationState()
	for _, item := range []struct {
		id, text string
		at       int64
	}{{"a", "older", 1}, {"z", "newer", 2}} {
		tr := space.Apply(s, space.Command{Type: space.CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "save-" + item.id, MemoryID: item.id, Content: item.text, CreatedAt: item.at})
		if tr.Rejection != "" {
			t.Fatal(tr.Rejection)
		}
		s = tr.State
	}
	profile := space.DefaultRuntimeProfile()
	profile.MaxContextBytes = 50
	projection, err := Project(s, "c1", space.DefaultPersona(), profile, 3)
	if err != nil || !strings.Contains(projection.Context, "newer") || strings.Contains(projection.Context, "older") {
		t.Fatalf("context=%q err=%v", projection.Context, err)
	}
}

func TestProjectionContextNeverExceedsByteBudget(t *testing.T) {
	records := map[string]space.ContextRecord{
		"a": {ID: "a", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "owner", Classification: "private", AcceptedAt: 1, Digest: space.DigestText(`{"locale":"en"}`), Payload: json.RawMessage(`{"locale":"en"}`)},
		"b": {ID: "b", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "owner", Classification: "private", AcceptedAt: 1, Digest: space.DigestText(`{"locale":"en"}`), Payload: json.RawMessage(`{"locale":"en"}`)},
	}
	const budget = 74
	if contextText := projectedPreferences(records, budget, 100); len(contextText) > budget {
		t.Fatalf("projected context bytes=%d, want at most %d: %q", len(contextText), budget, contextText)
	}
}

func TestProjectionExcludesExpiredNonPrivateAndTamperedPreferences(t *testing.T) {
	valid := space.ContextRecord{ID: "valid", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "owner", Classification: "private", AcceptedAt: 1, Digest: space.DigestText(`{"preferred_name":"Ada"}`), Payload: json.RawMessage(`{"preferred_name":"Ada"}`)}
	records := map[string]space.ContextRecord{"valid": valid}
	expired := valid
	expired.ID, expired.RetentionUntil = "expired", 99
	records["expired"] = expired
	public := valid
	public.ID, public.Classification = "public", "public"
	records["public"] = public
	tampered := valid
	tampered.ID, tampered.Payload = "tampered", json.RawMessage(`{"preferred_name":"Mallory"}`)
	records["tampered"] = tampered
	contextText := projectedPreferences(records, 4096, 100)
	if !strings.Contains(contextText, `"id":"valid"`) || strings.Contains(contextText, `"id":"expired"`) || strings.Contains(contextText, `"id":"public"`) || strings.Contains(contextText, `"id":"tampered"`) {
		t.Fatalf("unsafe preference projected: %s", contextText)
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

func TestCertifiedChatRejectsMismatchedAdapterEndpoint(t *testing.T) {
	state := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{endpoint: "endpoint:general", run: hermes.Run{RunID: "run-1"}}
	svc := NewService(state, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if !errors.Is(err, ErrRuntimeProfileUnverified) || runtime.created != 0 || state.state.Tasks["task-1"].Status != space.OutcomeFailed {
		t.Fatalf("mismatched endpoint used for chat: err=%v creates=%d task=%s", err, runtime.created, state.state.Tasks["task-1"].Status)
	}
}

func TestCertifiedChatRejectsChangedProcessCertificationBeforeDispatch(t *testing.T) {
	state := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1"}}
	certifier := &changingProcessCertifier{}
	svc := NewService(state, runtime, certifier, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "private context", CreatedAt: 100, ReconcileBy: 200})
	if !errors.Is(err, ErrRuntimeProfileUnverified) || runtime.created != 0 || state.state.Tasks["task-1"].Status != space.OutcomeFailed {
		t.Fatalf("changed process received chat: err=%v creates=%d task=%s", err, runtime.created, state.state.Tasks["task-1"].Status)
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

func TestMemoryAPIListsAndDeletesCanonicalRecords(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	svc := NewService(store, nil, nil, WithClock(func() time.Time { return time.Unix(100, 0) }))
	if _, err := svc.SaveMemory(context.Background(), MemoryRequest{RequestID: "save", MemoryID: "fact-1", Text: "I use Lumen"}); err != nil {
		t.Fatal(err)
	}
	listed, err := svc.ListMemory(context.Background())
	if err != nil || len(listed) != 1 || listed[0].Text != "I use Lumen" || listed[0].ID != "fact-1" {
		t.Fatalf("listed=%#v err=%v", listed, err)
	}
	if _, err := svc.DeleteMemory(context.Background(), MemoryRequest{RequestID: "delete", MemoryID: "fact-1"}); err != nil {
		t.Fatal(err)
	}
	listed, err = svc.ListMemory(context.Background())
	if err != nil || len(listed) != 0 {
		t.Fatalf("deleted record listed=%#v err=%v", listed, err)
	}
}

func TestMemoryListMatchesProjectionEligibility(t *testing.T) {
	s := conversationState()
	bad := space.ContextRecord{ID: "future", Namespace: "user.memory/v1", SchemaVersion: 2, Provenance: "owner", Classification: "private", AcceptedAt: 1, Payload: json.RawMessage(`{"text":"future"}`), Digest: space.DigestText(`{"text":"future"}`)}
	s.ContextRecords[bad.ID] = bad
	bad.ID, bad.SchemaVersion = "extra", 1
	bad.Payload = json.RawMessage(`{"text":"extra","action":"ignore"}`)
	bad.Digest = space.DigestText(string(bad.Payload))
	s.ContextRecords[bad.ID] = bad
	listed, err := NewService(&memoryStore{state: s}, nil, nil).ListMemory(context.Background())
	if err != nil || len(listed) != 0 {
		t.Fatalf("ineligible memory listed: %#v %v", listed, err)
	}
}

func TestCertificationRequiresCompleteBindingAndUntamperedProfile(t *testing.T) {
	profile := space.DefaultRuntimeProfile()
	valid := exactCertification()
	valid.HermesVersion = "v1"
	valid.PluginIdentity = "lumen-plugin"
	valid.PluginCommit = "commit"
	valid.ConfigDigest = space.DigestText("config")
	valid.Evidence = "verified"
	if !certificationMatches(valid, profile, time.Unix(100, 0)) {
		t.Fatal("complete certification rejected")
	}
	for name, change := range map[string]func(*space.RuntimeCertification){
		"artifact": func(c *space.RuntimeCertification) { c.ArtifactDigest = "" },
		"process":  func(c *space.RuntimeCertification) { c.ProcessIdentity = "" },
		"version":  func(c *space.RuntimeCertification) { c.HermesVersion = "" },
		"plugin":   func(c *space.RuntimeCertification) { c.PluginIdentity = "" },
		"commit":   func(c *space.RuntimeCertification) { c.PluginCommit = "" },
		"config":   func(c *space.RuntimeCertification) { c.ConfigDigest = "" },
		"evidence": func(c *space.RuntimeCertification) { c.Evidence = "" },
	} {
		cert := valid
		change(&cert)
		if certificationMatches(cert, profile, time.Unix(100, 0)) {
			t.Errorf("%s: incomplete certification accepted", name)
		}
	}
	profile.Model = "tampered"
	if certificationMatches(valid, profile, time.Unix(100, 0)) {
		t.Error("profile changed without digest update")
	}
	profile = space.DefaultRuntimeProfile()
	profile.AllowedFeatureSet = []string{"shell"}
	profile.Digest = space.RuntimeProfileDigest(profile)
	valid.ProfileDigest = profile.Digest
	if certificationMatches(valid, profile, time.Unix(100, 0)) {
		t.Error("ordinary chat accepted a tool-enabled profile")
	}
}

func TestUnexpectedToolEventCannotAppendAssistantMessage(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "running"}, events: []hermes.Event{{Type: "tool_call", Data: json.RawMessage(`{"tool":"shell"}`)}, {Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"unsafe answer"}`)}}}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err == nil || runtime.stopped != 1 || len(store.state.Messages["c1"]) != 1 || store.state.Tasks["task-1"].Status == space.OutcomeCompleted {
		t.Fatalf("unexpected tool event accepted: err=%v stopped=%d messages=%d task=%q", err, runtime.stopped, len(store.state.Messages["c1"]), store.state.Tasks["task-1"].Status)
	}
}

func TestRuntimeViolationInvalidatesCertificationForLaterSend(t *testing.T) {
	state := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "unsafe"}, events: []hermes.Event{{Type: "failed", Data: json.RawMessage(`{"status":"failed"}`)}, {Type: "tool_call", Data: json.RawMessage(`{"tool":"secret-command"}`)}}}
	cert := exactCertification()
	runtime.beforeStop = func() {
		if !space.RuntimeCertificationInvalidated(state.state, cert) {
			t.Fatal("Stop I/O started before durable certification invalidation")
		}
	}
	svc := NewService(state, runtime, fakeCertifier{cert: cert}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, _ = svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if !space.RuntimeCertificationInvalidated(state.state, cert) {
		t.Fatal("runtime violation did not invalidate exact certification")
	}
	for _, event := range state.state.Audit {
		if strings.Contains(event.RequestID+event.Outcome, "secret-command") {
			t.Fatal("runtime evidence leaked into durable audit")
		}
	}
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-2", ConversationID: "c1", SurfaceID: "web", TaskID: "task-2", Input: "again", CreatedAt: 101, ReconcileBy: 201})
	if !errors.Is(err, ErrRuntimeProfileUnverified) || runtime.created != 1 || state.state.Tasks["task-2"].Status != space.OutcomeFailed {
		t.Fatalf("invalidated certificate reused: err=%v creates=%d task=%q", err, runtime.created, state.state.Tasks["task-2"].Status)
	}
}

func TestRuntimeViolationInvalidationSurvivesEncryptedStoreReopen(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	durable, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := durable.Initialize(conversationState()); err != nil {
		t.Fatal(err)
	}
	cert := exactCertification()
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "running"}, events: []hermes.Event{{Type: "tool_call", Data: json.RawMessage(`{"tool":"private"}`)}}}
	svc := NewService(durable, runtime, fakeCertifier{cert: cert}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, _ = svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err := durable.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	state, err := reopened.Read()
	if err != nil || !space.RuntimeCertificationInvalidated(state, cert) {
		t.Fatalf("certification invalidation lost after reopen: invalidated=%v err=%v", space.RuntimeCertificationInvalidated(state, cert), err)
	}
	runtime.events = nil
	svc = NewService(reopened, runtime, fakeCertifier{cert: cert}, WithClock(func() time.Time { return time.Unix(101, 0) }))
	_, err = svc.Send(context.Background(), SendRequest{RequestID: "send-2", ConversationID: "c1", SurfaceID: "web", TaskID: "task-2", Input: "again", CreatedAt: 101, ReconcileBy: 201})
	if !errors.Is(err, ErrRuntimeProfileUnverified) || runtime.created != 1 {
		t.Fatalf("reopened state reused invalidated cert: err=%v creates=%d", err, runtime.created)
	}
}

func TestTerminalStatusStillRejectsHiddenToolEvent(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "unsafe answer"}, events: []hermes.Event{
		{Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"unsafe answer"}`)},
		{Type: "progress", Data: json.RawMessage(`{"status":"progress","details":{"tool_call":"shell"}}`)},
	}}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err == nil || runtime.stopped != 1 || len(store.state.Messages["c1"]) != 1 || store.state.Tasks["task-1"].Status == space.OutcomeCompleted {
		t.Fatalf("hidden tool event accepted: err=%v stopped=%d messages=%d task=%q", err, runtime.stopped, len(store.state.Messages["c1"]), store.state.Tasks["task-1"].Status)
	}
}

func TestDisconnectedEventStreamWithoutTerminalEvidenceIsUncertain(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "unverified"}, eventsErr: hermes.ErrEventStreamDisconnected}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err == nil || store.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(store.state.Messages["c1"]) != 1 {
		t.Fatalf("disconnected evidence accepted: err=%v task=%q messages=%d", err, store.state.Tasks["task-1"].Status, len(store.state.Messages["c1"]))
	}
}

func TestConflictingTerminalEvidenceCannotCompleteConversation(t *testing.T) {
	for name, run := range map[string]hermes.Run{
		"event conflict":  {RunID: "run-1", Status: "running"},
		"status conflict": {RunID: "run-1", Status: "failed"},
	} {
		t.Run(name, func(t *testing.T) {
			state := &memoryStore{state: conversationState()}
			runtime := &fakeRuntime{run: run, events: []hermes.Event{
				{Type: "failed", Data: json.RawMessage(`{"status":"failed"}`)},
				{Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"unsafe"}`)},
			}}
			svc := NewService(state, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
			_, _ = svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
			if state.state.Tasks["task-1"].Status == space.OutcomeCompleted || len(state.state.Messages["c1"]) != 1 {
				t.Fatalf("conflicting terminal evidence accepted: task=%q messages=%d", state.state.Tasks["task-1"].Status, len(state.state.Messages["c1"]))
			}
		})
	}
}

func TestDuplicateEventKeyCannotHideToolEvidence(t *testing.T) {
	if allowedChatEvent(hermes.Event{Type: "completed", Data: json.RawMessage(`{"status":"tool_call","status":"completed","output":"unsafe"}`)}) {
		t.Fatal("duplicate status hid tool evidence")
	}
}

func TestEventTypeCannotContradictPayloadStatus(t *testing.T) {
	if allowedChatEvent(hermes.Event{Type: "failed", Data: json.RawMessage(`{"status":"completed","output":"unsafe"}`)}) {
		t.Fatal("terminal type contradicted by payload status")
	}
}

func TestToolStatusHiddenInProgressCannotAppendAssistantMessage(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "running"}, events: []hermes.Event{{Type: "progress", Data: json.RawMessage(`{"status":"tool_call"}`)}, {Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"unsafe answer"}`)}}}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err == nil || runtime.stopped != 1 || len(store.state.Messages["c1"]) != 1 {
		t.Fatalf("tool status accepted: err=%v stopped=%d messages=%d", err, runtime.stopped, len(store.state.Messages["c1"]))
	}
}

func TestLateTerminalResultDoesNotAppendAssistantMessage(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	now := int64(100)
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "running"}, events: []hermes.Event{{Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"late"}`)}}}
	runtime.beforeEvents = func() { now = 201 }
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(now, 0) }))
	_, _ = svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if got := store.state.Tasks["task-1"].Status; got != space.OutcomeUnknown || len(store.state.Messages["c1"]) != 1 {
		t.Fatalf("late result accepted: task=%q messages=%d", got, len(store.state.Messages["c1"]))
	}
}

func TestReplayedSendReportsDurableTaskOutcome(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "running"}}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	req := SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200}
	_, _ = svc.Send(context.Background(), req)
	replayed, err := svc.Send(context.Background(), req)
	if err != nil || !replayed.Replayed || replayed.Receipt.Outcome != space.OutcomeUnknown || runtime.created != 1 {
		t.Fatalf("replay hid durable outcome or redispatched: receipt=%#v err=%v creates=%d", replayed.Receipt, err, runtime.created)
	}
}

func TestSessionIntentIsDurableBeforeCertificationIO(t *testing.T) {
	store := &memoryStore{state: conversationState()}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "hello"}}
	certifier := fakeCertifier{cert: exactCertification(), before: func() {
		mapping, ok := store.state.RuntimeSessions["c1"]
		if !ok || !mapping.Pending || mapping.HermesSessionID != "lumen-session:c1" || mapping.ProfileDigest != space.DefaultRuntimeProfile().Digest {
			t.Fatalf("certification began before durable session intent: %#v", mapping)
		}
	}}
	svc := NewService(store, runtime, certifier, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err != nil || store.state.RuntimeSessions["c1"].Pending {
		t.Fatalf("certified session did not bind: err=%v mapping=%#v", err, store.state.RuntimeSessions["c1"])
	}
}

func TestSessionReservationSurvivesEncryptedStoreReopen(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	durable, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := durable.Initialize(conversationState()); err != nil {
		t.Fatal(err)
	}
	certifier := fakeCertifier{err: ErrRuntimeProfileUnverified, before: func() {
		state, readErr := durable.Read()
		if readErr != nil || !state.RuntimeSessions["c1"].Pending || state.Tasks["task-1"].Status != space.OutcomeQueued {
			t.Fatalf("session intent not durable before certification: state=%#v err=%v", state.RuntimeSessions["c1"], readErr)
		}
	}}
	svc := NewService(durable, &fakeRuntime{}, certifier, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, _ = svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err := durable.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	state, err := reopened.Read()
	if err != nil || !state.RuntimeSessions["c1"].Pending || state.Tasks["task-1"].Status != space.OutcomeFailed {
		t.Fatalf("session reservation lost after reopen: mapping=%#v task=%#v err=%v", state.RuntimeSessions["c1"], state.Tasks["task-1"], err)
	}
}

func TestRestartReconcilesOnlyDurablyMappedConversationRun(t *testing.T) {
	state := conversationState()
	apply := func(command space.Command) {
		command.SpaceID, command.HostID, command.Epoch = state.SpaceID, state.HostID, state.Epoch
		tr := space.Apply(state, command)
		if tr.Rejection != "" {
			t.Fatalf("setup rejected: %s", tr.Rejection)
		}
		state = tr.State
	}
	apply(space.Command{Type: space.CommandSendConversation, ActorID: "owner", RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Content: "hello", RuntimeIdempotencyKey: "send-1", CreatedAt: 100, ReconcileBy: 200})
	apply(space.Command{Type: space.CommandReserveRuntimeSession, ActorID: "host", RequestID: "reserve-1", ConversationID: "c1", HermesSessionID: "lumen-session:c1", RuntimeProfileDigest: space.DefaultRuntimeProfile().Digest})
	apply(space.Command{Type: space.CommandBindRuntimeSession, ActorID: "host", RequestID: "bind-1", ConversationID: "c1", HermesSessionID: "lumen-session:c1", RuntimeIdentity: "hermes:test", RuntimeProfileDigest: space.DefaultRuntimeProfile().Digest})
	apply(space.Command{Type: space.CommandDispatchHostRun, ActorID: "host", RequestID: "dispatch-1", TaskID: "task-1", RuntimeRunID: "run-1", RuntimeIdempotencyKey: "send-1", RuntimeProfileDigest: space.DefaultRuntimeProfile().Digest, CertificationID: "cert-1", EndpointIdentity: "endpoint:test", DispatchedAt: 100, ReconcileBy: 200})
	store := &memoryStore{state: state}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "answer"}, events: []hermes.Event{{Type: "completed", Data: json.RawMessage(`{"status":"completed","output":"answer"}`)}}}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(101, 0) }))
	if err := svc.ReconcilePending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.state.Tasks["task-1"].Status != space.OutcomeCompleted || len(store.state.Messages["c1"]) != 2 || store.state.Messages["c1"][1].Content != "answer" || store.state.Messages["c1"][1].CreatedAt != 101 || runtime.created != 0 {
		t.Fatalf("mapped run did not recover honestly: task=%#v messages=%#v creates=%d", store.state.Tasks["task-1"], store.state.Messages["c1"], runtime.created)
	}
	replayed, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err != nil || replayed.Receipt.Outcome != space.OutcomeCompleted || runtime.created != 0 {
		t.Fatalf("replay redispatched recovered run: %#v %v creates=%d", replayed, err, runtime.created)
	}
	mismatchStore := &memoryStore{state: state}
	mismatchRuntime := &fakeRuntime{run: hermes.Run{RunID: "other-run", Status: "completed", Output: "wrong"}}
	if err := NewService(mismatchStore, mismatchRuntime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || mismatchStore.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(mismatchStore.state.Messages["c1"]) != 1 || mismatchRuntime.created != 0 {
		t.Fatalf("mismatched run recovered as success: task=%q messages=%d creates=%d err=%v", mismatchStore.state.Tasks["task-1"].Status, len(mismatchStore.state.Messages["c1"]), mismatchRuntime.created, err)
	}
	misbound := state
	badMapping := state.HostRuns["task-1"]
	badMapping.TaskID = "other-task"
	misbound.HostRuns = map[string]space.HostRun{"task-1": badMapping}
	misboundStore := &memoryStore{state: misbound}
	if err := NewService(misboundStore, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || misboundStore.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(misboundStore.state.Messages["c1"]) != 1 {
		t.Fatalf("misbound run recovered as success: task=%q messages=%d err=%v", misboundStore.state.Tasks["task-1"].Status, len(misboundStore.state.Messages["c1"]), err)
	}
	changedEndpoint := exactCertification()
	changedEndpoint.EndpointIdentity = "replacement-endpoint"
	replacementStore := &memoryStore{state: state}
	replacementRuntime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "wrong runtime"}}
	if err := NewService(replacementStore, replacementRuntime, fakeCertifier{cert: changedEndpoint}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || replacementStore.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(replacementStore.state.Messages["c1"]) != 1 {
		t.Fatalf("replacement endpoint supplied recovered answer: task=%q messages=%d err=%v", replacementStore.state.Tasks["task-1"].Status, len(replacementStore.state.Messages["c1"]), err)
	}
	wrongAdapterStore := &memoryStore{state: state}
	wrongAdapter := &fakeRuntime{endpoint: "endpoint:general", run: hermes.Run{RunID: "run-1", Status: "completed", Output: "wrong adapter"}}
	if err := NewService(wrongAdapterStore, wrongAdapter, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || wrongAdapterStore.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(wrongAdapterStore.state.Messages["c1"]) != 1 {
		t.Fatalf("wrong adapter supplied recovered answer: task=%q messages=%d err=%v", wrongAdapterStore.state.Tasks["task-1"].Status, len(wrongAdapterStore.state.Messages["c1"]), err)
	}
	deadlineStore := &memoryStore{state: state}
	if err := NewService(deadlineStore, &fakeRuntime{}, deadlineCheckingCertifier{t}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || deadlineStore.state.Tasks["task-1"].Status != space.OutcomeUnknown {
		t.Fatalf("unavailable certification blocked recovery: task=%q err=%v", deadlineStore.state.Tasks["task-1"].Status, err)
	}
	violationStore := &memoryStore{state: state}
	violationRuntime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "running"}, events: []hermes.Event{{Type: "tool_call", Data: json.RawMessage(`{"tool":"shell"}`)}}}
	if err := NewService(violationStore, violationRuntime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || violationStore.state.Tasks["task-1"].Status != space.OutcomeFailed || len(violationStore.state.Messages["c1"]) != 1 || violationRuntime.stopped != 1 || !space.RuntimeCertificationInvalidated(violationStore.state, space.RuntimeCertification{ID: "cert-1", RuntimeIdentity: "hermes:test", ProfileDigest: space.DefaultRuntimeProfile().Digest}) {
		t.Fatalf("unsafe recovered run accepted: task=%q messages=%d stopped=%d err=%v", violationStore.state.Tasks["task-1"].Status, len(violationStore.state.Messages["c1"]), violationRuntime.stopped, err)
	}
	oversized := strings.Repeat("x", space.MaxChatMessageBytes+1)
	oversizedData, _ := json.Marshal(map[string]string{"status": "completed", "output": oversized})
	oversizedStore := &memoryStore{state: state}
	oversizedRuntime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: oversized}, events: []hermes.Event{{Type: "completed", Data: oversizedData}}}
	if err := NewService(oversizedStore, oversizedRuntime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || oversizedStore.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(oversizedStore.state.Messages["c1"]) != 1 {
		t.Fatalf("oversized output blocked recovery: task=%q messages=%d err=%v", oversizedStore.state.Tasks["task-1"].Status, len(oversizedStore.state.Messages["c1"]), err)
	}
}

func TestRestartWithoutDurableRunMappingRemainsUnknown(t *testing.T) {
	state := conversationState()
	queued := space.Apply(state, space.Command{Type: space.CommandSendConversation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Content: "hello", CreatedAt: 100, ReconcileBy: 200})
	if queued.Rejection != "" {
		t.Fatal(queued.Rejection)
	}
	store := &memoryStore{state: queued.State}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed", Output: "unproven"}}
	if err := NewService(store, runtime, nil, WithClock(func() time.Time { return time.Unix(101, 0) })).ReconcilePending(context.Background()); err != nil || store.state.Tasks["task-1"].Status != space.OutcomeUnknown || len(store.state.Messages["c1"]) != 1 || runtime.created != 0 {
		t.Fatalf("unmapped run was redispatched or completed: task=%q messages=%d creates=%d err=%v", store.state.Tasks["task-1"].Status, len(store.state.Messages["c1"]), runtime.created, err)
	}
}

func TestRejectedSessionReservationTerminatesQueuedConversation(t *testing.T) {
	state := conversationState()
	profile := space.DefaultRuntimeProfile()
	state.RuntimeSessions["c1"] = space.RuntimeSessionMapping{ConversationID: "c1", RuntimeIdentity: "hermes:test", HermesSessionID: "different-session", PersonaDigest: profile.PersonaDigest, ProfileDigest: profile.Digest, HostEpoch: 1}
	store := &memoryStore{state: state}
	runtime := &fakeRuntime{run: hermes.Run{RunID: "run-1", Status: "completed"}}
	svc := NewService(store, runtime, fakeCertifier{cert: exactCertification()}, WithClock(func() time.Time { return time.Unix(100, 0) }))
	_, err := svc.Send(context.Background(), SendRequest{RequestID: "send-1", ConversationID: "c1", SurfaceID: "web", TaskID: "task-1", Input: "hello", CreatedAt: 100, ReconcileBy: 200})
	if err == nil || store.state.Tasks["task-1"].Status != space.OutcomeFailed || runtime.created != 0 {
		t.Fatalf("rejected reservation stranded task: err=%v task=%q creates=%d", err, store.state.Tasks["task-1"].Status, runtime.created)
	}
}

func conversationState() space.State {
	persona := space.DefaultPersona()
	profile := space.DefaultRuntimeProfile()
	return space.State{SchemaVersion: space.StateSchemaVersionV2, SpaceID: "space", OwnerID: "owner", HostID: "host", Epoch: 1, Audit: []space.AuditEvent{}, Conversations: map[string]space.Conversation{"c1": {ID: "c1", OwnerID: "owner", SurfaceID: "web", PersonaID: persona.ID, Status: space.ConversationActive, CreatedAt: 1, UpdatedAt: 1, NextSequence: 1}}, Messages: map[string][]space.Message{"c1": {}}, Personas: map[string]space.Persona{persona.ID: persona}, Surfaces: map[string]space.Surface{"web": {ID: "web", SchemaVersion: 1, Type: "web"}}, ContextRecords: map[string]space.ContextRecord{}, RuntimeSessions: map[string]space.RuntimeSessionMapping{}, RuntimeProfiles: map[string]space.RuntimeProfile{profile.ID: profile}, RuntimeCertifications: map[string]space.RuntimeCertification{}, Tasks: map[string]space.Task{}, HostCreates: map[string]space.HostCreate{}, HostRuns: map[string]space.HostRun{}, Commands: map[string]space.RecordedCommand{}}
}

func exactCertification() space.RuntimeCertification {
	p := space.DefaultRuntimeProfile()
	return space.RuntimeCertification{ID: "cert-1", RuntimeIdentity: "hermes:test", EndpointIdentity: "endpoint:test", ArtifactDigest: space.DigestText("patched-artifact"), ProcessIdentity: "process:test", HermesVersion: "v1", PluginIdentity: "lumen-plugin", PluginCommit: "commit", ConfigDigest: space.DigestText("config"), Evidence: "verified", ProfileDigest: p.Digest, EffectiveToolsets: []string{}, MemoryRead: false, MemoryWrite: false, Limits: space.RuntimeProfileLimits{Version: 1, MaxTurns: p.MaxTurns, MaxMessages: p.MaxMessages, MaxContextBytes: p.MaxContextBytes, MaxInputTokens: p.MaxInputTokens, MaxOutputTokens: p.MaxOutputTokens, MaxTotalTokens: p.MaxTotalTokens, DeadlineSeconds: p.DeadlineSeconds}, ExpiresAt: 1000}
}
