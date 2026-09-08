package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

type Config struct {
	DataDir, SocketPath, CredentialPath  string
	HermesBaseURL, HermesProfile         string
	HermesBearerPath                     string
	HermesCAPath, HermesClientCertPath   string
	HermesClientKeyPath, HermesServerPin string
}

var (
	markerSyncFile   = func(f *os.File) error { return f.Sync() }
	markerSyncParent = syncMarkerParent
)

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
	p := os.Getenv("LUMEN_HERMES_PROFILE")
	if p == "" {
		p = hermes.ProfileHardened
	}
	bearer := os.Getenv("LUMEN_HERMES_BEARER_FILE")
	if bearer == "" {
		bearer = filepath.Join(d, "hermes.token")
	}
	return Config{DataDir: d, SocketPath: s, CredentialPath: c, HermesBaseURL: os.Getenv("LUMEN_HERMES_BASE_URL"), HermesProfile: p, HermesBearerPath: bearer, HermesCAPath: os.Getenv("LUMEN_HERMES_CA_FILE"), HermesClientCertPath: os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"), HermesClientKeyPath: os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"), HermesServerPin: os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN")}, nil
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
	if c.HermesProfile != "" && c.HermesProfile != hermes.ProfileDevelopment && c.HermesProfile != hermes.ProfileHardened {
		return errors.New("invalid Hermes profile")
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
	if err := os.Chmod(c.DataDir, 0700); err != nil {
		return err
	}
	marker := filepath.Join(c.DataDir, "initialized")
	if exists(marker) {
		return errors.New("already initialized")
	}
	statePath := filepath.Join(c.DataDir, "state.json")
	keyPath := filepath.Join(c.DataDir, "state.key")
	keyBefore := exists(keyPath)
	credentialBefore := exists(c.CredentialPath)
	stateStore, err := store.New(statePath, keyPath)
	if err != nil {
		return err
	}
	rollback := func() error {
		var rollbackErr error
		_ = stateStore.Close()
		if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		if err := os.Remove(statePath + ".lock"); err != nil && !os.IsNotExist(err) {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		if !keyBefore {
			if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		if !credentialBefore {
			if err := os.Remove(c.CredentialPath); err != nil && !os.IsNotExist(err) {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		if err := syncMarkerParentReal(c.DataDir); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		if filepath.Dir(c.CredentialPath) != c.DataDir {
			if err := syncMarkerParentReal(filepath.Dir(c.CredentialPath)); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		return rollbackErr
	}
	fail := func(primary error) error {
		if rollbackErr := rollback(); rollbackErr != nil {
			return fmt.Errorf("%w; initialization rollback failed: %v", primary, rollbackErr)
		}
		return primary
	}
	if err := stateStore.Initialize(space.State{SchemaVersion: 1, Audit: []space.AuditEvent{}}); err != nil {
		_ = stateStore.Close()
		return err
	}
	if err := stateStore.Close(); err != nil {
		return fail(err)
	}
	cred, err := control.NewCredential()
	if err != nil {
		return fail(err)
	}
	if err := control.WriteCredential(c.CredentialPath, cred); err != nil {
		return fail(err)
	}
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fail(err)
	}
	_, err = f.WriteString("lumen-host v1\n")
	if err == nil {
		err = markerSyncFile(f)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = markerSyncParent(marker)
	}
	if err != nil {
		return fail(err)
	}
	return err
}

func syncMarkerParent(path string) error {
	return syncMarkerParentReal(path)
}

func syncMarkerParentReal(path string) error {
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	err = d.Sync()
	if closeErr := d.Close(); err == nil {
		err = closeErr
	}
	return err
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

type Service struct {
	cfg      Config
	server   *control.Server
	state    *store.Store
	executor *executor
	ready    chan struct{}
	stop     chan struct{}
	once     sync.Once
}

func New(c Config) (*Service, error) {
	return NewWithRuntime(c, configuredRuntime(c))
}
func (s *Service) Start() error {
	srv, err := control.NewServer(s.cfg.SocketPath, s.cfg.CredentialPath, s.handle)
	if err != nil {
		if s.state != nil {
			_ = s.state.Close()
		}
		if s.executor != nil {
			s.executor.cancel()
		}
		return err
	}
	s.server = srv
	srv.OnResponse(func(q control.Request, r control.Response) {
		if q.Command == "shutdown" && r.OK {
			go s.Shutdown()
		}
	})
	if err := srv.Listen(); err != nil {
		if s.state != nil {
			_ = s.state.Close()
		}
		if s.executor != nil {
			s.executor.cancel()
		}
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
		if s.executor != nil {
			s.executor.cancel()
			s.executor.waitConsumers()
		}
		if s.server != nil {
			_ = s.server.Close()
		}
		if s.state != nil {
			_ = s.state.Close()
		}
		close(s.stop)
	})
}
func (s *Service) handle(ctx context.Context, q control.Request) control.Response {
	switch q.Command {
	case "status":
		state, err := s.state.Read()
		if err != nil {
			return control.Response{Error: "state unavailable"}
		}
		return control.Response{OK: true, Data: map[string]any{"status": "ready", "space_id": state.SpaceID, "tasks": state.Tasks}}
	case "shutdown":
		return control.Response{OK: true, Data: map[string]string{"status": "shutting_down"}}
	case "task submit":
		return s.handleTaskSubmit(ctx, q.Arguments)
	case "task show":
		return s.handleTaskShow(q.Arguments)
	case "task cancel":
		return s.handleTaskCancel(ctx, q.Arguments)
	case "approval resolve":
		return s.handleApprovalResolve(ctx, q.Arguments)
	default:
		return control.Response{Error: "unsupported command"}
	}
}
