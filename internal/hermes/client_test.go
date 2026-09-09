package hermes

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func testConfig(baseURL string) Config {
	return Config{
		BaseURL:              baseURL,
		ProfileMode:          ProfileDevelopment,
		MaxResponseBytes:     64 * 1024,
		MaxEventBytes:        8 * 1024,
		RequestTimeout:       time.Second,
		EventTimeout:         time.Second,
		MaxEvents:            16,
		MaxEventStreamBytes:  64 * 1024,
		BearerTokenSource:    func() (string, error) { return "test-token", nil },
		RequiredCapabilities: []string{CapabilityRunSubmission, CapabilityRunStatus, CapabilityRunEvents, CapabilityRunStop},
	}
}

func TestCapabilitiesRejectMissingRequiredFeature(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/capabilities" {
			t.Fatal("unexpected path: " + r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"hermes.api_server.capabilities","platform":"hermes-agent","model":"hermes-agent","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":false,"run_events_sse":true,"run_stop":true}}`))
	}))

	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Capabilities(context.Background())
	if !errors.Is(err, ErrCapabilityMismatch) {
		t.Fatalf("expected capability mismatch, got %v", err)
	}
}

func TestCapabilitiesAcceptsHermesApprovalResponseAlias(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"hermes.api_server.capabilities","platform":"hermes-agent","model":"hermes-agent","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval_response":true,"run_stop":true,"session_continuity_header":"X-Hermes-Session-Id","browser_extension_control":{"enabled":false}},"endpoints":{"runs":"/v1/runs"},"runtime":{"version":"live"}}`))
	}))
	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Capabilities(context.Background()); err != nil {
		t.Fatalf("live Hermes approval capability rejected: %v", err)
	}
}

func TestHealthSendsScopedBearerAndExactHeaders(t *testing.T) {
	var got http.Header
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","platform":"hermes-agent","version":"0.20.6"}`))
	}))

	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got.Get("Authorization") != "Bearer test-token" {
		t.Fatalf("authorization = %q", got.Get("Authorization"))
	}
	if got.Get("Accept") != "application/json" {
		t.Fatalf("accept = %q", got.Get("Accept"))
	}
}

func TestCreateRunUsesIdempotencyKeyAndDecodesRun(t *testing.T) {
	var gotKey string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/runs" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		gotKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"run","run_id":"run_1","status":"started","created_at":100.5,"updated_at":101.5,"last_event":"run.started"}`))
	}))

	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	run, err := c.CreateRun(context.Background(), CreateRunRequest{Input: "hello"}, "idem-1")
	if err != nil {
		t.Fatal(err)
	}
	if run.RunID != "run_1" || gotKey != "idem-1" {
		t.Fatalf("run=%#v key=%q", run, gotKey)
	}
}

func TestCreateRunClassifiesProvenAndAmbiguousFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		want error
	}{
		{name: "proven rejection", code: http.StatusBadRequest, want: ErrCreateRejected},
		{name: "ambiguous server failure", code: http.StatusBadGateway, want: ErrCreateAmbiguous},
		{name: "conflict is ambiguous", code: http.StatusConflict, want: ErrCreateAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte("failure"))
			}))
			c, err := New(testConfig(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.CreateRun(context.Background(), CreateRunRequest{Input: "x"}, "create-key")
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
		})
	}
}

func TestEventsParseBoundedSSEWithoutDeduplicatingEvidence(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("id: 2\nevent: completed\ndata: {\"status\":\"completed\"}\n\nid: 1\nevent: progress\ndata: {\"delta\":\"x\"}\n\nid: 1\nevent: progress\ndata: {\"delta\":\"x\"}\n\n"))
	}))

	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	events, err := c.Events(context.Background(), "run_1")
	if !errors.Is(err, ErrEventStreamDisconnected) {
		t.Fatalf("expected disconnect after finite stream, got %v", err)
	}
	if len(events) != 3 || events[0].ID != "2" || events[1].ID != "1" || events[2].ID != "1" {
		t.Fatalf("events = %#v", events)
	}
}

func TestEventsStreamDeliversApprovalBeforeConnectionCloses(t *testing.T) {
	release := make(chan struct{})
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("test server does not support flush")
		}
		_, _ = w.Write([]byte("id: approval\nevent: approval.requested\ndata: {\"status\":\"awaiting_approval\"}\n\n"))
		flusher.Flush()
		<-release
	}))
	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	stream, err := c.EventsStream(context.Background(), "run_1")
	if err != nil {
		t.Fatal(err)
	}
	event, err := stream.Next(context.Background())
	if err != nil || event.ID != "approval" {
		t.Fatalf("first incremental event = %#v, %v", event, err)
	}
	close(release)
	_ = stream.Close()
}

func TestEventsRejectsIncompleteRecordAndBoundsStream(t *testing.T) {
	events, err := parseSSEBounded(strings.NewReader("id: 1\ndata: {\"status\":\"completed\"}"), 1024, 10, 1024)
	if !errors.Is(err, ErrEventStreamDisconnected) || len(events) != 0 {
		t.Fatalf("incomplete event = %#v, %v", events, err)
	}
	stream := strings.Repeat("data: {}\n\n", 20)
	if _, err := parseSSEBounded(strings.NewReader(stream), 128, 2, 1024); !errors.Is(err, ErrEventStreamTooLarge) {
		t.Fatalf("expected event-count bound, got %v", err)
	}
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(stream))
	}))
	cfg := testConfig(srv.URL)
	cfg.MaxEvents = 2
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Events(context.Background(), "run_1"); !errors.Is(err, ErrEventStreamTooLarge) {
		t.Fatalf("fake-server stream bound = %v", err)
	}
}

func TestSSERejectsUnknownAndMalformedRetryFields(t *testing.T) {
	if _, err := parseSSEBounded(strings.NewReader("retry: nope\ndata: {}\n\n"), 1024, 10, 1024); err == nil {
		t.Fatal("malformed retry accepted")
	}
	if _, err := parseSSEBounded(strings.NewReader("unknown: value\ndata: {}\n\n"), 1024, 10, 1024); err == nil {
		t.Fatal("unknown SSE field accepted")
	}
	events, err := parseSSEBounded(strings.NewReader("id: 1\nevent: progress\nretry: 250\ndata: {\"delta\":\ndata: \"x\"}\n\n"), 1024, 10, 1024)
	if err != ErrEventStreamDisconnected || len(events) != 1 || events[0].Retry != 250 {
		t.Fatalf("retry event = %#v, %v", events, err)
	}
}

func TestApprovalOnlyAcceptsOnceOrDeny(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("invalid approval reached server")
	}))

	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range []string{"session", "always", "allow", ""} {
		if err := c.ResolveApproval(context.Background(), "run_1", decision); !errors.Is(err, ErrInvalidApproval) {
			t.Fatalf("decision %q: expected invalid approval, got %v", decision, err)
		}
	}
}

func TestValidApprovalUsesDocumentedValues(t *testing.T) {
	var body string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runs/run_1/approval" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"hermes.run.approval_response","run_id":"run_1","choice":"once","resolved":1,"vendor_meta":{"version":2}}`))
	}))
	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ResolveApproval(context.Background(), "run_1", "once"); err != nil {
		t.Fatal(err)
	}
	if body != `{"choice":"once"}` {
		t.Fatalf("approval body = %q", body)
	}
}

func TestRunStatusAcceptsLiveWaitingForApproval(t *testing.T) {
	if err := validateRun(Run{RunID: "run_1", Status: "waiting_for_approval"}, true); err != nil {
		t.Fatal(err)
	}
}

func TestRunStatusAndSSEBoundsRejectMalformedEvidence(t *testing.T) {
	if _, err := parseSSEBounded(strings.NewReader("data: {\"too\":\"large\"}\n\n"), 8, 10, 1024); !errors.Is(err, ErrEventTooLarge) {
		t.Fatalf("expected event bound error, got %v", err)
	}
	if _, err := parseSSE(strings.NewReader("data: not-json\n\n"), 1024); err == nil {
		t.Fatal("invalid SSE JSON accepted")
	}
}

func TestSteerUsesDocumentedRunsEndpoint(t *testing.T) {
	var body string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/capabilities" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"hermes.api_server.capabilities","platform":"hermes-agent","model":"hermes-agent","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval":true,"run_stop":true,"run_steer":true}}`))
			return
		}
		if r.URL.Path != "/v1/runs/run_1/steer" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"steered"}`))
	}))
	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Steer(context.Background(), "run_1", SteerRequest{Input: "more"}); err != nil {
		t.Fatalf("steer failed: %v", err)
	}
	if body != `{"input":"more"}` {
		t.Fatalf("steer body = %q", body)
	}
}

func TestSteerIsOptionalAndFailsClosedWhenCapabilityIsAbsent(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/capabilities" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"hermes.api_server.capabilities","platform":"hermes-agent","model":"hermes-agent","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval":true,"run_stop":true}}`))
			return
		}
		t.Fatalf("unsupported steer request reached server: %s %s", r.Method, r.URL.Path)
	}))
	cfg := testConfig(srv.URL)
	cfg.RequiredCapabilities = nil
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Capabilities(context.Background()); err != nil {
		t.Fatalf("missing optional steer rejected core capabilities: %v", err)
	}
	if err := c.Steer(context.Background(), "run_1", SteerRequest{Input: "more"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected optional unsupported steer, got %v", err)
	}
}

func TestStopUsesRunsEndpoint(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/runs/run_1/stop" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"stopping"}`))
	}))
	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Stop(context.Background(), "run_1")
	if err != nil || got.Status != "stopping" {
		t.Fatalf("stop=%#v err=%v", got, err)
	}
}

func TestRejectsRedirectAndURLUserinfo(t *testing.T) {
	final := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("redirect target was contacted")
	}))
	redirect := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))

	c, err := New(testConfig(redirect.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); !errors.Is(err, ErrRedirectRejected) {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
	bad := testConfig("http://user:pass@example.test")
	if _, err := New(bad); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected userinfo rejection, got %v", err)
	}
	if _, err := New(testConfig("http://localhost:8642")); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("hostname loopback was accepted")
	}
}

func TestRunIDsAreOpaqueAndCannotNormalizePaths(t *testing.T) {
	suffixes := []string{"", "/events", "/approval", "/stop"}
	for _, id := range []string{".", "..", "a/../b", "%2e%2e", "run.1"} {
		for _, suffix := range suffixes {
			if _, err := runPath(id, suffix); !errors.Is(err, ErrInvalidRunID) {
				t.Fatalf("run ID %q accepted for %s: %v", id, suffix, err)
			}
		}
	}
}

func TestBoundsAndTimeouts(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(strings.Repeat("x", 100)))
			return
		}
		select {}
	}))

	cfg := testConfig(srv.URL)
	cfg.MaxResponseBytes = 32
	cfg.RequestTimeout = 20 * time.Millisecond
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected oversized response, got %v", err)
	}

	cfg.MaxResponseBytes = 1024
	c, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := c.Health(ctx); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestHardenedConfigRequiresTLS13PinAndClientCertificate(t *testing.T) {
	cfg := testConfig("https://example.test")
	cfg.ProfileMode = ProfileHardened
	if _, err := New(cfg); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected hardened TLS requirements, got %v", err)
	}
	cfg.TLS = TLSConfig{MinVersion: tls.VersionTLS13}
	if _, err := New(cfg); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected hardened identity requirements, got %v", err)
	}
}

func TestResponsesRequireKnownFieldsAndStatuses(t *testing.T) {
	for name, response := range map[string]string{
		"health":   `{}`,
		"run":      `{"run_id":"","status":"started"}`,
		"status":   `{"run_id":"run_1"}`,
		"approval": `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(response))
			}))
			c, err := New(testConfig(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			switch name {
			case "health":
				if _, err = c.Health(ctx); err == nil {
					t.Fatal("empty health accepted")
				}
			case "run":
				if _, err = c.CreateRun(ctx, CreateRunRequest{Input: "x"}, "key"); err == nil {
					t.Fatal("empty run ID accepted")
				}
			case "status":
				if _, err = c.RunStatus(ctx, "run_1"); err == nil {
					t.Fatal("empty status accepted")
				}
			case "approval":
				if err = c.ResolveApproval(ctx, "run_1", "once"); err == nil {
					t.Fatal("empty approval accepted")
				}
			}
		})
	}
}

func TestErrorDoesNotExposeResponseBody(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("echo Bearer test-token and secret response"))
	}))
	c, err := New(testConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Health(context.Background())
	if err == nil || strings.Contains(err.Error(), "test-token") || strings.Contains(err.Error(), "secret response") {
		t.Fatalf("unsafe error: %v", err)
	}
}

func TestHardenedTLSRejectsPinAndClientCertificateMismatch(t *testing.T) {
	pki := makeTestPKI(t)
	srv := newMutualTLSServer(t, pki, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	pin := sha256.Sum256(pki.ServerLeaf.Raw)
	cfg := testConfig(srv.URL)
	cfg.ProfileMode = ProfileHardened
	cfg.TLS = TLSConfig{RootCAs: pki.Roots, ClientCertificate: pki.Client, ServerCertSHA256: pin[:], MinVersion: tls.VersionTLS13}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); err != nil {
		t.Fatalf("matching hardened TLS failed: %v", err)
	}

	cfg.TLS.ServerCertSHA256 = make([]byte, sha256.Size)
	c, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); err == nil {
		t.Fatal("certificate pin mismatch was accepted")
	}

	cfg.TLS.ServerCertSHA256 = pin[:]
	cfg.TLS.ClientCertificate = pki.OtherClient
	c, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); err == nil {
		t.Fatal("client certificate mismatch was accepted")
	}
}

func TestHardenedTLSRetainsHostnameAndValidityVerification(t *testing.T) {
	pki := makeTestPKI(t)
	pin := sha256.Sum256(pki.ServerLeaf.Raw)
	server := newMutualTLSServer(t, pki, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	cfg := testConfig(server.URL)
	cfg.ProfileMode = ProfileHardened
	cfg.TLS = TLSConfig{RootCAs: pki.Roots, ClientCertificate: pki.Client, ServerCertSHA256: pin[:], ServerName: "wrong.example", MinVersion: tls.VersionTLS13}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); err == nil {
		t.Fatal("hostname mismatch was accepted")
	}

	caCert, caKey := makeCA(t, "expired-ca")
	expiredServer, expiredLeaf := makeSignedCertificateWindow(t, caCert, caKey, "127.0.0.1", x509.ExtKeyUsageServerAuth, time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour))
	expiredRoots := x509.NewCertPool()
	expiredRoots.AddCert(caCert)
	expiredPKI := testPKI{Roots: expiredRoots, Server: expiredServer, ServerLeaf: expiredLeaf, Client: pki.Client}
	expired := newMutualTLSServer(t, expiredPKI, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	expiredPin := sha256.Sum256(expiredLeaf.Raw)
	cfg = testConfig(expired.URL)
	cfg.ProfileMode = ProfileHardened
	cfg.TLS = TLSConfig{RootCAs: expiredRoots, ClientCertificate: pki.Client, ServerCertSHA256: expiredPin[:], MinVersion: tls.VersionTLS13}
	c, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Health(context.Background()); err == nil {
		t.Fatal("expired certificate was accepted")
	}
}
