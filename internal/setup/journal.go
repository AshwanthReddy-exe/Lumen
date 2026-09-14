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
	"sort"
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
	dir, path   string
	bindingPath string
	evidence    []StageEvidence
	binding     *JournalBinding
	syncParent  func(string) error
}

type JournalBinding struct {
	Profile                Profile           `json:"profile"`
	Topology               Topology          `json:"topology"`
	Supervisor             Supervisor        `json:"supervisor,omitempty"`
	ComposeProject         string            `json:"composeProject,omitempty"`
	EndpointOriginDigest   string            `json:"endpointOriginDigest,omitempty"`
	EndpointIdentityDigest string            `json:"endpointIdentityDigest,omitempty"`
	ReferenceDigests       map[string]string `json:"referenceDigests,omitempty"`
	ArtifactPaths          map[string]string `json:"artifactPaths,omitempty"`
	ArtifactDigests        map[string]string `json:"artifactDigests,omitempty"`
	ArtifactRefs           map[string]string `json:"artifactRefs,omitempty"`
	PlanDigest             string            `json:"planDigest"`
}

func NewJournal(d string) (*Journal, error) {
	j := &Journal{dir: d, path: filepath.Join(d, "setup-journal.json"), bindingPath: filepath.Join(d, "setup-binding.json"), syncParent: syncDirectory}
	if b, e := os.ReadFile(j.bindingPath); e == nil {
		var binding JournalBinding
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if dec.Decode(&binding) != nil || binding.Topology.Validate() != nil || !validProfile(binding.Profile) || (binding.Supervisor != "" && binding.Supervisor.Validate() != nil) || validateComposeBinding(binding.Supervisor, binding.ComposeProject) != nil || !validDigest(binding.PlanDigest) || (binding.EndpointOriginDigest != "" && !validDigest(binding.EndpointOriginDigest)) || (binding.EndpointIdentityDigest != "" && !validDigest(binding.EndpointIdentityDigest)) {
			return nil, ErrInvalidJournal
		}
		for _, digest := range binding.ArtifactDigests {
			if !validDigest(digest) {
				return nil, ErrInvalidJournal
			}
		}
		for _, digest := range binding.ReferenceDigests {
			if !validDigest(digest) {
				return nil, ErrInvalidJournal
			}
		}
		if !validArtifactPaths(binding.ArtifactPaths) {
			return nil, ErrInvalidJournal
		}
		if !validArtifactRefs(binding.ArtifactRefs) || !disjointArtifactBindings(binding.ArtifactPaths, binding.ArtifactRefs) {
			return nil, ErrInvalidJournal
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			return nil, ErrInvalidJournal
		}
		j.binding = &binding
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
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
func (j *Journal) Bind(b JournalBinding) error {
	if !validProfile(b.Profile) || b.Topology.Validate() != nil || (b.Supervisor != "" && b.Supervisor.Validate() != nil) || validateComposeBinding(b.Supervisor, b.ComposeProject) != nil || (b.Supervisor == SupervisorDocker && b.ComposeProject == "") || !validDigest(b.PlanDigest) || (b.EndpointOriginDigest != "" && !validDigest(b.EndpointOriginDigest)) || (b.EndpointIdentityDigest != "" && !validDigest(b.EndpointIdentityDigest)) {
		return ErrInvalidJournal
	}
	for _, d := range b.ArtifactDigests {
		if !validDigest(d) {
			return ErrInvalidJournal
		}
	}
	for _, d := range b.ReferenceDigests {
		if !validDigest(d) {
			return ErrInvalidJournal
		}
	}
	if !validArtifactPaths(b.ArtifactPaths) {
		return ErrInvalidJournal
	}
	if !validArtifactRefs(b.ArtifactRefs) || !disjointArtifactBindings(b.ArtifactPaths, b.ArtifactRefs) {
		return ErrInvalidJournal
	}
	if j.binding != nil {
		if ((j.binding.Supervisor == "" && b.Supervisor != "") || (j.binding.Supervisor == SupervisorDocker && j.binding.ComposeProject == "" && b.ComposeProject != "")) && legacyBindingCanUpgrade(*j.binding, b) {
			return j.writeBinding(b)
		}
		if !bindingsEqual(*j.binding, b) {
			return ErrInputChanged
		}
		return nil
	}
	if err := os.MkdirAll(j.dir, 0700); err != nil {
		return err
	}
	return j.writeBinding(b)
}

// legacyBindingCanUpgrade accepts only fields absent from the old binding as
// new evidence. Any value the old binding did record remains immutable.
func legacyBindingCanUpgrade(old, next JournalBinding) bool {
	if old.Profile != next.Profile || old.Topology != next.Topology || old.PlanDigest != next.PlanDigest {
		return false
	}
	if old.ComposeProject != "" && old.ComposeProject != next.ComposeProject {
		return false
	}
	if old.EndpointOriginDigest != "" && old.EndpointOriginDigest != next.EndpointOriginDigest {
		return false
	}
	if old.EndpointIdentityDigest != "" && old.EndpointIdentityDigest != next.EndpointIdentityDigest {
		return false
	}
	if old.ReferenceDigests != nil && !ReferenceDigestsMatch(old.ReferenceDigests, next.ReferenceDigests) {
		return false
	}
	if old.ArtifactDigests != nil && !ReferenceDigestsMatch(old.ArtifactDigests, next.ArtifactDigests) {
		return false
	}
	if old.ArtifactPaths != nil && !ReferenceDigestsMatch(old.ArtifactPaths, next.ArtifactPaths) {
		return false
	}
	if old.ArtifactRefs != nil && !ReferenceDigestsMatch(old.ArtifactRefs, next.ArtifactRefs) {
		return false
	}
	return true
}

func (j *Journal) writeBinding(b JournalBinding) error {
	stored := b
	if b.ReferenceDigests != nil {
		stored.ReferenceDigests = make(map[string]string, len(b.ReferenceDigests))
		for k, v := range b.ReferenceDigests {
			stored.ReferenceDigests[k] = v
		}
	}
	if b.ArtifactDigests != nil {
		stored.ArtifactDigests = make(map[string]string, len(b.ArtifactDigests))
		for k, v := range b.ArtifactDigests {
			stored.ArtifactDigests[k] = v
		}
	}
	if b.ArtifactPaths != nil {
		stored.ArtifactPaths = make(map[string]string, len(b.ArtifactPaths))
		for k, v := range b.ArtifactPaths {
			stored.ArtifactPaths[k] = v
		}
	}
	if b.ArtifactRefs != nil {
		stored.ArtifactRefs = make(map[string]string, len(b.ArtifactRefs))
		for k, v := range b.ArtifactRefs {
			stored.ArtifactRefs[k] = v
		}
	}
	data, _ := json.Marshal(stored)
	tmp, err := os.CreateTemp(j.dir, ".binding-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, j.bindingPath); err != nil {
		return err
	}
	if err = j.syncParent(j.dir); err != nil {
		return fmt.Errorf("binding durability: %w", err)
	}
	j.binding = &stored
	return nil
}

// Binding returns the immutable setup selection without exposing journal-owned
// maps to callers. A missing binding is distinct from an empty binding.
func (j *Journal) Binding() (JournalBinding, bool) {
	if j == nil || j.binding == nil {
		return JournalBinding{}, false
	}
	b := *j.binding
	if j.binding.ReferenceDigests != nil {
		b.ReferenceDigests = make(map[string]string, len(j.binding.ReferenceDigests))
		for k, v := range j.binding.ReferenceDigests {
			b.ReferenceDigests[k] = v
		}
	}
	if j.binding.ArtifactDigests != nil {
		b.ArtifactDigests = make(map[string]string, len(j.binding.ArtifactDigests))
		for k, v := range j.binding.ArtifactDigests {
			b.ArtifactDigests[k] = v
		}
	}
	if j.binding.ArtifactPaths != nil {
		b.ArtifactPaths = make(map[string]string, len(j.binding.ArtifactPaths))
		for k, v := range j.binding.ArtifactPaths {
			b.ArtifactPaths[k] = v
		}
	}
	if j.binding.ArtifactRefs != nil {
		b.ArtifactRefs = make(map[string]string, len(j.binding.ArtifactRefs))
		for k, v := range j.binding.ArtifactRefs {
			b.ArtifactRefs[k] = v
		}
	}
	return b, true
}

func bindingsEqual(a, b JournalBinding) bool {
	if a.Profile != b.Profile || a.Topology != b.Topology || a.Supervisor != b.Supervisor || a.ComposeProject != b.ComposeProject || a.EndpointOriginDigest != b.EndpointOriginDigest || a.EndpointIdentityDigest != b.EndpointIdentityDigest || a.PlanDigest != b.PlanDigest || !ReferenceDigestsMatch(a.ReferenceDigests, b.ReferenceDigests) || !ReferenceDigestsMatch(a.ArtifactPaths, b.ArtifactPaths) || !ReferenceDigestsMatch(a.ArtifactRefs, b.ArtifactRefs) {
		return false
	}
	if len(a.ArtifactDigests) != len(b.ArtifactDigests) {
		return false
	}
	keys := make([]string, 0, len(a.ArtifactDigests))
	for k := range a.ArtifactDigests {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if a.ArtifactDigests[k] != b.ArtifactDigests[k] {
			return false
		}
	}
	return true
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
		if i >= len(stageOrder) || v.Stage != stageOrder[i] || !validDigest(v.InputDigest) || (v.Profile != "" && !validProfile(v.Profile)) || (v.PlanDigest != "" && !validDigest(v.PlanDigest)) {
			return ErrInvalidJournal
		}
	}
	return nil
}

func validProfile(p Profile) bool { return p == Development || p == PersonalAlpha || p == Hardened }

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func validDigest(s string) bool {
	if !digestPattern.MatchString(s) {
		return false
	}
	for _, c := range s[len("sha256:"):] {
		if c != '0' {
			return true
		}
	}
	return false
}

func validArtifactPaths(paths map[string]string) bool {
	for name, path := range paths {
		if name == "" || path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return false
		}
	}
	return true
}

func validArtifactRefs(refs map[string]string) bool {
	for name, ref := range refs {
		if name == "" || !validPinnedImageRef(ref) {
			return false
		}
	}
	return true
}

func disjointArtifactBindings(paths, refs map[string]string) bool {
	for name := range refs {
		if _, ok := paths[name]; ok {
			return false
		}
	}
	return true
}

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
