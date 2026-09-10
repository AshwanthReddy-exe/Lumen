package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
)

var ErrHermesIncompatible = errors.New("Hermes installation is incompatible")

type HermesCandidate struct {
	Path                     string
	ExpectedVersion          string
	Profile                  Profile
	Platform                 Platform
	OwnerUID                 uint32
	OwnerKnown               bool
	EndpointIdentityVerified bool
	CredentialsSeparated     bool
	TLSVerified              bool
	LeafPinVerified          bool
	MutualTLSConfigured      bool
	Adapter                  hermes.Adapter
}

// AdoptHermes only observes a candidate. It never changes files, credentials,
// policy, or the advertised capability set.
func AdoptHermes(ctx context.Context, c HermesCandidate) error {
	if c.Path == "" || c.ExpectedVersion == "" || c.Adapter == nil {
		return ErrHermesIncompatible
	}
	if c.Profile != Development && c.Profile != PersonalAlpha && c.Profile != Hardened {
		return ErrHermesIncompatible
	}
	if c.Platform != PlatformMacOS && c.Platform != PlatformLinux && c.Platform != PlatformTermux {
		return ErrHermesIncompatible
	}
	if !c.EndpointIdentityVerified || !c.CredentialsSeparated {
		return ErrHermesIncompatible
	}
	if c.Profile == Hardened && (!c.TLSVerified || !c.LeafPinVerified || !c.MutualTLSConfigured) {
		return ErrHermesIncompatible
	}
	st, err := os.Lstat(c.Path)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&0111 == 0 {
		return ErrHermesIncompatible
	}
	if st.Mode()&0077 != 0 {
		return fmt.Errorf("%w: executable permissions", ErrHermesIncompatible)
	}
	if c.OwnerKnown {
		if !sameOwner(st, c.OwnerUID) {
			return fmt.Errorf("%w: owner", ErrHermesIncompatible)
		}
	}
	if c.Profile == Hardened && c.Platform == PlatformTermux {
		return fmt.Errorf("%w: hardened Termux isolation unavailable", ErrHermesIncompatible)
	}
	h, err := c.Adapter.Health(ctx)
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if err != nil || h.Status != "ok" || h.Version != c.ExpectedVersion {
		return fmt.Errorf("%w: health", ErrHermesIncompatible)
	}
	caps, err := c.Adapter.Capabilities(ctx)
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if err != nil || caps.Auth.Type != "bearer" || !caps.Auth.Required {
		return fmt.Errorf("%w: authentication", ErrHermesIncompatible)
	}
	for _, name := range []string{hermes.CapabilityRunSubmission, hermes.CapabilityRunStatus, hermes.CapabilityRunEvents, hermes.CapabilityRunApproval, hermes.CapabilityRunStop} {
		if !caps.Features[name] {
			return fmt.Errorf("%w: required capability", ErrHermesIncompatible)
		}
	}
	return nil
}

func sameOwner(st os.FileInfo, want uint32) bool {
	v := reflect.ValueOf(st.Sys())
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Struct {
		f := v.FieldByName("Uid")
		if f.IsValid() {
			return uint32(f.Uint()) == want
		}
	}
	return false
}
