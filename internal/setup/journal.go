package setup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var (
	ErrInputChanged        = errors.New("setup input changed")
	ErrInvalidJournal      = errors.New("invalid setup journal")
	ErrInvalidDigest       = errors.New("invalid input digest")
	ErrDurabilityUncertain = errors.New("durability uncertain")
)
var stageOrder = []Stage{Detected, ArtifactsReady, DirectoriesReady, CredentialsReady, ConfigurationReady, HostInitialized, ServicesInstalled, ServicesStarted, Validated}

type Journal struct {
	dir, path  string
	evidence   []StageEvidence
	syncParent func(string) error
}

func NewJournal(d string) (*Journal, error) {
	j := &Journal{dir: d, path: filepath.Join(d, "setup-journal.json"), syncParent: syncDirectory}
	b, e := os.ReadFile(j.path)
	if errors.Is(e, os.ErrNotExist) {
		return j, nil
	}
	if e != nil {
		return nil, e
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&j.evidence); e != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJournal, e)
	}
	var extra any
	if e = dec.Decode(&extra); e != io.EOF {
		return nil, fmt.Errorf("%w: trailing content", ErrInvalidJournal)
	}
	return j, validateEvidence(j.evidence)
}
func (j *Journal) Next() Stage {
	if len(j.evidence) < len(stageOrder) {
		return stageOrder[len(j.evidence)]
	}
	return Validated
}
func (j *Journal) Record(e StageEvidence) error {
	if !validStage(e.Stage) || !validDigest(e.InputDigest) {
		return ErrInvalidDigest
	}
	if len(j.evidence) > 0 && e.Stage == j.evidence[len(j.evidence)-1].Stage {
		if e.InputDigest != j.evidence[len(j.evidence)-1].InputDigest {
			return ErrInputChanged
		}
		return nil
	}
	if e.Stage != j.Next() {
		return ErrInvalidJournal
	}
	if e.CompletedAt == 0 {
		e.CompletedAt = time.Now().Unix()
	}
	c := append(append([]StageEvidence(nil), j.evidence...), e)
	ren, er := persist(j.dir, j.path, c, j.syncParent)
	if er != nil && !ren {
		return er
	}
	j.evidence = c
	if er != nil {
		return fmt.Errorf("%w: %v", ErrDurabilityUncertain, er)
	}
	return nil
}
func persist(d, p string, e []StageEvidence, sync func(string) error) (bool, error) {
	if x := os.MkdirAll(d, 0700); x != nil {
		return false, x
	}
	if x := os.Chmod(d, 0700); x != nil {
		return false, x
	}
	b, x := json.Marshal(e)
	if x != nil {
		return false, x
	}
	f, x := os.CreateTemp(d, ".journal-")
	if x != nil {
		return false, x
	}
	n := f.Name()
	defer os.Remove(n)
	if x = f.Chmod(0600); x == nil {
		var written int
		written, x = f.Write(b)
		if x == nil && written != len(b) {
			x = io.ErrShortWrite
		}
	}
	if x == nil {
		x = f.Sync()
	}
	if c := f.Close(); x == nil {
		x = c
	}
	if x != nil {
		return false, x
	}
	if x = os.Rename(n, p); x != nil {
		return false, x
	}
	return true, sync(d)
}
func validateEvidence(e []StageEvidence) error {
	for i, v := range e {
		if i >= len(stageOrder) || v.Stage != stageOrder[i] || !validDigest(v.InputDigest) {
			return ErrInvalidJournal
		}
	}
	return nil
}

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func validDigest(s string) bool { return digestPattern.MatchString(s) }
func validStage(s Stage) bool {
	for _, v := range stageOrder {
		if s == v {
			return true
		}
	}
	return false
}
func syncDirectory(d string) error {
	f, e := os.Open(d)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
