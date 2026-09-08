package control

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	MaxFrameSize   = 64 * 1024
	CredentialSize = 32
)

var ErrFrameTooLarge = errors.New("control request exceeds 64 KiB")

type Request struct {
	Credential string            `json:"credential,omitempty"`
	Command    string            `json:"command"`
	Arguments  map[string]string `json:"arguments,omitempty"`
}
type Response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

func ReadRequest(r io.Reader) (Request, error) {
	var q Request
	b, err := bufio.NewReader(io.LimitReader(r, MaxFrameSize+1)).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return q, err
	}
	if len(b) > MaxFrameSize || (err == nil && len(b) == MaxFrameSize && b[len(b)-1] != '\n') {
		return q, ErrFrameTooLarge
	}
	if len(b) == 0 {
		return q, io.ErrUnexpectedEOF
	}
	if err := json.Unmarshal(b, &q); err != nil {
		return q, fmt.Errorf("invalid request: %w", err)
	}
	return q, nil
}
func WriteResponse(w io.Writer, v Response) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > MaxFrameSize {
		return ErrFrameTooLarge
	}
	_, err = w.Write(b)
	return err
}
func CredentialMatches(encoded string, expected []byte) bool {
	got, err := base64.RawStdEncoding.DecodeString(encoded)
	// Compare fixed-size buffers even for malformed or short credentials so the
	// authentication path does not branch on attacker-controlled length.
	var candidate [CredentialSize]byte
	if err == nil && len(got) == len(candidate) {
		copy(candidate[:], got)
	}
	if subtle.ConstantTimeCompare(candidate[:], expected) != 1 {
		return false
	}
	return err == nil && len(got) == len(candidate)
}
func ReadCredential(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm() != 0600 {
		return nil, fmt.Errorf("credential file must be mode 0600")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) != CredentialSize {
		return nil, fmt.Errorf("credential must be exactly %d bytes", CredentialSize)
	}
	return b, nil
}
func WriteCredential(path string, b []byte) error {
	if len(b) != CredentialSize {
		return fmt.Errorf("credential must be exactly %d bytes", CredentialSize)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(b); err != nil {
		return err
	}
	return f.Chmod(0600)
}
func NewCredential() ([]byte, error) {
	b := make([]byte, CredentialSize)
	_, err := rand.Read(b)
	return b, err
}
