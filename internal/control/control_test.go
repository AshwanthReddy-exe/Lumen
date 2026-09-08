package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
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

func TestReadRequestReturnsAtDelimiterWithoutWaitingForEOF(t *testing.T) {
	r, w := net.Pipe()
	defer r.Close()
	defer w.Close()
	done := make(chan error, 1)
	go func() {
		_, err := ReadRequest(r)
		done <- err
	}()
	_, _ = w.Write([]byte(`{"command":"status"}` + "\n"))
	_ = w.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("ReadRequest waited for EOF after a complete frame")
	}
}

func TestReadResponseRequiresNewlineAndRejectsTrailingData(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{name: "missing delimiter", data: `{"ok":true}`},
		{name: "trailing frame", data: `{"ok":true}` + "\n{}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, w := net.Pipe()
			defer r.Close()
			defer w.Close()
			done := make(chan error, 1)
			go func() {
				_, err := ReadResponse(r)
				done <- err
			}()
			go func() {
				_, _ = w.Write([]byte(tc.data))
				_ = w.Close()
			}()
			if err := <-done; err == nil {
				t.Fatal("expected strict response framing error")
			}
		})
	}
}

func TestReadRequestRejectsDelayedTrailingData(t *testing.T) {
	r, w := net.Pipe()
	defer r.Close()
	done := make(chan error, 1)
	go func() {
		_, err := ReadRequest(r)
		done <- err
	}()
	_, _ = w.Write([]byte(`{"command":"status"}` + "\n"))
	select {
	case err := <-done:
		t.Fatalf("returned before connection end: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	_, _ = w.Write([]byte(`{}`))
	_ = w.Close()
	if err := <-done; err == nil {
		t.Fatal("expected delayed trailing request rejection")
	}
}

func TestReadResponseRejectsDelayedTrailingData(t *testing.T) {
	r, w := net.Pipe()
	defer r.Close()
	done := make(chan error, 1)
	go func() {
		_, err := ReadResponse(r)
		done <- err
	}()
	_, _ = w.Write([]byte(`{"ok":true}` + "\n"))
	select {
	case err := <-done:
		t.Fatalf("returned before connection end: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	_, _ = w.Write([]byte(`{}`))
	_ = w.Close()
	if err := <-done; err == nil {
		t.Fatal("expected delayed trailing response rejection")
	}
}

func TestServerRejectsSymlinkedParent(t *testing.T) {
	d := t.TempDir()
	private := filepath.Join(d, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(d, "operator")
	if err := WriteCredential(credential, bytes.Repeat([]byte{1}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(filepath.Join(link, "host.sock"), credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Listen(); err == nil {
		t.Fatal("expected symlinked parent rejection")
	}
}

func TestServerRejectsSymlinkedParentComponent(t *testing.T) {
	d := shortPrivateDir(t)
	private := filepath.Join(d, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(d, "operator")
	if err := WriteCredential(credential, bytes.Repeat([]byte{5}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(filepath.Join(link, "nested", "host.sock"), credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Listen(); err == nil {
		t.Fatal("expected symlinked parent component rejection")
	}
}

func TestLiveSocketCannotBeStartedTwice(t *testing.T) {
	d := shortPrivateDir(t)
	credential := filepath.Join(d, "operator")
	if err := WriteCredential(credential, bytes.Repeat([]byte{3}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(d, "host.sock")
	first, err := NewServer(path, credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Listen(); err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := NewServer(path, credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Listen(); err == nil {
		t.Fatal("expected live socket ownership rejection")
	}
}

func TestStaleSocketIsReclaimedAfterOwnershipLock(t *testing.T) {
	d := shortPrivateDir(t)
	credential := filepath.Join(d, "operator")
	if err := WriteCredential(credential, bytes.Repeat([]byte{4}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(d, "host.sock")
	first, err := NewServer(path, credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Listen(); err != nil {
		t.Fatal(err)
	}
	if err := first.ln.Close(); err != nil {
		t.Fatal(err)
	}
	first.releaseOwnership()
	s, err := NewServer(path, credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
}

func TestUnknownSocketIsNotEvictedAfterProbeFailure(t *testing.T) {
	d := shortPrivateDir(t)
	credential := filepath.Join(d, "operator")
	if err := WriteCredential(credential, bytes.Repeat([]byte{6}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(d, "host.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if unixListener, ok := listener.(*net.UnixListener); ok {
		unixListener.SetUnlinkOnClose(false)
	}
	s, err := NewServer(path, credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Listen(); err == nil {
		t.Fatal("expected unknown socket ownership rejection")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("unknown socket was evicted: %v", err)
	}
}

func TestAcceptReportsPermanentErrorInsteadOfSpinning(t *testing.T) {
	s := &Server{done: make(chan struct{}), failed: make(chan error, 1), connections: make(map[net.Conn]struct{})}
	failing := &failingListener{err: errors.New("permanent accept failure")}
	s.ln = failing
	go s.accept()
	select {
	case err := <-s.Errors():
		if !errors.Is(err, failing.err) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("accept loop did not report permanent failure")
	}
	if calls := atomic.LoadInt32(&failing.calls); calls != 1 {
		t.Fatalf("accept called %d times after permanent failure", calls)
	}
}

func shortPrivateDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/private/tmp", "lh-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

type failingListener struct {
	calls int32
	err   error
}

func (l *failingListener) Accept() (net.Conn, error) {
	atomic.AddInt32(&l.calls, 1)
	return nil, l.err
}
func (l *failingListener) Close() error   { return nil }
func (l *failingListener) Addr() net.Addr { return &net.UnixAddr{Name: "test", Net: "unix"} }

func TestCloseDoesNotRemoveReplacedSocket(t *testing.T) {
	private, err := os.MkdirTemp("/private/tmp", "lh-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(private)
	if err := os.Chmod(private, 0700); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(private, "operator")
	if err := WriteCredential(credential, bytes.Repeat([]byte{2}, CredentialSize)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(private, "s")
	s, err := NewServer(path, credential, func(_ context.Context, _ Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("replacement setup: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("replacement socket was removed: %v", err)
	}
}

func TestWriteResponseUsesBoundedNewlineFrame(t *testing.T) {
	var b bytes.Buffer
	if err := WriteResponse(&b, Response{OK: true}); err != nil {
		t.Fatal(err)
	}
	var decoded Response
	if err := json.Unmarshal(bytes.TrimSuffix(b.Bytes(), []byte{'\n'}), &decoded); err != nil {
		t.Fatal(err)
	}
}
