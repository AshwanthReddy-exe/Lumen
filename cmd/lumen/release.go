package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

// The immutable setup binding owns a deployment's identity. An explicit owner
// update advances the installed artifact generation through a separate durable
// release record so a rerun can never silently swap artifacts, and so the
// previously installed generation stays available for an offline rollback.

var (
	errReleasePlanUnavailable = errors.New("release plan unavailable")
	errImageUpdateUnsupported = errors.New("docker image update requires deployment tooling")
	errArtifactPathMissing    = errors.New("artifact path unavailable")
)

func releaseRecordPath(dataDir string) string {
	return filepath.Join(dataDir, "setup", "release-record.json")
}

func pendingReleasePath(dataDir string) string {
	return filepath.Join(dataDir, "setup", "release-pending.json")
}

func lockRelease(dataDir string) (*os.File, error) {
	if dataDir == "" || !filepath.IsAbs(dataDir) || filepath.Clean(dataDir) != dataDir {
		return nil, setup.ErrReleaseRecordInvalid
	}
	if _, err := setup.NewJournal(filepath.Join(dataDir, "setup")); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "setup", "release.lock")
	if st, err := os.Lstat(path); err == nil {
		if !privateReleaseLock(st) {
			return nil, setup.ErrReleaseRecordInvalid
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	st, statErr := file.Stat()
	pathSt, pathErr := os.Lstat(path)
	if statErr != nil || pathErr != nil || !privateReleaseLock(st) || !os.SameFile(st, pathSt) {
		_ = file.Close()
		return nil, setup.ErrReleaseRecordInvalid
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func privateReleaseLock(st os.FileInfo) bool {
	owner, ok := st.Sys().(*syscall.Stat_t)
	return ok && st.Mode().IsRegular() && st.Mode().Perm() == 0600 && owner.Uid == uint32(os.Getuid())
}

func unlockRelease(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	_ = file.Close()
}

func updateCommand(ctx context.Context) setup.Report { return releaseChangeCommand(ctx, false) }

func rollbackCommand(ctx context.Context) setup.Report { return releaseChangeCommand(ctx, true) }

func releaseChangeCommand(ctx context.Context, rollback bool) setup.Report {
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	if dataDir == "" {
		return placeholder("configuration_required")
	}
	lock, err := lockRelease(dataDir)
	if err != nil {
		return deploymentReport(err, "", "")
	}
	defer unlockRelease(lock)
	d, err := loadDeploymentLocked(dataDir)
	if err != nil {
		return deploymentReport(err, "", "")
	}
	if d.journal.Next() != setup.Validated {
		return setup.Report{Outcome: setup.ActionRequired, Profile: d.binding.Profile, Topology: d.binding.Topology, Stage: d.journal.Next(), Actions: []setup.Action{{Code: "setup_incomplete"}}}
	}
	current, err := boundRelease(d)
	if err != nil {
		return releaseReport(d, "release_record_unavailable")
	}
	next := current
	change := map[string]setup.Artifact{}
	if rollback {
		if next, err = setup.RollbackRelease(current); err != nil {
			return releaseReport(d, "no_rollback_generation")
		}
	} else {
		target, changed, planErr := manifestRelease(ctx, d, current)
		if planErr != nil {
			if errors.Is(planErr, errImageUpdateUnsupported) {
				return releaseReport(d, "image_update_requires_deployment")
			}
			return releaseReport(d, "artifacts_unavailable")
		}
		if next, err = setup.NextRelease(current, target); err != nil {
			return releaseReport(d, "release_not_advanceable")
		}
		change = changed
	}
	if err := setup.ReleaseMatchesBinding(next, d.binding); err != nil {
		return releaseReport(d, "release_binding_drift")
	}
	storeDir := setup.ReleaseStoreDir(d.config.DataDir)
	// Preserve the live bytes before replacing them so this change can itself
	// be undone without network access.
	if err := setup.RetainExecutableGeneration(storeDir, current); err != nil {
		return releaseReport(d, "release_retention_failed")
	}
	if err := setup.SaveReleaseRecord(pendingReleasePath(d.config.DataDir), current); err != nil {
		return releaseReport(d, "release_record_unavailable")
	}
	if rollback {
		if err := setup.RestoreRetainedGeneration(storeDir, next, 0700); err != nil {
			return failedRelease(d, "rollback_failed")
		}
	} else if err := installRelease(ctx, d, change); err != nil {
		return failedRelease(d, "update_failed")
	}
	if err := setup.VerifyReleaseGeneration(next); err != nil {
		return failedRelease(d, "release_verification_failed")
	}
	if err := setup.SaveReleaseRecord(releaseRecordPath(d.config.DataDir), next); err != nil {
		return failedRelease(d, "release_record_unavailable")
	}
	if err := clearPendingRelease(d.config.DataDir); err != nil {
		return releaseReport(d, "release_record_unavailable")
	}
	return restartDeployment(ctx, d)
}

func failedRelease(d deployment, code string) setup.Report {
	if err := recoverPendingRelease(d.config.DataDir, d.binding, true); err != nil {
		return releaseReport(d, "release_recovery_failed")
	}
	return releaseReport(d, code)
}

// A pending record contains the last committed generation, retained before a
// file swap. Recovery rolls an incomplete swap back, including after a crash.
func recoverPendingRelease(dataDir string, binding setup.JournalBinding, validated bool) error {
	pendingPath := pendingReleasePath(dataDir)
	if _, err := os.Lstat(pendingPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if !validated {
		return setup.ErrReleaseRecordInvalid
	}
	previous, err := setup.LoadReleaseRecord(pendingPath)
	if err != nil {
		return err
	}
	if err := setup.ReleaseMatchesBinding(previous, binding); err != nil {
		return err
	}
	recordPath := releaseRecordPath(dataDir)
	committed := false
	if _, err := os.Lstat(recordPath); err == nil {
		record, err := setup.LoadReleaseRecord(recordPath)
		if err != nil {
			return err
		}
		if err := setup.ReleaseMatchesBinding(record, binding); err != nil {
			return err
		}
		if record.Generation == previous.Generation+1 && reflect.DeepEqual(record.Rollback, previous.Current) {
			if err := setup.VerifyReleaseGeneration(record); err != nil {
				return err
			}
			committed = true
		} else if !reflect.DeepEqual(record, previous) {
			return setup.ErrReleaseRecordInvalid
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !committed {
		if err := setup.RestoreRetainedGeneration(setup.ReleaseStoreDir(dataDir), previous, 0700); err != nil {
			return err
		}
		if err := setup.VerifyReleaseGeneration(previous); err != nil {
			return err
		}
		if err := setup.SaveReleaseRecord(recordPath, previous); err != nil {
			return err
		}
	}
	return clearPendingRelease(dataDir)
}

func clearPendingRelease(dataDir string) error {
	path := pendingReleasePath(dataDir)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	err = f.Sync()
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// boundRelease returns the durable release record, or derives generation one
// from the immutable binding for a deployment that predates release records.
func boundRelease(d deployment) (setup.ReleaseRecord, error) {
	path := releaseRecordPath(d.config.DataDir)
	if _, err := os.Lstat(path); err == nil {
		record, loadErr := setup.LoadReleaseRecord(path)
		if loadErr != nil {
			return setup.ReleaseRecord{}, loadErr
		}
		if matchErr := setup.ReleaseMatchesBinding(record, d.binding); matchErr != nil {
			return setup.ReleaseRecord{}, matchErr
		}
		if verifyErr := setup.VerifyReleaseGeneration(record); verifyErr != nil {
			return setup.ReleaseRecord{}, verifyErr
		}
		return record, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return setup.ReleaseRecord{}, err
	}
	return bindingRelease(d)
}

func bindingRelease(d deployment) (setup.ReleaseRecord, error) {
	artifacts := make(map[string]setup.ReleaseArtifact, 2)
	for _, name := range artifactNames(d.binding.Topology) {
		if ref, ok := d.binding.ArtifactRefs[name]; ok {
			artifacts[name] = setup.ReleaseArtifact{Kind: setup.ArtifactDockerImage, Digest: durableImageDigest(ref), Ref: ref}
			continue
		}
		artifacts[name] = setup.ReleaseArtifact{Kind: setup.ArtifactExecutable, Digest: d.binding.ArtifactDigests[name], Path: d.binding.ArtifactPaths[name]}
	}
	return setup.ReleaseRecord{Generation: 1, Profile: d.binding.Profile, Topology: d.binding.Topology, Current: artifacts}, nil
}

// manifestRelease selects the target generation for the bound topology,
// profile, platform, and architecture. It never consults the environment for
// profile or topology, so an update cannot change a deployment's identity.
func manifestRelease(ctx context.Context, d deployment, current setup.ReleaseRecord) (map[string]setup.ReleaseArtifact, map[string]setup.Artifact, error) {
	probe := cliProbe{dataDir: d.config.DataDir, preferredSupervisor: d.supervisor}
	plan, err := setup.Plan(ctx, setup.Request{Topology: d.binding.Topology, Profile: d.binding.Profile}, probe)
	if err != nil || plan.Outcome != setup.Ready {
		return nil, nil, errReleasePlanUnavailable
	}
	state := &setupState{plan: plan, dataDir: d.config.DataDir, profile: d.binding.Profile, topology: d.binding.Topology, composeProject: d.composeProject}
	selected, err := state.selectedArtifacts()
	if err != nil {
		return nil, nil, err
	}
	identity := make(map[string]setup.ReleaseArtifact, len(selected))
	change := map[string]setup.Artifact{}
	for name, a := range selected {
		ra, err := setup.ReleaseArtifactFor(a)
		if err != nil {
			return nil, nil, err
		}
		if ra.Kind == setup.ArtifactExecutable {
			path := d.binding.ArtifactPaths[name]
			if path == "" || filepath.Base(path) != name {
				return nil, nil, errArtifactPathMissing
			}
			ra.Path = path
		}
		identity[name] = ra
		if existing, ok := current.Current[name]; ok && existing.Digest == ra.Digest && existing.Kind == ra.Kind {
			continue
		}
		if ra.Kind == setup.ArtifactDockerImage {
			// Replacing a container image is a deployment operation, not an
			// in-place file swap; report it honestly instead of faking it.
			return nil, nil, errImageUpdateUnsupported
		}
		change[name] = a
	}
	return identity, change, nil
}

func installRelease(ctx context.Context, d deployment, change map[string]setup.Artifact) error {
	if len(change) == 0 {
		return nil
	}
	installDir := ""
	for name := range change {
		path := d.binding.ArtifactPaths[name]
		if path == "" || filepath.Base(path) != name {
			return errArtifactPathMissing
		}
		if installDir == "" {
			installDir = filepath.Dir(path)
		} else if filepath.Dir(path) != installDir {
			return setup.ErrInvalidArtifact
		}
	}
	installer := setup.Installer{StageDir: filepath.Join(d.config.DataDir, "setup", "stage"), InstallDir: installDir}
	for name, a := range change {
		if local := os.Getenv("LUMEN_" + strings.ToUpper(name) + "_ARTIFACT"); local != "" {
			f, err := os.Open(local)
			if err != nil {
				return err
			}
			installErr := installer.Install(ctx, a, f)
			_ = f.Close()
			if installErr != nil {
				return installErr
			}
			continue
		}
		if err := setup.DownloadAndInstall(ctx, a, setup.Downloader{MaxBytes: setup.MaxArtifactSize}, installer); err != nil {
			return err
		}
	}
	return nil
}

func restartDeployment(ctx context.Context, d deployment) setup.Report {
	states, err := (setup.CommandSupervisor{Manager: d.supervisor, Topology: d.binding.Topology, Compose: setup.ComposeConfig{Project: d.composeProject}}).Control(ctx, setup.Action{Code: "restart"}, d.services)
	if err != nil {
		return releaseReport(d, "service_unavailable")
	}
	r := setup.Report{Outcome: setup.Ready, Profile: d.binding.Profile, Topology: d.binding.Topology, Stage: setup.Validated, States: map[string]string{}}
	for _, s := range states {
		r.States[string(s.Name)] = s.State
		if s.State != setup.StateRunning {
			r.Outcome = setup.ActionRequired
			r.Actions = append(r.Actions, setup.Action{Code: "service_unavailable"})
		}
	}
	return r
}

func releaseReport(d deployment, code string) setup.Report {
	return setup.Report{Outcome: setup.ActionRequired, Profile: d.binding.Profile, Topology: d.binding.Topology, Stage: setup.Validated, Actions: []setup.Action{{Code: code}}}
}
