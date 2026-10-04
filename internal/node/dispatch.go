package node

import (
	"context"
	"errors"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

type StatusStore interface {
	Read() (space.State, error)
	Update(func(space.State) space.Transition) (space.Transition, error)
}

// DispatchStatus is a read-only cross-node journey: Host policy is checked
// before network I/O and again before the verified receipt becomes canonical.
func DispatchStatus(ctx context.Context, store StatusStore, client StatusClient, taskID string, now time.Time) (space.Transition, error) {
	state, err := store.Read()
	if err != nil {
		return space.Transition{}, err
	}
	request, err := authorizedStatus(state, taskID, now)
	if err != nil {
		return space.Transition{}, err
	}
	if _, err := client.Read(ctx, request); err != nil {
		return space.Transition{}, err
	}
	var rejected bool
	transition, err := store.Update(func(current space.State) space.Transition {
		currentRequest, authErr := authorizedStatus(current, taskID, now)
		if authErr != nil || currentRequest != request {
			rejected = true
			return space.Transition{State: current, Rejection: "node_authority_changed"}
		}
		return space.Apply(current, space.Command{Type: space.CommandComplete, SpaceID: current.SpaceID, HostID: current.HostID, Epoch: current.Epoch, ActorID: request.NodeID, TaskID: taskID, Outcome: space.OutcomeCompleted, RequestID: "node-status:" + taskID})
	})
	if err != nil {
		return transition, err
	}
	if rejected || transition.Rejection != "" {
		return transition, errors.New("node completion rejected")
	}
	return transition, nil
}

func authorizedStatus(state space.State, taskID string, now time.Time) (StatusRequest, error) {
	task, ok := state.Tasks[taskID]
	if !ok || task.ID != taskID || task.HostEpoch != state.Epoch || task.Status != space.OutcomeQueued || task.CapabilityID != "node.status/read" || task.Action != "read" || task.ActionFingerprint == "" || state.SpaceID == "" || state.HostID == "" || state.Epoch <= 0 || now.Unix() <= 0 {
		return StatusRequest{}, errors.New("node task unavailable")
	}
	target, targetOK := state.Nodes[task.TargetNodeID]
	origin, originOK := state.Nodes[task.OriginNodeID]
	if !targetOK || target.ID != task.TargetNodeID || target.Status != "paired" || !originOK || origin.ID != task.OriginNodeID || origin.Status != "paired" || state.Grants[task.TargetNodeID+"|node.status/read|read"] != space.GrantAllow {
		return StatusRequest{}, errors.New("node grant unavailable")
	}
	advertised := false
	for _, capability := range state.Advertisements {
		if capability.NodeID == task.TargetNodeID && capability.CapabilityID == "node.status/read" && capability.Action == "read" {
			advertised = true
			break
		}
	}
	if !advertised {
		return StatusRequest{}, errors.New("node capability unavailable")
	}
	return StatusRequest{Version: 1, SpaceID: state.SpaceID, HostID: state.HostID, HostEpoch: state.Epoch, NodeID: task.TargetNodeID, TaskID: taskID, CapabilityID: task.CapabilityID, Action: task.Action, ActionFingerprint: task.ActionFingerprint, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}, nil
}
