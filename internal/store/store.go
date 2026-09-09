package store

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

type Hooks struct {
	Write   func(*os.File, []byte) error
	Sync    func(*os.File) error
	Rename  func(string, string) error
	DirSync func(*os.File) error
	Remove  func(string) error
}

type Store struct {
	path      string
	key       string
	stateName string
	keyName   string
	root      *os.Root
	keyRoot   *os.Root
	lock      *os.File
	hooks     Hooks
	mu        sync.Mutex
	close     sync.Once
	closed    bool
}

// New opens the store lock. With one argument it treats the argument as a
// private data directory; with two it accepts explicit state and key paths.
func New(path string, keyPath ...string) (*Store, error) {
	if len(keyPath) == 0 {
		keyPath = []string{filepath.Join(path, "state.key")}
		path = filepath.Join(path, "state.json")
	} else if len(keyPath) != 1 {
		return nil, errors.New("store expects a state path and key path")
	}
	if err := validPath(path); err != nil {
		return nil, err
	}
	if err := validPath(keyPath[0]); err != nil {
		return nil, err
	}
	parent, err := openParent(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	keyParent, err := openParent(filepath.Dir(keyPath[0]))
	if err != nil {
		parent.Close()
		return nil, err
	}
	lockPath := filepath.Base(path) + ".lock"
	lock, err := parent.OpenFile(lockPath, os.O_RDWR|os.O_CREATE|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		keyParent.Close()
		parent.Close()
		return nil, err
	}
	if err := checkFile(lock); err != nil {
		lock.Close()
		keyParent.Close()
		parent.Close()
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		keyParent.Close()
		parent.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errors.New("state store already active")
		}
		return nil, err
	}
	if err := verifyRoot(parent, filepath.Dir(path)); err != nil {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
		keyParent.Close()
		parent.Close()
		return nil, errors.New("store parent changed during open")
	}
	if filepath.Dir(path) == filepath.Dir(keyPath[0]) {
		if err := verifyRoot(keyParent, filepath.Dir(keyPath[0])); err != nil {
			_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			_ = lock.Close()
			keyParent.Close()
			parent.Close()
			return nil, errors.New("key parent changed during open")
		}
	}
	return &Store{path: path, key: keyPath[0], stateName: filepath.Base(path), keyName: filepath.Base(keyPath[0]), root: parent, keyRoot: keyParent, lock: lock}, nil
}

func Open(path string, keyPath ...string) (*Store, error) { return New(path, keyPath...) }

func NewWithHooks(path string, hooks Hooks, keyPath ...string) (*Store, error) {
	s, err := New(path, keyPath...)
	if err == nil {
		s.hooks = hooks
	}
	return s, err
}

func (s *Store) Close() error {
	var err error
	s.close.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed {
			return
		}
		s.closed = true
		if s.lock != nil {
			if e := syscall.Flock(int(s.lock.Fd()), syscall.LOCK_UN); e != nil {
				err = e
			}
			if e := s.lock.Close(); err == nil {
				err = e
			}
		}
		if s.root != nil {
			if e := s.root.Close(); err == nil {
				err = e
			}
		}
		if s.keyRoot != nil {
			if e := s.keyRoot.Close(); err == nil {
				err = e
			}
		}
	})
	return err
}

func (s *Store) Initialize(state space.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("store is closed")
	}
	if err := validateState(state); err != nil {
		return err
	}
	var stateErr error
	_, stateErr = rootLstat(s.root, s.stateName)
	if stateErr == nil {
		return errors.New("state already initialized")
	} else if !errors.Is(stateErr, os.ErrNotExist) {
		return stateErr
	}
	key, err := s.loadOrCreateKey(true)
	if err != nil {
		return err
	}
	return s.commit(key, state, false)
}

func (s *Store) Read() (space.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return space.State{}, errors.New("store is closed")
	}
	return s.read()
}

func (s *Store) Update(fn func(space.State) space.Transition) (space.Transition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return space.Transition{}, errors.New("store is closed")
	}
	state, err := s.read()
	if err != nil {
		return space.Transition{}, err
	}
	if fn == nil {
		return space.Transition{}, errors.New("nil state transition")
	}
	var tr space.Transition
	func() {
		defer func() {
			if recover() != nil {
				err = errors.New("state transition panicked")
			}
		}()
		tr = fn(state)
	}()
	if err != nil {
		return space.Transition{}, err
	}
	key, err := s.loadKey()
	if err != nil {
		return space.Transition{}, err
	}
	if err := s.commit(key, tr.State, true); err != nil {
		return space.Transition{}, err
	}
	return tr, nil
}

func (s *Store) read() (space.State, error) {
	key, err := s.loadKey()
	if err != nil {
		return space.State{}, err
	}
	f, err := s.root.OpenFile(s.stateName, os.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return space.State{}, err
	}
	defer f.Close()
	if err := checkFile(f); err != nil {
		return space.State{}, err
	}
	b, err := io.ReadAll(io.LimitReader(f, 8<<20+1))
	if err != nil {
		return space.State{}, err
	}
	if len(b) == 0 || len(b) > 8<<20 {
		return space.State{}, errors.New("invalid state envelope size")
	}
	return decodeEnvelope(key, b)
}

func (s *Store) loadKey() ([]byte, error) { return readKey(s.keyRoot, s.keyName) }

func (s *Store) loadOrCreateKey(create bool) ([]byte, error) {
	key, err := s.loadKey()
	if err == nil || !create || !errors.Is(err, os.ErrNotExist) {
		return key, err
	}
	b := make([]byte, keySize)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	f, err := s.keyRoot.OpenFile(s.keyName, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err == nil {
		if _, err = f.Write(b); err == nil {
			err = f.Sync()
		}
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Store) commit(key []byte, state space.State, preserve bool) error {
	if err := validateState(state); err != nil {
		return err
	}
	b, err := encodeEnvelope(key, state)
	if err != nil {
		return err
	}
	tmpName := ".state.tmp-" + randomName()
	backupName := tmpName + ".old"
	recoveryName := backupName + ".recovery"
	backupMade := false
	syncDir := s.syncParent
	err = func() error {
		tmp, e := s.root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
		if e != nil {
			return e
		}
		cleanup := func() { _ = tmp.Close(); _ = s.root.Remove(tmpName) }
		if s.hooks.Write != nil {
			e = s.hooks.Write(tmp, b)
		} else {
			var n int
			n, e = tmp.Write(b)
			if e == nil && n != len(b) {
				e = io.ErrShortWrite
			}
		}
		if e != nil {
			cleanup()
			return e
		}
		if s.hooks.Sync != nil {
			e = s.hooks.Sync(tmp)
		} else {
			e = tmp.Sync()
		}
		if e != nil {
			cleanup()
			return e
		}
		if e = tmp.Close(); e != nil {
			_ = s.root.Remove(tmpName)
			return e
		}
		if preserve {
			if info, statErr := s.root.Lstat(s.stateName); statErr == nil {
				if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
					_ = s.root.Remove(tmpName)
					return errors.New("state file has invalid type or mode")
				}
				if e = s.root.Rename(s.stateName, backupName); e != nil {
					_ = s.root.Remove(tmpName)
					return e
				}
				backupMade = true
			} else if !errors.Is(statErr, os.ErrNotExist) {
				_ = s.root.Remove(tmpName)
				return statErr
			}
		}
		if info, statErr := s.root.Lstat(s.stateName); statErr == nil && (!info.Mode().IsRegular() || info.Mode().Perm() != 0600) {
			_ = s.root.Remove(tmpName)
			return errors.New("state file has invalid type or mode")
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			_ = s.root.Remove(tmpName)
			return statErr
		}
		if s.hooks.Rename != nil {
			e = s.hooks.Rename(filepath.Join(filepath.Dir(s.path), tmpName), s.path)
		} else {
			e = s.root.Rename(tmpName, s.stateName)
		}
		if e != nil {
			_ = s.root.Remove(tmpName)
			return e
		}
		return nil
	}()
	if err != nil {
		if backupMade {
			if rollbackErr := s.rollback(backupName, recoveryName, syncDir); rollbackErr != nil {
				return fmt.Errorf("commit failed: %w; rollback failed: %v", err, rollbackErr)
			}
		}
		return err
	}
	if err = syncDir(); err != nil {
		if rollbackErr := s.rollback(backupName, recoveryName, syncDir); rollbackErr != nil {
			return fmt.Errorf("directory sync failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}
	if backupMade {
		// The replacement is already the durable authority. Cleanup is
		// non-authoritative: retain recovery material if removal fails and still
		// report the committed transition honestly.
		remove := func(name string) error { return s.root.Remove(name) }
		if s.hooks.Remove != nil {
			remove = s.hooks.Remove
		}
		_ = remove(backupName)
	}
	return nil
}

func (s *Store) rollback(backupName, recoveryName string, syncDir func() error) error {
	err := func() error {
		if backupName != "" {
			if _, statErr := s.root.Lstat(backupName); statErr == nil {
				if err := s.root.Link(backupName, recoveryName); err != nil {
					return err
				}
				return s.root.Rename(backupName, s.stateName)
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return statErr
			}
		}
		return s.root.Remove(s.stateName)
	}()
	if err != nil {
		return err
	}
	if err := syncDir(); err != nil {
		return err
	}
	_ = s.root.Remove(recoveryName)
	return nil
}

func (s *Store) syncParent() error {
	f, err := s.root.Open(".")
	if err != nil {
		return err
	}
	defer f.Close()
	if s.hooks.DirSync != nil {
		return s.hooks.DirSync(f)
	}
	return f.Sync()
}

func strictJSON(data []byte, target any) error {
	if len(data) == 0 {
		return io.ErrUnexpectedEOF
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("trailing data")
		}
		return err
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return errors.New("trailing data")
		}
		return err
	}
	return nil
}

func scanJSONValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch delim := tok.(type) {
	case json.Delim:
		switch delim {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("duplicate or invalid object key")
				}
				seen[name] = true
				if err := scanJSONValue(dec); err != nil {
					return err
				}
			}
			_, err = dec.Token()
		case '[':
			for dec.More() {
				if err := scanJSONValue(dec); err != nil {
					return err
				}
			}
			_, err = dec.Token()
		default:
			return errors.New("invalid JSON delimiter")
		}
	}
	return err
}

func validPath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("store paths must be absolute and clean")
	}
	return nil
}

func openParent(path string) (*os.Root, error) {
	if err := rejectSymlinkComponents(path); err != nil {
		return nil, err
	}
	if st, err := os.Lstat(path); err != nil || !st.IsDir() || st.Mode()&0077 != 0 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("store parent must be private directory")
	}
	expected, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	f, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	info, statErr := f.Stat()
	closeErr := f.Close()
	if statErr != nil || closeErr != nil || !os.SameFile(expected, info) {
		root.Close()
		return nil, errors.New("store root changed during open")
	}
	if err := verifyRoot(root, path); err != nil {
		root.Close()
		return nil, err
	}
	return root, nil
}

func checkPrivate(f *os.File) error {
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() && !st.IsDir() {
		return errors.New("store object has invalid type")
	}
	if st.Mode().Perm() != 0600 && st.Mode().Perm()&0077 != 0 {
		return errors.New("store object must not be accessible by group or others")
	}
	if stat, ok := st.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Getuid()) {
		return errors.New("store object owner mismatch")
	}
	return nil
}

func checkFile(f *os.File) error {
	if err := checkPrivate(f); err != nil {
		return err
	}
	if st, err := f.Stat(); err != nil {
		return err
	} else if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return errors.New("store file must be regular mode 0600")
	}
	return nil
}

func readKey(root *os.Root, name string) ([]byte, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := checkFile(f); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(f, keySize+1))
	if err != nil {
		return nil, err
	}
	if len(b) != keySize {
		return nil, fmt.Errorf("state key must be exactly %d bytes", keySize)
	}
	return b, nil
}

func randomName() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func validateState(state space.State) error {
	if state.SchemaVersion != FormatVersion {
		return fmt.Errorf("unsupported state schema version %d", state.SchemaVersion)
	}
	return nil
}

func rootLstat(root *os.Root, name string) (os.FileInfo, error) { return root.Lstat(name) }

func verifyRoot(root *os.Root, path string) error {
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	expected, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, expected) {
		return errors.New("store root changed during open")
	}
	return checkPrivate(f)
}

func rejectSymlinkComponents(path string) error {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		st, err := os.Lstat(current)
		if err == nil && st.Mode()&os.ModeSymlink != 0 && current != "/tmp" && current != "/var" {
			return errors.New("store path must not contain symlinks")
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func randomRead(b []byte) (int, error)               { return rand.Read(b) }
func newCipher(key []byte) (cipher.Block, error)     { return aes.NewCipher(key) }
func newGCM(block cipher.Block) (cipher.AEAD, error) { return cipher.NewGCM(block) }
