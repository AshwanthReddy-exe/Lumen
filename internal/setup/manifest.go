package setup

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strings"
)

const ManifestSchemaVersion = 1
const MaxArtifactSize int64 = 2 << 30

type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	Topology      Topology   `json:"topology"`
	Artifacts     []Artifact `json:"artifacts"`
}
type Artifact struct {
	Name            string  `json:"name"`
	Version         string  `json:"version"`
	OS              string  `json:"os"`
	Architecture    string  `json:"architecture"`
	Profile         Profile `json:"profile"`
	URL             string  `json:"url"`
	Size            int64   `json:"size"`
	SHA256          string  `json:"sha256"`
	ContractVersion int     `json:"contractVersion"`
	ExecutableMode  uint32  `json:"executableMode"`
	Ownership       string  `json:"ownership"`
}

var semver = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
var ErrArtifactNotFound = errors.New("artifact not found")

func LoadManifest(r io.Reader) (Manifest, error) {
	var m Manifest
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return Manifest{}, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return Manifest{}, errors.New("manifest has trailing JSON")
	}
	if m.SchemaVersion != ManifestSchemaVersion || len(m.Artifacts) == 0 || m.Topology.Validate() != nil {
		return Manifest{}, errors.New("invalid manifest schema or empty artifacts")
	}
	seen := map[string]bool{}
	targets := map[string]map[string]bool{}
	for _, a := range m.Artifacts {
		if err := a.validate(); err != nil {
			return Manifest{}, err
		}
		if !validDigest(a.SHA256) {
			return Manifest{}, errors.New("invalid artifact digest")
		}
		u, _ := url.Parse(a.URL)
		if u.Host == "example.invalid" || (a.Name == "hermes" && (strings.HasSuffix(u.Path, "/main.tar.gz") || strings.HasSuffix(u.Path, "/master.tar.gz"))) {
			return Manifest{}, errors.New("invalid artifact source")
		}
		if (a.Name == "lumen" && a.Ownership != "lumen") || (a.Name == "hermes" && a.Ownership != "hermes") {
			return Manifest{}, errors.New("invalid artifact ownership")
		}
		k := a.Name + "\x00" + a.OS + "\x00" + a.Architecture + "\x00" + string(a.Profile)
		if seen[k] {
			return Manifest{}, errors.New("duplicate artifact target")
		}
		seen[k] = true
		target := a.OS + "\x00" + a.Architecture + "\x00" + string(a.Profile)
		if targets[target] == nil {
			targets[target] = map[string]bool{}
		}
		targets[target][a.Name] = true
	}
	for _, names := range targets {
		if m.Topology == TopologyCombined && (!names["lumen"] || !names["hermes"]) {
			return Manifest{}, errors.New("combined target requires lumen and hermes artifacts")
		}
		if m.Topology == TopologyExternal && (names["hermes"] || !names["lumen"]) {
			return Manifest{}, errors.New("external target requires lumen artifact only")
		}
	}
	return m, nil
}
func (a Artifact) validate() error {
	if (a.Name != "lumen" && a.Name != "hermes") || !semver.MatchString(a.Version) || a.ContractVersion != 1 || a.Size <= 0 || a.Size > MaxArtifactSize || !digestPattern.MatchString(a.SHA256) {
		return errors.New("invalid artifact fields")
	}
	if a.Profile != Development && a.Profile != PersonalAlpha && a.Profile != Hardened {
		return errors.New("invalid artifact profile")
	}
	if (a.OS != "darwin" && a.OS != "linux" && a.OS != "android") || (a.Architecture != "amd64" && a.Architecture != "arm64") || (a.OS == "android" && a.Architecture != "arm64") {
		return errors.New("unsupported artifact target")
	}
	u, err := url.Parse(a.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("invalid artifact URL")
	}
	if a.ExecutableMode != 0700 && a.ExecutableMode != 0755 {
		return errors.New("invalid executable mode")
	}
	return nil
}
func (m Manifest) Select(name, os, arch string, profile Profile) (Artifact, error) {
	for _, a := range m.Artifacts {
		if a.Name == name && a.OS == os && a.Architecture == arch && a.Profile == profile {
			return a, nil
		}
	}
	return Artifact{}, ErrArtifactNotFound
}
