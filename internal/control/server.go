package control

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Handler func(context.Context, Request) Response
type Server struct {
	path        string
	credential  []byte
	handler     Handler
	ln          net.Listener
	done        chan struct{}
	failed      chan error
	connections map[net.Conn]struct{}
	parent      *os.File
	lock        *os.File
	socketInfo  os.FileInfo
	onResponse  func(Request, Response)
	mu          sync.Mutex
	ownershipMu sync.Mutex
	closeOnce   sync.Once
}

func NewServer(path, credentialPath string, h Handler) (*Server, error) {
	c, err := ReadCredential(credentialPath)
	if err != nil {
		return nil, err
	}
	return &Server{path: path, credential: c, handler: h, done: make(chan struct{}), failed: make(chan error, 1), connections: make(map[net.Conn]struct{})}, nil
}
func (s *Server) Listen() error {
	parent, err := openPrivateParent(filepath.Dir(s.path))
	if err != nil {
		return err
	}
	lock, err := os.OpenFile(s.path+".lock", os.O_RDWR|os.O_CREATE|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		parent.Close()
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		parent.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fmt.Errorf("control socket already active")
		}
		return err
	}
	s.parent, s.lock = parent, lock
	parentInfo, err := parent.Stat()
	if err != nil {
		s.releaseOwnership()
		return err
	}
	if st, err := os.Lstat(s.path); err == nil {
		if st.Mode()&os.ModeSocket == 0 {
			s.releaseOwnership()
			return fmt.Errorf("control path is not a socket")
		}
		owner, ownerErr := readLockIdentity(lock)
		if ownerErr != nil || !sameIdentity(st, owner) {
			s.releaseOwnership()
			return fmt.Errorf("control socket ownership is unknown")
		}
		if e := os.Remove(s.path); e != nil && !errors.Is(e, os.ErrNotExist) {
			s.releaseOwnership()
			return e
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		s.releaseOwnership()
		return err
	}
	ln, err := net.Listen("unix", s.path)
	if err != nil {
		s.releaseOwnership()
		return err
	}
	s.ln = ln
	s.socketInfo, err = os.Lstat(s.path)
	if err != nil {
		_ = ln.Close()
		s.releaseOwnership()
		return err
	}
	if unixListener, ok := ln.(*net.UnixListener); ok {
		unixListener.SetUnlinkOnClose(false)
	}
	if err := writeLockIdentity(lock, s.socketInfo); err != nil {
		_ = ln.Close()
		s.removeOwnedSocket()
		s.releaseOwnership()
		return err
	}
	if current, statErr := os.Stat(filepath.Dir(s.path)); statErr != nil || !os.SameFile(current, parentInfo) {
		_ = ln.Close()
		s.removeOwnedSocket()
		s.releaseOwnership()
		return fmt.Errorf("control parent changed during bind")
	}
	if err := os.Chmod(s.path, 0600); err != nil {
		_ = ln.Close()
		s.removeOwnedSocket()
		s.releaseOwnership()
		return err
	}
	go s.accept()
	return nil
}

func openPrivateParent(dir string) (*os.File, error) {
	if err := rejectSymlinkComponents(dir); err != nil {
		return nil, err
	}
	st, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("control parent must not be a symlink")
	}
	f, err := os.OpenFile(dir, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	st, err = f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !st.IsDir() || st.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, fmt.Errorf("control parent must be private")
	}
	if stat, ok := st.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Getuid()) {
		f.Close()
		return nil, fmt.Errorf("control parent owner mismatch")
	}
	return f, nil
}

func rejectSymlinkComponents(dir string) error {
	for current := filepath.Clean(dir); ; current = filepath.Dir(current) {
		st, err := os.Lstat(current)
		if err == nil {
			if st.Mode()&os.ModeSymlink != 0 && current != "/tmp" && current != "/var" {
				return fmt.Errorf("control parent must not contain symlinks")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func fileIdentity(info os.FileInfo) (uint64, uint64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return uint64(stat.Dev), uint64(stat.Ino), true
}

func sameIdentity(info os.FileInfo, identity [2]uint64) bool {
	dev, ino, ok := fileIdentity(info)
	return ok && dev == identity[0] && ino == identity[1]
}

func readLockIdentity(lock *os.File) ([2]uint64, error) {
	var identity [2]uint64
	if _, err := lock.Seek(0, 0); err != nil {
		return identity, err
	}
	b, err := io.ReadAll(io.LimitReader(lock, 128))
	if err != nil {
		return identity, err
	}
	if _, err := fmt.Sscanf(string(b), "%d:%d", &identity[0], &identity[1]); err != nil {
		return identity, err
	}
	return identity, nil
}

func writeLockIdentity(lock *os.File, info os.FileInfo) error {
	dev, ino, ok := fileIdentity(info)
	if !ok {
		return fmt.Errorf("control socket identity unavailable")
	}
	if err := lock.Truncate(0); err != nil {
		return err
	}
	if _, err := lock.Seek(0, 0); err != nil {
		return err
	}
	_, err := fmt.Fprintf(lock, "%d:%d\n", dev, ino)
	return err
}

func (s *Server) releaseOwnership() {
	s.ownershipMu.Lock()
	defer s.ownershipMu.Unlock()
	if s.lock != nil {
		_ = syscall.Flock(int(s.lock.Fd()), syscall.LOCK_UN)
		_ = s.lock.Close()
		s.lock = nil
	}
	if s.parent != nil {
		_ = s.parent.Close()
		s.parent = nil
	}
}

func (s *Server) Errors() <-chan error { return s.failed }

func (s *Server) OnResponse(f func(Request, Response)) { s.onResponse = f }

func (s *Server) accept() {
	var delay time.Duration
	for {
		c, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
					if delay > 250*time.Millisecond {
						delay = 250 * time.Millisecond
					}
				}
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
				case <-s.done:
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					return
				}
				continue
			}
			select {
			case s.failed <- err:
			default:
			}
			_ = s.Close()
			return
		}
		delay = 0
		go s.handle(c)
	}
}
func (s *Server) handle(c net.Conn) {
	defer c.Close()
	s.mu.Lock()
	s.connections[c] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.connections, c); s.mu.Unlock() }()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q, err := ReadRequest(c)
	if err != nil {
		_ = WriteResponse(c, Response{Error: err.Error()})
		return
	}
	if !CredentialMatches(q.Credential, s.credential) {
		_ = WriteResponse(c, Response{Error: "unauthorized"})
		return
	}
	resp := Response{Error: "handler failure"}
	func() { defer func() { _ = recover() }(); resp = s.handler(ctx, q) }()
	_ = WriteResponse(c, resp)
	if s.onResponse != nil {
		s.onResponse(q, resp)
	}
}
func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.done)
		if s.ln != nil {
			err = s.ln.Close()
		}
		s.mu.Lock()
		for c := range s.connections {
			_ = c.Close()
		}
		s.mu.Unlock()
		s.removeOwnedSocket()
		s.releaseOwnership()
	})
	return err
}

func (s *Server) removeOwnedSocket() {
	if s.socketInfo == nil {
		return
	}
	if current, err := os.Lstat(s.path); err == nil && current.Mode()&os.ModeSocket != 0 && os.SameFile(current, s.socketInfo) {
		_ = os.Remove(s.path)
	}
}
