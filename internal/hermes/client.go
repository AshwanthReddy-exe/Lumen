// Package hermes contains the evidence-only boundary to Hermes's documented
// API server. It does not persist state or authorize a Lumen operation.
package hermes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ProfileDevelopment = "development"
	ProfileHardened    = "hardened"

	CapabilityRunSubmission       = "run_submission"
	CapabilityRunStatus           = "run_status"
	CapabilityRunEvents           = "run_events_sse"
	CapabilityRunApproval         = "run_approval"
	capabilityRunApprovalResponse = "run_approval_response"
	CapabilityRunSteer            = "run_steer"
	CapabilityRunStop             = "run_stop"
	defaultMaxEvents              = 1024
)

var (
	ErrInvalidConfig           = errors.New("invalid Hermes client configuration")
	ErrCapabilityMismatch      = errors.New("Hermes capabilities do not satisfy the required contract")
	ErrInvalidApproval         = errors.New("approval decision must be once or deny")
	ErrUnsupported             = errors.New("Hermes operation is unsupported by the advertised capability set")
	ErrRedirectRejected        = errors.New("Hermes redirect rejected")
	ErrResponseTooLarge        = errors.New("Hermes response exceeds configured bound")
	ErrEventTooLarge           = errors.New("Hermes SSE event exceeds configured bound")
	ErrEventStreamTooLarge     = errors.New("Hermes SSE stream exceeds configured bound")
	ErrEventStreamDisconnected = errors.New("Hermes SSE stream disconnected")
	ErrInvalidRunID            = errors.New("invalid Hermes run id")
	ErrInvalidEvidence         = errors.New("invalid Hermes evidence")
	ErrCreateRejected          = errors.New("Hermes run creation was proven rejected")
	ErrCreateAmbiguous         = errors.New("Hermes run creation outcome is ambiguous")
)

// TLSConfig is the identity material for a hardened Hermes endpoint. The
// server pin is the SHA-256 digest of the leaf certificate's DER bytes.
type TLSConfig struct {
	RootCAs                *x509.CertPool
	ClientCertificate      tls.Certificate
	ClientCert             tls.Certificate
	ServerCertSHA256       []byte
	PinnedServerCertSHA256 []byte
	ServerCertPin          string
	ServerName             string
	MinVersion             uint16
}

type Config struct {
	BaseURL              string
	ProfileMode          string
	MaxResponseBytes     int64
	MaxEventBytes        int64
	RequestTimeout       time.Duration
	EventTimeout         time.Duration
	MaxEvents            int
	MaxEventStreamBytes  int64
	BearerToken          string
	BearerTokenSource    func() (string, error)
	RequiredCapabilities []string
	TLS                  TLSConfig
}

type Capabilities struct {
	Object   string          `json:"object"`
	Platform string          `json:"platform"`
	Model    string          `json:"model"`
	Auth     CapabilityAuth  `json:"auth"`
	Features map[string]bool `json:"features"`
}

func (c *Capabilities) UnmarshalJSON(data []byte) error {
	type wireCapabilities struct {
		Object    string                     `json:"object"`
		Platform  string                     `json:"platform"`
		Model     string                     `json:"model"`
		Auth      CapabilityAuth             `json:"auth"`
		Features  map[string]json.RawMessage `json:"features"`
		Endpoints json.RawMessage            `json:"endpoints"`
		Runtime   json.RawMessage            `json:"runtime"`
	}
	var wire wireCapabilities
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	features := make(map[string]bool)
	for name, raw := range wire.Features {
		var enabled bool
		if json.Unmarshal(raw, &enabled) == nil {
			features[name] = enabled
		}
	}
	*c = Capabilities{Object: wire.Object, Platform: wire.Platform, Model: wire.Model, Auth: wire.Auth, Features: features}
	return nil
}

type CapabilityAuth struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type Health struct {
	Status   string `json:"status"`
	Platform string `json:"platform,omitempty"`
	Version  string `json:"version,omitempty"`
}

type ApprovalResponse struct {
	Status string `json:"status"`
}

type SteerResponse struct {
	Status string `json:"status"`
}

type CreateRunRequest struct {
	Input               string          `json:"input"`
	SessionID           string          `json:"session_id,omitempty"`
	Instructions        string          `json:"instructions,omitempty"`
	ConversationHistory json.RawMessage `json:"conversation_history,omitempty"`
	PreviousResponseID  string          `json:"previous_response_id,omitempty"`
	Model               string          `json:"model,omitempty"`
	Provider            string          `json:"provider,omitempty"`
	ModelOptions        json.RawMessage `json:"model_options,omitempty"`
}

type SteerRequest struct {
	Input string `json:"input"`
}

type Run struct {
	Object    string  `json:"object,omitempty"`
	RunID     string  `json:"run_id"`
	Status    string  `json:"status"`
	SessionID string  `json:"session_id,omitempty"`
	Model     string  `json:"model,omitempty"`
	Output    string  `json:"output,omitempty"`
	Usage     *Usage  `json:"usage,omitempty"`
	CreatedAt float64 `json:"created_at,omitempty"`
	UpdatedAt float64 `json:"updated_at,omitempty"`
	LastEvent string  `json:"last_event,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type Event struct {
	ID    string
	Type  string
	Retry int
	Data  json.RawMessage
}

// Adapter is the narrow evidence contract consumed by Host orchestration.
// Implementations must not mutate Space state or make policy decisions.
type Adapter interface {
	Capabilities(context.Context) (Capabilities, error)
	Health(context.Context) (Health, error)
	CreateRun(context.Context, CreateRunRequest, string) (Run, error)
	RunStatus(context.Context, string) (Run, error)
	Events(context.Context, string) ([]Event, error)
	ResolveApproval(context.Context, string, string) error
	Steer(context.Context, string, SteerRequest) error
	Stop(context.Context, string) (Run, error)
}

type RuntimeAdapter = Adapter

type Client struct {
	baseURL              *url.URL
	http                 *http.Client
	maxResponseBytes     int64
	maxEventBytes        int64
	maxEvents            int
	maxEventStreamBytes  int64
	requestTimeout       time.Duration
	eventTimeout         time.Duration
	bearerToken          string
	bearerTokenSource    func() (string, error)
	requiredCapabilities []string
}

func New(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u == nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%w: base URL must be an absolute URL without userinfo, query, or fragment", ErrInvalidConfig)
	}
	if cfg.ProfileMode != ProfileDevelopment && cfg.ProfileMode != ProfileHardened {
		return nil, fmt.Errorf("%w: unknown profile mode", ErrInvalidConfig)
	}
	if cfg.MaxResponseBytes <= 0 || cfg.MaxEventBytes <= 0 || cfg.RequestTimeout <= 0 || cfg.EventTimeout <= 0 {
		return nil, fmt.Errorf("%w: response/event bounds and timeouts must be positive", ErrInvalidConfig)
	}
	if cfg.MaxResponseBytes >= 1<<63-1 || cfg.MaxEventBytes >= 1<<63-1 || cfg.MaxEvents < 0 || cfg.MaxEventStreamBytes < 0 || cfg.MaxEventStreamBytes >= 1<<63-1 {
		return nil, fmt.Errorf("%w: response/event bounds are too large", ErrInvalidConfig)
	}
	if cfg.BearerTokenSource == nil && cfg.BearerToken == "" {
		return nil, fmt.Errorf("%w: bearer credential source is required", ErrInvalidConfig)
	}
	if cfg.BearerToken != "" && !visibleToken(cfg.BearerToken) {
		return nil, fmt.Errorf("%w: bearer credential contains invalid characters", ErrInvalidConfig)
	}
	if cfg.ProfileMode == ProfileDevelopment {
		if u.Scheme != "http" || !isLoopback(u.Hostname()) {
			return nil, fmt.Errorf("%w: development profile only permits loopback HTTP", ErrInvalidConfig)
		}
	} else {
		if u.Scheme != "https" {
			return nil, fmt.Errorf("%w: hardened profile requires HTTPS", ErrInvalidConfig)
		}
		if err := validateTLSIdentity(cfg.TLS); err != nil {
			return nil, err
		}
		if cfg.TLS.RootCAs == nil {
			return nil, fmt.Errorf("%w: hardened profile requires explicit CA roots", ErrInvalidConfig)
		}
	}

	required := append([]string(nil), cfg.RequiredCapabilities...)
	if required == nil {
		required = []string{CapabilityRunSubmission, CapabilityRunStatus, CapabilityRunEvents, CapabilityRunApproval, CapabilityRunStop}
	}
	maxEvents := cfg.MaxEvents
	if maxEvents == 0 {
		maxEvents = defaultMaxEvents
	}
	maxStreamBytes := cfg.MaxEventStreamBytes
	if maxStreamBytes == 0 {
		maxStreamBytes = cfg.MaxResponseBytes
	}
	tlsConfig := &tls.Config{MinVersion: cfg.TLS.MinVersion, RootCAs: cfg.TLS.RootCAs, ServerName: cfg.TLS.ServerName}
	if cfg.ProfileMode == ProfileHardened {
		if tlsConfig.MinVersion == 0 {
			tlsConfig.MinVersion = tls.VersionTLS13
		}
		cert := cfg.TLS.ClientCertificate
		if len(cert.Certificate) == 0 {
			cert = cfg.TLS.ClientCert
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		pin := cfg.TLS.ServerCertSHA256
		if len(pin) == 0 {
			pin = cfg.TLS.PinnedServerCertSHA256
		}
		if len(pin) == 0 && cfg.TLS.ServerCertPin != "" {
			pin, err = hex.DecodeString(cfg.TLS.ServerCertPin)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid server certificate pin", ErrInvalidConfig)
			}
		}
		want := append([]byte(nil), pin...)
		previous := tlsConfig.VerifyConnection
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			if previous != nil {
				if err := previous(cs); err != nil {
					return err
				}
			}
			if len(cs.PeerCertificates) == 0 {
				return errors.New("Hermes TLS peer supplied no certificate")
			}
			sum := sha256.Sum256(cs.PeerCertificates[0].Raw)
			if !bytes.Equal(sum[:], want) {
				return errors.New("Hermes TLS server certificate pin mismatch")
			}
			return nil
		}
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		transport = &http.Transport{}
	} else {
		transport = transport.Clone()
	}
	transport.TLSClientConfig = tlsConfig
	return &Client{
		baseURL: u,
		// Each request carries its own bounded context; Events uses the configured
		// event timeout instead of inheriting the ordinary request timeout.
		http:                 &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrRedirectRejected }},
		maxResponseBytes:     cfg.MaxResponseBytes,
		maxEventBytes:        cfg.MaxEventBytes,
		maxEvents:            maxEvents,
		maxEventStreamBytes:  maxStreamBytes,
		requestTimeout:       cfg.RequestTimeout,
		eventTimeout:         cfg.EventTimeout,
		bearerToken:          cfg.BearerToken,
		bearerTokenSource:    cfg.BearerTokenSource,
		requiredCapabilities: required,
	}, nil
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	var out Capabilities
	b, err := c.request(ctx, http.MethodGet, "/v1/capabilities", nil, "application/json", "")
	if err != nil {
		return out, err
	}
	if err := decodeJSON(b, &out); err != nil {
		return out, err
	}
	if out.Object == "" || out.Platform == "" || out.Auth.Type == "" || out.Features == nil {
		return out, fmt.Errorf("%w: incomplete capabilities", ErrCapabilityMismatch)
	}
	if out.Auth.Type != "bearer" || !out.Auth.Required {
		return out, fmt.Errorf("%w: Hermes must require bearer authentication", ErrCapabilityMismatch)
	}
	// Hermes names this capability run_approval_response in its live API;
	// normalize that documented wire alias to Lumen's stable contract name.
	for _, feature := range c.requiredCapabilities {
		enabled := featureEnabled(out.Features, feature)
		if feature == CapabilityRunApproval {
			enabled = enabled || featureEnabled(out.Features, capabilityRunApprovalResponse)
		}
		if !enabled {
			return out, fmt.Errorf("%w: missing %s", ErrCapabilityMismatch, feature)
		}
	}
	return out, nil
}

func (c *Client) Health(ctx context.Context) (Health, error) {
	var out Health
	b, err := c.request(ctx, http.MethodGet, "/health", nil, "application/json", "")
	if err != nil {
		return out, err
	}
	if err := decodeJSON(b, &out); err != nil {
		return out, err
	}
	if out.Status != "ok" {
		return out, fmt.Errorf("%w: invalid health status", ErrInvalidEvidence)
	}
	return out, nil
}

func (c *Client) CreateRun(ctx context.Context, in CreateRunRequest, idempotencyKey string) (Run, error) {
	var out Run
	if !visibleToken(idempotencyKey) || len(idempotencyKey) > 255 {
		return out, fmt.Errorf("%w: invalid idempotency key", ErrCreateRejected)
	}
	b, err := c.request(ctx, http.MethodPost, "/v1/runs", in, "application/json", idempotencyKey)
	if err != nil {
		if provenHTTPRejection(err) {
			return out, fmt.Errorf("%w: %v", ErrCreateRejected, err)
		}
		return out, fmt.Errorf("%w: %v", ErrCreateAmbiguous, err)
	}
	if err := decodeJSON(b, &out); err != nil {
		return out, fmt.Errorf("%w: %v", ErrCreateAmbiguous, err)
	}
	if err := validateRun(out, true); err != nil {
		return out, fmt.Errorf("%w: %v", ErrCreateAmbiguous, err)
	}
	return out, nil
}

func (c *Client) RunStatus(ctx context.Context, runID string) (Run, error) {
	var out Run
	path, err := runPath(runID, "")
	if err != nil {
		return out, err
	}
	b, err := c.request(ctx, http.MethodGet, path, nil, "application/json", "")
	if err != nil {
		return out, err
	}
	if err := decodeJSON(b, &out); err != nil {
		return out, err
	}
	if err := validateRun(out, true); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) ResolveApproval(ctx context.Context, runID, decision string) error {
	if decision != "once" && decision != "deny" {
		return ErrInvalidApproval
	}
	path, err := runPath(runID, "/approval")
	if err != nil {
		return err
	}
	b, err := c.request(ctx, http.MethodPost, path, struct {
		Decision string `json:"decision"`
	}{decision}, "application/json", "")
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return fmt.Errorf("%w: empty approval response", ErrInvalidEvidence)
	}
	var response ApprovalResponse
	if err := decodeJSON(b, &response); err != nil {
		return err
	}
	if !validApprovalStatus(response.Status) {
		return fmt.Errorf("%w: invalid approval status", ErrInvalidEvidence)
	}
	return nil
}

func (c *Client) Steer(ctx context.Context, runID string, in SteerRequest) error {
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return err
	}
	if !featureEnabled(caps.Features, CapabilityRunSteer) {
		return fmt.Errorf("%w: missing %s", ErrUnsupported, CapabilityRunSteer)
	}
	path, err := runPath(runID, "/steer")
	if err != nil {
		return err
	}
	b, err := c.request(ctx, http.MethodPost, path, in, "application/json", "")
	if err != nil {
		return err
	}
	var response SteerResponse
	if err := decodeJSON(b, &response); err != nil {
		return err
	}
	if !validSteerStatus(response.Status) {
		return fmt.Errorf("%w: invalid steer status", ErrInvalidEvidence)
	}
	return nil
}

func featureEnabled(features map[string]bool, name string) bool {
	return features[name]
}

func (c *Client) Stop(ctx context.Context, runID string) (Run, error) {
	var out Run
	path, err := runPath(runID, "/stop")
	if err != nil {
		return out, err
	}
	b, err := c.request(ctx, http.MethodPost, path, struct{}{}, "application/json", "")
	if err != nil {
		return out, err
	}
	if err := decodeJSON(b, &out); err != nil {
		return out, err
	}
	if err := validateRun(out, false); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) request(ctx context.Context, method, path string, body any, accept, idempotencyKey string) ([]byte, error) {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawPath = ""
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	token, err := c.token()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", accept)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, readErr := boundedRead(resp.Body, c.maxResponseBytes, ErrResponseTooLarge)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, safeHTTPError(resp.StatusCode, b)
	}
	return b, nil
}

func (c *Client) token() (string, error) {
	token := c.bearerToken
	if c.bearerTokenSource != nil {
		var err error
		token, err = c.bearerTokenSource()
		if err != nil {
			return "", err
		}
	}
	if !visibleToken(token) {
		return "", fmt.Errorf("%w: bearer credential source returned invalid credential", ErrInvalidConfig)
	}
	return token, nil
}

func decodeJSON(b []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("invalid Hermes JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("invalid Hermes JSON: trailing data")
		}
		return fmt.Errorf("invalid Hermes JSON: trailing data: %w", err)
	}
	return nil
}

func boundedRead(r io.Reader, max int64, tooLarge error) ([]byte, error) {
	if max <= 0 || max >= 1<<63-1 {
		return nil, tooLarge
	}
	b, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, tooLarge
	}
	return b, nil
}

func runPath(runID, suffix string) (string, error) {
	if !opaqueRunID(runID) {
		return "", ErrInvalidRunID
	}
	return "/v1/runs/" + url.PathEscape(runID) + suffix, nil
}

func validateTLSIdentity(cfg TLSConfig) error {
	if cfg.MinVersion != 0 && cfg.MinVersion < tls.VersionTLS13 {
		return fmt.Errorf("%w: hardened TLS requires TLS 1.3", ErrInvalidConfig)
	}
	cert := cfg.ClientCertificate
	if len(cert.Certificate) == 0 {
		cert = cfg.ClientCert
	}
	if len(cert.Certificate) == 0 {
		return fmt.Errorf("%w: hardened TLS requires a client certificate", ErrInvalidConfig)
	}
	pin := cfg.ServerCertSHA256
	if len(pin) == 0 {
		pin = cfg.PinnedServerCertSHA256
	}
	if len(pin) == 0 && cfg.ServerCertPin != "" {
		var err error
		pin, err = hex.DecodeString(cfg.ServerCertPin)
		if err != nil {
			return fmt.Errorf("%w: invalid server certificate pin", ErrInvalidConfig)
		}
	}
	if len(pin) != sha256.Size {
		return fmt.Errorf("%w: hardened TLS requires a 32-byte server certificate pin", ErrInvalidConfig)
	}
	return nil
}

func visibleToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

func isLoopback(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func opaqueRunID(id string) bool {
	if id == "" {
		return false
	}
	for i := 0; i < len(id); i++ {
		if (id[i] < 'a' || id[i] > 'z') && (id[i] < 'A' || id[i] > 'Z') && (id[i] < '0' || id[i] > '9') && id[i] != '_' && id[i] != '-' {
			return false
		}
	}
	return true
}

func validateRun(run Run, requireID bool) error {
	if requireID && !opaqueRunID(run.RunID) {
		return fmt.Errorf("%w: missing or invalid run ID", ErrInvalidEvidence)
	}
	if !validRunStatus(run.Status) {
		return fmt.Errorf("%w: invalid run status", ErrInvalidEvidence)
	}
	return nil
}

func validRunStatus(status string) bool {
	switch status {
	case "queued", "started", "running", "stopping", "awaiting_approval", "completed", "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func validApprovalStatus(status string) bool {
	switch status {
	case "ok", "accepted", "resolved", "denied":
		return true
	default:
		return false
	}
}

func validSteerStatus(status string) bool {
	switch status {
	case "ok", "steered", "queued":
		return true
	default:
		return false
	}
}

func safeHTTPError(status int, body []byte) error {
	digest := sha256.Sum256(body)
	return &HTTPError{Status: status, Digest: digest[:8]}
}

type HTTPError struct {
	Status int
	Digest []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Hermes HTTP status %d (body sha256=%x)", e.Status, e.Digest)
}

func provenHTTPRejection(err error) bool {
	var response *HTTPError
	if !errors.As(err, &response) {
		return false
	}
	switch response.Status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}
