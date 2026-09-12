package setup

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ConfigRequest struct {
	DataDir, HermesDir                                                            string
	Profile                                                                       Profile
	HermesBaseURL                                                                 string
	HermesBearer, HermesCA, HermesClientCert, HermesClientKey, HermesServerPin    string
	Journal                                                                       *Journal
	Topology                                                                      Topology
	HermesCredentialFile, HermesCAFile, HermesClientCertFile, HermesClientKeyFile string
}
type ConfigPaths struct{ Lumen, Hermes, Bearer, CA, ClientCert, ClientKey string }

func WriteConfig(r ConfigRequest) (ConfigPaths, error) {
	if r.DataDir == "" || r.HermesDir == "" || !filepath.IsAbs(r.DataDir) || !filepath.IsAbs(r.HermesDir) || filepath.Clean(r.DataDir) != r.DataDir || filepath.Clean(r.HermesDir) != r.HermesDir {
		return ConfigPaths{}, errors.New("absolute configuration directories required")
	}
	if r.Profile != Development && r.Profile != PersonalAlpha && r.Profile != Hardened {
		return ConfigPaths{}, errors.New("invalid profile")
	}
	if r.Topology == "" {
		r.Topology = TopologyCombined
	}
	if err := r.Topology.Validate(); err != nil {
		return ConfigPaths{}, err
	}
	u, err := url.Parse(r.HermesBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ConfigPaths{}, errors.New("invalid Hermes endpoint")
	}
	secureExternal := r.Topology == TopologyExternal && (r.Profile == PersonalAlpha || r.Profile == Hardened)
	secureCombined := r.Topology != TopologyExternal && r.Profile == Hardened
	if (secureExternal || secureCombined) && (u.Scheme != "https" || strings.HasPrefix(u.Hostname(), "127.") || u.Hostname() == "localhost" || u.Hostname() == "::1" || r.HermesServerPin == "") {
		return ConfigPaths{}, errors.New("hardened configuration requires non-loopback TLS identity")
	}
	if secureCombined && (r.HermesCA == "" || r.HermesClientCert == "" || r.HermesClientKey == "") {
		return ConfigPaths{}, errors.New("hardened configuration requires TLS material")
	}
	if err := safeDir(r.DataDir); err != nil {
		return ConfigPaths{}, err
	}
	if err := os.Chmod(r.DataDir, 0700); err != nil {
		return ConfigPaths{}, err
	}
	if r.Topology != TopologyExternal {
		if err := safeDir(r.HermesDir); err != nil {
			return ConfigPaths{}, err
		}
		if err := os.Chmod(r.HermesDir, 0700); err != nil {
			return ConfigPaths{}, err
		}
	}
	if r.Journal != nil {
		if err := r.Journal.Record(StageEvidence{Stage: DirectoriesReady, InputDigest: "sha256:" + strings.Repeat("0", 64)}); err != nil {
			return ConfigPaths{}, err
		}
	}
	p := ConfigPaths{Lumen: filepath.Join(r.DataDir, "lumen.json"), Hermes: filepath.Join(r.HermesDir, "hermes.json"), Bearer: filepath.Join(r.HermesDir, "hermes.token"), CA: filepath.Join(r.HermesDir, "ca.pem"), ClientCert: filepath.Join(r.HermesDir, "client.crt"), ClientKey: filepath.Join(r.HermesDir, "client.key")}
	if r.Topology == TopologyExternal {
		if r.HermesCredentialFile == "" {
			return ConfigPaths{}, errors.New("external topology requires credential reference")
		}
		if secureExternal {
			if err := validateEndpointFiles(HermesCandidate{CredentialFile: r.HermesCredentialFile, CAFile: r.HermesCAFile, ClientCertFile: r.HermesClientCertFile, ClientKeyFile: r.HermesClientKeyFile, OwnerKnown: true, OwnerUID: uint32(os.Getuid())}); err != nil {
				return ConfigPaths{}, errors.New("hardened external configuration requires valid credential references")
			}
		}
		p.Bearer, p.CA, p.ClientCert, p.ClientKey = r.HermesCredentialFile, r.HermesCAFile, r.HermesClientCertFile, r.HermesClientKeyFile
		p.Hermes = ""
	}
	items := []struct{ path, body string }{{p.Bearer, r.HermesBearer}, {p.CA, r.HermesCA}, {p.ClientCert, r.HermesClientCert}, {p.ClientKey, r.HermesClientKey}}
	if r.Topology == TopologyExternal {
		items = nil
	}
	for _, item := range items {
		path, body := item.path, item.body
		if body != "" {
			if err := writePrivate(path, []byte(body)); err != nil {
				return ConfigPaths{}, err
			}
		}
	}
	if r.Journal != nil {
		if err := r.Journal.Record(StageEvidence{Stage: CredentialsReady, InputDigest: "sha256:" + strings.Repeat("1", 64)}); err != nil {
			return ConfigPaths{}, err
		}
	}
	profile := string(r.Profile)
	if r.Profile == PersonalAlpha && r.Topology == TopologyExternal {
		profile = "hardened"
	}
	lumen := map[string]any{"version": 1, "data_dir": r.DataDir, "hermes": map[string]string{"base_url": r.HermesBaseURL, "profile": profile, "bearer_file": p.Bearer, "ca_file": p.CA, "client_cert_file": p.ClientCert, "client_key_file": p.ClientKey, "server_cert_pin": r.HermesServerPin}}
	hermesCfg := map[string]any{"version": 1, "profile": string(r.Profile), "base_url": r.HermesBaseURL, "bearer_file": p.Bearer}
	if err := writeJSON(p.Lumen, lumen); err != nil {
		return ConfigPaths{}, err
	}
	if r.Topology != TopologyExternal {
		if err := writeJSON(p.Hermes, hermesCfg); err != nil {
			return ConfigPaths{}, err
		}
	}
	if r.Journal != nil {
		if err := r.Journal.Record(StageEvidence{Stage: ConfigurationReady, InputDigest: "sha256:" + strings.Repeat("2", 64)}); err != nil {
			return ConfigPaths{}, err
		}
	}
	return p, nil
}
func safeDir(p string) error {
	cur := string(filepath.Separator)
	for _, part := range strings.Split(filepath.Clean(p), string(filepath.Separator))[1:] {
		cur = filepath.Join(cur, part)
		if s, err := os.Lstat(cur); err == nil {
			if s.Mode()&os.ModeSymlink != 0 {
				return errors.New("configuration directory must not contain symlinks")
			}
			if !s.IsDir() {
				return errors.New("configuration directory must not contain symlinks")
			}
			if cur == filepath.Clean(p) && !sameOwner(s, uint32(os.Getuid())) {
				return errors.New("configuration directory must be private and owner-controlled")
			}
		} else if os.IsNotExist(err) {
			if err := os.Mkdir(cur, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return os.Chmod(p, 0700)
}

func validatePrivatePath(path string, owner uint32, file bool) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("private path must be absolute and clean")
	}
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))[1:]
	cur := string(filepath.Separator)
	for i, part := range parts {
		cur = filepath.Join(cur, part)
		st, err := os.Lstat(cur)
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("private path contains unsafe component")
		}
		if i < len(parts)-1 {
			privateOwner := sameOwner(st, owner)
			trustedSystem := sameOwner(st, 0) && st.Mode().Perm()&0022 == 0
			if !st.IsDir() || (privateOwner && st.Mode().Perm()&0022 != 0) || (!privateOwner && !trustedSystem) {
				return errors.New("private path parent is not a directory")
			}
		} else if file {
			if !st.Mode().IsRegular() || !sameOwner(st, owner) || st.Mode().Perm()&0077 != 0 {
				return errors.New("private path is not a regular file")
			}
		} else if !st.IsDir() || !sameOwner(st, owner) || st.Mode().Perm()&0077 != 0 {
			return errors.New("private path is not a directory")
		}
	}
	return nil
}
func writePrivate(p string, b []byte) error {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	if _, e = f.Write(b); e != nil {
		return e
	}
	return f.Sync()
}
func writeJSON(p string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return writePrivate(p, append(b, '\n'))
}
