package setup

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	ErrDigestMismatch             = errors.New("artifact digest mismatch")
	ErrSizeMismatch               = errors.New("artifact size mismatch")
	ErrInstallDurabilityUncertain = errors.New("artifact install durability uncertain")
	ErrInvalidArtifact            = errors.New("invalid artifact")
)

type StagedArtifact struct {
	Path     string
	Artifact Artifact
}
type Installer struct {
	StageDir, InstallDir string
	Rename               func(string, string) error
	SyncDir              func(string) error
}

func (i Installer) Stage(ctx context.Context, a Artifact, src io.Reader) (StagedArtifact, error) {
	if validateArtifact(a) != nil || src == nil {
		return StagedArtifact{}, ErrInvalidArtifact
	}
	if e := os.MkdirAll(i.StageDir, 0700); e != nil {
		return StagedArtifact{}, e
	}
	if e := os.Chmod(i.StageDir, 0700); e != nil {
		return StagedArtifact{}, e
	}
	f, e := os.CreateTemp(i.StageDir, ".artifact-")
	if e != nil {
		return StagedArtifact{}, e
	}
	p := f.Name()
	fail := func(e error) (StagedArtifact, error) { _ = f.Close(); _ = os.Remove(p); return StagedArtifact{}, e }
	if e = f.Chmod(0600); e != nil {
		return fail(e)
	}
	h := sha256.New()
	var n int64
	b := make([]byte, 32768)
	for n <= a.Size {
		select {
		case <-ctx.Done():
			return fail(ctx.Err())
		default:
		}
		w := int64(len(b))
		if r := a.Size + 1 - n; r < w {
			w = r
		}
		m, re := src.Read(b[:w])
		if m > 0 {
			if _, e = f.Write(b[:m]); e != nil {
				return fail(e)
			}
			_, _ = h.Write(b[:m])
			n += int64(m)
		}
		if re == io.EOF {
			break
		}
		if re != nil {
			return fail(re)
		}
		if m == 0 {
			return fail(io.ErrNoProgress)
		}
	}
	if n != a.Size {
		return fail(ErrSizeMismatch)
	}
	if subtle.ConstantTimeCompare([]byte("sha256:"+hexDigest(h.Sum(nil))), []byte(a.SHA256)) != 1 {
		return fail(ErrDigestMismatch)
	}
	if e = f.Sync(); e != nil {
		return fail(e)
	}
	if e = f.Close(); e != nil {
		_ = os.Remove(p)
		return StagedArtifact{}, e
	}
	if e = os.Chmod(p, os.FileMode(a.ExecutableMode)); e != nil {
		_ = os.Remove(p)
		return StagedArtifact{}, e
	}
	return StagedArtifact{Path: p, Artifact: a}, nil
}
func (i Installer) Install(ctx context.Context, a Artifact, src io.Reader) error {
	s, e := i.Stage(ctx, a, src)
	if e != nil {
		return e
	}
	return i.InstallStaged(a, s)
}
func (i Installer) InstallStaged(a Artifact, s StagedArtifact) error {
	defer os.Remove(s.Path)
	if validateArtifact(a) != nil || s.Path == "" || !reflect.DeepEqual(a, s.Artifact) {
		return ErrInvalidArtifact
	}
	root, _ := filepath.Abs(i.StageDir)
	path, _ := filepath.Abs(s.Path)
	rel, e := filepath.Rel(root, path)
	if e != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.Contains(rel, string(filepath.Separator)) {
		return ErrInvalidArtifact
	}
	st, e := os.Lstat(s.Path)
	if e != nil || st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
		return ErrInvalidArtifact
	}
	if e = os.MkdirAll(i.InstallDir, 0700); e != nil {
		return e
	}
	if e = os.Chmod(i.InstallDir, 0700); e != nil {
		return e
	}
	target := filepath.Join(i.InstallDir, a.Name)
	if filepath.Dir(target) != filepath.Clean(i.InstallDir) {
		return ErrInvalidArtifact
	}
	if old, e := os.Lstat(target); e == nil && (old.Mode()&os.ModeSymlink != 0 || !old.Mode().IsRegular()) {
		return ErrInvalidArtifact
	} else if e != nil && !os.IsNotExist(e) {
		return e
	}
	rn := i.Rename
	if rn == nil {
		rn = os.Rename
	}
	sd := i.SyncDir
	if sd == nil {
		sd = syncDirectory
	}
	backup := target + ".rollback"
	if _, e = os.Lstat(backup); e == nil {
		return ErrInvalidArtifact
	} else if !os.IsNotExist(e) {
		return e
	}
	had := false
	if _, e = os.Lstat(target); e == nil {
		if e = rn(target, backup); e != nil {
			return e
		}
		had = true
	}
	if had {
		if e = sd(i.InstallDir); e != nil {
			_ = rn(backup, target)
			_ = sd(i.InstallDir)
			return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, e)
		}
	}
	if e = verifyStaged(s.Path, a); e != nil {
		if had {
			_ = rn(backup, target)
			_ = sd(i.InstallDir)
		}
		return e
	}
	if e = rn(s.Path, target); e != nil {
		if had {
			_ = rn(backup, target)
		}
		return e
	}
	if e = sd(i.InstallDir); e != nil {
		return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, e)
	}
	if had {
		_ = os.Remove(backup)
		if e = sd(i.InstallDir); e != nil {
			return fmt.Errorf("%w: %v", ErrInstallDurabilityUncertain, e)
		}
	}
	return nil
}
func validateArtifact(a Artifact) error {
	if a.validate() != nil || filepath.Base(a.Name) != a.Name || strings.ContainsAny(a.Name, `/\\`) {
		return ErrInvalidArtifact
	}
	return nil
}
func verifyStaged(path string, a Artifact) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.CopyN(h, f, a.Size+1)
	if err != nil && err != io.EOF {
		return err
	}
	if n != a.Size {
		return ErrSizeMismatch
	}
	if subtle.ConstantTimeCompare([]byte("sha256:"+hexDigest(h.Sum(nil))), []byte(a.SHA256)) != 1 {
		return ErrDigestMismatch
	}
	return nil
}

// VerifyArtifact checks an already installed artifact against its manifest
// without changing the installation.
func VerifyArtifact(path string, a Artifact) error {
	if validateArtifact(a) != nil {
		return ErrInvalidArtifact
	}
	return verifyStaged(path, a)
}
func hexDigest(b []byte) string {
	const d = "0123456789abcdef"
	o := make([]byte, len(b)*2)
	for i, v := range b {
		o[i*2] = d[v>>4]
		o[i*2+1] = d[v&15]
	}
	return string(o)
}
