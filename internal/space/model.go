package space

type Grant string

const (
	GrantDeny  Grant = "deny"
	GrantAsk   Grant = "ask"
	GrantAllow Grant = "allow"
)

type IdentityKind string

const (
	IdentityOwner IdentityKind = "owner"
	IdentityHost  IdentityKind = "host"
)

type CommandType string

const (
	CommandCreateSpace                CommandType = "create_space"
	CommandPairNode                   CommandType = "pair_node"
	CommandAdvertiseCapability        CommandType = "advertise_capability"
	CommandSetGrant                   CommandType = "set_grant"
	CommandSubmit                     CommandType = "submit"
	CommandApprove                    CommandType = "approve"
	CommandComplete                   CommandType = "complete"
	CommandRevokeNode                 CommandType = "revoke_node"
	CommandRecoverAfterRestart        CommandType = "recover_after_restart"
	CommandDispatchHostRun            CommandType = "dispatch_host_run"
	CommandReconcileHostRun           CommandType = "reconcile_host_run"
	CommandRequestHostRunCancellation CommandType = "request_host_run_cancellation"
	CommandCreateHostRun              CommandType = "create_host_run"
	CommandRequestRuntimeApproval     CommandType = "request_runtime_approval"
	CommandResolveRuntimeApproval     CommandType = "resolve_runtime_approval"
	CommandRecordRuntimeApproval      CommandType = "record_runtime_approval"
)

type Outcome string

const (
	OutcomeApplied            Outcome = "applied"
	OutcomeAwaitingPermission Outcome = "awaiting_permission"
	OutcomeQueued             Outcome = "queued"
	OutcomeCompleted          Outcome = "completed"
	OutcomeFailed             Outcome = "failed"
	OutcomeUnknown            Outcome = "unknown_outcome"
	OutcomeDispatched         Outcome = "dispatched"
	OutcomeRunning            Outcome = "running"
	OutcomeCancelling         Outcome = "cancelling"
	OutcomeCancelled          Outcome = "cancelled"
	OutcomeCreating           Outcome = "creating"
)

type EvidenceOutcome string

const (
	EvidenceRunning     EvidenceOutcome = "running"
	EvidenceCompleted   EvidenceOutcome = "completed"
	EvidenceFailed      EvidenceOutcome = "failed"
	EvidenceCancelled   EvidenceOutcome = "cancelled"
	EvidenceUnavailable EvidenceOutcome = "unavailable"
)

type AuditEventType string

const (
	AuditSpaceCreated    AuditEventType = "space.created"
	AuditCommandAccepted AuditEventType = "command.accepted"
	AuditCommandRejected AuditEventType = "command.rejected"
	AuditCommandReplayed AuditEventType = "command.replayed"
)

type State struct {
	SchemaVersion    int                        `json:"schemaVersion"`
	SpaceID          string                     `json:"spaceId,omitempty"`
	OwnerID          string                     `json:"ownerId,omitempty"`
	HostID           string                     `json:"hostId,omitempty"`
	Epoch            int                        `json:"epoch,omitempty"`
	Identities       []Identity                 `json:"identities,omitempty"`
	Capabilities     []Capability               `json:"capabilities,omitempty"`
	Audit            []AuditEvent               `json:"audit"`
	Nodes            map[string]Node            `json:"nodes,omitempty"`
	Advertisements   []CapabilityKey            `json:"advertisements,omitempty"`
	Grants           map[string]Grant           `json:"grants,omitempty"`
	Tasks            map[string]Task            `json:"tasks,omitempty"`
	Approvals        map[string]Approval        `json:"approvals,omitempty"`
	HostCreates      map[string]HostCreate      `json:"hostCreates,omitempty"`
	RuntimeApprovals map[string]RuntimeApproval `json:"runtimeApprovals,omitempty"`
	Commands         map[string]RecordedCommand `json:"commands,omitempty"`
	HostRuns         map[string]HostRun         `json:"hostRuns,omitempty"`
}
type Identity struct {
	ID   string       `json:"id"`
	Kind IdentityKind `json:"kind"`
}
type Capability struct {
	ID    string `json:"id"`
	Grant Grant  `json:"grant"`
}
type Node struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
type CapabilityKey struct {
	NodeID       string `json:"nodeId"`
	CapabilityID string `json:"capabilityId"`
	Action       string `json:"action"`
}
type Task struct {
	ID                string  `json:"id"`
	CommandID         string  `json:"commandId"`
	OriginNodeID      string  `json:"originNodeId"`
	TargetNodeID      string  `json:"targetNodeId"`
	CapabilityID      string  `json:"capabilityId"`
	Action            string  `json:"action"`
	ActionFingerprint string  `json:"actionFingerprint"`
	HostEpoch         int     `json:"hostEpoch"`
	Status            Outcome `json:"status"`
	TerminalReason    string  `json:"terminalReason,omitempty"`
	Output            string  `json:"output,omitempty"`
	OutputTruncated   bool    `json:"outputTruncated,omitempty"`
}
type HostRun struct {
	TaskID               string `json:"taskId"`
	RuntimeRunID         string `json:"runtimeRunId"`
	RuntimeProfileDigest string `json:"runtimeProfileDigest"`
	HostEpoch            int    `json:"hostEpoch"`
	DispatchedAt         int64  `json:"dispatchedAt"`
	ReconcileBy          int64  `json:"reconcileBy"`
}
type HostCreate struct {
	TaskID               string `json:"taskId"`
	IdempotencyKey       string `json:"idempotencyKey"`
	RuntimeProfileDigest string `json:"runtimeProfileDigest"`
	HostEpoch            int    `json:"hostEpoch"`
	CreatedAt            int64  `json:"createdAt"`
	ReconcileBy          int64  `json:"reconcileBy"`
}
type RuntimeApproval struct {
	ID                string `json:"id"`
	TaskID            string `json:"taskId"`
	RuntimeRunID      string `json:"runtimeRunId"`
	ActorNodeID       string `json:"actorNodeId"`
	TargetNodeID      string `json:"targetNodeId"`
	ActionFingerprint string `json:"actionFingerprint"`
	ExpiresAt         int64  `json:"expiresAt"`
	Decision          string `json:"decision,omitempty"`
	DeliveryState     string `json:"deliveryState,omitempty"`
	DeliveryAttempt   int    `json:"deliveryAttempt,omitempty"`
}
type Approval struct {
	ID                string `json:"id"`
	TaskID            string `json:"taskId"`
	ActorNodeID       string `json:"actorNodeId"`
	TargetNodeID      string `json:"targetNodeId"`
	ActionFingerprint string `json:"actionFingerprint"`
	ExpiresAt         int64  `json:"expiresAt"`
	ConsumedAt        int64  `json:"consumedAt"`
}
type RecordedCommand struct {
	Type      CommandType `json:"type"`
	Content   string      `json:"content"`
	Receipt   *Receipt    `json:"receipt,omitempty"`
	Rejection string      `json:"rejection,omitempty"`
}
type AuditEvent struct {
	Event     AuditEventType `json:"event"`
	RequestID string         `json:"requestId"`
	ActorID   string         `json:"actorId"`
	HostID    string         `json:"hostId"`
	Epoch     int            `json:"epoch"`
	Operation CommandType    `json:"operation"`
	Outcome   string         `json:"outcome"`
}
type Command struct {
	Type                  CommandType     `json:"type"`
	SpaceID               string          `json:"spaceId"`
	OwnerID               string          `json:"ownerId"`
	HostID                string          `json:"hostId"`
	RequestID             string          `json:"requestId"`
	Epoch                 int             `json:"epoch,omitempty"`
	OperationID           string          `json:"operationId,omitempty"`
	ActorID               string          `json:"actorId,omitempty"`
	NodeID                string          `json:"nodeId,omitempty"`
	CapabilityID          string          `json:"capabilityId,omitempty"`
	Action                string          `json:"action,omitempty"`
	ActionFingerprint     string          `json:"actionFingerprint,omitempty"`
	TaskID                string          `json:"taskId,omitempty"`
	OriginNodeID          string          `json:"originNodeId,omitempty"`
	TargetNodeID          string          `json:"targetNodeId,omitempty"`
	Grant                 Grant           `json:"grant,omitempty"`
	ApprovalID            string          `json:"approvalId,omitempty"`
	ExpiresAt             int64           `json:"expiresAt,omitempty"`
	ApprovedAt            int64           `json:"approvedAt,omitempty"`
	Outcome               Outcome         `json:"outcome,omitempty"`
	RuntimeRunID          string          `json:"runtimeRunId,omitempty"`
	RuntimeIdempotencyKey string          `json:"runtimeIdempotencyKey,omitempty"`
	RuntimeProfileDigest  string          `json:"runtimeProfileDigest,omitempty"`
	DispatchedAt          int64           `json:"dispatchedAt,omitempty"`
	ReconcileBy           int64           `json:"reconcileBy,omitempty"`
	ObservedAt            int64           `json:"observedAt,omitempty"`
	Evidence              EvidenceOutcome `json:"evidence,omitempty"`
	RuntimeApprovalID     string          `json:"runtimeApprovalId,omitempty"`
	Decision              string          `json:"decision,omitempty"`
	DeliveryState         string          `json:"deliveryState,omitempty"`
	DeliveryAttempt       int             `json:"deliveryAttempt,omitempty"`
	Output                string          `json:"output,omitempty"`
	OutputTruncated       bool            `json:"outputTruncated,omitempty"`
}

const MaxTaskOutputBytes = 8 << 10

type Transition struct {
	State     State
	Receipt   Receipt
	Rejection string
	Replayed  bool
}
type Receipt struct {
	RequestID string  `json:"requestId,omitempty"`
	Outcome   Outcome `json:"outcome,omitempty"`
	SubjectID string  `json:"subjectId,omitempty"`
}
type FixtureSuite struct {
	SchemaVersion int           `json:"schemaVersion"`
	Cases         []FixtureCase `json:"cases"`
}
type FixtureCase struct {
	Name     string          `json:"name"`
	Initial  State           `json:"initial"`
	Commands []Command       `json:"commands"`
	Expected FixtureExpected `json:"expected"`
}
type FixtureExpected struct {
	State       State                `json:"state"`
	Audit       []AuditEvent         `json:"audit"`
	Transitions []ExpectedTransition `json:"transitions"`
}
type ExpectedTransition struct {
	Receipt   Receipt `json:"receipt"`
	Rejection string  `json:"rejection"`
}

func commandID(c Command) string {
	if c.RequestID != "" {
		return c.RequestID
	}
	return c.OperationID
}
