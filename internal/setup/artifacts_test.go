package setup

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testArtifact(body string) Artifact {
	h := sha256.Sum256([]byte(body))
	return Artifact{Name: "lumen", Version: "1.0.0", OS: "linux", Architecture: "amd64", Profile: Development, URL: "https://example.invalid/lumen", Size: int64(len(body)), SHA256: "sha256:" + fmtHex(h[:]), ContractVersion: 1, ExecutableMode: 0700}
}
func fmtHex(b []byte) string {
	const x = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = x[v>>4]
		out[i*2+1] = x[v&15]
	}
	return string(out)
}
func TestStageVerifiesAndInstallsAtomically(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: filepath.Join(d, "stage"), InstallDir: filepath.Join(d, "bin")}
	a := testArtifact("new")
	s, err := i.Stage(context.Background(), a, strings.NewReader("new"))
	if err != nil {
		t.Fatal(err)
	}
	if err := i.InstallStaged(a, s); err != nil {
		t.Fatal(err)
	}
	b, e := os.ReadFile(filepath.Join(d, "bin", "lumen"))
	if e != nil || string(b) != "new" {
		t.Fatalf("%q %v", b, e)
	}
}
func TestStageRejectsSizeAndDigest(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: d}
	a := testArtifact("new")
	a.SHA256 = "sha256:" + strings.Repeat("0", 64)
	if !errors.Is(mustStage(i, a, "new"), ErrDigestMismatch) {
		t.Fatal("digest")
	}
}
func mustStage(i Installer, a Artifact, s string) error {
	_, e := i.Stage(context.Background(), a, strings.NewReader(s))
	return e
}
func TestInstallPreservesExistingOnFailure(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: filepath.Join(d, "stage"), InstallDir: filepath.Join(d, "bin")}
	os.MkdirAll(i.InstallDir, 0700)
	p := filepath.Join(i.InstallDir, "lumen")
	os.WriteFile(p, []byte("old"), 0700)
	a := testArtifact("bad")
	a.SHA256 = "sha256:" + strings.Repeat("0", 64)
	if !errors.Is(i.Install(context.Background(), a, strings.NewReader("bad")), ErrDigestMismatch) {
		t.Fatal("want mismatch")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "old" {
		t.Fatal(string(b))
	}
}
func TestStageCancellation(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: d}
	a := testArtifact("x")
	ctx, c := context.WithCancel(context.Background())
	c()
	_, e := i.Stage(ctx, a, strings.NewReader("x"))
	if !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(d, ".stage")); !os.IsNotExist(e) {
		t.Fatal("left staging")
	}
	_ = io.EOF
}

func TestStageRejectsShortAndOversized(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: d}
	short := testArtifact("abcd")
	if err := mustStage(i, short, "abc"); !errors.Is(err, ErrSizeMismatch) {
		t.Fatal(err)
	}
	over := testArtifact("abc")
	if err := mustStage(i, over, "abcd"); !errors.Is(err, ErrSizeMismatch) {
		t.Fatal(err)
	}
}

func TestVerifyArtifactUsesManifestDigestWithoutMutation(t *testing.T) {
	d := t.TempDir()
	body := "verified"
	a := testArtifact(body)
	p := filepath.Join(d, a.Name)
	if err := os.WriteFile(p, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyArtifact(p, a); err != nil {
		t.Fatal(err)
	}
	a.SHA256 = "sha256:" + strings.Repeat("0", 64)
	if err := VerifyArtifact(p, a); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("mismatch=%v", err)
	}
}

func TestStagePermissionsAndInstallCleanup(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: filepath.Join(d, "stage"), InstallDir: filepath.Join(d, "bin")}
	a := testArtifact("new")
	a.ExecutableMode = 0755
	s, err := i.Stage(context.Background(), a, strings.NewReader("new"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(i.StageDir); err != nil {
		t.Fatal(err)
	}
	if mode := mustMode(t, i.StageDir); mode != 0700 {
		t.Fatalf("stage %o", mode)
	}
	if mode := mustMode(t, s.Path); mode != 0755 {
		t.Fatalf("file %o", mode)
	}
	if err := i.InstallStaged(a, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Path); !os.IsNotExist(err) {
		t.Fatal("staged file remains")
	}
	if _, err := os.Stat(filepath.Join(i.InstallDir, "lumen.rollback")); !os.IsNotExist(err) {
		t.Fatal("rollback remains")
	}
}

func TestInstallRejectsSymlinksAndTraversal(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: filepath.Join(d, "stage"), InstallDir: filepath.Join(d, "bin")}
	a := testArtifact("x")
	s, _ := i.Stage(context.Background(), a, strings.NewReader("x"))
	os.Remove(s.Path)
	os.Symlink(filepath.Join(d, "elsewhere"), s.Path)
	if err := i.InstallStaged(a, s); !errors.Is(err, ErrInvalidArtifact) {
		t.Fatal(err)
	}
	os.Remove(s.Path)
	os.MkdirAll(i.InstallDir, 0700)
	os.Symlink(filepath.Join(d, "elsewhere"), filepath.Join(i.InstallDir, "lumen"))
	s, _ = i.Stage(context.Background(), a, strings.NewReader("x"))
	if err := i.InstallStaged(a, s); !errors.Is(err, ErrInvalidArtifact) {
		t.Fatal(err)
	}
	a.Name = "../escape"
	if err := mustStage(i, a, "x"); !errors.Is(err, ErrInvalidArtifact) {
		t.Fatal(err)
	}
}

func TestInstallRenameFailureRestoresPrior(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: filepath.Join(d, "stage"), InstallDir: filepath.Join(d, "bin")}
	os.MkdirAll(i.InstallDir, 0700)
	p := filepath.Join(i.InstallDir, "lumen")
	os.WriteFile(p, []byte("old"), 0641)
	a := testArtifact("new")
	calls := 0
	i.Rename = func(from, to string) error {
		calls++
		if calls == 2 {
			return errors.New("replace")
		}
		return os.Rename(from, to)
	}
	err := i.Install(context.Background(), a, strings.NewReader("new"))
	if err == nil {
		t.Fatal("expected rename failure")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "old" {
		t.Fatal(string(b))
	}
	if mustMode(t, p) != 0641 {
		t.Fatal("mode changed")
	}
	if _, err := os.Stat(p + ".rollback"); !os.IsNotExist(err) {
		t.Fatal("rollback remains")
	}
}

func TestInstallSyncFailureIsUncertain(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: filepath.Join(d, "stage"), InstallDir: filepath.Join(d, "bin"), SyncDir: func(string) error { return errors.New("sync") }}
	a := testArtifact("new")
	err := i.Install(context.Background(), a, strings.NewReader("new"))
	if !errors.Is(err, ErrInstallDurabilityUncertain) {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(i.InstallDir, "lumen"))
	if string(b) != "new" {
		t.Fatalf("disk=%q", b)
	}
}

type cancelReader struct {
	n      int
	cancel context.CancelFunc
}

func (r *cancelReader) Read(p []byte) (int, error) {
	if r.n > 0 {
		return 0, io.EOF
	}
	r.n++
	copy(p, "a")
	r.cancel()
	return 1, nil
}
func TestStageMidReadCancellationCleansTemp(t *testing.T) {
	d := t.TempDir()
	i := Installer{StageDir: d}
	a := testArtifact("aa")
	ctx, cancel := context.WithCancel(context.Background())
	_, err := i.Stage(ctx, a, &cancelReader{cancel: cancel})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(d, ".artifact-*"))
	if len(matches) != 0 {
		t.Fatal(matches)
	}
}

func mustMode(t *testing.T, p string) os.FileMode {
	t.Helper()
	s, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return s.Mode().Perm()
}
