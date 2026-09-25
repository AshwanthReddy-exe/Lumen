package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

// StatusRequest is the first bounded node invocation. A task ID is its
// idempotency key; this read-only action has no retry side effect.
type StatusRequest struct {
	Version           int    `json:"version"`
	SpaceID           string `json:"spaceId"`
	HostID            string `json:"hostId"`
	HostEpoch         int    `json:"hostEpoch"`
	NodeID            string `json:"nodeId"`
	TaskID            string `json:"taskId"`
	CapabilityID      string `json:"capabilityId"`
	Action            string `json:"action"`
	ActionFingerprint string `json:"actionFingerprint"`
	IssuedAt          int64  `json:"issuedAt"`
	ExpiresAt         int64  `json:"expiresAt"`
}

type SignedStatusRequest struct {
	Request   StatusRequest `json:"request"`
	Signature []byte        `json:"signature"`
}

type StatusReceipt struct {
	Version       int    `json:"version"`
	TaskID        string `json:"taskId"`
	NodeID        string `json:"nodeId"`
	HostEpoch     int    `json:"hostEpoch"`
	RequestDigest string `json:"requestDigest"`
	Status        string `json:"status"`
	ObservedAt    int64  `json:"observedAt"`
}

type SignedStatusReceipt struct {
	Receipt   StatusReceipt `json:"receipt"`
	Signature []byte        `json:"signature"`
}

func SignStatusRequest(request StatusRequest, key ed25519.PrivateKey) (SignedStatusRequest, error) {
	if !validStatusRequest(request) || len(key) != ed25519.PrivateKeySize {
		return SignedStatusRequest{}, errors.New("invalid node status request")
	}
	data, err := json.Marshal(request)
	if err != nil {
		return SignedStatusRequest{}, err
	}
	return SignedStatusRequest{Request: request, Signature: ed25519.Sign(key, data)}, nil
}

func validStatusRequest(r StatusRequest) bool {
	return r.Version == 1 && r.SpaceID != "" && r.HostID != "" && r.HostEpoch > 0 && r.NodeID != "" && r.TaskID != "" && r.CapabilityID == "node.status/read" && r.Action == "read" && r.ActionFingerprint != "" && r.IssuedAt > 0 && r.ExpiresAt > r.IssuedAt && r.ExpiresAt-r.IssuedAt <= 300
}

func statusDigest(r StatusRequest) string {
	data, _ := json.Marshal(r)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func strictJSON(reader io.Reader, out any) error {
	data, err := io.ReadAll(io.LimitReader(reader, 16<<10+1))
	if err != nil || len(data) > 16<<10 {
		return errors.New("oversized or unreadable JSON body")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON or oversized body")
	}
	return nil
}

// StatusHandler accepts only a signed, fresh read-only status invocation and
// rechecks local permission on every request. The caller supplies HTTPS.
type StatusHandler struct {
	SpaceID, HostID, NodeID string
	HostEpoch               int
	HostPublicKey           ed25519.PublicKey
	NodePrivateKey          ed25519.PrivateKey
	LocalAllowed            func() bool
	Now                     func() time.Time
}

func (h StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/node/status" {
		http.NotFound(w, r)
		return
	}
	if r.TLS == nil {
		http.Error(w, "denied", http.StatusForbidden)
		return
	}
	var signed SignedStatusRequest
	if strictJSON(r.Body, &signed) != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	data, _ := json.Marshal(signed.Request)
	if len(h.HostPublicKey) != ed25519.PublicKeySize || !ed25519.Verify(h.HostPublicKey, data, signed.Signature) {
		http.Error(w, "denied", http.StatusForbidden)
		return
	}
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}
	q := signed.Request
	if !validStatusRequest(q) || q.SpaceID != h.SpaceID || q.HostID != h.HostID || q.HostEpoch != h.HostEpoch || q.NodeID != h.NodeID || q.IssuedAt > now.Unix()+30 || q.ExpiresAt <= now.Unix() || h.LocalAllowed == nil || !h.LocalAllowed() || len(h.NodePrivateKey) != ed25519.PrivateKeySize {
		http.Error(w, "denied", http.StatusForbidden)
		return
	}
	receipt := StatusReceipt{Version: 1, TaskID: q.TaskID, NodeID: h.NodeID, HostEpoch: h.HostEpoch, RequestDigest: statusDigest(q), Status: "online", ObservedAt: now.Unix()}
	proof, _ := json.Marshal(receipt)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SignedStatusReceipt{Receipt: receipt, Signature: ed25519.Sign(h.NodePrivateKey, proof)})
}

// StatusClient requires an HTTPS client configured to verify the target's TLS
// identity; it also verifies the node's signed, request-bound receipt.
type StatusClient struct {
	HTTPClient     *http.Client
	URL            string
	HostPrivateKey ed25519.PrivateKey
	NodePublicKey  ed25519.PublicKey
	Now            func() time.Time
}

func (c StatusClient) Read(ctx context.Context, request StatusRequest) (StatusReceipt, error) {
	u, err := url.Parse(c.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || c.HTTPClient == nil || len(c.NodePublicKey) != ed25519.PublicKeySize {
		return StatusReceipt{}, errors.New("invalid node endpoint")
	}
	signed, err := SignStatusRequest(request, c.HostPrivateKey)
	if err != nil {
		return StatusReceipt{}, err
	}
	body, _ := json.Marshal(signed)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/v1/node/status", bytes.NewReader(body))
	if err != nil {
		return StatusReceipt{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	client := *c.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("node redirect denied") }
	response, err := client.Do(httpRequest)
	if err != nil {
		return StatusReceipt{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return StatusReceipt{}, errors.New("node status unavailable or denied")
	}
	var result SignedStatusReceipt
	if err := strictJSON(response.Body, &result); err != nil {
		return StatusReceipt{}, err
	}
	proof, _ := json.Marshal(result.Receipt)
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	if !ed25519.Verify(c.NodePublicKey, proof, result.Signature) || result.Receipt.Version != 1 || result.Receipt.TaskID != request.TaskID || result.Receipt.NodeID != request.NodeID || result.Receipt.HostEpoch != request.HostEpoch || result.Receipt.RequestDigest != statusDigest(request) || result.Receipt.Status != "online" || result.Receipt.ObservedAt < request.IssuedAt-30 || result.Receipt.ObservedAt > now.Unix()+30 {
		return StatusReceipt{}, errors.New("invalid node receipt")
	}
	return result.Receipt, nil
}
