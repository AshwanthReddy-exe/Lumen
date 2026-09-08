package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
)

type Config struct{ DataDir, SocketPath, CredentialPath string }

func LoadConfig() (Config, error) {
	d := os.Getenv("LUMEN_DATA_DIR")
	if d == "" {
		return Config{}, errors.New("LUMEN_DATA_DIR is required")
	}
	s := os.Getenv("LUMEN_SOCKET_PATH")
	if s == "" {
		s = filepath.Join(d, "host.sock")
	}
	c := os.Getenv("LUMEN_OPERATOR_CREDENTIAL_FILE")
	if c == "" {
		c = filepath.Join(d, "operator.credential")
	}
	return Config{d, s, c}, nil
}
func (c Config) valid() error {
	if c.DataDir == "" || c.SocketPath == "" || c.CredentialPath == "" {
		return errors.New("incomplete configuration")
	}
	for _, p := range []string{c.DataDir, c.SocketPath, c.CredentialPath} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return errors.New("configuration paths must be absolute and clean")
		}
	}
	return nil
}
func Initialize(c Config) error {
	if err := c.valid(); err != nil {
		return err
	}
	if err := os.MkdirAll(c.DataDir, 0700); err != nil {
		return err
	}
	marker := filepath.Join(c.DataDir, "initialized")
	if _, err := os.Stat(marker); err == nil {
		return errors.New("already initialized")
	}
	cred, err := control.NewCredential()
	if err != nil {
		return err
	}
	if err := control.WriteCredential(c.CredentialPath, cred); err != nil {
		return err
	}
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("lumen-host v1\n")
	return err
}

type Service struct {
	cfg    Config
	server *control.Server
	ready  chan struct{}
	stop   chan struct{}
	once   sync.Once
}

func New(c Config) (*Service, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(c.DataDir, "initialized")); err != nil {
		return nil, fmt.Errorf("state unavailable: %w", err)
	}
	return &Service{cfg: c, ready: make(chan struct{}), stop: make(chan struct{})}, nil
}
func (s *Service) Start() error {
	srv, err := control.NewServer(s.cfg.SocketPath, s.cfg.CredentialPath, s.handle)
	if err != nil {
		return err
	}
	s.server = srv
	srv.OnResponse(func(q control.Request, r control.Response) {
		if q.Command == "shutdown" && r.OK {
			go s.Shutdown()
		}
	})
	if err := srv.Listen(); err != nil {
		return err
	}
	close(s.ready)
	return nil
}
func (s *Service) Ready() <-chan struct{} { return s.ready }
func (s *Service) Wait(ctx context.Context) error {
	select {
	case <-s.stop:
		return nil
	case err := <-s.server.Errors():
		return fmt.Errorf("control listener failed: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Service) Shutdown() {
	s.once.Do(func() {
		if s.server != nil {
			_ = s.server.Close()
		}
		close(s.stop)
	})
}
func (s *Service) handle(_ context.Context, q control.Request) control.Response {
	switch q.Command {
	case "status":
		return control.Response{OK: true, Data: map[string]string{"status": "ready"}}
	case "shutdown":
		return control.Response{OK: true, Data: map[string]string{"status": "shutting_down"}}
	default:
		return control.Response{Error: "unsupported command"}
	}
}
