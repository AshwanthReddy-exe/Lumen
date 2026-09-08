package control

import (
	"context"
	"net"
	"os"
	"path/filepath"
)

type Handler func(context.Context, Request) Response
type Server struct {
	path       string
	credential []byte
	handler    Handler
	ln         net.Listener
	done       chan struct{}
}

func NewServer(path, credentialPath string, h Handler) (*Server, error) {
	c, err := ReadCredential(credentialPath)
	if err != nil {
		return nil, err
	}
	return &Server{path: path, credential: c, handler: h, done: make(chan struct{})}, nil
}
func (s *Server) Listen() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	_ = os.Remove(s.path)
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
	q, err := ReadRequest(c)
	if err != nil {
		_ = WriteResponse(c, Response{Error: err.Error()})
		return
	}
	if !CredentialMatches(q.Credential, s.credential) {
		_ = WriteResponse(c, Response{Error: "unauthorized"})
		return
	}
	_ = WriteResponse(c, s.handler(context.Background(), q))
}
func (s *Server) Close() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}
