package setup

import (
	"context"
	"errors"
	"testing"
)

type fakeProbe struct {
	isolation          bool
	goos, arch, prefix string
	supervisor         Supervisor
	available          bool
	calls              int
	versions           [2]string
	adopted            bool
	setup              string
	cancel             func()
}

func (p *fakeProbe) HardenedIsolation() bool { p.calls++; return p.isolation }
func (p *fakeProbe) GOOS() string {
	p.calls++
	if p.cancel != nil {
		p.cancel()
	}
	if p.goos == "" {
		return "linux"
	}
	return p.goos
}
func (p *fakeProbe) GOARCH() string {
	p.calls++
	if p.arch == "" {
		return "amd64"
	}
	return p.arch
}
func (p *fakeProbe) TermuxPrefix() string           { p.calls++; return p.prefix }
func (p *fakeProbe) Supervisor() (Supervisor, bool) { p.calls++; return p.supervisor, p.available }
func (p *fakeProbe) SetupDir() string               { p.calls++; return p.setup }
func (p *fakeProbe) InstalledVersions() (string, string, bool) {
	p.calls++
	return p.versions[0], p.versions[1], p.adopted
}
func TestHardenedNeverDowngrades(t *testing.T) {
	_, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: Hardened}, &fakeProbe{})
	if !errors.Is(e, ErrIsolationUnavailable) {
		t.Fatal(e)
	}
}
func TestPlanFacts(t *testing.T) {
	p := &fakeProbe{goos: "linux", arch: "arm64", supervisor: SupervisorRunit, available: true, setup: "/x", versions: [2]string{"l", "h"}, adopted: true, isolation: true}
	g, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: PersonalAlpha}, p)
	if e != nil || g.Platform != PlatformLinux || g.Architecture != "arm64" || g.SetupDir != "/x" || g.LumenVersion != "l" || g.HermesVersion != "h" || g.Supervisor != SupervisorRunit || g.Outcome != Ready {
		t.Fatalf("%#v %v", g, e)
	}
}
func TestPlanPlatforms(t *testing.T) {
	for _, x := range []struct {
		o, p string
		w    Platform
	}{{"darwin", "", PlatformMacOS}, {"linux", "", PlatformLinux}, {"android", "", PlatformTermux}, {"linux", "/data/data/com.termux/files/usr", PlatformTermux}} {
		g, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: Development}, &fakeProbe{goos: x.o, prefix: x.p, available: true})
		if e != nil || g.Platform != x.w {
			t.Fatalf("%#v %v", g, e)
		}
	}
}
func TestPlanValidationCancellationSupervisor(t *testing.T) {
	p := &fakeProbe{goos: "darwin"}
	if _, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: Profile("bad")}, p); e == nil || p.calls != 0 {
		t.Fatal(e, p.calls)
	}
	if _, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: Development}, nil); e == nil {
		t.Fatal("nil")
	}
	if _, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: Development}, &fakeProbe{goos: "linux", arch: "mips"}); e == nil {
		t.Fatal("arch")
	}
	c, f := context.WithCancel(context.Background())
	f()
	p = &fakeProbe{}
	if _, e := Plan(c, Request{Topology: TopologyCombined, Profile: Development}, p); !errors.Is(e, context.Canceled) || p.calls != 0 {
		t.Fatal(e, p.calls)
	}
	g, e := Plan(context.Background(), Request{Topology: TopologyCombined, Profile: Development}, &fakeProbe{goos: "linux"})
	if e != nil || g.Outcome != ActionRequired || g.Actions[0].Code != "supervisor_unavailable" {
		t.Fatalf("%#v %v", g, e)
	}
}

func TestPlanCancellationBetweenProbeCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &fakeProbe{cancel: cancel}
	if _, err := Plan(ctx, Request{Topology: TopologyCombined, Profile: Development}, p); !errors.Is(err, context.Canceled) || p.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, p.calls)
	}
}

func TestPlanRequiresExplicitSupportedTopology(t *testing.T) {
	for _, topology := range []Topology{"", "local", "adopted"} {
		if _, err := Plan(context.Background(), Request{Profile: Development, Topology: topology}, &fakeProbe{}); err == nil {
			t.Fatalf("accepted topology %q", topology)
		}
	}
	for _, topology := range []Topology{TopologyCombined, TopologyExternal} {
		if _, err := Plan(context.Background(), Request{Profile: Development, Topology: topology}, &fakeProbe{}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPlanKeepsTopologyIndependentFromProfile(t *testing.T) {
	for _, profile := range []Profile{Development, PersonalAlpha, Hardened} {
		p := &fakeProbe{isolation: true}
		for _, topology := range []Topology{TopologyCombined, TopologyExternal} {
			got, err := Plan(context.Background(), Request{Profile: profile, Topology: topology}, p)
			if err != nil || got.Topology != topology || got.Profile != profile {
				t.Fatalf("profile=%q topology=%q result=%#v err=%v", profile, topology, got, err)
			}
		}
	}
}
