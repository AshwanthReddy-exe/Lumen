package host

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
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

// ConfigFromFile loads the non-secret settings emitted by lumen setup.
func ConfigFromFile(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var w struct {
		Version int    `json:"version"`
		DataDir string `json:"data_dir"`
		Hermes  struct {
			BaseURL        string `json:"base_url"`
			Profile        string `json:"profile"`
			BearerFile     string `json:"bearer_file"`
			CAFile         string `json:"ca_file"`
			ClientCertFile string `json:"client_cert_file"`
			ClientKeyFile  string `json:"client_key_file"`
			ServerCertPin  string `json:"server_cert_pin"`
		} `json:"hermes"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	if w.Version != 1 || w.DataDir == "" || w.Hermes.BaseURL == "" || w.Hermes.Profile == "" || w.Hermes.BearerFile == "" {
		return Config{}, errors.New("incomplete generated configuration")
	}
	c := Config{DataDir: w.DataDir, SocketPath: filepath.Join(w.DataDir, "host.sock"), CredentialPath: filepath.Join(w.DataDir, "operator.credential"), HermesBaseURL: w.Hermes.BaseURL, HermesProfile: w.Hermes.Profile, HermesBearerPath: w.Hermes.BearerFile, HermesCAPath: w.Hermes.CAFile, HermesClientCertPath: w.Hermes.ClientCertFile, HermesClientKeyPath: w.Hermes.ClientKeyFile, HermesServerPin: w.Hermes.ServerCertPin}
	if err := c.valid(); err != nil {
		return Config{}, err
	}
	return c, nil
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
	stateExists, keyExists := exists(statePath), exists(keyPath)
	if stateExists || keyExists {
		if !stateExists || !keyExists {
			return errors.New("incomplete initialization state")
		}
		return resumeInitialization(c, marker, statePath, keyPath)
	}
	if exists(c.CredentialPath) {
		return errors.New("operator credential exists without initialized state")
	}
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
		if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		if err := os.Remove(c.CredentialPath); err != nil && !os.IsNotExist(err) {
			rollbackErr = errors.Join(rollbackErr, err)
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
	spaceID, err := newBootstrapID("space")
	if err != nil {
		_ = stateStore.Close()
		return err
	}
	ownerID, err := newBootstrapID("owner")
	if err != nil {
		_ = stateStore.Close()
		return err
	}
	hostID, err := newBootstrapID("host")
	if err != nil {
		_ = stateStore.Close()
		return err
	}
	if ownerID == hostID {
		_ = stateStore.Close()
		return errors.New("bootstrap identities collided")
	}
	if err := stateStore.Initialize(space.State{SchemaVersion: 1, Audit: []space.AuditEvent{}}); err != nil {
		_ = stateStore.Close()
		return err
	}
	if err := bootstrapState(stateStore, spaceID, ownerID, hostID); err != nil {
		return fail(err)
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
	err = writeInitializedMarker(marker)
	if err != nil {
		return fail(err)
	}
	return err
}

func resumeInitialization(c Config, marker, statePath, keyPath string) error {
	stateStore, err := store.Open(statePath, keyPath)
	if err != nil {
		return fmt.Errorf("state unavailable: %w", err)
	}
	state, readErr := stateStore.Read()
	closeErr := stateStore.Close()
	if readErr != nil {
		return fmt.Errorf("state unavailable: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("state unavailable: %w", closeErr)
	}
	if err := validateBootstrapState(state); err != nil {
		return fmt.Errorf("incomplete initialization state: %w", err)
	}
	if exists(c.CredentialPath) {
		if _, err := control.ReadCredential(c.CredentialPath); err != nil {
			return fmt.Errorf("operator credential unavailable: %w", err)
		}
	} else {
		credential, err := control.NewCredential()
		if err != nil {
			return err
		}
		if err := control.WriteCredential(c.CredentialPath, credential); err != nil {
			return err
		}
	}
	if err := writeInitializedMarker(marker); err != nil {
		return err
	}
	return nil
}

func writeInitializedMarker(marker string) error {
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
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
	return err
}

func validateBootstrapState(state space.State) error {
	if state.SchemaVersion != 1 || state.SpaceID == "" || state.OwnerID == "" || state.HostID == "" || state.SpaceID == state.OwnerID || state.SpaceID == state.HostID || state.OwnerID == state.HostID || state.Epoch != 1 {
		return errors.New("invalid bootstrap identities or epoch")
	}
	if len(state.Identities) != 2 || !hasIdentity(state, state.OwnerID, space.IdentityOwner) || !hasIdentity(state, state.HostID, space.IdentityHost) {
		return errors.New("invalid bootstrap identities")
	}
	if len(state.Nodes) != 2 || state.Nodes[state.OwnerID].Status != "paired" || state.Nodes[state.HostID].Status != "paired" {
		return errors.New("invalid bootstrap nodes")
	}
	if len(state.Capabilities) != 1 || state.Capabilities[0] != (space.Capability{ID: "agent.run/execute", Grant: space.GrantAsk}) {
		return errors.New("invalid bootstrap capability")
	}
	if len(state.Advertisements) != 1 || state.Advertisements[0] != (space.CapabilityKey{NodeID: state.HostID, CapabilityID: "agent.run/execute", Action: "run"}) {
		return errors.New("invalid bootstrap advertisement")
	}
	if len(state.Grants) != 1 || state.Grants[state.HostID+"|agent.run/execute|run"] != space.GrantAsk {
		return errors.New("invalid bootstrap grant")
	}
	if len(state.Tasks) != 0 || len(state.Approvals) != 0 || len(state.HostCreates) != 0 || len(state.RuntimeApprovals) != 0 || len(state.HostRuns) != 0 {
		return errors.New("bootstrap contains execution state")
	}
	if len(state.Commands) != 3 || !validBootstrapCommand(state, "bootstrap:create-space", space.CommandCreateSpace) || !validBootstrapCommand(state, "bootstrap:advertise-execute", space.CommandAdvertiseCapability) || !validBootstrapCommand(state, "bootstrap:grant-execute", space.CommandSetGrant) {
		return errors.New("invalid bootstrap command records")
	}
	if len(state.Audit) != 3 {
		return errors.New("invalid bootstrap audit")
	}
	expectedAudit := map[string]struct {
		event     space.AuditEventType
		actor     string
		operation space.CommandType
	}{
		"bootstrap:create-space":      {space.AuditSpaceCreated, state.OwnerID, space.CommandCreateSpace},
		"bootstrap:advertise-execute": {space.AuditCommandAccepted, state.HostID, space.CommandAdvertiseCapability},
		"bootstrap:grant-execute":     {space.AuditCommandAccepted, state.OwnerID, space.CommandSetGrant},
	}
	for _, audit := range state.Audit {
		want, ok := expectedAudit[audit.RequestID]
		if !ok || audit.Event != want.event || audit.ActorID != want.actor || audit.HostID != state.HostID || audit.Epoch != 1 || audit.Operation != want.operation || audit.Outcome != string(space.OutcomeApplied) {
			return errors.New("invalid bootstrap audit")
		}
		delete(expectedAudit, audit.RequestID)
	}
	if len(expectedAudit) != 0 {
		return errors.New("invalid bootstrap audit")
	}
	return nil
}

func hasIdentity(state space.State, id string, kind space.IdentityKind) bool {
	for _, identity := range state.Identities {
		if identity.ID == id && identity.Kind == kind {
			return true
		}
	}
	return false
}

func validBootstrapCommand(state space.State, requestID string, commandType space.CommandType) bool {
	recorded, ok := state.Commands[requestID]
	if !ok || recorded.Type != commandType || recorded.Rejection != "" || recorded.Receipt == nil || recorded.Receipt.RequestID != requestID || recorded.Receipt.Outcome != space.OutcomeApplied {
		return false
	}
	var command space.Command
	if json.Unmarshal([]byte(recorded.Content), &command) != nil || command.RequestID != requestID || command.Type != commandType || command.SpaceID != state.SpaceID || command.HostID != state.HostID {
		return false
	}
	switch commandType {
	case space.CommandCreateSpace:
		return command.OwnerID == state.OwnerID && command.ActorID == state.OwnerID
	case space.CommandAdvertiseCapability:
		return command.Epoch == 1 && command.ActorID == state.HostID && command.NodeID == state.HostID && command.CapabilityID == "agent.run/execute" && command.Action == "run"
	case space.CommandSetGrant:
		return command.Epoch == 1 && command.ActorID == state.OwnerID && command.NodeID == state.HostID && command.CapabilityID == "agent.run/execute" && command.Action == "run" && command.Grant == space.GrantAsk
	default:
		return false
	}
}

func newBootstrapID(prefix string) (string, error) {
	bytes, err := control.NewCredential()
	if err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(bytes), nil
}

func bootstrapState(stateStore *store.Store, spaceID, ownerID, hostID string) error {
	commands := []space.Command{
		{Type: space.CommandCreateSpace, SpaceID: spaceID, OwnerID: ownerID, HostID: hostID, ActorID: ownerID, RequestID: "bootstrap:create-space"},
		{Type: space.CommandAdvertiseCapability, SpaceID: spaceID, HostID: hostID, Epoch: 1, ActorID: hostID, NodeID: hostID, CapabilityID: "agent.run/execute", Action: "run", RequestID: "bootstrap:advertise-execute"},
		{Type: space.CommandSetGrant, SpaceID: spaceID, HostID: hostID, Epoch: 1, ActorID: ownerID, NodeID: hostID, CapabilityID: "agent.run/execute", Action: "run", Grant: space.GrantAsk, RequestID: "bootstrap:grant-execute"},
	}
	for _, command := range commands {
		command := command
		tr, err := stateStore.Update(func(state space.State) space.Transition { return space.Apply(state, command) })
		if err != nil {
			return err
		}
		if tr.Rejection != "" {
			return fmt.Errorf("bootstrap %s rejected: %s", command.Type, tr.Rejection)
		}
	}
	return nil
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
	runtime, err := configuredRuntime(c)
	if err != nil {
		return nil, fmt.Errorf("Hermes configuration unavailable: %w", err)
	}
	return NewWithRuntime(c, runtime)
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
			s.executor.shutdown()
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
		return control.Response{OK: true, Data: map[string]any{"status": "ready", "space_id": state.SpaceID, "owner_id": state.OwnerID, "host_id": state.HostID, "active_host_id": state.HostID, "epoch": state.Epoch, "tasks": state.Tasks}}
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
