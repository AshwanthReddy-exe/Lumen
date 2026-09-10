package setup

import (
	"context"
	"strings"
	"testing"
)

func TestDoctorDistinguishesDegradedAndRedactsDetails(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{HostReady: true, HermesReady: false, SupervisorReady: true, BootReady: true, CredentialsReady: true, IsolationReady: true}
	}}.Check(context.Background())
	if r.Outcome != Degraded {
		t.Fatalf("got %#v", r)
	}
	r = Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{HostReady: false, HermesReady: true, SupervisorReady: false, BootReady: false, CredentialsReady: false, IsolationReady: false}
	}}.Check(context.Background())
	if r.Outcome != ActionRequired || len(r.Actions) == 0 {
		t.Fatalf("got %#v", r)
	}
	for _, a := range r.Actions {
		if strings.Contains(a.Detail, "/Users/") {
			t.Fatal("path leaked")
		}
	}
}
