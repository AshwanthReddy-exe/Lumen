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

func goodExternalCandidate(t *testing.T) (HermesCandidate, string) {
	t.Helper()
	d := t.TempDir()
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
	p := t.TempDir() + "/hermes"
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
