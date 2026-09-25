package setup

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// The setup journal binding is immutable by design: a rerun must never
// silently change the artifacts, topology, or profile a deployment was bound
// to. An explicit owner-initiated update must still be able to advance the
// installed artifact generation and later roll it back. ReleaseRecord is that
// separate, explicit record. It never rewrites the binding; it records which
// generation is live and which generation is available for rollback.
var (
	ErrReleaseRecordInvalid  = errors.New("invalid release record")
	ErrReleaseNotAdvanceable = errors.New("release generation cannot advance")
	ErrReleaseNoRollback     = errors.New("no release generation to roll back to")
	ErrReleaseBindingDrift   = errors.New("release record does not match the bound deployment")
)

// ReleaseArtifact is the durable identity of one installed artifact. It stores
// digests and references only, never credential material or file content.
type ReleaseArtifact struct {
	Kind    ArtifactKind `json:"kind"`
	Version string       `json:"version"`
	Digest  string       `json:"digest"`
	Path    string       `json:"path,omitempty"`
	Ref     string       `json:"ref,omitempty"`
}

// ReleaseRecord is append-only evidence of the installed release generation.
// Generation 1 is written by the first explicit update; the initial setup
// generation stays owned by the immutable journal binding.
type ReleaseRecord struct {
	Generation uint64                     `json:"generation"`
	Profile    Profile                    `json:"profile"`
	Topology   Topology                   `json:"topology"`
	Current    map[string]ReleaseArtifact `json:"current"`
	Rollback   map[string]ReleaseArtifact `json:"rollback,omitempty"`
}

// ReleaseArtifactFor derives the durable identity of a manifest artifact. It
// refuses any artifact without a real, non-placeholder identity so a fixture
// can never be recorded as an installed generation.
func ReleaseArtifactFor(a Artifact) (ReleaseArtifact, error) {
	if a.validate() != nil {
		return ReleaseArtifact{}, ErrReleaseRecordInvalid
	}
	switch a.normalizedKind() {
	case ArtifactExecutable:
		if !validDigest(a.SHA256) {
			return ReleaseArtifact{}, ErrReleaseRecordInvalid
		}
		return ReleaseArtifact{Kind: ArtifactExecutable, Version: a.Version, Digest: a.SHA256}, nil
	case ArtifactDockerImage:
		digest := imageDigestFromRef(a.ImageRef)
		if !validDigest(digest) {
			return ReleaseArtifact{}, ErrReleaseRecordInvalid
		}
		return ReleaseArtifact{Kind: ArtifactDockerImage, Version: a.Version, Digest: digest, Ref: a.ImageRef}, nil
	default:
		return ReleaseArtifact{}, ErrReleaseRecordInvalid
	}
}

// NextRelease advances a record by one generation. The bound profile and
// topology can never change, and the artifact set must stay identical: an
// update replaces an artifact, it does not add or remove one. A changed
// artifact set is a topology or profile change that must go through setup.
func NextRelease(current ReleaseRecord, target map[string]ReleaseArtifact) (ReleaseRecord, error) {
	if err := validReleaseRecord(current); err != nil {
		return ReleaseRecord{}, err
	}
	// An update replaces the bytes of an artifact. Adding or removing one is a
	// topology change that must go through setup, so it is rejected before the
	// target is otherwise validated.
	if !sameArtifactNames(current.Current, target) {
		return ReleaseRecord{}, ErrReleaseNotAdvanceable
	}
	next := ReleaseRecord{
		Generation: current.Generation + 1,
		Profile:    current.Profile,
		Topology:   current.Topology,
		Current:    copyReleaseArtifacts(target),
		Rollback:   copyReleaseArtifacts(current.Current),
	}
	if err := validReleaseRecord(next); err != nil {
		return ReleaseRecord{}, err
	}
	// Nothing to do when every artifact identity is unchanged and no rollback
	// generation is waiting to be consumed. Version metadata alone is not a
	// reason to advance a generation.
	if len(current.Rollback) == 0 && sameArtifactIdentities(current.Current, next.Current) {
		return ReleaseRecord{}, ErrReleaseNotAdvanceable
	}
	return next, nil
}

func sameArtifactIdentities(a, b map[string]ReleaseArtifact) bool {
	if len(a) != len(b) {
		return false
	}
	for name, x := range a {
		y, ok := b[name]
		if !ok || x.Kind != y.Kind || x.Digest != y.Digest {
			return false
		}
	}
	return true
}

// RollbackRelease swaps the live generation with the recorded rollback
// generation. It is the inverse of NextRelease and preserves both identities.
func RollbackRelease(current ReleaseRecord) (ReleaseRecord, error) {
	if err := validReleaseRecord(current); err != nil {
		return ReleaseRecord{}, err
	}
	if len(current.Rollback) == 0 || !sameArtifactNames(current.Current, current.Rollback) {
		return ReleaseRecord{}, ErrReleaseNoRollback
	}
	next := ReleaseRecord{
		Generation: current.Generation + 1,
		Profile:    current.Profile,
		Topology:   current.Topology,
		Current:    copyReleaseArtifacts(current.Rollback),
		Rollback:   copyReleaseArtifacts(current.Current),
	}
	if err := validReleaseRecord(next); err != nil {
		return ReleaseRecord{}, err
	}
	return next, nil
}

// ReleaseMatchesBinding proves a record describes the same deployment the
// immutable binding describes, and that the record cannot redirect which file
// or image a deployment verifies. The profile, topology, and every install
// path or pinned image reference must be identical to the binding; only an
// artifact digest may differ, because advancing a digest is exactly what an
// explicit update does. It is the authority check an update must pass before
// touching an installed artifact.
func ReleaseMatchesBinding(r ReleaseRecord, b JournalBinding) error {
	if err := validReleaseRecord(r); err != nil {
		return err
	}
	if r.Profile != b.Profile || r.Topology != b.Topology {
		return ErrReleaseBindingDrift
	}
	for _, name := range artifactNameSet(b.Topology) {
		a, ok := r.Current[name]
		if !ok {
			return ErrReleaseBindingDrift
		}
		path, hasPath := b.ArtifactPaths[name]
		ref, hasRef := b.ArtifactRefs[name]
		switch a.Kind {
		case ArtifactExecutable:
			if hasRef || !hasPath || a.Path != path {
				return ErrReleaseBindingDrift
			}
		case ArtifactDockerImage:
			if hasPath || !hasRef || a.Ref != ref {
				return ErrReleaseBindingDrift
			}
		default:
			return ErrReleaseBindingDrift
		}
	}
	return nil
}

func validReleaseRecord(r ReleaseRecord) error {
	if r.Generation == 0 || !validProfile(r.Profile) || r.Topology.Validate() != nil {
		return ErrReleaseRecordInvalid
	}
	if err := validReleaseArtifactSet(r.Current, r.Topology); err != nil {
		return err
	}
	if len(r.Rollback) > 0 {
		if err := validReleaseArtifactSet(r.Rollback, r.Topology); err != nil {
			return err
		}
	}
	return nil
}

// validReleaseArtifactSet requires the complete artifact set for the topology.
// A partial record would describe a deployment that cannot actually run, so it
// is rejected rather than repaired.
func validReleaseArtifactSet(set map[string]ReleaseArtifact, topology Topology) error {
	want := artifactNameSet(topology)
	if len(want) == 0 || len(set) != len(want) {
		return ErrReleaseRecordInvalid
	}
	for _, name := range want {
		a, ok := set[name]
		if !ok || validReleaseArtifact(a) != nil {
			return ErrReleaseRecordInvalid
		}
	}
	return nil
}

// artifactNameSet is the authoritative artifact set per topology. It mirrors
// the manifest LoadManifest topology rules without importing CLI concerns.
func artifactNameSet(topology Topology) []string {
	switch topology {
	case TopologyCombined:
		return []string{"lumen", "hermes"}
	case TopologyExternal:
		return []string{"lumen"}
	default:
		return nil
	}
}

func validReleaseArtifact(a ReleaseArtifact) error {
	// Version is optional human metadata; the digest is the identity. The
	// immutable setup binding records no version, so a generation synthesized
	// from an already-installed deployment legitimately has none.
	if a.Version != "" && !semver.MatchString(a.Version) {
		return ErrReleaseRecordInvalid
	}
	switch a.Kind {
	case ArtifactExecutable:
		if !validDigest(a.Digest) || a.Path == "" || !filepath.IsAbs(a.Path) || filepath.Clean(a.Path) != a.Path || a.Ref != "" {
			return ErrReleaseRecordInvalid
		}
		return nil
	case ArtifactDockerImage:
		if a.Path != "" || !validPinnedImageRef(a.Ref) || imageDigestFromRef(a.Ref) != a.Digest {
			return ErrReleaseRecordInvalid
		}
		return nil
	default:
		return ErrReleaseRecordInvalid
	}
}

func imageDigestFromRef(ref string) string {
	if i := strings.LastIndex(ref, "@"); i >= 0 {
		return ref[i+1:]
	}
	return ""
}

func copyReleaseArtifacts(in map[string]ReleaseArtifact) map[string]ReleaseArtifact {
	if in == nil {
		return nil
	}
	out := make(map[string]ReleaseArtifact, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sameArtifactNames(a, b map[string]ReleaseArtifact) bool {
	if len(a) != len(b) {
		return false
	}
	for name := range a {
		if _, ok := b[name]; !ok {
			return false
		}
	}
	return true
}

// SaveReleaseRecord durably writes the record into an owner-only private
// directory. A failure after the rename reports uncertain durability rather
// than claiming success.
func SaveReleaseRecord(path string, r ReleaseRecord) error {
	if err := validReleaseRecord(r); err != nil {
		return err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ErrReleaseRecordInvalid
	}
	parent := filepath.Dir(path)
	if err := validatePrivatePath(parent, uint32(os.Getuid()), false); err != nil {
		return ErrReleaseRecordInvalid
	}
	if err := os.Chmod(parent, 0700); err != nil {
		return err
	}
	if old, err := os.Lstat(path); err == nil {
		if old.Mode()&os.ModeSymlink != 0 || !old.Mode().IsRegular() || old.Mode().Perm()&0077 != 0 || !sameOwner(old, uint32(os.Getuid())) {
			return ErrReleaseRecordInvalid
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(parent, ".release-")
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
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	backup := ""
	if _, err := os.Lstat(path); err == nil {
		backup = path + ".rollback"
		if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err = os.Rename(path, backup); err != nil {
			return err
		}
	}
	if err = os.Rename(tmp, path); err != nil {
		if backup != "" {
			_ = os.Rename(backup, path)
			_ = syncDirectory(parent)
		}
		return err
	}
	if err = syncDirectory(parent); err != nil {
		_ = os.Remove(path)
		if backup != "" {
			_ = os.Rename(backup, path)
		}
		_ = syncDirectory(parent)
		return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, err)
	}
	if backup != "" {
		if err = os.Remove(backup); err != nil {
			_ = os.Remove(path)
			_ = os.Rename(backup, path)
			_ = syncDirectory(parent)
			return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, err)
		}
		_ = syncDirectory(parent)
	}
	return nil
}

// LoadReleaseRecord returns the durable record. A missing record is reported
// distinctly so callers can fall back to the binding's generation.
func LoadReleaseRecord(path string) (ReleaseRecord, error) {
	if err := validatePrivatePath(path, uint32(os.Getuid()), true); err != nil {
		return ReleaseRecord{}, ErrReleaseRecordInvalid
	}
	st, err := os.Lstat(path)
	if err != nil {
		return ReleaseRecord{}, err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 || !sameOwner(st, uint32(os.Getuid())) {
		return ReleaseRecord{}, ErrReleaseRecordInvalid
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ReleaseRecord{}, err
	}
	var r ReleaseRecord
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return ReleaseRecord{}, fmt.Errorf("%w: %v", ErrReleaseRecordInvalid, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return ReleaseRecord{}, ErrReleaseRecordInvalid
	}
	if err := validReleaseRecord(r); err != nil {
		return ReleaseRecord{}, err
	}
	return r, nil
}

// VerifyReleaseGeneration checks that every live executable artifact still
// matches its recorded digest on disk. Docker images are verified by the
// caller, which owns the container runtime.
func VerifyReleaseGeneration(r ReleaseRecord) error {
	if err := validReleaseRecord(r); err != nil {
		return err
	}
	for _, a := range r.Current {
		if a.Kind != ArtifactExecutable {
			continue
		}
		if err := executableDigestMatches(a.Path, a.Digest); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseStoreDir is the private directory that retains artifact bytes for a
// later offline rollback. It lives beside the setup journal inside the
// deployment data directory.
func ReleaseStoreDir(dataDir string) string { return filepath.Join(dataDir, "releases") }

func retainedArtifactPath(storeDir, name, digest string) string {
	return filepath.Join(storeDir, name, strings.TrimPrefix(digest, "sha256:"))
}

// RetainExecutableGeneration copies the live executable artifacts of a
// generation into the private store so a later rollback can restore them
// without network access. Every copy is proven against the recorded digest
// before it is accepted, so a tampered live artifact is never retained as a
// trusted rollback source. Retention is idempotent and never removes other
// generations.
func RetainExecutableGeneration(storeDir string, r ReleaseRecord) error {
	if err := validReleaseRecord(r); err != nil {
		return err
	}
	for name, a := range r.Current {
		if a.Kind != ArtifactExecutable {
			continue
		}
		if err := executableDigestMatches(a.Path, a.Digest); err != nil {
			return err
		}
		dir := filepath.Join(storeDir, name)
		if err := safeDir(dir); err != nil {
			return ErrReleaseRecordInvalid
		}
		if err := os.Chmod(dir, 0700); err != nil {
			return err
		}
		dest := retainedArtifactPath(storeDir, name, a.Digest)
		if err := privateRetainedDigestMatches(dest, a.Digest); err == nil {
			continue
		}
		src, err := os.Open(a.Path)
		if err != nil {
			return err
		}
		err = writeVerified(dest, src, a.Digest, 0600)
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

// RestoreRetainedGeneration writes retained bytes back to an installed
// generation using atomic replace, verifying every digest before and after the
// swap. A failure on any artifact leaves the previously live bytes in place;
// the caller owns journaling whether the restore completed.
func RestoreRetainedGeneration(storeDir string, r ReleaseRecord, mode os.FileMode) error {
	if err := validReleaseRecord(r); err != nil {
		return err
	}
	if mode != 0700 && mode != 0755 {
		return ErrReleaseRecordInvalid
	}
	for name, a := range r.Current {
		if a.Kind != ArtifactExecutable {
			continue
		}
		srcPath := retainedArtifactPath(storeDir, name, a.Digest)
		if err := privateRetainedDigestMatches(srcPath, a.Digest); err != nil {
			return err
		}
		src, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		err = writeVerified(a.Path, src, a.Digest, mode)
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

// writeVerified atomically replaces dest with the bytes read from src after
// proving they match the expected digest. Any failure leaves the previous dest
// content untouched, so an interrupted update or rollback preserves the live
// artifact.
func writeVerified(dest string, src io.Reader, expected string, mode os.FileMode) error {
	if !validDigest(expected) || src == nil || (mode != 0600 && mode != 0700 && mode != 0755) {
		return ErrReleaseRecordInvalid
	}
	if !filepath.IsAbs(dest) || filepath.Clean(dest) != dest {
		return ErrReleaseRecordInvalid
	}
	parent := filepath.Dir(dest)
	if err := validatePrivatePath(parent, uint32(os.Getuid()), false); err != nil {
		return ErrReleaseRecordInvalid
	}
	if old, err := os.Lstat(dest); err == nil {
		if old.Mode()&os.ModeSymlink != 0 || !old.Mode().IsRegular() || !sameOwner(old, uint32(os.Getuid())) {
			return ErrReleaseRecordInvalid
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(parent, ".release-gen-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(src, MaxArtifactSize+1))
	if err != nil {
		_ = f.Close()
		return err
	}
	if n > MaxArtifactSize {
		_ = f.Close()
		return ErrSizeMismatch
	}
	if subtle.ConstantTimeCompare([]byte("sha256:"+hexDigest(h.Sum(nil))), []byte(expected)) != 1 {
		_ = f.Close()
		return ErrDigestMismatch
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, dest); err != nil {
		return err
	}
	if err = syncDirectory(parent); err != nil {
		return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, err)
	}
	return nil
}

// executableDigestMatches proves an installed artifact still matches its
// recorded digest and is a regular, owner-controlled executable.
func executableDigestMatches(path, want string) error {
	if err := digestFileMatches(path, want); err != nil {
		return err
	}
	st, err := os.Lstat(path)
	if err != nil || st.Mode()&0111 == 0 {
		return ErrReleaseRecordInvalid
	}
	return nil
}

// privateRetainedDigestMatches proves retained bytes are a regular,
// owner-only file with the expected digest. Retained bytes are never
// executable and never readable by another principal.
func privateRetainedDigestMatches(path, want string) error {
	if err := digestFileMatches(path, want); err != nil {
		return err
	}
	st, err := os.Lstat(path)
	if err != nil || st.Mode().Perm()&0077 != 0 {
		return ErrReleaseRecordInvalid
	}
	return nil
}

func digestFileMatches(path, want string) error {
	if !validDigest(want) {
		return ErrReleaseRecordInvalid
	}
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || !sameOwner(st, uint32(os.Getuid())) {
		return ErrReleaseRecordInvalid
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, MaxArtifactSize+1))
	if err != nil || n > MaxArtifactSize {
		return ErrReleaseRecordInvalid
	}
	if subtle.ConstantTimeCompare([]byte("sha256:"+hexDigest(h.Sum(nil))), []byte(want)) != 1 {
		return ErrDigestMismatch
	}
	return nil
}
