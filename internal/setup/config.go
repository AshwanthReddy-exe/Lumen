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
	DataDir, HermesDir                                                         string
	Profile                                                                    Profile
	HermesBaseURL                                                              string
	HermesBearer, HermesCA, HermesClientCert, HermesClientKey, HermesServerPin string
	Journal                                                                    *Journal
}
type ConfigPaths struct{ Lumen, Hermes, Bearer, CA, ClientCert, ClientKey string }

func WriteConfig(r ConfigRequest) (ConfigPaths, error) {
	if r.DataDir == "" || r.HermesDir == "" || !filepath.IsAbs(r.DataDir) || !filepath.IsAbs(r.HermesDir) || filepath.Clean(r.DataDir) != r.DataDir || filepath.Clean(r.HermesDir) != r.HermesDir {
		return ConfigPaths{}, errors.New("absolute configuration directories required")
	}
	if r.Profile != Development && r.Profile != PersonalAlpha && r.Profile != Hardened {
		return ConfigPaths{}, errors.New("invalid profile")
	}
	u, err := url.Parse(r.HermesBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ConfigPaths{}, errors.New("invalid Hermes endpoint")
	}
	if r.Profile == Hardened && (u.Scheme != "https" || strings.HasPrefix(u.Hostname(), "127.") || u.Hostname() == "localhost" || u.Hostname() == "::1" || r.HermesServerPin == "" || r.HermesCA == "" || r.HermesClientCert == "" || r.HermesClientKey == "") {
		return ConfigPaths{}, errors.New("hardened configuration requires non-loopback TLS identity")
	}
	if err := safeDir(r.DataDir); err != nil {
		return ConfigPaths{}, err
	}
	if err := os.Chmod(r.DataDir, 0700); err != nil {
		return ConfigPaths{}, err
	}
	if err := safeDir(r.HermesDir); err != nil {
		return ConfigPaths{}, err
	}
	if err := os.Chmod(r.HermesDir, 0700); err != nil {
		return ConfigPaths{}, err
	}
	if r.Journal != nil {
		if err := r.Journal.Record(StageEvidence{Stage: DirectoriesReady, InputDigest: "sha256:" + strings.Repeat("0", 64)}); err != nil {
			return ConfigPaths{}, err
		}
	}
	p := ConfigPaths{Lumen: filepath.Join(r.DataDir, "lumen.json"), Hermes: filepath.Join(r.HermesDir, "hermes.json"), Bearer: filepath.Join(r.HermesDir, "hermes.token"), CA: filepath.Join(r.HermesDir, "ca.pem"), ClientCert: filepath.Join(r.HermesDir, "client.crt"), ClientKey: filepath.Join(r.HermesDir, "client.key")}
	for _, item := range []struct{ path, body string }{{p.Bearer, r.HermesBearer}, {p.CA, r.HermesCA}, {p.ClientCert, r.HermesClientCert}, {p.ClientKey, r.HermesClientKey}} {
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
	lumen := map[string]any{"version": 1, "data_dir": r.DataDir, "hermes": map[string]string{"base_url": r.HermesBaseURL, "profile": string(r.Profile), "bearer_file": p.Bearer, "ca_file": p.CA, "client_cert_file": p.ClientCert, "client_key_file": p.ClientKey, "server_cert_pin": r.HermesServerPin}}
	hermesCfg := map[string]any{"version": 1, "profile": string(r.Profile), "base_url": r.HermesBaseURL, "bearer_file": p.Bearer}
	if err := writeJSON(p.Lumen, lumen); err != nil {
		return ConfigPaths{}, err
	}
	if err := writeJSON(p.Hermes, hermesCfg); err != nil {
		return ConfigPaths{}, err
	}
	if r.Journal != nil {
		if err := r.Journal.Record(StageEvidence{Stage: ConfigurationReady, InputDigest: "sha256:" + strings.Repeat("2", 64)}); err != nil {
			return ConfigPaths{}, err
		}
	}
	return p, nil
}
func safeDir(p string) error {
	if s, err := os.Lstat(p); err == nil {
		if s.Mode()&os.ModeSymlink != 0 || !s.IsDir() {
			return errors.New("configuration directory must not contain symlinks")
		}
		return os.Chmod(p, 0700)
	}
	cur := string(filepath.Separator)
	for _, part := range strings.Split(filepath.Clean(p), string(filepath.Separator))[1:] {
		cur = filepath.Join(cur, part)
		if s, err := os.Lstat(cur); err == nil {
			if s.Mode()&os.ModeSymlink != 0 {
				if cur != "/var" && cur != "/private" {
					return errors.New("configuration directory must not contain symlinks")
				}
			}
			if !s.IsDir() {
				return errors.New("configuration directory must not contain symlinks")
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
