package store

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

const (
	FormatVersion = 1
	CipherSuite   = "AES-256-GCM"
	keySize       = 32
	nonceSize     = 12
)

// Envelope is the on-disk representation. The header is authenticated as GCM
// additional data, so changing any visible field invalidates the ciphertext.
type Envelope struct {
	FormatVersion int               `json:"formatVersion"`
	KeyID         string            `json:"keyId"`
	CipherSuite   string            `json:"cipherSuite"`
	Nonce         string            `json:"nonce"`
	Metadata      map[string]string `json:"metadata"`
	Ciphertext    string            `json:"ciphertext"`
}

type envelopeHeader struct {
	FormatVersion int               `json:"formatVersion"`
	KeyID         string            `json:"keyId"`
	CipherSuite   string            `json:"cipherSuite"`
	Nonce         string            `json:"nonce"`
	Metadata      map[string]string `json:"metadata"`
}

func keyID(key []byte) string {
	h := sha256.Sum256(key)
	return fmt.Sprintf("sha256:%x", h[:])
}

func (e Envelope) header() (envelopeHeader, []byte, error) {
	if e.FormatVersion != FormatVersion || e.CipherSuite != CipherSuite || e.KeyID == "" {
		return envelopeHeader{}, nil, fmt.Errorf("unsupported envelope")
	}
	nonce, err := base64.RawStdEncoding.DecodeString(e.Nonce)
	if err != nil || len(nonce) != nonceSize {
		return envelopeHeader{}, nil, fmt.Errorf("invalid nonce")
	}
	if e.Metadata == nil {
		return envelopeHeader{}, nil, fmt.Errorf("missing metadata")
	}
	h := envelopeHeader{e.FormatVersion, e.KeyID, e.CipherSuite, e.Nonce, e.Metadata}
	b, err := json.Marshal(h)
	if err != nil {
		return envelopeHeader{}, nil, err
	}
	return h, b, nil
}

func encodeEnvelope(key []byte, state space.State) ([]byte, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("state key must be exactly %d bytes", keySize)
	}
	nonce := make([]byte, nonceSize)
	if _, err := randomRead(nonce); err != nil {
		return nil, err
	}
	metadata := map[string]string{"stateSchemaVersion": fmt.Sprint(state.SchemaVersion)}
	e := Envelope{FormatVersion: FormatVersion, KeyID: keyID(key), CipherSuite: CipherSuite, Nonce: base64.RawStdEncoding.EncodeToString(nonce), Metadata: metadata}
	_, aad, err := e.header()
	if err != nil {
		return nil, err
	}
	block, err := newCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := newGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	e.Ciphertext = base64.RawStdEncoding.EncodeToString(aead.Seal(nil, nonce, plain, aad))
	return json.Marshal(e)
}

func decodeEnvelope(key []byte, data []byte) (space.State, error) {
	var e Envelope
	if err := strictJSON(data, &e); err != nil {
		return space.State{}, fmt.Errorf("invalid envelope: %w", err)
	}
	if len(key) != keySize || e.KeyID != keyID(key) {
		return space.State{}, fmt.Errorf("state key mismatch")
	}
	_, aad, err := e.header()
	if err != nil {
		return space.State{}, err
	}
	nonce, _ := base64.RawStdEncoding.DecodeString(e.Nonce)
	ciphertext, err := base64.RawStdEncoding.DecodeString(e.Ciphertext)
	if err != nil || len(ciphertext) == 0 {
		return space.State{}, fmt.Errorf("invalid ciphertext")
	}
	block, err := newCipher(key)
	if err != nil {
		return space.State{}, err
	}
	aead, err := newGCM(block)
	if err != nil {
		return space.State{}, err
	}
	plain, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return space.State{}, fmt.Errorf("ciphertext authentication failed")
	}
	var state space.State
	if err := strictJSON(plain, &state); err != nil {
		return space.State{}, fmt.Errorf("invalid state: %w", err)
	}
	if state.SchemaVersion != FormatVersion {
		return space.State{}, fmt.Errorf("unsupported state schema")
	}
	return state, nil
}
