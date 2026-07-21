package contract

import "time"

// DesktopStateSafeNumber is an unsigned wire integer that remains exact in JavaScript.
type DesktopStateSafeNumber uint64

// DesktopStateMaxSafeNumber is the largest integer exactly representable by JavaScript numbers.
const DesktopStateMaxSafeNumber DesktopStateSafeNumber = 1<<53 - 1

// DesktopStateErrorCode is a stable desktop-state failure identifier shared by every transport.
type DesktopStateErrorCode string

const (
	DesktopStateErrorNotFound      DesktopStateErrorCode = "desktop_state_not_found"
	DesktopStateErrorWorkspace     DesktopStateErrorCode = "workspace_not_found"
	DesktopStateErrorRevConflict   DesktopStateErrorCode = "desktop_state_rev_conflict"
	DesktopStateErrorValueTooLarge DesktopStateErrorCode = "desktop_state_value_too_large"
	DesktopStateErrorKeyQuota      DesktopStateErrorCode = "desktop_state_key_quota_exceeded"
	DesktopStateErrorInvalidKey    DesktopStateErrorCode = "desktop_state_invalid_key"
	DesktopStateErrorInvalidValue  DesktopStateErrorCode = "desktop_state_invalid_value"
	DesktopStateErrorSlowConsumer  DesktopStateErrorCode = "desktop_state_slow_consumer"
)

// DesktopStateErrorCodeValues returns every public desktop-state error code.
func DesktopStateErrorCodeValues() []string {
	return []string{
		string(DesktopStateErrorNotFound),
		string(DesktopStateErrorWorkspace),
		string(DesktopStateErrorRevConflict),
		string(DesktopStateErrorValueTooLarge),
		string(DesktopStateErrorKeyQuota),
		string(DesktopStateErrorInvalidKey),
		string(DesktopStateErrorInvalidValue),
		string(DesktopStateErrorSlowConsumer),
	}
}

// DesktopStateOpKind identifies one mutation in an atomic apply request.
type DesktopStateOpKind string

const (
	DesktopStateOpPut    DesktopStateOpKind = "put"
	DesktopStateOpDelete DesktopStateOpKind = "delete"
)

// DesktopStateOpKindValues returns every public mutation kind.
func DesktopStateOpKindValues() []string {
	return []string{string(DesktopStateOpPut), string(DesktopStateOpDelete)}
}

// DesktopStateEntry is the canonical desktop-state wire envelope.
type DesktopStateEntry struct {
	Key       string                 `json:"key"`
	Value     map[string]any         `json:"value"`
	Rev       DesktopStateSafeNumber `json:"rev"`
	Seq       DesktopStateSafeNumber `json:"seq"`
	Deleted   bool                   `json:"deleted"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// DesktopStateListResponse is a gap-free snapshot fence and its live entries.
type DesktopStateListResponse struct {
	AsOfSeq DesktopStateSafeNumber `json:"as_of_seq"`
	Entries []DesktopStateEntry    `json:"entries"`
}

// DesktopStatePutRequest replaces one value, optionally using compare-and-swap.
type DesktopStatePutRequest struct {
	Value map[string]any          `json:"value"`
	IfRev *DesktopStateSafeNumber `json:"if_rev,omitempty"`
}

// DesktopStateApplyOp is one mutation in an atomic batch.
type DesktopStateApplyOp struct {
	Kind  DesktopStateOpKind      `json:"kind"`
	Key   string                  `json:"key"`
	Value *map[string]any         `json:"value,omitempty"`
	IfRev *DesktopStateSafeNumber `json:"if_rev,omitempty"`
}

// DesktopStateApplyRequest atomically applies all supplied operations.
type DesktopStateApplyRequest struct {
	Ops []DesktopStateApplyOp `json:"ops"`
}

// DesktopStateApplyResponse returns the committed canonical envelopes.
type DesktopStateApplyResponse struct {
	Results []DesktopStateEntry `json:"results"`
}

// DesktopStateErrorPayload is the deterministic error body shared by HTTP, UDS, CLI, and WS.
type DesktopStateErrorPayload struct {
	Error string                `json:"error"`
	Code  DesktopStateErrorCode `json:"code"`
	Key   string                `json:"key,omitempty"`
}

// DesktopStateSubscribeFrame starts one snapshot-plus-delta subscription.
type DesktopStateSubscribeFrame struct {
	Op string `json:"op"`
}

// DesktopStateApplyFrame carries one client mutation and correlation id.
type DesktopStateApplyFrame struct {
	Op  string                `json:"op"`
	Req string                `json:"req"`
	Ops []DesktopStateApplyOp `json:"ops"`
}

// DesktopStatePingFrame requests an application-level pong.
type DesktopStatePingFrame struct {
	Op string `json:"op"`
}

// DesktopStateSnapshotFrame is the first server frame after a subscription.
type DesktopStateSnapshotFrame struct {
	Op      string                 `json:"op"`
	AsOfSeq DesktopStateSafeNumber `json:"as_of_seq"`
	Entries []DesktopStateEntry    `json:"entries"`
}

// DesktopStateEventFrame carries one committed mutation in workspace sequence order.
type DesktopStateEventFrame struct {
	Op     string            `json:"op"`
	Entry  DesktopStateEntry `json:"entry"`
	Origin string            `json:"origin"`
}

// DesktopStateAckResult is the revision/sequence summary for one committed key.
type DesktopStateAckResult struct {
	Key string                 `json:"key"`
	Rev DesktopStateSafeNumber `json:"rev"`
	Seq DesktopStateSafeNumber `json:"seq"`
}

// DesktopStateAckFrame correlates a successful mutation with its client request.
type DesktopStateAckFrame struct {
	Op      string                  `json:"op"`
	Req     string                  `json:"req"`
	Results []DesktopStateAckResult `json:"results"`
}

// DesktopStateErrorFrame carries a stable failure code without closing a healthy connection.
type DesktopStateErrorFrame struct {
	Op   string                `json:"op"`
	Req  string                `json:"req,omitempty"`
	Code DesktopStateErrorCode `json:"code"`
	Key  string                `json:"key,omitempty"`
}

// DesktopStatePongFrame answers an application-level ping.
type DesktopStatePongFrame struct {
	Op string `json:"op"`
}

// DesktopStateWebSocketContract registers all frame schemas in the generated API contract.
type DesktopStateWebSocketContract struct {
	Subscribe DesktopStateSubscribeFrame `json:"subscribe"`
	Apply     DesktopStateApplyFrame     `json:"apply"`
	Ping      DesktopStatePingFrame      `json:"ping"`
	Snapshot  DesktopStateSnapshotFrame  `json:"snapshot"`
	Event     DesktopStateEventFrame     `json:"event"`
	Ack       DesktopStateAckFrame       `json:"ack"`
	Error     DesktopStateErrorFrame     `json:"error"`
	Pong      DesktopStatePongFrame      `json:"pong"`
}
