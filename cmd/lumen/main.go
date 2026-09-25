package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

const actionRequired = setup.ActionRequired
const lumenVersion = "0.0.0"

type deployment struct {
	binding        setup.JournalBinding
	config         host.Config
	adoption       *setup.ExternalAdoption
	supervisor     setup.Supervisor
	composeProject string
	services       []setup.ServiceName
	journal        *setup.Journal
}

type hermesObservation struct {
	Ready               bool
	Version             string
	IdentityMatch       bool
	AuthenticationValid bool
	CompatibilityValid  bool
	Unavailable         bool
	Err                 error
	Error               error
}

type doctorDeps struct {
	hostStatus                  func(context.Context, host.Config) (string, error)
	supervisorStatus            func(context.Context, setup.Supervisor, []setup.ServiceName) ([]setup.ServiceState, error)
	supervisorStatusWithProject func(context.Context, setup.Supervisor, string, []setup.ServiceName) ([]setup.ServiceState, error)
	hermesProbe                 func(context.Context, host.Config, *setup.ExternalAdoption) hermesObservation
	bootStatus                  func(context.Context, setup.Supervisor, []setup.ServiceName) (bool, error)
	bootStatusWithProject       func(context.Context, setup.Supervisor, string, []setup.ServiceName) (bool, error)
}

var doctorDepsOverride *doctorDeps

var dockerImageInspect = inspectDockerImage

var (
	errDeploymentUnavailable = errors.New("deployment unavailable")
	errDeploymentBinding     = errors.New("deployment binding mismatch")
)

type deploymentError struct {
	cause error
}

func (e deploymentError) Error() string { return "deployment unavailable" }
func (e deploymentError) Unwrap() error { return e.cause }

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
		case "update":
			return updateCommand(ctx)
		case "rollback":
			return rollbackCommand(ctx)
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

func inspectDockerImage(imageRef string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{json .RepoDigests}}", imageRef).Output()
	if err == nil {
		var repoDigests []string
		if json.Unmarshal([]byte(strings.TrimSpace(string(output))), &repoDigests) == nil {
			for _, repoDigest := range repoDigests {
				if repoDigest == imageRef {
					return nil
				}
			}
		}
	}
	if err != nil {
		return errors.New("docker image unavailable")
	}
	return errors.New("docker image identity mismatch")
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
	preferredSupervisor := setup.Supervisor("")
	var existingBinding setup.JournalBinding
	existingBindingOK := false
	if existingJournal, journalErr := setup.NewJournal(filepath.Join(dataDir, "setup")); journalErr == nil {
		if binding, ok := existingJournal.Binding(); ok {
			existingBinding, existingBindingOK = binding, true
			if binding.Supervisor != "" {
				preferredSupervisor = binding.Supervisor
			}
		}
	}
	probe := cliProbe{dataDir: dataDir, preferredSupervisor: preferredSupervisor}
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
	if plan.Supervisor == setup.SupervisorDocker {
		state.composeProject = composeProjectForSetup(dataDir, os.Getenv("COMPOSE_PROJECT_NAME"))
	}
	journal, err := setup.NewJournal(plan.SetupDir)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "journal_unavailable"}}}
	}
	composeProject := state.composeProject
	if existingBindingOK && existingBinding.Supervisor == setup.SupervisorDocker {
		if existingBinding.ComposeProject != "" {
			composeProject = existingBinding.ComposeProject
		}
	}
	if plan.Supervisor == setup.SupervisorDocker && composeProject == "" {
		return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "setup_identity_mismatch"}}}
	}
	state.composeProject = composeProject
	if existingBinding, ok := journal.Binding(); ok {
		if existingBinding.Profile != profile || existingBinding.Topology != topology || (existingBinding.Supervisor != "" && existingBinding.Supervisor != plan.Supervisor) {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "setup_identity_mismatch"}}}
		}
		if topology == setup.TopologyExternal {
			endpointDigest, digestErr := setup.EndpointOriginDigest(hermesEndpoint())
			candidateReferences := setup.DigestReferences(map[string]string{
				"credential":  os.Getenv("LUMEN_HERMES_CREDENTIAL_FILE"),
				"ca":          os.Getenv("LUMEN_HERMES_CA_FILE"),
				"client_cert": os.Getenv("LUMEN_HERMES_CLIENT_CERT_FILE"),
				"client_key":  os.Getenv("LUMEN_HERMES_CLIENT_KEY_FILE"),
				"server_pin":  os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN"),
			})
			if digestErr != nil || endpointDigest != existingBinding.EndpointOriginDigest || !setup.ReferenceDigestsMatch(existingBinding.ReferenceDigests, candidateReferences) {
				return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "setup_identity_mismatch"}}}
			}
		}
	}
	if topology == setup.TopologyCombined {
		if _, selectErr := state.selectedArtifacts(); selectErr == nil {
			if err := state.checkArtifacts(ctx); err != nil {
				return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "artifacts_unavailable"}}}
			}
		}
	}
	if topology == setup.TopologyExternal {
		adoption, err := adoptExternalHermes(ctx, plan, profile)
		if err != nil {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "external_adoption_failed"}}}
		}
		if existingBindingOK && (adoption.EndpointOriginDigest != existingBinding.EndpointOriginDigest || adoption.EndpointIdentityDigest != existingBinding.EndpointIdentityDigest || !setup.ReferenceDigestsMatch(existingBinding.ReferenceDigests, adoptionReferenceDigests(adoption))) {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "setup_identity_mismatch"}}}
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
	request := setup.Request{Topology: topology, Profile: profile, Supervisor: plan.Supervisor, ComposeProject: composeProject, ArtifactPaths: state.artifactPaths(), ArtifactDigests: state.artifactDigests(), ArtifactRefs: state.artifactRefs()}
	if topology == setup.TopologyCombined {
		endpoint := hermesEndpoint()
		if os.Getenv("LUMEN_HERMES_BASE_URL") == "" {
			if cfg, cfgErr := state.hostConfig(); cfgErr == nil && cfg.HermesBaseURL != "" {
				endpoint = cfg.HermesBaseURL
			}
		}
		request.EndpointOriginDigest, err = setup.EndpointOriginDigest(endpoint)
		if err != nil {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "configuration_required"}}}
		}
		request.ReferenceDigests = setup.DigestReferences(map[string]string{
			"credential": filepath.Join(dataDir, "hermes", "hermes.token"),
		})
		if profile == setup.Hardened {
			request.ReferenceDigests = setup.DigestReferences(map[string]string{
				"credential":  filepath.Join(dataDir, "hermes", "hermes.token"),
				"ca":          filepath.Join(dataDir, "hermes", "ca.pem"),
				"client_cert": filepath.Join(dataDir, "hermes", "client.crt"),
				"client_key":  filepath.Join(dataDir, "hermes", "client.key"),
				"server_pin":  os.Getenv("LUMEN_HERMES_SERVER_CERT_PIN"),
			})
		}
	}
	if state.adoption != nil {
		request.EndpointOriginDigest = state.adoption.EndpointOriginDigest
		request.EndpointIdentityDigest = state.adoption.EndpointIdentityDigest
		request.ReferenceDigests = setup.DigestReferences(map[string]string{
			"credential":  state.adoption.CredentialFile,
			"ca":          state.adoption.CAFile,
			"client_cert": state.adoption.ClientCertFile,
			"client_key":  state.adoption.ClientKeyFile,
			"server_pin":  state.adoption.ServerCertPin,
		})
	}
	if existingBinding, ok := journal.Binding(); ok && existingBinding.Supervisor == "" && existingBinding.EndpointOriginDigest == "" && topology == setup.TopologyCombined {
		// A legacy combined binding had no endpoint digest. Upgrade it only
		// when the already-written config proves the same endpoint; an absent
		// config is ambiguous and must not be silently rebound.
		existingConfig, configErr := host.ConfigFromFile(filepath.Join(dataDir, "lumen.json"))
		existingDigest, digestErr := setup.EndpointOriginDigest(existingConfig.HermesBaseURL)
		if configErr != nil || digestErr != nil || existingDigest != request.EndpointOriginDigest {
			return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: "setup_identity_mismatch"}}}
		}
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

func adoptionReferenceDigests(adoption setup.ExternalAdoption) map[string]string {
	return setup.DigestReferences(map[string]string{
		"credential":  adoption.CredentialFile,
		"ca":          adoption.CAFile,
		"client_cert": adoption.ClientCertFile,
		"client_key":  adoption.ClientKeyFile,
		"server_pin":  adoption.ServerCertPin,
	})
}

func composeProjectForSetup(dataDir, configured string) string {
	if configured != "" {
		return configured
	}
	sum := sha256.Sum256([]byte(filepath.Clean(dataDir)))
	return "lumen-" + hex.EncodeToString(sum[:6])
}

func loadDeployment(dataDir string) (deployment, error) {
	if dataDir == "" || !filepath.IsAbs(dataDir) || filepath.Clean(dataDir) != dataDir {
		return deployment{}, deploymentError{cause: errDeploymentUnavailable}
	}
	j, err := setup.NewJournal(filepath.Join(dataDir, "setup"))
	if err != nil {
		return deployment{}, deploymentError{cause: errDeploymentUnavailable}
	}
	binding, ok := j.Binding()
	if !ok || binding.Supervisor == "" || binding.Supervisor.Validate() != nil {
		return deployment{}, deploymentError{cause: errDeploymentBinding}
	}
	if binding.Supervisor == setup.SupervisorDocker && binding.ComposeProject == "" {
		return deployment{}, deploymentError{cause: errDeploymentBinding}
	}
	digests, digestErr := effectiveArtifactDigests(binding, dataDir)
	if digestErr != nil || !durableEndpointEvidence(binding) || !durableReferenceEvidence(binding) || !verifyDurableArtifacts(binding, digests) {
		return deployment{}, deploymentError{cause: errDeploymentBinding}
	}
	cfg, err := host.ConfigFromFile(filepath.Join(dataDir, "lumen.json"))
	if err != nil || cfg.DataDir != dataDir || cfg.SocketPath != filepath.Join(dataDir, "host.sock") || cfg.CredentialPath != filepath.Join(dataDir, "operator.credential") || !durableConfigProfileMatches(binding.Profile, binding.Topology, cfg.HermesProfile) {
		return deployment{}, deploymentError{cause: errDeploymentBinding}
	}
	d := deployment{binding: binding, config: cfg, supervisor: binding.Supervisor, composeProject: binding.ComposeProject, services: ownedServices(binding.Topology), journal: j}
	if len(d.services) == 0 {
		return deployment{}, deploymentError{cause: errDeploymentBinding}
	}
	adoptionPath := filepath.Join(dataDir, "setup", "external-adoption.json")
	if binding.Topology == setup.TopologyExternal {
		adoption, loadErr := setup.LoadExternalAdoption(adoptionPath)
		refs := setup.DigestReferences(map[string]string{"credential": adoption.CredentialFile, "ca": adoption.CAFile, "client_cert": adoption.ClientCertFile, "client_key": adoption.ClientKeyFile, "server_pin": adoption.ServerCertPin})
		if loadErr != nil || cfg.HermesBaseURL != adoption.Endpoint || !endpointDigestMatches(binding.EndpointOriginDigest, adoption.Endpoint) || cfg.HermesBearerPath != adoption.CredentialFile || cfg.HermesCAPath != adoption.CAFile || cfg.HermesClientCertPath != adoption.ClientCertFile || cfg.HermesClientKeyPath != adoption.ClientKeyFile || cfg.HermesServerPin != adoption.ServerCertPin || adoption.EndpointOriginDigest != binding.EndpointOriginDigest || adoption.EndpointIdentityDigest != binding.EndpointIdentityDigest || !setup.ReferenceDigestsMatch(binding.ReferenceDigests, refs) {
			return deployment{}, deploymentError{cause: errDeploymentBinding}
		}
		if _, statErr := os.Lstat(filepath.Join(dataDir, "hermes")); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
			return deployment{}, deploymentError{cause: errDeploymentBinding}
		}
		d.adoption = &adoption
	} else {
		if _, statErr := os.Lstat(adoptionPath); statErr == nil || !errors.Is(statErr, os.ErrNotExist) || binding.EndpointIdentityDigest != "" {
			return deployment{}, deploymentError{cause: errDeploymentBinding}
		}
		localHermesDir := filepath.Join(dataDir, "hermes")
		refs := setup.DigestReferences(map[string]string{"credential": cfg.HermesBearerPath})
		if binding.Profile == setup.Hardened {
			refs = setup.DigestReferences(map[string]string{"credential": cfg.HermesBearerPath, "ca": cfg.HermesCAPath, "client_cert": cfg.HermesClientCertPath, "client_key": cfg.HermesClientKeyPath, "server_pin": cfg.HermesServerPin})
		}
		if cfg.HermesBearerPath != filepath.Join(localHermesDir, "hermes.token") || !endpointDigestMatches(binding.EndpointOriginDigest, cfg.HermesBaseURL) || !setup.ReferenceDigestsMatch(binding.ReferenceDigests, refs) {
			return deployment{}, deploymentError{cause: errDeploymentBinding}
		}
	}
	return d, nil
}

func durableConfigProfileMatches(profile setup.Profile, topology setup.Topology, got string) bool {
	if got == string(profile) {
		return true
	}
	return topology == setup.TopologyExternal && profile == setup.PersonalAlpha && got == hermes.ProfileHardened
}

func endpointDigestMatches(want, endpoint string) bool {
	if want == "" {
		return false
	}
	got, err := setup.EndpointOriginDigest(endpoint)
	return err == nil && got == want
}

func durableEndpointEvidence(binding setup.JournalBinding) bool {
	if binding.EndpointOriginDigest == "" {
		return false
	}
	if binding.Topology == setup.TopologyExternal {
		return binding.EndpointIdentityDigest != ""
	}
	return binding.EndpointIdentityDigest == ""
}

func durableReferenceEvidence(binding setup.JournalBinding) bool {
	want := []string{"credential"}
	if binding.Topology == setup.TopologyExternal && (binding.Profile == setup.PersonalAlpha || binding.Profile == setup.Hardened) {
		want = append(want, "ca", "client_cert", "client_key", "server_pin")
	}
	if len(binding.ReferenceDigests) != len(want) {
		return false
	}
	for _, name := range want {
		if !validDurableDigest(binding.ReferenceDigests[name]) {
			return false
		}
	}
	return true
}

func validDurableDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[len("sha256:"):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return value != "sha256:"+strings.Repeat("0", sha256.Size*2)
}

// effectiveArtifactDigests resolves which artifact identities a bound
// deployment must currently satisfy. The immutable binding owns profile,
// topology, install paths, and image references; an explicit owner update may
// advance only the digest, recorded in the release record beside the binding.
// A record that does not describe the same deployment is refused rather than
// silently ignored, so a tampered record fails the deployment closed.
func effectiveArtifactDigests(binding setup.JournalBinding, dataDir string) (map[string]string, error) {
	path := releaseRecordPath(dataDir)
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return binding.ArtifactDigests, nil
		}
		return nil, err
	}
	record, err := setup.LoadReleaseRecord(path)
	if err != nil {
		return nil, err
	}
	if err := setup.ReleaseMatchesBinding(record, binding); err != nil {
		return nil, err
	}
	digests := make(map[string]string, len(record.Current))
	for name, artifact := range record.Current {
		digests[name] = artifact.Digest
	}
	return digests, nil
}

func verifyDurableArtifacts(binding setup.JournalBinding, digests map[string]string) bool {
	names := artifactNames(binding.Topology)
	if len(names) == 0 || len(binding.ArtifactPaths)+len(binding.ArtifactRefs) != len(names) || len(digests) != len(names) {
		return false
	}
	for _, name := range names {
		if ref, ok := binding.ArtifactRefs[name]; ok {
			if _, pathOK := binding.ArtifactPaths[name]; pathOK || !validDurableImageRef(ref) || digests[name] != durableImageDigest(ref) || dockerImageInspect(ref) != nil {
				return false
			}
			continue
		}
		path, ok := binding.ArtifactPaths[name]
		want, digestOK := digests[name]
		if !ok || !digestOK || path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || !validDurableDigest(want) {
			return false
		}
		st, err := os.Lstat(path)
		if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || st.Mode()&0111 == 0 || st.Mode()&0022 != 0 {
			return false
		}
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		h := sha256.New()
		n, copyErr := io.Copy(h, io.LimitReader(f, setup.MaxArtifactSize+1))
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil || n > setup.MaxArtifactSize || "sha256:"+hex.EncodeToString(h.Sum(nil)) != want {
			return false
		}
	}
	for name := range binding.ArtifactPaths {
		if !containsArtifactName(names, name) {
			return false
		}
	}
	for name := range binding.ArtifactRefs {
		if !containsArtifactName(names, name) {
			return false
		}
	}
	return true
}

func containsArtifactName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func durableImageDigest(imageRef string) string {
	if i := strings.LastIndex(imageRef, "@"); i >= 0 {
		return imageRef[i+1:]
	}
	return ""
}

func validDurableImageRef(imageRef string) bool {
	digest := durableImageDigest(imageRef)
	return digest != "" && validDurableDigest(digest) && !strings.ContainsAny(imageRef, "?# \t\r\n")
}

func deploymentReport(err error, profile setup.Profile, topology setup.Topology) setup.Report {
	code := "setup_identity_mismatch"
	if errors.Is(err, errDeploymentUnavailable) {
		code = "journal_unavailable"
	}
	return setup.Report{Outcome: setup.ActionRequired, Profile: profile, Topology: topology, Actions: []setup.Action{{Code: code}}}
}

func doctorCommand(ctx context.Context) setup.Report {
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	if dataDir == "" {
		return placeholder("configuration_required")
	}
	d, err := loadDeployment(dataDir)
	if err != nil {
		return deploymentReport(err, "", "")
	}
	deps := defaultDoctorDeps()
	return (setup.Doctor{Observe: func(ctx context.Context) setup.DoctorEvidence {
		return observeDeployment(ctx, d, deps)
	}}).Check(ctx)
}

func serviceCommand(ctx context.Context, action string) setup.Report {
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	if dataDir == "" {
		return placeholder("configuration_required")
	}
	d, err := loadDeployment(dataDir)
	if err != nil {
		return deploymentReport(err, "", "")
	}
	if d.journal.Next() != setup.Validated {
		return setup.Report{Outcome: setup.ActionRequired, Profile: d.binding.Profile, Topology: d.binding.Topology, Stage: d.journal.Next(), Actions: []setup.Action{{Code: "setup_incomplete"}}}
	}
	states, err := (setup.CommandSupervisor{Manager: d.supervisor, Topology: d.binding.Topology, Compose: setup.ComposeConfig{Project: d.composeProject}}).Control(ctx, setup.Action{Code: action}, d.services)
	if err != nil {
		return setup.Report{Outcome: setup.ActionRequired, Profile: d.binding.Profile, Topology: d.binding.Topology, Actions: []setup.Action{{Code: "service_unavailable"}}}
	}
	r := setup.Report{Outcome: setup.Ready, Profile: d.binding.Profile, Topology: d.binding.Topology, Stage: setup.Validated, States: map[string]string{}}
	for _, s := range states {
		r.States[string(s.Name)] = s.State
		if (action == "stop" && s.State != setup.StateStopped) || (action != "stop" && s.State != setup.StateRunning) {
			r.Outcome = setup.ActionRequired
			code := "service_unavailable"
			if action == "stop" {
				code = "service_not_stopped"
			}
			r.Actions = append(r.Actions, setup.Action{Code: code})
		}
	}
	return r
}

func defaultDoctorDeps() doctorDeps {
	d := doctorDeps{
		hostStatus:                  defaultHostStatus,
		supervisorStatus:            defaultSupervisorStatus,
		supervisorStatusWithProject: defaultSupervisorStatusWithProject,
		hermesProbe:                 defaultHermesProbe,
		bootStatus:                  defaultBootStatus,
		bootStatusWithProject:       defaultBootStatusWithProject,
	}
	if doctorDepsOverride != nil {
		if doctorDepsOverride.hostStatus != nil {
			d.hostStatus = doctorDepsOverride.hostStatus
		}
		if doctorDepsOverride.supervisorStatus != nil {
			d.supervisorStatus = doctorDepsOverride.supervisorStatus
			d.supervisorStatusWithProject = nil
		}
		if doctorDepsOverride.hermesProbe != nil {
			d.hermesProbe = doctorDepsOverride.hermesProbe
		}
		if doctorDepsOverride.bootStatus != nil {
			d.bootStatus = doctorDepsOverride.bootStatus
			d.bootStatusWithProject = nil
		}
	}
	return d
}

func observeDeployment(ctx context.Context, d deployment, deps doctorDeps) setup.DoctorEvidence {
	e := setup.DoctorEvidence{
		Stage:               d.journal.Next(),
		Profile:             d.binding.Profile,
		Topology:            d.binding.Topology,
		Platform:            "",
		LumenVersion:        lumenVersion,
		HostState:           setup.StateUnknown,
		HermesState:         "unavailable",
		ArtifactState:       setup.StateUnknown,
		SupervisorState:     setup.StateUnknown,
		BootState:           setup.StateUnknown,
		IsolationReady:      deploymentIsolationReady(d),
		BindingMatch:        true,
		PreviouslyValidated: d.journal.Next() == setup.Validated,
	}
	if d.adoption != nil {
		e.HermesVersion = d.adoption.Version
	}

	state, liveErr := deps.hostStatus(ctx, d.config)
	if liveErr == nil && (state == setup.StateRunning || state == "ready") {
		e.CredentialsReady = privateFile(d.config.CredentialPath) && privateFile(d.config.HermesBearerPath)
		e.HostReady, e.HostState = true, setup.StateRunning
	} else if _, err := host.VerifyInitialized(d.config); err != nil {
		e.HostError = err
	} else {
		e.CredentialsReady = privateFile(d.config.CredentialPath) && privateFile(d.config.HermesBearerPath)
		e.HostState = setup.StateStopped
		e.HostError = liveErr
	}

	var states []setup.ServiceState
	var statusErr error
	if deps.supervisorStatusWithProject != nil {
		states, statusErr = deps.supervisorStatusWithProject(ctx, d.supervisor, d.composeProject, d.services)
	} else {
		states, statusErr = deps.supervisorStatus(ctx, d.supervisor, d.services)
	}
	e.SupervisorReady, e.SupervisorState = aggregateOwnedSupervisorStates(states, d.services, statusErr)
	// A live, complete local service set is the only artifact evidence available
	// to doctor; setup history alone is not proof that artifacts remain usable.
	e.ArtifactsReady = e.SupervisorReady
	if e.ArtifactsReady {
		e.ArtifactState = "installed"
	} else {
		e.ArtifactState = setup.StateUnknown
	}
	var bootReady bool
	var bootErr error
	if deps.bootStatusWithProject != nil {
		bootReady, bootErr = deps.bootStatusWithProject(ctx, d.supervisor, d.composeProject, d.services)
	} else {
		bootReady, bootErr = deps.bootStatus(ctx, d.supervisor, d.services)
	}
	if bootErr != nil {
		e.BootState = setup.StateUnknown
	} else if bootReady {
		e.BootReady, e.BootState = true, setup.StateRunning
	} else {
		e.BootState = setup.StateStopped
	}

	observation := deps.hermesProbe(ctx, d.config, d.adoption)
	e.HermesReady = observation.Ready
	e.HermesVersion = observation.Version
	e.EndpointIdentityMatch = observation.IdentityMatch
	e.AuthenticationValid = observation.AuthenticationValid
	e.CompatibilityValid = observation.CompatibilityValid
	e.HermesError = observation.Err
	if e.HermesError == nil {
		e.HermesError = observation.Error
	}
	if observation.Unavailable && e.PreviouslyValidated {
		// A bounded transport outage does not invalidate the durable adoption.
		e.EndpointIdentityMatch = true
		e.AuthenticationValid = true
		e.CompatibilityValid = true
		e.BindingMatch = true
	}
	if e.HermesReady {
		e.HermesState = setup.StateRunning
	}
	return e
}

func deploymentIsolationReady(d deployment) bool {
	if d.binding.Profile != setup.Hardened {
		return true
	}
	if d.adoption != nil {
		return d.adoption.CAFile != "" && d.adoption.ClientCertFile != "" && d.adoption.ClientKeyFile != "" && d.adoption.ServerCertPin != ""
	}
	return d.supervisor == setup.SupervisorDocker
}

func aggregateOwnedSupervisorStates(states []setup.ServiceState, want []setup.ServiceName, statusErr error) (bool, string) {
	if statusErr != nil || len(states) != len(want) || len(want) == 0 {
		if statusErr != nil {
			return false, setup.StateFailed
		}
		return false, setup.StateUnknown
	}
	wanted := make(map[setup.ServiceName]bool, len(want))
	for _, name := range want {
		if wanted[name] {
			return false, setup.StateUnknown
		}
		wanted[name] = true
	}
	seen := make(map[setup.ServiceName]bool, len(states))
	state := setup.StateRunning
	for _, service := range states {
		if !wanted[service.Name] || seen[service.Name] || service.Validate() != nil {
			return false, setup.StateUnknown
		}
		seen[service.Name] = true
		switch service.State {
		case setup.StateFailed:
			state = setup.StateFailed
		case setup.StateStopped:
			if state == setup.StateRunning {
				state = setup.StateStopped
			}
		case setup.StateUnknown:
			if state == setup.StateRunning {
				state = setup.StateUnknown
			}
		}
	}
	if len(seen) != len(want) {
		return false, setup.StateUnknown
	}
	return state == setup.StateRunning, state
}

func defaultHostStatus(ctx context.Context, cfg host.Config) (string, error) {
	response, err := control.CallContext(ctx, cfg.SocketPath, cfg.CredentialPath, control.Request{Command: "status"})
	if err != nil {
		return setup.StateUnknown, err
	}
	if !response.OK {
		return setup.StateUnknown, errors.New("Host status rejected")
	}
	data, ok := response.Data.(map[string]any)
	if !ok || data["status"] != "ready" {
		return setup.StateUnknown, errors.New("Host status is not ready")
	}
	return setup.StateRunning, nil
}

func defaultSupervisorStatus(ctx context.Context, manager setup.Supervisor, services []setup.ServiceName) ([]setup.ServiceState, error) {
	return (setup.CommandSupervisor{Manager: manager, Topology: topologyForOwnedServices(services)}).Control(ctx, setup.Action{Code: "status"}, services)
}

func defaultSupervisorStatusWithProject(ctx context.Context, manager setup.Supervisor, project string, services []setup.ServiceName) ([]setup.ServiceState, error) {
	return (setup.CommandSupervisor{Manager: manager, Topology: topologyForOwnedServices(services), Compose: setup.ComposeConfig{Project: project}}).Control(ctx, setup.Action{Code: "status"}, services)
}

func defaultBootStatus(ctx context.Context, manager setup.Supervisor, services []setup.ServiceName) (bool, error) {
	return (setup.CommandSupervisor{Manager: manager, Topology: topologyForOwnedServices(services)}).BootStatus(ctx, services)
}

func defaultBootStatusWithProject(ctx context.Context, manager setup.Supervisor, project string, services []setup.ServiceName) (bool, error) {
	return (setup.CommandSupervisor{Manager: manager, Topology: topologyForOwnedServices(services), Compose: setup.ComposeConfig{Project: project}}).BootStatus(ctx, services)
}

func topologyForOwnedServices(services []setup.ServiceName) setup.Topology {
	if len(services) == 1 && services[0] == setup.ServiceHost {
		return setup.TopologyExternal
	}
	return setup.TopologyCombined
}

func defaultHermesProbe(ctx context.Context, cfg host.Config, adoption *setup.ExternalAdoption) hermesObservation {
	client, err := newHermesClient(cfg)
	if err != nil {
		return hermesObservation{Err: errors.New("Hermes configuration mismatch")}
	}
	observer, ok := any(client).(hermes.EndpointIdentityObserver)
	if !ok {
		return hermesObservation{Err: errors.New("Hermes identity unavailable")}
	}
	identity, err := observer.VerifiedEndpointIdentity(ctx)
	if err != nil {
		return hermesObservation{Unavailable: hermesTransportUnavailable(err), Err: hermesProbeError(err)}
	}
	if adoption != nil {
		if err := setup.CheckExternalBinding(*adoption, cfg.HermesBaseURL, identity); err != nil {
			return hermesObservation{IdentityMatch: false, Err: errors.New("Hermes identity mismatch")}
		}
	}
	observation := hermesObservation{IdentityMatch: true}
	health, err := client.Health(ctx)
	if err != nil {
		if hermesAuthenticationError(err) {
			observation.Err = errors.New("Hermes authentication rejected")
		} else if hermesTransportUnavailable(err) {
			observation.Unavailable = true
			observation.Err = errors.New("Hermes endpoint unavailable")
		} else {
			observation.Err = errors.New("Hermes compatibility mismatch")
		}
		return observation
	}
	observation.Version = health.Version
	caps, err := client.Capabilities(ctx)
	if err != nil {
		if hermesAuthenticationError(err) {
			observation.AuthenticationValid = false
			observation.Err = errors.New("Hermes authentication incompatible")
		} else if hermesTransportUnavailable(err) {
			observation.Unavailable = true
			observation.Err = errors.New("Hermes endpoint unavailable")
		} else if caps.Auth.Type != "bearer" || !caps.Auth.Required {
			observation.AuthenticationValid = false
			observation.Err = errors.New("Hermes authentication incompatible")
		} else {
			observation.AuthenticationValid = true
			observation.Err = errors.New("Hermes capabilities incompatible")
		}
		return observation
	}
	observation.AuthenticationValid = true
	observation.CompatibilityValid = true
	observation.Ready = true
	return observation
}

func hermesProbeError(err error) error {
	if hermesAuthenticationError(err) {
		return errors.New("Hermes authentication rejected")
	}
	if hermesTransportUnavailable(err) {
		return errors.New("Hermes endpoint unavailable")
	}
	return errors.New("Hermes identity mismatch")
}

func hermesTransportUnavailable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var response *hermes.HTTPError
	if errors.As(err, &response) {
		return response.Status >= 500
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		var tlsVerify *tls.CertificateVerificationError
		if errors.As(urlErr.Err, &tlsVerify) || strings.Contains(strings.ToLower(urlErr.Err.Error()), "tls") {
			return false
		}
		var opErr *net.OpError
		return errors.As(urlErr.Err, &opErr)
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func hermesAuthenticationError(err error) bool {
	var response *hermes.HTTPError
	return errors.As(err, &response) && (response.Status == 401 || response.Status == 403)
}

func newHermesClient(cfg host.Config) (*hermes.Client, error) {
	token, err := readPrivateSecret(cfg.HermesBearerPath)
	if err != nil {
		return nil, err
	}
	profile := cfg.HermesProfile
	if profile == "personal-alpha" {
		profile = hermes.ProfileHardened
	}
	hcfg := hermes.Config{BaseURL: cfg.HermesBaseURL, ProfileMode: profile, BearerToken: strings.TrimSpace(string(token)), MaxResponseBytes: 8 << 20, MaxEventBytes: 1 << 20, MaxEventStreamBytes: 8 << 20, RequestTimeout: 10 * time.Second, EventTimeout: 30 * time.Second}
	if profile == hermes.ProfileHardened {
		caPEM, err := readPrivateSecret(cfg.HermesCAPath)
		if err != nil {
			return nil, err
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("invalid Hermes CA")
		}
		certPEM, err := readPrivateSecret(cfg.HermesClientCertPath)
		if err != nil {
			return nil, err
		}
		keyPEM, err := readPrivateSecret(cfg.HermesClientKeyPath)
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(cfg.HermesBaseURL)
		if err != nil {
			return nil, err
		}
		hcfg.TLS = hermes.TLSConfig{RootCAs: roots, ClientCertificate: cert, ServerCertPin: cfg.HermesServerPin, ServerName: u.Hostname()}
	}
	return hermes.New(hcfg)
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

type cliProbe struct {
	dataDir             string
	preferredSupervisor setup.Supervisor
}

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
	if p.preferredSupervisor != "" {
		return p.preferredSupervisor, true
	}
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
	plan           setup.PlanResult
	dataDir        string
	profile        setup.Profile
	topology       setup.Topology
	composeProject string
	adoption       *setup.ExternalAdoption
	manager        setup.Supervisor
	installed      bool
	started        bool
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
		return s.checkArtifacts(ctx)
	case setup.DirectoriesReady:
		return s.makeDirectories()
	case setup.CredentialsReady:
		return s.ensureCredentials()
	case setup.ConfigurationReady:
		return s.writeConfig()
	case setup.ServicesInstalled:
		return s.installServices(ctx)
	case setup.ServicesStarted:
		manager := s.manager
		if manager == "" {
			manager = s.plan.Supervisor
		}
		if manager.Validate() != nil {
			return errors.New("supervisor unavailable")
		}
		s.manager = manager
		if _, err := (setup.CommandSupervisor{Manager: manager, Topology: s.topology, Compose: setup.ComposeConfig{Project: s.composeProject}}).Control(ctx, setup.Action{Code: "start"}, ownedServices(s.topology)); err != nil {
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

func (s *setupState) verifyStage(ctx context.Context, stage setup.Stage) error {
	switch stage {
	case setup.Detected:
		if s.plan.Platform == "" {
			return errors.New("platform unavailable")
		}
	case setup.ArtifactsReady:
		return s.checkArtifacts(ctx)
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
	artifacts, err := s.selectedArtifacts()
	if err != nil {
		return digests
	}
	for name, artifact := range artifacts {
		if artifact.Kind == setup.ArtifactDockerImage {
			if digest := durableImageDigest(artifact.ImageRef); digest != "" {
				digests[name] = digest
			}
			continue
		}
		digests[name] = artifact.SHA256
	}
	return digests
}

func (s *setupState) artifactPaths() map[string]string {
	paths := make(map[string]string)
	artifacts, err := s.selectedArtifacts()
	if err != nil {
		return paths
	}
	for _, name := range artifactNames(s.topology) {
		if artifacts[name].Kind == setup.ArtifactDockerImage {
			continue
		}
		if path := os.Getenv("LUMEN_" + strings.ToUpper(name) + "_ARTIFACT"); path != "" {
			paths[name] = path
		}
	}
	return paths
}

func (s *setupState) artifactRefs() map[string]string {
	refs := make(map[string]string)
	artifacts, err := s.selectedArtifacts()
	if err != nil {
		return refs
	}
	for name, artifact := range artifacts {
		if artifact.Kind == setup.ArtifactDockerImage && artifact.ImageRef != "" {
			refs[name] = artifact.ImageRef
		}
	}
	return refs
}

func (s *setupState) selectedArtifacts() (map[string]setup.Artifact, error) {
	path := os.Getenv("LUMEN_MANIFEST")
	if path == "" {
		path = filepath.Join("deploy", "manifest-v1.json")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	manifest, err := setup.LoadManifest(f)
	if err != nil || manifest.Topology != s.topology || manifest.Topology != s.plan.Topology {
		if err == nil {
			err = errors.New("manifest topology mismatch")
		}
		return nil, err
	}
	osName := "linux"
	if s.plan.Platform == setup.PlatformMacOS {
		osName = "darwin"
	} else if s.plan.Platform == setup.PlatformTermux {
		osName = "android"
	}
	selected := make(map[string]setup.Artifact, len(artifactNames(s.topology)))
	for _, name := range artifactNames(s.topology) {
		artifact, selectErr := manifest.Select(name, osName, s.plan.Architecture, s.profile)
		if selectErr != nil {
			return nil, selectErr
		}
		selected[name] = artifact
	}
	return selected, nil
}

func artifactNames(topology setup.Topology) []string {
	if topology == setup.TopologyCombined {
		return []string{"lumen", "hermes"}
	}
	if topology == setup.TopologyExternal {
		return []string{"lumen"}
	}
	return nil
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

func (s *setupState) checkArtifacts(_ context.Context) error {
	artifacts, err := s.selectedArtifacts()
	if err != nil {
		return err
	}
	for _, name := range artifactNames(s.topology) {
		a := artifacts[name]
		if a.Kind == setup.ArtifactDockerImage {
			if dockerImageInspect(a.ImageRef) != nil {
				return errors.New("docker image unavailable")
			}
			continue
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
	manager := s.manager
	if manager == "" {
		manager = s.plan.Supervisor
	}
	if manager.Validate() != nil {
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
	if err := setup.InstallServices(ctx, setup.CommandSupervisor{Manager: manager, Topology: s.topology, Compose: setup.ComposeConfig{Project: s.composeProject}}, plan); err != nil {
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
	d, err := loadDeployment(s.dataDir)
	if err != nil {
		return setup.DoctorEvidence{Profile: s.profile, Topology: s.topology, Platform: s.plan.Platform, HostError: err}
	}
	return observeDeployment(ctx, d, defaultDoctorDeps())
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
	observation := defaultHermesProbe(ctx, cfg, nil)
	return observation.Ready, observation.Version, observation.Err
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
