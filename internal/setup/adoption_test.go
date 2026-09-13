package setup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
)

func TestExternalAdoptionSaveLoadIsPrivateAndRedacted(t *testing.T) {
	p := filepath.Join(realTempDir(t), "adoption.json")
	if err := os.Chmod(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	cred := filepath.Join(filepath.Dir(p), "credential")
	if err := os.WriteFile(cred, []byte("opaque"), 0600); err != nil {
		t.Fatal(err)
	}
	a := ExternalAdoption{Endpoint: "https://hermes.example", EndpointOriginDigest: "sha256:" + strings.Repeat("a", 64), EndpointIdentityDigest: "sha256:" + strings.Repeat("b", 64), Version: "1.0.0", CredentialFile: "/tmp/credential"}
	a.CredentialFile = cred
	if err := SaveExternalAdoption(p, a); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0600 {
		t.Fatalf("mode=%o", st.Mode().Perm())
	}
	got, err := LoadExternalAdoption(p)
	if err != nil || got != a {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "secret") {
		t.Fatal("secret persisted")
	}
}

func TestExternalAdoptionLoadRejectsUnknownAndInvalidJSON(t *testing.T) {
	p := filepath.Join(realTempDir(t), "adoption.json")
	os.WriteFile(p, []byte(`{"endpoint":"x","extra":true}`), 0600)
	if _, err := LoadExternalAdoption(p); err == nil {
		t.Fatal("accepted invalid record")
	}
}

func TestExternalAdoptionLoadRejectsTrailingJSON(t *testing.T) {
	d := realTempDir(t)
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "adoption.json")
	if err := os.WriteFile(p, []byte(`{} {}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadExternalAdoption(p); err == nil {
		t.Fatal("accepted trailing JSON")
	}
}

func TestExternalAdoptionRequiresCredentialReference(t *testing.T) {
	a := ExternalAdoption{Endpoint: "https://hermes.example", EndpointOriginDigest: "sha256:" + strings.Repeat("a", 64), EndpointIdentityDigest: "sha256:" + strings.Repeat("b", 64), Version: "1.0.0"}
	if validAdoption(a) {
		t.Fatal("accepted adoption without credential reference")
	}
}

func TestSaveExternalAdoptionRejectsSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	cred := filepath.Join(root, "credential")
	if err := os.WriteFile(cred, []byte("opaque"), 0600); err != nil {
		t.Fatal(err)
	}
	a := ExternalAdoption{Endpoint: "https://hermes.example", EndpointOriginDigest: "sha256:" + strings.Repeat("a", 64), EndpointIdentityDigest: "sha256:" + strings.Repeat("b", 64), Version: "1.0.0", CredentialFile: cred}
	if err := SaveExternalAdoption(filepath.Join(link, "adoption.json"), a); err == nil {
		t.Fatal("accepted adoption path through symlinked parent")
	}
}

func TestExternalAdoptionRejectsUnsafeCredentialParent(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	if err := os.Chmod(filepath.Dir(c.CredentialFile), 0770); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("accepted credential in unsafe parent: %v", err)
	}
}

func TestPersonalAlphaExternalAdoptionRequiresTLSMaterial(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	c.Profile = PersonalAlpha
	c.Endpoint = "https://hermes.example/runs"
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("accepted personal-alpha external transport without TLS material: %v", err)
	}
}

func TestCheckExternalBindingMatchesEndpointAndIdentity(t *testing.T) {
	endpoint := "https://hermes.example/api"
	identity := "tls-leaf-sha256:leaf"
	a := ExternalAdoption{
		Endpoint:               endpoint,
		EndpointOriginDigest:   digest(endpoint),
		EndpointIdentityDigest: digest(identity),
		Version:                "1.0.0",
		CredentialFile:         "/tmp/credential",
	}
	if err := CheckExternalBinding(a, endpoint, identity); err != nil {
		t.Fatal(err)
	}
	if err := CheckExternalBinding(a, "https://other.example/api", identity); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("endpoint mismatch = %v", err)
	}
	if err := CheckExternalBinding(a, endpoint, "tls-leaf-sha256:other"); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("identity mismatch = %v", err)
	}
}

func TestReferenceDigestsBindCredentialAndTLSReferences(t *testing.T) {
	a := ExternalAdoption{
		CredentialFile: "/state/hermes.token",
		CAFile:         "/state/ca.pem",
		ClientCertFile: "/state/client.crt",
		ClientKeyFile:  "/state/client.key",
		ServerCertPin:  "pin",
	}
	want := DigestReferences(map[string]string{
		"credential":  a.CredentialFile,
		"ca":          a.CAFile,
		"client_cert": a.ClientCertFile,
		"client_key":  a.ClientKeyFile,
		"server_pin":  a.ServerCertPin,
	})
	if !ReferenceDigestsMatch(want, DigestReferences(map[string]string{
		"credential":  a.CredentialFile,
		"ca":          a.CAFile,
		"client_cert": a.ClientCertFile,
		"client_key":  a.ClientKeyFile,
		"server_pin":  a.ServerCertPin,
	})) {
		t.Fatal("matching references were rejected")
	}
	changed := DigestReferences(map[string]string{
		"credential":  "/state/other.token",
		"ca":          a.CAFile,
		"client_cert": a.ClientCertFile,
		"client_key":  a.ClientKeyFile,
		"server_pin":  a.ServerCertPin,
	})
	if ReferenceDigestsMatch(want, changed) {
		t.Fatal("changed credential reference was accepted")
	}
}

func TestExternalAdoptionDirectorySyncFailureRestoresPrior(t *testing.T) {
	d := realTempDir(t)
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	cred := filepath.Join(d, "credential")
	if err := os.WriteFile(cred, []byte("opaque"), 0600); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "adoption.json")
	old := ExternalAdoption{Endpoint: "https://old.example", EndpointOriginDigest: "sha256:" + strings.Repeat("a", 64), EndpointIdentityDigest: "sha256:" + strings.Repeat("b", 64), Version: "1.0.0", CredentialFile: cred}
	if err := SaveExternalAdoption(p, old); err != nil {
		t.Fatal(err)
	}
	originalSync := adoptionSyncDirectory
	calls := 0
	adoptionSyncDirectory = func(path string) error {
		calls++
		if calls == 1 {
			return errors.New("injected sync failure")
		}
		return originalSync(path)
	}
	t.Cleanup(func() { adoptionSyncDirectory = originalSync })
	next := old
	next.Endpoint = "https://new.example"
	if err := SaveExternalAdoption(p, next); !errors.Is(err, ErrInstallDurabilityUncertain) {
		t.Fatalf("got %v", err)
	}
	got, err := LoadExternalAdoption(p)
	if err != nil || got != old {
		t.Fatalf("prior record not restored: got=%#v err=%v", got, err)
	}
}

func TestExternalAdoptionRenameFailureRestoresPriorDurably(t *testing.T) {
	d := realTempDir(t)
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	cred := filepath.Join(d, "credential")
	if err := os.WriteFile(cred, []byte("opaque"), 0600); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "adoption.json")
	old := ExternalAdoption{Endpoint: "https://old.example", EndpointOriginDigest: "sha256:" + strings.Repeat("a", 64), EndpointIdentityDigest: "sha256:" + strings.Repeat("b", 64), Version: "1.0.0", CredentialFile: cred}
	if err := SaveExternalAdoption(p, old); err != nil {
		t.Fatal(err)
	}
	originalRename, originalSync := adoptionRename, adoptionSyncDirectory
	renames, syncs := 0, 0
	adoptionRename = func(oldPath, newPath string) error {
		renames++
		if renames == 2 {
			return errors.New("injected rename failure")
		}
		return originalRename(oldPath, newPath)
	}
	adoptionSyncDirectory = func(path string) error { syncs++; return originalSync(path) }
	t.Cleanup(func() { adoptionRename, adoptionSyncDirectory = originalRename, originalSync })
	next := old
	next.Endpoint = "https://new.example"
	if err := SaveExternalAdoption(p, next); err == nil {
		t.Fatal("expected rename failure")
	}
	got, err := LoadExternalAdoption(p)
	if err != nil || got != old {
		t.Fatalf("prior record not restored: got=%#v err=%v", got, err)
	}
	if syncs == 0 {
		t.Fatal("restored directory was not synced")
	}
}

type scalarSysFileInfo struct{}

func (scalarSysFileInfo) Name() string       { return "hermes" }
func (scalarSysFileInfo) Size() int64        { return 1 }
func (scalarSysFileInfo) Mode() os.FileMode  { return 0700 }
func (scalarSysFileInfo) ModTime() time.Time { return time.Time{} }
func (scalarSysFileInfo) IsDir() bool        { return false }
func (scalarSysFileInfo) Sys() any           { return uint32(1) }

func TestSameOwnerFailsClosedForUnexpectedMetadata(t *testing.T) {
	if sameOwner(scalarSysFileInfo{}, 1) {
		t.Fatal("unexpected metadata accepted")
	}
}

type adoptionFake struct {
	h         hermes.Health
	caps      hermes.Capabilities
	identity  string
	err       error
	cancelled bool
	calls     *int
}

func (f adoptionFake) Capabilities(ctx context.Context) (hermes.Capabilities, error) {
	if f.calls != nil {
		*f.calls++
	}
	if f.cancelled {
		<-ctx.Done()
		return hermes.Capabilities{}, ctx.Err()
	}
	return f.caps, f.err
}
func (f adoptionFake) Health(ctx context.Context) (hermes.Health, error) {
	if f.calls != nil {
		*f.calls++
	}
	if f.cancelled {
		<-ctx.Done()
		return hermes.Health{}, ctx.Err()
	}
	return f.h, f.err
}
func (f adoptionFake) VerifiedEndpointIdentity(context.Context) (string, error) {
	if f.identity != "" {
		return f.identity, nil
	}
	return "authenticated-test-peer", nil
}

func goodExternalCandidate(t *testing.T) (HermesCandidate, string) {
	t.Helper()
	d := realTempDir(t)
	cred := filepath.Join(d, "bearer")
	if err := os.WriteFile(cred, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	caps := map[string]bool{}
	for _, n := range []string{hermes.CapabilityRunSubmission, hermes.CapabilityRunStatus, hermes.CapabilityRunEvents, hermes.CapabilityRunApproval, hermes.CapabilityRunStop} {
		caps[n] = true
	}
	f := adoptionFake{h: hermes.Health{Status: "ok", Version: "1.2.3"}, caps: hermes.Capabilities{Auth: hermes.CapabilityAuth{Type: "bearer", Required: true}, Features: caps}}
	return HermesCandidate{Endpoint: "https://hermes.example/runs", VerifiedEndpointIdentity: "leaf-identity", CredentialFile: cred, ExpectedVersion: "1.2.3", Profile: Development, Platform: PlatformLinux, OwnerKnown: true, OwnerUID: uint32(os.Getuid()), EndpointIdentityVerified: true, CredentialsSeparated: true, Adapter: f}, cred
}

func realTempDir(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	d, err := os.MkdirTemp(root, ".lumen-adoption-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

func boundedContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second)
}

func TestExternalAdoptionNeverInstallsOrControlsHermes(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(c.Path); !errors.Is(err, os.ErrNotExist) && c.Path != "" {
		t.Fatalf("unexpected Hermes artifact mutation: %v", err)
	}
}

func TestExternalAdoptionRequiresAuthenticatedCompatibility(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	f := c.Adapter.(adoptionFake)
	f.caps.Auth.Required = false
	c.Adapter = f
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("got %v", err)
	}
}

func TestExternalAdoptionRejectsEndpointSubstitution(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	calls := 0
	f := c.Adapter.(adoptionFake)
	f.calls = &calls
	c.Adapter = f
	c.ExpectedOriginDigest = digest("https://other.example/runs")
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("got %v", err)
	}
	if calls != 0 {
		t.Fatalf("adapter called before endpoint mismatch: %d", calls)
	}
}

func TestExternalAdoptionUsesAuthenticatedAdapterIdentity(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	c.VerifiedEndpointIdentity = "caller-supplied-identity"
	f := c.Adapter.(adoptionFake)
	f.identity = "authenticated-peer-identity"
	c.Adapter = f
	ctx, cancel := boundedContext()
	defer cancel()
	a, err := AdoptExternalHermes(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if a.EndpointIdentityDigest != digest("authenticated-peer-identity") {
		t.Fatalf("identity digest = %q, caller claim was trusted", a.EndpointIdentityDigest)
	}
}

func TestExternalAdoptionPreservesCredentialReferences(t *testing.T) {
	c, cred := goodExternalCandidate(t)
	ctx, cancel := boundedContext()
	defer cancel()
	a, err := AdoptExternalHermes(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(a)
	if string(b) == "" || strings.Contains(string(b), "secret") {
		t.Fatalf("credential leaked: %s", b)
	}
	var round ExternalAdoption
	if err := json.Unmarshal(b, &round); err != nil || round.CredentialFile != cred {
		t.Fatalf("round trip: %#v %v", round, err)
	}
}

func TestHardenedExternalAdoptionRequiresPinnedMutualTLS(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	c.Profile = Hardened
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("got %v", err)
	}
}

func TestExternalAdoptionRejectsUnsupportedProfileAndPlatform(t *testing.T) {
	for _, tc := range []struct {
		name     string
		profile  Profile
		platform Platform
	}{{"profile", Profile("invalid"), PlatformLinux}, {"platform", Development, Platform("invalid")}} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := goodExternalCandidate(t)
			c.Profile, c.Platform = tc.profile, tc.platform
			ctx, cancel := boundedContext()
			defer cancel()
			if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestExternalAdoptionRejectsWrongOwnerBeforeAdapter(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	calls := 0
	f := c.Adapter.(adoptionFake)
	f.calls = &calls
	c.Adapter = f
	c.OwnerUID++
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("got %v", err)
	}
	if calls != 0 {
		t.Fatalf("adapter called: %d", calls)
	}
}

func TestExternalAdoptionRejectsUnknownOwnerBeforeAdapter(t *testing.T) {
	c, _ := goodExternalCandidate(t)
	calls := 0
	f := c.Adapter.(adoptionFake)
	f.calls = &calls
	c.Adapter = f
	c.OwnerKnown = false
	ctx, cancel := boundedContext()
	defer cancel()
	if _, err := AdoptExternalHermes(ctx, c); !errors.Is(err, ErrHermesIncompatible) {
		t.Fatalf("got %v", err)
	}
	if calls != 0 {
		t.Fatalf("adapter called: %d", calls)
	}
}
func (f adoptionFake) CreateRun(context.Context, hermes.CreateRunRequest, string) (hermes.Run, error) {
	return hermes.Run{}, nil
}
func (f adoptionFake) RunStatus(context.Context, string) (hermes.Run, error) {
	return hermes.Run{}, nil
}
func (f adoptionFake) Events(context.Context, string) ([]hermes.Event, error)   { return nil, nil }
func (f adoptionFake) ResolveApproval(context.Context, string, string) error    { return nil }
func (f adoptionFake) Steer(context.Context, string, hermes.SteerRequest) error { return nil }
func (f adoptionFake) Stop(context.Context, string) (hermes.Run, error)         { return hermes.Run{}, nil }

func goodAdoption(t *testing.T) (HermesCandidate, string) {
	t.Helper()
	p := filepath.Join(realTempDir(t), "hermes")
	if err := os.WriteFile(p, []byte("keep"), 0700); err != nil {
		t.Fatal(err)
	}
	caps := map[string]bool{}
	for _, n := range []string{hermes.CapabilityRunSubmission, hermes.CapabilityRunStatus, hermes.CapabilityRunEvents, hermes.CapabilityRunApproval, hermes.CapabilityRunStop} {
		caps[n] = true
	}
	return HermesCandidate{Path: p, ExpectedVersion: "1.2.3", Profile: Development, Platform: PlatformLinux, OwnerKnown: true, OwnerUID: uint32(os.Getuid()), EndpointIdentityVerified: true, CredentialsSeparated: true, Adapter: adoptionFake{h: hermes.Health{Status: "ok", Version: "1.2.3"}, caps: hermes.Capabilities{Auth: hermes.CapabilityAuth{Type: "bearer", Required: true}, Features: caps}}}, p
}
func TestAdoptHermesAcceptsAndPreserves(t *testing.T) {
	c, p := goodAdoption(t)
	before, _ := os.Stat(p)
	if err := AdoptHermes(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(p)
	b, _ := os.ReadFile(p)
	if string(b) != "keep" || before.Mode() != after.Mode() {
		t.Fatal("candidate mutated")
	}
}

func TestAdoptHermesAcceptsApprovalResponseCapabilityAlias(t *testing.T) {
	c, _ := goodAdoption(t)
	f := c.Adapter.(adoptionFake)
	delete(f.caps.Features, hermes.CapabilityRunApproval)
	f.caps.Features["run_approval_response"] = true
	c.Adapter = f
	if err := AdoptHermes(context.Background(), c); err != nil {
		t.Fatalf("documented approval response alias rejected: %v", err)
	}
}

func TestAdoptHermesRejectsMatrix(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*HermesCandidate, adoptionFake)
	}{{"version", func(c *HermesCandidate, f adoptionFake) { f.h.Version = "9.9.9"; c.Adapter = f }}, {"unhealthy", func(c *HermesCandidate, f adoptionFake) { f.h.Status = "down"; c.Adapter = f }}, {"auth", func(c *HermesCandidate, f adoptionFake) { f.caps.Auth.Type = ""; c.Adapter = f }}, {"owner", func(c *HermesCandidate, f adoptionFake) { c.OwnerUID++ }}, {"profile", func(c *HermesCandidate, f adoptionFake) { c.Profile = Profile("bad") }}, {"platform", func(c *HermesCandidate, f adoptionFake) { c.Platform = Platform("bad") }}, {"termux hardened", func(c *HermesCandidate, f adoptionFake) { c.Profile = Hardened; c.Platform = PlatformTermux }}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := goodAdoption(t)
			f := c.Adapter.(adoptionFake)
			tc.mutate(&c, f)
			if err := AdoptHermes(context.Background(), c); !errors.Is(err, ErrHermesIncompatible) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestAdoptHermesRejectsEachMissingCapability(t *testing.T) {
	for _, missing := range []string{hermes.CapabilityRunSubmission, hermes.CapabilityRunStatus, hermes.CapabilityRunEvents, hermes.CapabilityRunApproval, hermes.CapabilityRunStop} {
		t.Run(missing, func(t *testing.T) {
			c, _ := goodAdoption(t)
			f := c.Adapter.(adoptionFake)
			delete(f.caps.Features, missing)
			c.Adapter = f
			if err := AdoptHermes(context.Background(), c); !errors.Is(err, ErrHermesIncompatible) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestAdoptHermesCancellation(t *testing.T) {
	c, _ := goodAdoption(t)
	f := c.Adapter.(adoptionFake)
	f.cancelled = true
	c.Adapter = f
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := AdoptHermes(ctx, c); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
func TestAdoptHermesRejectsSymlinkAndModes(t *testing.T) {
	c, p := goodAdoption(t)
	os.Chmod(p, 0755)
	if err := AdoptHermes(context.Background(), c); err == nil {
		t.Fatal("permissive accepted")
	}
	os.Remove(p)
	os.Symlink("missing", p)
	if err := AdoptHermes(context.Background(), c); err == nil {
		t.Fatal("symlink accepted")
	}
}
