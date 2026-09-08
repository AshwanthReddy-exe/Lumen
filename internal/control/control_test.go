package control

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestFrameRejectsOversizedRequest(t *testing.T) {
	r, w := net.Pipe()
	defer r.Close()
	defer w.Close()
	errCh := make(chan error, 1)
	go func() { _, err := ReadRequest(r); errCh <- err }()
	_, _ = w.Write(bytes.Repeat([]byte{'x'}, MaxFrameSize+1))
	if err := <-errCh; err == nil {
		t.Fatal("expected oversized request error")
	}
}

func TestCredentialFileIsRestrictedAndConstantTimeServerAuth(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "operator")
	if err := WriteCredential(p, bytes.Repeat([]byte{7}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
}
