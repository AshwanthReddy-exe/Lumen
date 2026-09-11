package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
)

func SaveExternalAdoption(path string, adoption ExternalAdoption) error {
	if !validAdoption(adoption) {
		return fmt.Errorf("%w: adoption", ErrHermesIncompatible)
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("%w: target", ErrHermesIncompatible)
	}
	parent := filepath.Dir(path)
	st, err := os.Stat(parent)
	if err != nil || !st.IsDir() || st.Mode().Perm()&0077 != 0 || !sameOwner(st, uint32(os.Getuid())) {
		return fmt.Errorf("%w: parent", ErrHermesIncompatible)
	}
	if old, err := os.Lstat(path); err == nil && (old.Mode()&0077 != 0 || old.Mode()&os.ModeSymlink != 0 || !old.Mode().IsRegular() || !sameOwner(old, uint32(os.Getuid()))) {
		return ErrHermesIncompatible
	}
	b, err := json.Marshal(adoption)
	if err != nil {
		return err
	}
	f, err := adoptionCreateTemp(parent, ".adoption-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = adoptionSyncFile(f); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		return err
	}
	backup := ""
	if _, err := os.Stat(path); err == nil {
		backup = path + ".bak"
		if err := adoptionRemove(backup); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err = adoptionRename(path, backup); err != nil {
			return err
		}
	}
	if err = adoptionRename(tmp, path); err != nil {
		if backup != "" {
			_ = adoptionRename(backup, path)
			_ = adoptionSyncDirectory(parent)
		}
		return err
	}
	if err = adoptionSyncDirectory(parent); err != nil {
		_ = adoptionRemove(path)
		if backup != "" {
			_ = adoptionRename(backup, path)
		}
		_ = adoptionSyncDirectory(parent)
		return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, err)
	}
	if backup != "" {
		if err := adoptionRemove(backup); err != nil {
			_ = adoptionRemove(path)
			_ = adoptionRename(backup, path)
			_ = adoptionSyncDirectory(parent)
			return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, err)
		}
	}
	return nil
}

func LoadExternalAdoption(path string) (ExternalAdoption, error) {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&0077 != 0 || st.Mode()&os.ModeSymlink != 0 || !sameOwner(st, uint32(os.Getuid())) {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ExternalAdoption{}, err
	}
	var a ExternalAdoption
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return ExternalAdoption{}, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if !validAdoption(a) {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	return a, nil
}

func validAdoption(a ExternalAdoption) bool {
	u, err := url.Parse(a.Endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	canonical := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + strings.TrimRight(u.EscapedPath(), "/")
	if a.Endpoint != canonical {
		return false
	}
	if !validDigest(a.EndpointOriginDigest) || !validDigest(a.EndpointIdentityDigest) || a.Version == "" {
		return false
	}
	if a.CredentialFile == "" {
		return false
	}
	for _, p := range []string{a.CredentialFile, a.CAFile, a.ClientCertFile, a.ClientKeyFile} {
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return false
		}
		st, err := os.Lstat(p)
		if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 || !sameOwner(st, uint32(os.Getuid())) {
			return false
		}
	}
	return true
}

var (
	adoptionCreateTemp    = os.CreateTemp
	adoptionRename        = os.Rename
	adoptionRemove        = os.Remove
	adoptionSyncFile      = func(f *os.File) error { return f.Sync() }
	adoptionSyncDirectory = syncDirectory
)

var ErrHermesIncompatible = errors.New("Hermes installation is incompatible")
var ErrHermesUnavailable = errors.New("Hermes endpoint unavailable")

type HermesCandidate struct {
	Endpoint                                     string
	VerifiedEndpointIdentity                     string
	ExpectedOriginDigest, ExpectedIdentityDigest string
	CredentialFile                               string
	CAFile, ClientCertFile, ClientKeyFile        string
	Path                                         string
	ExpectedVersion                              string
	Profile                                      Profile
	Platform                                     Platform
	OwnerUID                                     uint32
	OwnerKnown                                   bool
	EndpointIdentityVerified                     bool
	CredentialsSeparated                         bool
	TLSVerified                                  bool
	LeafPinVerified                              bool
	MutualTLSConfigured                          bool
	Adapter                                      hermes.Adapter
}

// ExternalAdoption is the only durable evidence needed for an independently
// managed Hermes. It deliberately contains references and digests, never
// credential material.
type ExternalAdoption struct {
	Endpoint               string `json:"endpoint"`
	EndpointOriginDigest   string `json:"endpoint_origin_digest"`
	EndpointIdentityDigest string `json:"endpoint_identity_digest"`
	Version                string `json:"version"`
	CredentialFile         string `json:"credential_file,omitempty"`
	CAFile                 string `json:"ca_file,omitempty"`
	ClientCertFile         string `json:"client_cert_file,omitempty"`
	ClientKeyFile          string `json:"client_key_file,omitempty"`
}

func AdoptExternalHermes(ctx context.Context, c HermesCandidate) (ExternalAdoption, error) {
	if _, ok := ctx.Deadline(); !ok {
		return ExternalAdoption{}, fmt.Errorf("%w: bounded context required", ErrHermesIncompatible)
	}
	if c.Endpoint == "" || c.CredentialFile == "" {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if !c.OwnerKnown {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if c.Profile != Development && c.Profile != PersonalAlpha && c.Profile != Hardened {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if c.Platform != PlatformLinux && c.Platform != PlatformMacOS && c.Platform != PlatformTermux {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.User != nil || u.Scheme == "" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(u.Host, "@") {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if c.Profile != Development && u.Scheme != "https" {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if c.Profile == Hardened && (!c.TLSVerified || !c.LeafPinVerified || !c.MutualTLSConfigured || c.CAFile == "" || c.ClientCertFile == "" || c.ClientKeyFile == "") {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	if err := validateEndpointFiles(c); err != nil {
		return ExternalAdoption{}, err
	}
	canonical := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + strings.TrimRight(u.EscapedPath(), "/")
	originDigest := digest(canonical)
	identityDigest := digest(c.VerifiedEndpointIdentity)
	if c.VerifiedEndpointIdentity == "" || identityDigest == "sha256:"+strings.Repeat("0", 64) || (c.ExpectedOriginDigest != "" && c.ExpectedOriginDigest != originDigest) || (c.ExpectedIdentityDigest != "" && c.ExpectedIdentityDigest != identityDigest) {
		return ExternalAdoption{}, fmt.Errorf("%w: endpoint identity", ErrHermesIncompatible)
	}
	if c.Adapter == nil || !c.EndpointIdentityVerified || !c.CredentialsSeparated {
		return ExternalAdoption{}, ErrHermesIncompatible
	}
	h, err := c.Adapter.Health(ctx)
	if errors.Is(err, context.Canceled) {
		return ExternalAdoption{}, err
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ExternalAdoption{}, err
		}
		return ExternalAdoption{}, fmt.Errorf("%w: health: %v", ErrHermesUnavailable, err)
	}
	if h.Status != "ok" || (c.ExpectedVersion != "" && h.Version != c.ExpectedVersion) {
		return ExternalAdoption{}, fmt.Errorf("%w: health", ErrHermesIncompatible)
	}
	caps, err := c.Adapter.Capabilities(ctx)
	if errors.Is(err, context.Canceled) {
		return ExternalAdoption{}, err
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ExternalAdoption{}, err
		}
		return ExternalAdoption{}, fmt.Errorf("%w: capabilities: %v", ErrHermesUnavailable, err)
	}
	if caps.Auth.Type != "bearer" || !caps.Auth.Required {
		return ExternalAdoption{}, fmt.Errorf("%w: authentication", ErrHermesIncompatible)
	}
	for _, n := range []string{hermes.CapabilityRunSubmission, hermes.CapabilityRunStatus, hermes.CapabilityRunEvents, hermes.CapabilityRunApproval, hermes.CapabilityRunStop} {
		if !caps.Features[n] && !(n == hermes.CapabilityRunApproval && caps.Features["run_approval_response"]) {
			return ExternalAdoption{}, fmt.Errorf("%w: required capability", ErrHermesIncompatible)
		}
	}
	return ExternalAdoption{Endpoint: canonical, EndpointOriginDigest: originDigest, EndpointIdentityDigest: identityDigest, Version: h.Version, CredentialFile: c.CredentialFile, CAFile: c.CAFile, ClientCertFile: c.ClientCertFile, ClientKeyFile: c.ClientKeyFile}, nil
}

func digest(s string) string {
	h := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(h[:])
}
func validateEndpointFiles(c HermesCandidate) error {
	for _, p := range []string{c.CredentialFile, c.CAFile, c.ClientCertFile, c.ClientKeyFile} {
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return ErrHermesIncompatible
		}
		s, err := os.Lstat(p)
		if err != nil || !s.Mode().IsRegular() || s.Mode()&os.ModeSymlink != 0 || s.Mode().Perm()&0077 != 0 {
			return ErrHermesIncompatible
		}
		if c.OwnerKnown && !sameOwner(s, c.OwnerUID) {
			return fmt.Errorf("%w: credential owner", ErrHermesIncompatible)
		}
	}
	return nil
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
