package conversation

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

// ReconcilePending finishes interrupted conversations from Host-owned evidence.
// It never submits a run, including when Hermes accepted a create before the
// Host could persist its run ID.
func (s *Service) ReconcilePending(ctx context.Context) error {
	state, err := s.state.Read()
	if err != nil {
		return err
	}
	if state.SchemaVersion != space.StateSchemaVersionV2 {
		return nil
	}
	ids := make([]string, 0, len(state.Tasks))
	for id, task := range state.Tasks {
		if task.CapabilityID == "conversation.chat/respond" && (task.Status == space.OutcomeQueued || task.Status == space.OutcomeCreating || task.Status == space.OutcomeDispatched || task.Status == space.OutcomeRunning) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, taskID := range ids {
		state, err = s.state.Read()
		if err != nil {
			return err
		}
		task := state.Tasks[taskID]
		if task.Status == space.OutcomeCompleted || task.Status == space.OutcomeFailed || task.Status == space.OutcomeUnknown {
			continue
		}
		var resultErr error
		run, mapped := state.HostRuns[taskID]
		conversationID := conversationForTask(state, taskID)
		session := state.RuntimeSessions[conversationID]
		profile, profiled := profileForDigest(state, run.RuntimeProfileDigest)
		cert := space.RuntimeCertification{ID: run.CertificationID, RuntimeIdentity: session.RuntimeIdentity, ProfileDigest: run.RuntimeProfileDigest}
		if !mapped || task.Status != space.OutcomeDispatched && task.Status != space.OutcomeRunning || run.TaskID != taskID || run.RuntimeRunID == "" || run.CertificationID == "" || run.EndpointIdentity == "" || run.HostEpoch != state.Epoch || task.HostEpoch != state.Epoch || run.DispatchedAt <= 0 || run.ReconcileBy <= run.DispatchedAt || conversationID == "" || session.Pending || session.RuntimeIdentity == "" || session.ProfileDigest != run.RuntimeProfileDigest || session.HostEpoch != state.Epoch || !profiled || space.RuntimeCertificationInvalidated(state, cert) || s.now().Unix() >= run.ReconcileBy || s.certifier == nil {
			_, resultErr = s.complete(space.Transition{}, state, taskID, space.OutcomeUnknown, "", errors.New("conversation recovery lacks verified run mapping"))
		} else {
			reconcileCtx, cancel := context.WithTimeout(ctx, time.Unix(run.ReconcileBy, 0).Sub(s.now()))
			liveCert, certifyErr := s.certifier.Certify(reconcileCtx, state, profile)
			if certifyErr != nil || reconcileCtx.Err() != nil || !certificationMatches(liveCert, profile, s.now()) || liveCert.ID != run.CertificationID || liveCert.RuntimeIdentity != session.RuntimeIdentity || liveCert.EndpointIdentity != run.EndpointIdentity || !s.endpointMatches(reconcileCtx, liveCert) {
				_, resultErr = s.complete(space.Transition{}, state, taskID, space.OutcomeUnknown, "", errors.New("conversation recovery runtime binding unverified"))
			} else {
				_, resultErr = s.reconcileRun(reconcileCtx, space.Transition{}, state, taskID, hermes.Run{RunID: run.RuntimeRunID}, liveCert, profile, run.ReconcileBy)
			}
			cancel()
		}
		updated, readErr := s.state.Read()
		if readErr != nil {
			return errors.Join(resultErr, readErr)
		}
		status := updated.Tasks[taskID].Status
		if status != space.OutcomeCompleted && status != space.OutcomeFailed && status != space.OutcomeUnknown {
			return errors.Join(resultErr, errors.New("conversation recovery did not persist a terminal outcome"))
		}
		if errors.Is(resultErr, ErrRuntimeProfileUnverified) && !space.RuntimeCertificationInvalidated(updated, cert) && mapped {
			return resultErr
		}
	}
	return nil
}

func conversationForTask(state space.State, taskID string) string {
	for id, messages := range state.Messages {
		for _, message := range messages {
			if message.TaskID == taskID && message.Role == space.MessageUser {
				return id
			}
		}
	}
	return ""
}

func profileForDigest(state space.State, digest string) (space.RuntimeProfile, bool) {
	for _, profile := range state.RuntimeProfiles {
		if profile.Digest == digest {
			return profile, true
		}
	}
	return space.RuntimeProfile{}, false
}
