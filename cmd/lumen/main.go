package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

const actionRequired = setup.ActionRequired
const lumenVersion = "0.0.0"

func runForTest(args []string) setup.Report { return run(args) }

func run(args []string) setup.Report {
	ctx := context.Background()
	if len(args) == 1 {
		switch args[0] {
		case "connect":
			return setup.Report{Outcome: setup.ActionRequired, AvailableInBlock: 3}
		case "setup":
			return setupCommand(ctx)
		case "doctor":
			return doctorCommand(ctx)
		}
	}
	if len(args) == 2 && args[0] == "service" {
		switch args[1] {
		case "start", "stop", "restart", "status":
			return serviceCommand(ctx, args[1])
		}
	}
	return placeholder("invalid_command")
}

func setupCommand(ctx context.Context) setup.Report {
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	if dataDir == "" {
		return placeholder("configuration_required")
	}
	profile, err := requestedProfile()
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Actions: []setup.Action{{Code: "invalid_profile"}}}
	}
	topology, err := requestedTopology()
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "invalid_topology"}}}
	}
	probe := cliProbe{dataDir: dataDir}
	plan, err := setup.Plan(ctx, setup.Request{Topology: topology, Profile: profile}, probe)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: setupErrorCode(err)}}}
	}
	if plan.Outcome != setup.Ready {
		r := setup.Report{Outcome: plan.Outcome, Profile: profile, Topology: topology}
		for _, action := range plan.Actions {
			r.Actions = append(r.Actions, setup.Action{Code: action.Code})
		}
		if len(r.Actions) == 0 {
			r.Actions = []setup.Action{{Code: "setup_plan_requires_action"}}
		}
		return r
	}
	state := &setupState{plan: plan, dataDir: dataDir, profile: profile, topology: topology}
	if topology == setup.TopologyExternal {
		adoption, err := adoptExternalHermes(ctx, plan, profile)
		if err != nil {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "external_adoption_failed"}}}
		}
		adoptionPath := filepath.Join(plan.SetupDir, "external-adoption.json")
		if _, statErr := os.Lstat(adoptionPath); statErr == nil {
			existing, loadErr := setup.LoadExternalAdoption(adoptionPath)
			if loadErr != nil || existing != adoption {
				return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "external_adoption_failed"}}}
			}
			state.adoption = &existing
		} else if errors.Is(statErr, os.ErrNotExist) {
			if err := ensurePrivateDir(plan.SetupDir); err != nil {
				return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "journal_unavailable"}}}
			}
			if err := setup.SaveExternalAdoption(adoptionPath, adoption); err != nil {
				return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "external_adoption_failed"}}}
			}
			state.adoption = &adoption
		} else {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "external_adoption_failed"}}}
		}
	}
	journal, err := setup.NewJournal(plan.SetupDir)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "journal_unavailable"}}}
	}
	request := setup.Request{Topology: topology, Profile: profile, ArtifactDigests: state.artifactDigests()}
	if state.adoption != nil {
		request.EndpointOriginDigest = state.adoption.EndpointOriginDigest
		request.EndpointIdentityDigest = state.adoption.EndpointIdentityDigest
	}
	runner := setup.Runner{
		Journal: journal,
		Plan:    &plan,
		Initializer: setup.HostInitializerFuncs{
			InitializeFunc: func(context.Context) error {
				cfg, e := state.hostConfig()
				if e != nil {
					return e
				}
				e = host.Initialize(cfg)
				if errors.Is(e, host.ErrAlreadyInitialized) {
					return nil
				}
				return e
			},
			VerifyFunc: func(context.Context) error {
				cfg, e := state.hostConfig()
				if e != nil {
					return e
				}
				_, e = host.VerifyInitialized(cfg)
				return e
			},
		},
		RunStage: state.runStage,
		Verify:   state.verifyStage,
	}
	report, _ := runner.Run(ctx, request)
	return report
}

func doctorCommand(ctx context.Context) setup.Report {
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	if dataDir == "" {
		return placeholder("configuration_required")
	}
	profile, err := requestedProfile()
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Actions: []setup.Action{{Code: "invalid_profile"}}}
	}
	topology, err := requestedTopology()
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "invalid_topology"}}}
	}
	state := &setupState{dataDir: dataDir, profile: profile, topology: topology}
	if plan, e := setup.Plan(ctx, setup.Request{Topology: topology, Profile: profile}, cliProbe{dataDir: dataDir}); e == nil {
		state.plan = plan
	}
	return (setup.Doctor{Observe: state.observe}).Check(ctx)
}

func serviceCommand(ctx context.Context, action string) setup.Report {
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	if dataDir == "" {
		return placeholder("configuration_required")
	}
	profile, err := requestedProfile()
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Actions: []setup.Action{{Code: "invalid_profile"}}}
	}
	topology, err := requestedTopology()
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "invalid_topology"}}}
	}
	services := ownedServices(topology)
	if len(services) == 0 {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "invalid_topology"}}}
	}
	manager, available := (cliProbe{dataDir: dataDir}).Supervisor()
	if !available {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "supervisor_unavailable"}}}
	}
	states, err := (setup.CommandSupervisor{Manager: manager}).Control(ctx, setup.Action{Code: action}, services)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "service_unavailable"}}}
	}
	r := setup.Report{Outcome: setup.Ready, Profile: profile, States: map[string]string{}}
	for _, s := range states {
		r.States[string(s.Name)] = s.State
		if s.State != setup.StateRunning && action != "stop" {
			r.Outcome = setup.ActionRequired
		}
	}
	return r
}

func requestedTopology() (setup.Topology, error) {
	topology := setup.Topology(os.Getenv("LUMEN_SETUP_TOPOLOGY"))
	if err := topology.Validate(); err != nil {
		return "", err
	}
	return topology, nil
}

func ownedServices(topology setup.Topology) []setup.ServiceName {
	switch topology {
	case setup.TopologyCombined:
		return []setup.ServiceName{setup.ServiceHermes, setup.ServiceHost}
	case setup.TopologyExternal:
		return []setup.ServiceName{setup.ServiceHost}
	default:
		return nil
	}
}

func requestedProfile() (setup.Profile, error) {
	p := setup.Profile(os.Getenv("LUMEN_SETUP_PROFILE"))
	if p == "" {
		p = setup.Development
	}
	if p != setup.Development && p != setup.PersonalAlpha && p != setup.Hardened {
		return "", errors.New("invalid profile")
	}
	return p, nil
}

type cliProbe struct{ dataDir string }

func (p cliProbe) GOOS() string         { return runtime.GOOS }
func (p cliProbe) GOARCH() string       { return runtime.GOARCH }
func (p cliProbe) TermuxPrefix() string { return os.Getenv("PREFIX") }
func (p cliProbe) SetupDir() string     { return filepath.Join(p.dataDir, "setup") }
func (p cliProbe) HardenedIsolation() bool {
	return os.Getenv("LUMEN_HERMES_ISOLATED") == "1"
}
func (p cliProbe) InstalledVersions() (string, string, bool) {
	return os.Getenv("LUMEN_LUMEN_VERSION"), os.Getenv("LUMEN_HERMES_VERSION"), os.Getenv("LUMEN_HERMES_ADOPTED") == "1"
}
func (p cliProbe) Supervisor() (setup.Supervisor, bool) {
	if configured := setup.Supervisor(os.Getenv("LUMEN_SUPERVISOR")); configured != "" {
		switch configured {
		case setup.SupervisorLaunchd, setup.SupervisorSystemd, setup.SupervisorRunit, setup.SupervisorDocker:
			return configured, true
		default:
			return configured, false
		}
	}
	for _, candidate := range []struct {
		manager setup.Supervisor
		binary  string
	}{{setup.SupervisorLaunchd, "launchctl"}, {setup.SupervisorSystemd, "systemctl"}, {setup.SupervisorRunit, "sv"}, {setup.SupervisorDocker, "docker"}} {
		if _, err := exec.LookPath(candidate.binary); err == nil {
			return candidate.manager, true
		}
	}
	return "", false
}

type setupState struct {
	plan      setup.PlanResult
	dataDir   string
	profile   setup.Profile
	topology  setup.Topology
	adoption  *setup.ExternalAdoption
	manager   setup.Supervisor
	installed bool
	started   bool
}

func (s *setupState) hostConfig() (host.Config, error) {
	if p := filepath.Join(s.dataDir, "lumen.json"); fileExists(p) {
		return host.ConfigFromFile(p)
	}
	cfg := host.Config{
		DataDir: s.dataDir, SocketPath: filepath.Join(s.dataDir, "host.sock"), CredentialPath: filepath.Join(s.dataDir, "operator.credential"),
		HermesBaseURL: hermesEndpoint(), HermesProfile: string(hermesProfile(s.profile)), HermesBearerPath: filepath.Join(s.dataDir, "hermes", "hermes.token"),
		HermesCAPath: os.Getenv("LUMEN_HERMES_CA_FILE"), HermesClientCertPath: os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"), HermesClientKeyPath: os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"), HermesServerPin: os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN"),
	}
	if s.adoption != nil {
		cfg.HermesBaseURL = s.adoption.Endpoint
		if s.profile == setup.PersonalAlpha || s.profile == setup.Hardened {
			cfg.HermesProfile = hermes.ProfileHardened
		}
		cfg.HermesBearerPath = s.adoption.CredentialFile
		cfg.HermesCAPath = s.adoption.CAFile
		cfg.HermesClientCertPath = s.adoption.ClientCertFile
		cfg.HermesClientKeyPath = s.adoption.ClientKeyFile
		cfg.HermesServerPin = s.adoption.ServerCertPin
	}
	return cfg, nil
}

func (s *setupState) runStage(ctx context.Context, stage setup.Stage) error {
	switch stage {
	case setup.Detected:
		return nil
	case setup.ArtifactsReady:
		return s.checkArtifacts()
	case setup.DirectoriesReady:
		return s.makeDirectories()
	case setup.CredentialsReady:
		return s.ensureCredentials()
	case setup.ConfigurationReady:
		return s.writeConfig()
	case setup.ServicesInstalled:
		return s.installServices(ctx)
	case setup.ServicesStarted:
		manager, available := (cliProbe{dataDir: s.dataDir}).Supervisor()
		if !available {
			return errors.New("supervisor unavailable")
		}
		s.manager = manager
		if _, err := (setup.CommandSupervisor{Manager: manager}).Control(ctx, setup.Action{Code: "start"}, ownedServices(s.topology)); err != nil {
			return err
		}
		s.started = true
		return nil
	case setup.Validated:
		return nil
	default:
		return errors.New("unsupported setup stage")
	}
}

func (s *setupState) verifyStage(_ context.Context, stage setup.Stage) error {
	switch stage {
	case setup.Detected:
		if s.plan.Platform == "" {
			return errors.New("platform unavailable")
		}
	case setup.ArtifactsReady:
		return s.checkArtifacts()
	case setup.DirectoriesReady:
		if !fileExists(s.dataDir) || (s.topology == setup.TopologyCombined && !fileExists(filepath.Join(s.dataDir, "hermes"))) {
			return errors.New("directories unavailable")
		}
	case setup.CredentialsReady:
		if s.topology == setup.TopologyCombined && !privateFile(filepath.Join(s.dataDir, "hermes", "hermes.token")) {
			return errors.New("credentials unavailable")
		}
		if s.topology == setup.TopologyExternal && (s.adoption == nil || !privateFile(s.adoption.CredentialFile)) {
			return errors.New("credentials unavailable")
		}
	case setup.ConfigurationReady:
		if _, err := host.ConfigFromFile(filepath.Join(s.dataDir, "lumen.json")); err != nil {
			return err
		}
		if s.topology == setup.TopologyCombined && !fileExists(filepath.Join(s.dataDir, "hermes", "hermes.json")) {
			return errors.New("Hermes configuration unavailable")
		}
		if s.topology == setup.TopologyExternal && fileExists(filepath.Join(s.dataDir, "hermes")) {
			return errors.New("external topology has local Hermes configuration")
		}
	case setup.HostInitialized:
		cfg, err := s.hostConfig()
		if err != nil {
			return err
		}
		_, err = host.VerifyInitialized(cfg)
		return err
	case setup.ServicesInstalled:
		if !s.installed {
			return errors.New("services not installed")
		}
	case setup.ServicesStarted:
		if !s.started {
			return errors.New("services not started")
		}
	case setup.Validated:
		cfg, err := s.hostConfig()
		if err != nil {
			return err
		}
		_, err = host.VerifyInitialized(cfg)
		return err
	}
	return nil
}

func (s *setupState) writeConfig() error {
	if fileExists(filepath.Join(s.dataDir, "lumen.json")) {
		return nil
	}
	r := setup.ConfigRequest{DataDir: s.dataDir, HermesDir: filepath.Join(s.dataDir, "hermes"), Profile: s.profile, Topology: s.topology, HermesBaseURL: hermesEndpoint(), HermesServerPin: os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN")}
	if s.profile == setup.Hardened && s.topology == setup.TopologyCombined {
		var err error
		if r.HermesCA, err = readSetupTLSFile("LUMEN_HERMES_CA_FILE"); err != nil {
			return err
		}
		if r.HermesClientCert, err = readSetupTLSFile("LUMEN_HERMES_CLIENT_CERT_FILE"); err != nil {
			return err
		}
		if r.HermesClientKey, err = readSetupTLSFile("LUMEN_HERMES_CLIENT_KEY_FILE"); err != nil {
			return err
		}
	}
	if s.adoption != nil {
		r.HermesBaseURL = s.adoption.Endpoint
		r.HermesServerPin = s.adoption.ServerCertPin
		r.HermesCredentialFile = s.adoption.CredentialFile
		r.HermesCAFile = s.adoption.CAFile
		r.HermesClientCertFile = s.adoption.ClientCertFile
		r.HermesClientKeyFile = s.adoption.ClientKeyFile
	}
	_, err := setup.WriteConfig(r)
	return err
}

func readSetupTLSFile(env string) (string, error) {
	path := os.Getenv(env)
	if path == "" {
		return "", errors.New(env + " is required")
	}
	b, err := readPrivateSecret(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *setupState) makeDirectories() error {
	if err := ensurePrivateDir(s.dataDir); err != nil {
		return err
	}
	if s.topology == setup.TopologyCombined {
		return ensurePrivateDir(filepath.Join(s.dataDir, "hermes"))
	}
	return nil
}

func (s *setupState) ensureCredentials() error {
	if s.topology == setup.TopologyExternal {
		if s.adoption == nil || !privateFile(s.adoption.CredentialFile) {
			return errors.New("external credential reference unavailable")
		}
		return nil
	}
	_, err := setupSecret(filepath.Join(s.dataDir, "hermes", "hermes.token"))
	return err
}

func (s *setupState) artifactDigests() map[string]string {
	digests := map[string]string{}
	path := os.Getenv("LUMEN_MANIFEST")
	if path == "" {
		path = filepath.Join("deploy", "manifest-v1.json")
	}
	f, err := os.Open(path)
	if err != nil {
		return digests
	}
	defer f.Close()
	manifest, err := setup.LoadManifest(f)
	if err != nil || manifest.Topology != s.topology {
		return digests
	}
	osName := "linux"
	if s.plan.Platform == setup.PlatformMacOS {
		osName = "darwin"
	} else if s.plan.Platform == setup.PlatformTermux {
		osName = "android"
	}
	for _, name := range []string{"lumen", "hermes"} {
		if s.topology == setup.TopologyExternal && name == "hermes" {
			continue
		}
		if artifact, err := manifest.Select(name, osName, s.plan.Architecture, s.profile); err == nil {
			digests[name] = artifact.SHA256
		}
	}
	return digests
}

func adoptExternalHermes(ctx context.Context, plan setup.PlanResult, profile setup.Profile) (setup.ExternalAdoption, error) {
	credential := os.Getenv("LUMEN_HERMES_CREDENTIAL_FILE")
	endpoint := hermesEndpoint()
	token, err := readPrivateSecret(credential)
	if err != nil {
		return setup.ExternalAdoption{}, err
	}
	cfg := hermes.Config{BaseURL: endpoint, ProfileMode: externalHermesProfile(profile), BearerToken: strings.TrimSpace(string(token)), MaxResponseBytes: 8 << 20, MaxEventBytes: 8 << 20, MaxEventStreamBytes: 8 << 20, RequestTimeout: time.Second, EventTimeout: time.Second}
	if profile != setup.Development {
		u, parseErr := url.Parse(endpoint)
		if parseErr != nil {
			return setup.ExternalAdoption{}, parseErr
		}
		caPEM, readErr := readPrivateSecret(os.Getenv("LUMEN_HERMES_CA_FILE"))
		if readErr != nil {
			return setup.ExternalAdoption{}, readErr
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return setup.ExternalAdoption{}, errors.New("invalid Hermes CA file")
		}
		certPEM, readErr := readPrivateSecret(os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"))
		if readErr != nil {
			return setup.ExternalAdoption{}, readErr
		}
		keyPEM, readErr := readPrivateSecret(os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"))
		if readErr != nil {
			return setup.ExternalAdoption{}, readErr
		}
		clientCert, pairErr := tls.X509KeyPair(certPEM, keyPEM)
		if pairErr != nil {
			return setup.ExternalAdoption{}, pairErr
		}
		cfg.TLS = hermes.TLSConfig{RootCAs: roots, ClientCertificate: clientCert, ServerCertPin: os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN"), ServerName: u.Hostname()}
	}
	client, err := hermes.New(cfg)
	if err != nil {
		return setup.ExternalAdoption{}, err
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return setup.AdoptExternalHermes(bounded, setup.HermesCandidate{
		Endpoint: endpoint, CredentialFile: credential,
		CAFile: os.Getenv("LUMEN_HERMES_CA_FILE"), ClientCertFile: os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"), ClientKeyFile: os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"),
		ServerCertPin: cfg.TLS.ServerCertPin,
		Profile:       profile, Platform: plan.Platform, OwnerKnown: true, OwnerUID: uint32(os.Getuid()),
		Adapter: client,
	})
}

func (s *setupState) checkArtifacts() error {
	path := os.Getenv("LUMEN_MANIFEST")
	if path == "" {
		path = filepath.Join("deploy", "manifest-v1.json")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	manifest, err := setup.LoadManifest(f)
	if err != nil {
		return err
	}
	if manifest.Topology != s.plan.Topology {
		return errors.New("manifest topology mismatch")
	}
	osName := "linux"
	switch s.plan.Platform {
	case setup.PlatformMacOS:
		osName = "darwin"
	case setup.PlatformTermux:
		osName = "android"
	}
	names := []string{"lumen"}
	if s.plan.Topology == setup.TopologyCombined {
		names = append(names, "hermes")
	}
	for _, name := range names {
		a, err := manifest.Select(name, osName, s.plan.Architecture, s.profile)
		if err != nil {
			return err
		}
		if a.SHA256 == "sha256:"+strings.Repeat("0", 64) {
			return errors.New("artifact digest unavailable")
		}
		if s.plan.Platform == setup.PlatformTermux {
			bound := os.Getenv("LUMEN_" + strings.ToUpper(name) + "_SHA256")
			if bound == "" || bound != a.SHA256 {
				return errors.New("artifact digest unavailable")
			}
		}
		p := os.Getenv("LUMEN_" + strings.ToUpper(name) + "_ARTIFACT")
		if p == "" || !fileExists(p) {
			return errors.New("artifact unavailable")
		}
		if err := setup.VerifyArtifact(p, a); err != nil {
			return errors.New("artifact verification failed")
		}
	}
	return nil
}

func (s *setupState) installServices(ctx context.Context) error {
	manager, available := (cliProbe{dataDir: s.dataDir}).Supervisor()
	if !available {
		return errors.New("supervisor unavailable")
	}
	s.manager = manager
	root := filepath.Join(s.dataDir, "setup", "services")
	plan := setup.ServicePlan{
		Services: serviceDefinitions(manager, s.topology, root),
		Initializer: setup.HostInitializerFuncs{InitializeFunc: func(context.Context) error {
			cfg, err := s.hostConfig()
			if err != nil {
				return err
			}
			initErr := host.Initialize(cfg)
			if initErr != nil && !errors.Is(initErr, host.ErrAlreadyInitialized) {
				return initErr
			}
			return nil
		}, VerifyFunc: func(context.Context) error {
			cfg, err := s.hostConfig()
			if err != nil {
				return err
			}
			_, err = host.VerifyInitialized(cfg)
			return err
		}},
	}
	if err := setup.InstallServices(ctx, setup.CommandSupervisor{Manager: manager}, plan); err != nil {
		return err
	}
	s.installed = true
	return nil
}

func serviceDefinitions(manager setup.Supervisor, topology setup.Topology, root string) []setup.ServiceDefinition {
	names := ownedServices(topology)
	defs := make([]setup.ServiceDefinition, 0, len(names))
	for _, name := range names {
		d := setup.ServiceDefinition{Name: name}
		switch manager {
		case setup.SupervisorSystemd:
			d.Source = filepath.Join("deploy", "systemd", "lumen-"+string(name)+".service")
			d.Destination = filepath.Join(root, string(name)+".service")
		case setup.SupervisorLaunchd:
			d.Source = filepath.Join("deploy", "launchd", "dev.lumen."+string(name)+".plist")
			d.Destination = filepath.Join(root, "dev.lumen."+string(name)+".plist")
		case setup.SupervisorRunit:
			source := "run"
			if name == setup.ServiceHermes {
				source = "hermes-run"
			}
			d.Source = filepath.Join("deploy", "termux", source)
			d.Destination = filepath.Join(root, string(name), "run")
		case setup.SupervisorDocker:
			// Compose owns the service definitions; there is no unit to copy.
		default:
			return nil
		}
		defs = append(defs, d)
	}
	return defs
}

func (s *setupState) observe(ctx context.Context) setup.DoctorEvidence {
	e := setup.DoctorEvidence{Profile: s.profile, Platform: s.plan.Platform, LumenVersion: lumenVersion, HermesVersion: s.plan.HermesVersion, HostState: setup.StateUnknown, HermesState: "unavailable", ArtifactState: setup.StateUnknown, SupervisorState: setup.StateUnknown, BootState: setup.StateUnknown}
	if j, err := setup.NewJournal(filepath.Join(s.dataDir, "setup")); err == nil {
		e.Stage = j.Next()
	}
	if err := s.checkArtifacts(); err == nil {
		e.ArtifactsReady, e.ArtifactState = true, "installed"
	} else {
		e.ArtifactState = "missing"
	}
	cfg, err := s.hostConfig()
	if err == nil {
		if _, err = host.VerifyInitialized(cfg); err == nil {
			e.HostReady = true
			if fileExists(cfg.SocketPath) {
				e.HostState = setup.StateRunning
			} else {
				e.HostState = setup.StateStopped
			}
		} else {
			e.HostError = err
		}
		e.CredentialsReady = privateFile(cfg.CredentialPath) && privateFile(cfg.HermesBearerPath)
	}
	if manager, available := (cliProbe{dataDir: s.dataDir}).Supervisor(); available {
		states, statusErr := (setup.CommandSupervisor{Manager: manager}).Control(ctx, setup.Action{Code: "status"}, []setup.ServiceName{setup.ServiceHermes, setup.ServiceHost})
		e.SupervisorReady, e.SupervisorState = aggregateSupervisorStates(states, statusErr)
	}
	e.BootReady = e.SupervisorReady && (os.Getenv("LUMEN_BOOT_ENABLED") == "1" || stageIndex(e.Stage) >= stageIndex(setup.ServicesInstalled))
	if e.BootReady {
		e.BootState = setup.StateRunning
	}
	e.IsolationReady = s.profile != setup.Hardened || os.Getenv("LUMEN_HERMES_ISOLATED") == "1"
	if cfg != (host.Config{}) {
		e.HermesReady, e.HermesVersion, e.HermesError = hermesReady(ctx, cfg)
		if e.HermesReady {
			e.HermesState = setup.StateRunning
		}
	}
	if e.HermesVersion == "" {
		e.HermesVersion = s.plan.HermesVersion
	}
	return e
}

func aggregateSupervisorStates(states []setup.ServiceState, statusErr error) (bool, string) {
	if statusErr != nil {
		return false, setup.StateFailed
	}
	if len(states) != 2 {
		return false, setup.StateUnknown
	}
	seenHermes, seenHost := false, false
	state := setup.StateRunning
	for _, service := range states {
		switch service.Name {
		case setup.ServiceHermes:
			if seenHermes {
				return false, setup.StateUnknown
			}
			seenHermes = true
		case setup.ServiceHost:
			if seenHost {
				return false, setup.StateUnknown
			}
			seenHost = true
		default:
			return false, setup.StateUnknown
		}
		if err := service.Validate(); err != nil {
			return false, setup.StateUnknown
		}
		switch service.State {
		case setup.StateFailed:
			return false, setup.StateFailed
		case setup.StateStopped:
			state = setup.StateStopped
		case setup.StateUnknown:
			if state == setup.StateRunning {
				state = setup.StateUnknown
			}
		}
	}
	if !seenHermes || !seenHost {
		return false, setup.StateUnknown
	}
	return state == setup.StateRunning, state
}

func hermesReady(ctx context.Context, cfg host.Config) (bool, string, error) {
	token, err := os.ReadFile(cfg.HermesBearerPath)
	if err != nil {
		return false, "", err
	}
	client, err := hermes.New(hermes.Config{BaseURL: cfg.HermesBaseURL, ProfileMode: cfg.HermesProfile, BearerToken: strings.TrimSpace(string(token)), MaxResponseBytes: 8 << 20, MaxEventBytes: 1 << 20, MaxEventStreamBytes: 8 << 20, RequestTimeout: time.Second, EventTimeout: time.Second})
	if err != nil {
		return false, "", err
	}
	health, err := client.Health(ctx)
	if err != nil {
		return false, "", err
	}
	if _, err := client.Capabilities(ctx); err != nil {
		return false, health.Version, err
	}
	return true, health.Version, nil
}

func stageIndex(stage setup.Stage) int {
	for i, s := range []setup.Stage{setup.Detected, setup.ArtifactsReady, setup.DirectoriesReady, setup.CredentialsReady, setup.ConfigurationReady, setup.HostInitialized, setup.ServicesInstalled, setup.ServicesStarted, setup.Validated} {
		if stage == s {
			return i
		}
	}
	return -1
}
func hermesEndpoint() string {
	if v := os.Getenv("LUMEN_HERMES_BASE_URL"); v != "" {
		return v
	}
	return "http://127.0.0.1:8642"
}
func hermesProfile(p setup.Profile) string {
	if p == setup.Hardened {
		return hermes.ProfileHardened
	}
	return hermes.ProfileDevelopment
}

func externalHermesProfile(p setup.Profile) string {
	if p == setup.PersonalAlpha || p == setup.Hardened {
		return hermes.ProfileHardened
	}
	return hermes.ProfileDevelopment
}
func setupSecret(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		if !privateFile(path) {
			return "", errors.New("unsafe credential file")
		}
		return string(b), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(token); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

func readPrivateSecret(path string) ([]byte, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("private Hermes file is required")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("invalid private Hermes file")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return nil, errors.New("private Hermes file must be owner-only")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint32(stat.Uid) != uint32(os.Getuid()) {
		return nil, errors.New("private Hermes file has wrong owner")
	}
	return io.ReadAll(f)
}

func ensurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	st, err := os.Lstat(path)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe setup directory")
	}
	return os.Chmod(path, 0700)
}
func privateFile(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode().IsRegular() && st.Mode().Perm() == 0600
}
func fileExists(path string) bool { _, err := os.Stat(path); return err == nil }
func setupErrorCode(error) string { return "setup_unavailable" }
func placeholder(code string) setup.Report {
	return setup.Report{Outcome: setup.ActionRequired, Actions: []setup.Action{{Code: code}}}
}

func main() {
	report := run(os.Args[1:])
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
