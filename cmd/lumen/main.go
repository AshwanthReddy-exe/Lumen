package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	probe := cliProbe{dataDir: dataDir}
	plan, err := setup.Plan(ctx, setup.Request{Profile: profile}, probe)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: setupErrorCode(err)}}}
	}
	if plan.Outcome != setup.Ready {
		r := setup.Report{Outcome: plan.Outcome, Profile: profile}
		for _, action := range plan.Actions {
			r.Actions = append(r.Actions, setup.Action{Code: action.Code})
		}
		if len(r.Actions) == 0 {
			r.Actions = []setup.Action{{Code: "setup_plan_requires_action"}}
		}
		return r
	}
	journal, err := setup.NewJournal(plan.SetupDir)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "journal_unavailable"}}}
	}
	request := setup.Request{Profile: profile, ArtifactDigests: map[string]string{"lumen": os.Getenv("LUMEN_LUMEN_SHA256"), "hermes": os.Getenv("LUMEN_HERMES_SHA256")}}
	state := &setupState{plan: plan, dataDir: dataDir, profile: profile}
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
	state := &setupState{dataDir: dataDir, profile: profile}
	if plan, e := setup.Plan(ctx, setup.Request{Profile: profile}, cliProbe{dataDir: dataDir}); e == nil {
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
	manager, available := (cliProbe{dataDir: dataDir}).Supervisor()
	if !available {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Actions: []setup.Action{{Code: "supervisor_unavailable"}}}
	}
	states, err := (setup.CommandSupervisor{Manager: manager}).Control(ctx, setup.Action{Code: action}, []setup.ServiceName{setup.ServiceHermes, setup.ServiceHost})
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
	manager   setup.Supervisor
	installed bool
	started   bool
}

func (s *setupState) hostConfig() (host.Config, error) {
	if p := filepath.Join(s.dataDir, "lumen.json"); fileExists(p) {
		return host.ConfigFromFile(p)
	}
	return host.Config{
		DataDir: s.dataDir, SocketPath: filepath.Join(s.dataDir, "host.sock"), CredentialPath: filepath.Join(s.dataDir, "operator.credential"),
		HermesBaseURL: hermesEndpoint(), HermesProfile: string(hermesProfile(s.profile)), HermesBearerPath: filepath.Join(s.dataDir, "hermes", "hermes.token"),
		HermesCAPath: os.Getenv("LUMEN_HERMES_CA_FILE"), HermesClientCertPath: os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"), HermesClientKeyPath: os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"), HermesServerPin: os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN"),
	}, nil
}

func (s *setupState) runStage(ctx context.Context, stage setup.Stage) error {
	switch stage {
	case setup.Detected:
		return nil
	case setup.ArtifactsReady:
		return s.checkArtifacts()
	case setup.DirectoriesReady:
		return s.writeConfig()
	case setup.CredentialsReady, setup.ConfigurationReady:
		return nil
	case setup.ServicesInstalled:
		return s.installServices(ctx)
	case setup.ServicesStarted:
		manager, available := (cliProbe{dataDir: s.dataDir}).Supervisor()
		if !available {
			return errors.New("supervisor unavailable")
		}
		s.manager = manager
		if _, err := (setup.CommandSupervisor{Manager: manager}).Control(ctx, setup.Action{Code: "start"}, []setup.ServiceName{setup.ServiceHermes, setup.ServiceHost}); err != nil {
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
		if !fileExists(filepath.Join(s.dataDir, "hermes")) {
			return errors.New("directories unavailable")
		}
	case setup.CredentialsReady:
		if !fileExists(filepath.Join(s.dataDir, "hermes", "hermes.token")) {
			return errors.New("credentials unavailable")
		}
	case setup.ConfigurationReady:
		if _, err := host.ConfigFromFile(filepath.Join(s.dataDir, "lumen.json")); err != nil {
			return err
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
	if err := os.MkdirAll(filepath.Join(s.dataDir, "hermes"), 0700); err != nil {
		return err
	}
	token, err := setupSecret(filepath.Join(s.dataDir, "hermes", "hermes.token"))
	if err != nil {
		return err
	}
	_, err = setup.WriteConfig(setup.ConfigRequest{DataDir: s.dataDir, HermesDir: filepath.Join(s.dataDir, "hermes"), Profile: s.profile, HermesBaseURL: hermesEndpoint(), HermesBearer: token, HermesCA: os.Getenv("LUMEN_HERMES_CA_FILE"), HermesClientCert: os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"), HermesClientKey: os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"), HermesServerPin: os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN")})
	return err
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
	osName := "linux"
	switch s.plan.Platform {
	case setup.PlatformMacOS:
		osName = "darwin"
	case setup.PlatformTermux:
		osName = "android"
	}
	for _, name := range []string{"lumen", "hermes"} {
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
	servicePath := func(name string) string { return filepath.Join(root, name+".service") }
	plan := setup.ServicePlan{
		Hermes: setup.ServiceDefinition{Name: setup.ServiceHermes, Source: filepath.Join("deploy", "systemd", "lumen-hermes.service"), Destination: servicePath("hermes")},
		Host:   setup.ServiceDefinition{Name: setup.ServiceHost, Source: filepath.Join("deploy", "systemd", "lumen-host.service"), Destination: servicePath("host")},
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
		if statusErr == nil {
			e.SupervisorReady, e.SupervisorState = true, setup.StateRunning
			for _, st := range states {
				if st.State == setup.StateFailed {
					e.SupervisorState = setup.StateFailed
				}
			}
		} else {
			e.SupervisorState = setup.StateFailed
		}
	}
	e.BootReady = os.Getenv("LUMEN_BOOT_ENABLED") == "1" || (e.SupervisorReady && stageIndex(e.Stage) >= stageIndex(setup.ServicesInstalled))
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
func setupSecret(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		return string(b), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
