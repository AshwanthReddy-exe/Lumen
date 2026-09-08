package control

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Handler func(context.Context, Request) Response
type Server struct {
	path        string
	credential  []byte
	handler     Handler
	ln          net.Listener
	done        chan struct{}
	connections map[net.Conn]struct{}
	mu          sync.Mutex
}

func NewServer(path, credentialPath string, h Handler) (*Server, error) {
	c, err := ReadCredential(credentialPath)
	if err != nil {
		return nil, err
	}
	return &Server{path: path, credential: c, handler: h, done: make(chan struct{}), connections: make(map[net.Conn]struct{})}, nil
}
func (s *Server) Listen() error {
	if st, err := os.Stat(filepath.Dir(s.path)); err == nil && (!st.IsDir() || st.Mode().Perm()&0077 != 0) {
		return fmt.Errorf("control parent must be private")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	if st, err := os.Lstat(s.path); err == nil {
		if st.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("control path is not a socket")
		}
		probe, e := net.DialTimeout("unix", s.path, 100*time.Millisecond)
		if e == nil {
			_ = probe.Close()
			return fmt.Errorf("control socket already active")
		}
		if e := os.Remove(s.path); e != nil {
			return e
		}
	}
	ln, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	s.ln = ln
	if err := os.Chmod(s.path, 0600); err != nil {
		ln.Close()
		return err
	}
	go s.accept()
	return nil
}
func (s *Server) accept() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
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
}
func (s *Server) Close() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	if s.ln != nil {
		err := s.ln.Close()
		s.mu.Lock()
		for c := range s.connections {
			_ = c.Close()
		}
		s.mu.Unlock()
		_ = os.Remove(s.path)
		return err
	}
	return nil
}
