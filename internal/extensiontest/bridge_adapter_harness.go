package extensiontest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	bridgepkg "github.com/compozy/agh/internal/bridges"
	aghconfig "github.com/compozy/agh/internal/config"
	extensionpkg "github.com/compozy/agh/internal/extension"
	extensioncontract "github.com/compozy/agh/internal/extension/contract"
	extensionprotocol "github.com/compozy/agh/internal/extension/protocol"
	observepkg "github.com/compozy/agh/internal/observe"
	sandboxlocal "github.com/compozy/agh/internal/sandbox/local"
	"github.com/compozy/agh/internal/session"
	skillspkg "github.com/compozy/agh/internal/skills"
	"github.com/compozy/agh/internal/store/globaldb"
	"github.com/compozy/agh/internal/store/sessiondb"
	"github.com/compozy/agh/internal/subprocess"
	aghtestutil "github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

const (
	bridgeAdapterHarnessBrgTelegramReferenceValue = "brg-telegram-reference"
	bridgeAdapterHarnessCoderKey                  = "coder"
	bridgeAdapterHarnessHelloKey                  = "hello"
)

const (
	EnvHandshakePath = "AGH_BRIDGE_ADAPTER_HANDSHAKE_PATH"
	EnvOwnershipPath = "AGH_BRIDGE_ADAPTER_OWNERSHIP_PATH"
	EnvStatePath     = "AGH_BRIDGE_ADAPTER_STATE_PATH"
	EnvDeliveryPath  = "AGH_BRIDGE_ADAPTER_DELIVERY_PATH"
	EnvIngestPath    = "AGH_BRIDGE_ADAPTER_INGEST_PATH"
	EnvUpdatesPath   = "AGH_BRIDGE_ADAPTER_UPDATES_PATH"
	EnvStartsPath    = "AGH_BRIDGE_ADAPTER_STARTS_PATH"
	EnvShutdownPath  = "AGH_BRIDGE_ADAPTER_SHUTDOWN_PATH"
	EnvCrashOncePath = "AGH_BRIDGE_ADAPTER_CRASH_ONCE_PATH"
)

// MarkerPaths contains the standardized marker and control files used by the
// reference adapter and the conformance harness.
type MarkerPaths struct {
	Handshake string
	Ownership string
	State     string
	Delivery  string
	Ingest    string
	Updates   string
	Starts    string
	Shutdown  string
	CrashOnce string
}

// Env returns the environment expected by the reference adapter marker contract.
func (m MarkerPaths) Env() map[string]string {
	return map[string]string{
		EnvHandshakePath: m.Handshake,
		EnvOwnershipPath: m.Ownership,
		EnvStatePath:     m.State,
		EnvDeliveryPath:  m.Delivery,
		EnvIngestPath:    m.Ingest,
		EnvUpdatesPath:   m.Updates,
		EnvStartsPath:    m.Starts,
		EnvShutdownPath:  m.Shutdown,
		EnvCrashOncePath: m.CrashOnce,
	}
}

// NewMarkerPaths returns a temp-root-relative marker layout used by the
// subprocess-backed adapter tests.
func NewMarkerPaths(root string) MarkerPaths {
	return MarkerPaths{
		Handshake: filepath.Join(root, "adapter-handshake.json"),
		Ownership: filepath.Join(root, "adapter-ownership.json"),
		State:     filepath.Join(root, "adapter-states.jsonl"),
		Delivery:  filepath.Join(root, "adapter-deliveries.jsonl"),
		Ingest:    filepath.Join(root, "adapter-ingest.jsonl"),
		Updates:   filepath.Join(root, "adapter-updates.jsonl"),
		Starts:    filepath.Join(root, "adapter-starts.log"),
		Shutdown:  filepath.Join(root, "adapter-shutdown.log"),
		CrashOnce: filepath.Join(root, "adapter-crash-once.json"),
	}
}

// HandshakeRecord captures the adapter initialize marker.
type HandshakeRecord struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
}

// OwnershipRecord captures the provider-owned instance list/get evidence
// collected by the reference adapter during boot.
type OwnershipRecord struct {
	Listed  []bridgepkg.BridgeInstance `json:"listed,omitempty"`
	Fetched []bridgepkg.BridgeInstance `json:"fetched,omitempty"`
	Error   string                     `json:"error,omitempty"`
}

// DeliveryRecord captures one `bridges/deliver` request plus the adapter ack
// when the subprocess remained alive long enough to return it.
type DeliveryRecord struct {
	PID     int                       `json:"pid"`
	Request bridgepkg.DeliveryRequest `json:"request"`
	Ack     *bridgepkg.DeliveryAck    `json:"ack,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

// StateRecord captures one adapter-driven `bridges/instances/report_state`
// result marker.
type StateRecord struct {
	BridgeInstanceID string                   `json:"bridge_instance_id,omitempty"`
	Status           bridgepkg.BridgeStatus   `json:"status"`
	Instance         bridgepkg.BridgeInstance `json:"instance"`
	Error            string                   `json:"error,omitempty"`
}

// IngestRecord captures one fake inbound update mapped into a normalized ingest.
type IngestRecord struct {
	Envelope bridgepkg.InboundMessageEnvelope              `json:"envelope"`
	Result   extensioncontract.BridgesMessagesIngestResult `json:"result"`
	Error    string                                        `json:"error,omitempty"`
}

// ConformanceReport is the collected adapter evidence used by the validator.
type ConformanceReport struct {
	Handshake  *HandshakeRecord
	Ownership  *OwnershipRecord
	States     []StateRecord
	Deliveries []DeliveryRecord
	Ingests    []IngestRecord
}

// ManagedInstanceExpectation describes one provider-owned bridge instance that
// must appear in the negotiated runtime and conformance evidence.
type ManagedInstanceExpectation struct {
	InstanceID          string
	ExtensionName       string
	BoundSecretNames    []string
	ExpectedFinalStatus bridgepkg.BridgeStatus
}

// ConformanceExpectation configures the reusable adapter validator.
type ConformanceExpectation struct {
	Provider                  string
	Platform                  string
	ManagedInstances          []ManagedInstanceExpectation
	RequireOwnedInstanceList  bool
	RequireOwnedInstanceFetch bool
	RequireStateReport        bool
	RequireDelivery           bool
	RequireResume             bool
}

// ConformanceIssue reports one adapter contract failure.
type ConformanceIssue struct {
	Code    string
	Message string
}

// ConformanceError aggregates reusable harness failures.
type ConformanceError struct {
	Issues []ConformanceIssue
}

func (e *ConformanceError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Code, issue.Message))
	}
	return strings.Join(parts, "; ")
}

// ValidateConformance checks the adapter evidence against the reusable bridge
// adapter contract enforced by the harness.
func ValidateConformance(report ConformanceReport, expect ConformanceExpectation) error {
	expectedByID := make(map[string]ManagedInstanceExpectation, len(expect.ManagedInstances))
	for _, managed := range expect.ManagedInstances {
		expectedByID[strings.TrimSpace(managed.InstanceID)] = managed
	}

	issues := make([]ConformanceIssue, 0)
	issues = append(issues, validateHandshakeConformance(report.Handshake, expect, expectedByID)...)
	issues = append(issues, validateOwnershipConformance(report.Ownership, expect, expectedByID)...)
	issues = append(issues, validateStateConformance(report.States, expect, expectedByID)...)
	issues = append(issues, validateDeliveryConformance(report.Deliveries, expect, expectedByID)...)
	issues = append(issues, validateIngestConformance(report.Ingests, expectedByID)...)
	if len(issues) > 0 {
		return &ConformanceError{Issues: issues}
	}
	return nil
}

func validateHandshakeConformance(
	handshake *HandshakeRecord,
	expect ConformanceExpectation,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	if handshake == nil {
		return []ConformanceIssue{{
			Code:    "missing_handshake",
			Message: "adapter did not write an initialize marker",
		}}
	}

	issues := make([]ConformanceIssue, 0)
	request := handshake.Request
	if !slices.Contains(request.Methods.ExtensionServices, "bridges/deliver") {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_delivery_negotiation",
			Message: "initialize did not negotiate bridges/deliver",
		})
	}
	if request.Runtime.Bridge == nil {
		return append(issues, ConformanceIssue{
			Code:    "missing_bridge_runtime",
			Message: "initialize runtime.bridge was nil",
		})
	}

	runtime := request.Runtime.Bridge
	issues = append(issues, validateHandshakeRuntime(runtime, expect, expectedByID)...)
	issues = append(issues, validateGrantedActions(request.Capabilities.GrantedActions)...)
	return issues
}

func validateHandshakeRuntime(
	runtime *subprocess.InitializeBridgeRuntime,
	expect ConformanceExpectation,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if got, want := strings.TrimSpace(runtime.RuntimeVersion), subprocess.InitializeBridgeRuntimeVersion1; got != want {
		issues = append(issues, ConformanceIssue{
			Code:    "wrong_runtime_version",
			Message: fmt.Sprintf("initialize runtime bridge version = %q, want %q", got, want),
		})
	}
	if expect.Provider != "" && strings.TrimSpace(runtime.Provider) != strings.TrimSpace(expect.Provider) {
		issues = append(issues, ConformanceIssue{
			Code: "wrong_provider",
			Message: fmt.Sprintf(
				"initialize runtime bridge provider = %q, want %q",
				runtime.Provider,
				expect.Provider,
			),
		})
	}
	if expect.Platform != "" && strings.TrimSpace(runtime.Platform) != strings.TrimSpace(expect.Platform) {
		issues = append(issues, ConformanceIssue{
			Code: "wrong_platform",
			Message: fmt.Sprintf(
				"initialize runtime bridge platform = %q, want %q",
				runtime.Platform,
				expect.Platform,
			),
		})
	}
	issues = append(issues, validateExpectedManagedInstances(runtime, expect.ManagedInstances)...)
	issues = append(issues, validateUnexpectedManagedInstances(runtime, expectedByID)...)
	return issues
}

func validateExpectedManagedInstances(
	runtime *subprocess.InitializeBridgeRuntime,
	expected []ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	for _, managedExpect := range expected {
		managed, ok := runtime.ManagedInstance(managedExpect.InstanceID)
		if !ok {
			issues = append(issues, ConformanceIssue{
				Code: "missing_managed_instance",
				Message: fmt.Sprintf(
					"initialize runtime bridge did not include managed instance %q",
					managedExpect.InstanceID,
				),
			})
			continue
		}
		if managedExpect.ExtensionName != "" &&
			strings.TrimSpace(managed.Instance.ExtensionName) != strings.TrimSpace(managedExpect.ExtensionName) {
			issues = append(issues, ConformanceIssue{
				Code: "wrong_extension",
				Message: fmt.Sprintf(
					"initialize runtime bridge instance %q extension = %q, want %q",
					managedExpect.InstanceID,
					managed.Instance.ExtensionName,
					managedExpect.ExtensionName,
				),
			})
		}
		issues = append(issues, validateManagedBoundSecrets(managed, managedExpect)...)
	}
	return issues
}

func validateManagedBoundSecrets(
	managed *subprocess.InitializeBridgeManagedInstance,
	expect ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if managed == nil {
		return append(issues, ConformanceIssue{
			Code:    "missing_managed_instance",
			Message: fmt.Sprintf("initialize runtime omitted managed instance %q", expect.InstanceID),
		})
	}
	bound := make(map[string]struct{}, len(managed.BoundSecrets))
	for _, secret := range managed.BoundSecrets {
		bound[strings.TrimSpace(secret.BindingName)] = struct{}{}
	}
	for _, bindingName := range expect.BoundSecretNames {
		if _, ok := bound[strings.TrimSpace(bindingName)]; !ok {
			issues = append(issues, ConformanceIssue{
				Code: "missing_bound_secret",
				Message: fmt.Sprintf(
					"initialize runtime did not include bound secret %q for managed instance %q",
					bindingName,
					expect.InstanceID,
				),
			})
		}
	}
	return issues
}

func validateUnexpectedManagedInstances(
	runtime *subprocess.InitializeBridgeRuntime,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	if len(expectedByID) == 0 {
		return nil
	}
	issues := make([]ConformanceIssue, 0)
	for _, managed := range runtime.ManagedInstances {
		if _, ok := expectedByID[strings.TrimSpace(managed.Instance.ID)]; ok {
			continue
		}
		issues = append(issues, ConformanceIssue{
			Code: "unexpected_managed_instance",
			Message: fmt.Sprintf(
				"initialize runtime included unexpected managed instance %q",
				managed.Instance.ID,
			),
		})
	}
	return issues
}

func validateGrantedActions(actions []extensionprotocol.HostAPIMethod) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	for _, method := range actions {
		text := strings.ToLower(strings.TrimSpace(string(method)))
		if strings.Contains(text, "vault/") || strings.Contains(text, "secret/") {
			issues = append(issues, ConformanceIssue{
				Code:    "leaked_secret_surface",
				Message: fmt.Sprintf("initialize granted unexpected secret surface %q", method),
			})
		}
	}
	return issues
}

func validateOwnershipConformance(
	ownership *OwnershipRecord,
	expect ConformanceExpectation,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if expect.RequireOwnedInstanceList || expect.RequireOwnedInstanceFetch {
		if ownership == nil {
			return []ConformanceIssue{{
				Code:    "missing_ownership_marker",
				Message: "adapter did not write provider ownership markers",
			}}
		}
		if strings.TrimSpace(ownership.Error) != "" {
			issues = append(issues, ConformanceIssue{
				Code:    "owned_instance_lookup_error",
				Message: ownership.Error,
			})
		}
	}
	if expect.RequireOwnedInstanceList && ownership != nil {
		issues = append(issues, validateOwnedInstanceList(ownership.Listed, expectedByID)...)
	}
	if expect.RequireOwnedInstanceFetch && ownership != nil {
		issues = append(issues, validateOwnedInstanceFetch(ownership.Fetched, expectedByID)...)
	}
	return issues
}

func validateOwnedInstanceList(
	listedInstances []bridgepkg.BridgeInstance,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if len(listedInstances) == 0 {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_owned_instance_list",
			Message: "adapter did not list its owned bridge instances",
		})
	}

	listed := make(map[string]struct{}, len(listedInstances))
	for _, instance := range listedInstances {
		instanceID := strings.TrimSpace(instance.ID)
		listed[instanceID] = struct{}{}
		if len(expectedByID) > 0 {
			if _, ok := expectedByID[instanceID]; !ok {
				issues = append(issues, ConformanceIssue{
					Code:    "unexpected_owned_instance",
					Message: fmt.Sprintf("ownership list included unexpected instance %q", instanceID),
				})
			}
		}
	}
	for instanceID := range expectedByID {
		if _, ok := listed[instanceID]; !ok {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_owned_instance",
				Message: fmt.Sprintf("ownership list omitted expected instance %q", instanceID),
			})
		}
	}
	return issues
}

func validateOwnedInstanceFetch(
	fetchedInstances []bridgepkg.BridgeInstance,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if len(fetchedInstances) == 0 {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_owned_instance_fetch",
			Message: "adapter did not fetch owned bridge instances explicitly",
		})
	}

	fetched := make(map[string]struct{}, len(fetchedInstances))
	for _, instance := range fetchedInstances {
		fetched[strings.TrimSpace(instance.ID)] = struct{}{}
	}
	for instanceID := range expectedByID {
		if _, ok := fetched[instanceID]; !ok {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_owned_instance_fetch",
				Message: fmt.Sprintf("adapter did not fetch owned instance %q explicitly", instanceID),
			})
		}
	}
	return issues
}

func validateStateConformance(
	states []StateRecord,
	expect ConformanceExpectation,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if expect.RequireStateReport && len(states) == 0 {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_state_report",
			Message: "adapter did not report bridge state",
		})
	}

	lastStateByID := make(map[string]StateRecord)
	for _, record := range states {
		instanceID := strings.TrimSpace(record.BridgeInstanceID)
		if instanceID == "" {
			instanceID = strings.TrimSpace(record.Instance.ID)
		}
		if instanceID == "" {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_state_instance_id",
				Message: "adapter reported bridge state without bridge_instance_id",
			})
			continue
		}
		if len(expectedByID) > 0 {
			if _, ok := expectedByID[instanceID]; !ok {
				issues = append(issues, ConformanceIssue{
					Code:    "unexpected_state_instance",
					Message: fmt.Sprintf("state report targeted unexpected instance %q", instanceID),
				})
			}
		}
		record.BridgeInstanceID = instanceID
		lastStateByID[instanceID] = record
	}

	for _, managedExpect := range expect.ManagedInstances {
		if !expect.RequireStateReport && managedExpect.ExpectedFinalStatus == "" {
			continue
		}
		last, ok := lastStateByID[strings.TrimSpace(managedExpect.InstanceID)]
		if !ok {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_managed_state",
				Message: fmt.Sprintf("adapter did not report state for managed instance %q", managedExpect.InstanceID),
			})
			continue
		}
		if managedExpect.ExpectedFinalStatus != "" &&
			last.Status.Normalize() != managedExpect.ExpectedFinalStatus.Normalize() {
			issues = append(issues, ConformanceIssue{
				Code: "wrong_final_status",
				Message: fmt.Sprintf(
					"managed instance %q last reported status = %q, want %q",
					managedExpect.InstanceID,
					last.Status,
					managedExpect.ExpectedFinalStatus,
				),
			})
		}
	}
	return issues
}

func validateDeliveryConformance(
	deliveries []DeliveryRecord,
	expect ConformanceExpectation,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	if expect.RequireDelivery && len(deliveries) == 0 {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_delivery",
			Message: "adapter did not receive any delivery requests",
		})
	}

	lastSeq := make(map[string]int64)
	pendingAckResume := make(map[deliveryAckKey]struct{})
	sawResume := false
	for _, record := range deliveries {
		deliveryIssues, resumed := validateDeliveryRecord(record, expectedByID, lastSeq, pendingAckResume)
		issues = append(issues, deliveryIssues...)
		sawResume = sawResume || resumed
	}

	for key := range pendingAckResume {
		issues = append(issues, ConformanceIssue{
			Code: "missing_ack",
			Message: fmt.Sprintf(
				"delivery %q sequence %d did not return an ack or later resume",
				key.deliveryID,
				key.seq,
			),
		})
	}
	if expect.RequireResume && !sawResume {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_resume",
			Message: "adapter did not observe a resume delivery after restart",
		})
	}
	return issues
}

func validateDeliveryRecord(
	record DeliveryRecord,
	expectedByID map[string]ManagedInstanceExpectation,
	lastSeq map[string]int64,
	pendingAckResume map[deliveryAckKey]struct{},
) ([]ConformanceIssue, bool) {
	issues := make([]ConformanceIssue, 0)
	event := record.Request.Event
	deliveryID := strings.TrimSpace(event.DeliveryID)
	instanceID := strings.TrimSpace(event.BridgeInstanceID)
	eventType := normalizeEventType(event.EventType)
	sawResume := false

	if err := record.Request.Validate(); err != nil {
		issues = append(issues, ConformanceIssue{
			Code:    "invalid_delivery_request",
			Message: fmt.Sprintf("delivery %q failed request validation: %v", deliveryID, err),
		})
	}

	if len(expectedByID) > 0 {
		if _, ok := expectedByID[instanceID]; !ok {
			issues = append(issues, ConformanceIssue{
				Code:    "unexpected_delivery_instance",
				Message: fmt.Sprintf("delivery %q targeted unexpected instance %q", deliveryID, instanceID),
			})
		}
	}
	if targetID := strings.TrimSpace(event.DeliveryTarget.BridgeInstanceID); targetID != "" && targetID != instanceID {
		issues = append(issues, ConformanceIssue{
			Code: "mismatched_delivery_target",
			Message: fmt.Sprintf(
				"delivery %q event instance %q did not match target instance %q",
				deliveryID,
				instanceID,
				targetID,
			),
		})
	}
	if eventType == bridgepkg.DeliveryEventTypeResume {
		sawResume = true
		if record.Request.Snapshot == nil {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_resume_snapshot",
				Message: fmt.Sprintf("resume delivery %q omitted its snapshot", deliveryID),
			})
		}
		clearPendingDeliveryAcks(pendingAckResume, deliveryID)
	}
	if eventType != bridgepkg.DeliveryEventTypeResume {
		if previous, ok := lastSeq[deliveryID]; ok && event.Seq <= previous {
			issues = append(issues, ConformanceIssue{
				Code:    "out_of_order_delivery",
				Message: fmt.Sprintf("delivery %q sequence %d arrived after %d", deliveryID, event.Seq, previous),
			})
		}
		lastSeq[deliveryID] = event.Seq
	}

	ackIssues := validateDeliveryAck(record, deliveryID, eventType, pendingAckResume)
	issues = append(issues, ackIssues...)
	return issues, sawResume
}

func validateDeliveryAck(
	record DeliveryRecord,
	deliveryID string,
	eventType string,
	pendingAckResume map[deliveryAckKey]struct{},
) []ConformanceIssue {
	event := record.Request.Event
	ackKey := deliveryAckKey{deliveryID: deliveryID, seq: event.Seq}
	if record.Ack == nil {
		pendingAckResume[ackKey] = struct{}{}
		return nil
	}

	issues := make([]ConformanceIssue, 0)
	if err := record.Ack.ValidateFor(event); err != nil {
		issues = append(issues, ConformanceIssue{
			Code:    "invalid_ack",
			Message: err.Error(),
		})
	}
	delete(pendingAckResume, ackKey)
	if event.Seq > 0 && strings.TrimSpace(record.Ack.RemoteMessageID) == "" {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_remote_message_id",
			Message: fmt.Sprintf("delivery %q sequence %d did not return remote_message_id", deliveryID, event.Seq),
		})
	}
	if event.Seq > 1 && eventType != bridgepkg.DeliveryEventTypeResume &&
		strings.TrimSpace(record.Ack.ReplaceRemoteMessageID) == "" {
		issues = append(issues, ConformanceIssue{
			Code: "missing_replace_remote_message_id",
			Message: fmt.Sprintf(
				"delivery %q sequence %d did not return replace_remote_message_id",
				deliveryID,
				event.Seq,
			),
		})
	}
	return issues
}

type deliveryAckKey struct {
	deliveryID string
	seq        int64
}

func clearPendingDeliveryAcks(pending map[deliveryAckKey]struct{}, deliveryID string) {
	for key := range pending {
		if key.deliveryID == deliveryID {
			delete(pending, key)
		}
	}
}

func validateIngestConformance(
	ingests []IngestRecord,
	expectedByID map[string]ManagedInstanceExpectation,
) []ConformanceIssue {
	issues := make([]ConformanceIssue, 0)
	for _, record := range ingests {
		instanceID := strings.TrimSpace(record.Envelope.BridgeInstanceID)
		if instanceID == "" {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_ingest_instance_id",
				Message: "adapter ingested an inbound message without bridge_instance_id",
			})
			continue
		}
		if len(expectedByID) > 0 {
			if _, ok := expectedByID[instanceID]; !ok {
				issues = append(issues, ConformanceIssue{
					Code:    "unexpected_ingest_instance",
					Message: fmt.Sprintf("ingest targeted unexpected instance %q", instanceID),
				})
			}
		}
	}
	return issues
}

// ScriptedPromptEvent is one deterministic agent event emitted by the harness
// driver when the host creates a session prompt.
type ScriptedPromptEvent struct {
	Type  string
	Text  string
	Error string
	Delay time.Duration
}

// ScriptedPromptDriver is a deterministic in-process session driver used by
// the adapter harness instead of a real ACP subprocess.
type ScriptedPromptDriver struct {
	now       time.Time
	script    []ScriptedPromptEvent
	processes map[*session.AgentProcess]*scriptedPromptProcess
	prompts   []acp.PromptRequest
	mu        sync.Mutex
	startSeq  atomic.Int64
}

type scriptedPromptProcess struct {
	done sync.Once
	ch   chan struct{}
}

// NewScriptedPromptDriver constructs a session driver that replays the provided
// agent events for every prompt.
func NewScriptedPromptDriver(now time.Time, script []ScriptedPromptEvent) *ScriptedPromptDriver {
	return &ScriptedPromptDriver{
		now:       now,
		script:    append([]ScriptedPromptEvent(nil), script...),
		processes: make(map[*session.AgentProcess]*scriptedPromptProcess),
	}
}

// Start implements session.AgentDriver.
func (d *ScriptedPromptDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	seq := d.startSeq.Add(1)
	procState := &scriptedPromptProcess{ch: make(chan struct{})}
	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       int(seq),
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: fmt.Sprintf("acp-%d", seq),
		StartedAt: d.now.Add(time.Duration(seq) * time.Millisecond),
		Done:      procState.ch,
		Wait: func() error {
			<-procState.ch
			return nil
		},
	})

	d.mu.Lock()
	d.processes[proc] = procState
	d.mu.Unlock()
	return proc, nil
}

// Prompt implements session.AgentDriver.
func (d *ScriptedPromptDriver) Prompt(
	ctx context.Context,
	_ *session.AgentProcess,
	req acp.PromptRequest,
) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	d.prompts = append(d.prompts, req)
	script := d.script
	startedAt := d.now
	d.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}

	events := make(chan acp.AgentEvent, len(script))
	go func() {
		defer close(events)
		for idx, item := range script {
			if item.Delay > 0 {
				timer := time.NewTimer(item.Delay)
				select {
				case <-timer.C:
				case <-ctx.Done():
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					return
				}
			}
			event := acp.AgentEvent{
				Type:      item.Type,
				TurnID:    req.TurnID,
				Timestamp: startedAt.Add(time.Duration(idx+1) * time.Millisecond),
				Text:      item.Text,
				Error:     item.Error,
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

// Cancel implements session.AgentDriver.
func (d *ScriptedPromptDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

// Stop implements session.AgentDriver.
func (d *ScriptedPromptDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	state := d.processes[proc]
	d.mu.Unlock()
	if state == nil {
		return nil
	}
	state.done.Do(func() { close(state.ch) })
	return nil
}

// ManagedInstanceConfig configures one provider-owned bridge instance created by the harness.
type ManagedInstanceConfig struct {
	ID             string
	DisplayName    string
	DMPolicy       bridgepkg.BridgeDMPolicy
	RoutingPolicy  bridgepkg.RoutingPolicy
	ProviderConfig map[string]any
	BoundSecrets   []subprocess.InitializeBridgeBoundSecret
}

// HarnessConfig configures one subprocess-backed bridge adapter test harness.
type HarnessConfig struct {
	ExtensionDir             string
	ExtensionName            string
	DisplayName              string
	Platform                 string
	RoutingPolicy            bridgepkg.RoutingPolicy
	BoundSecrets             []subprocess.InitializeBridgeBoundSecret
	ManagedInstances         []ManagedInstanceConfig
	Driver                   session.AgentDriver
	StartTime                time.Time
	CrashOnceOnFirstDelivery bool
	BrokerOptions            []bridgepkg.DeliveryBrokerOption
	ExtraEnv                 map[string]string
}

// Harness wires the manager, host API, session manager, observer, and marker
// contract used to validate bridge adapters end to end.
type Harness struct {
	HomePaths aghconfig.HomePaths
	Markers   MarkerPaths
	Observer  *observepkg.Observer
	Bridges   *bridgepkg.Service
	Broker    *bridgepkg.Broker
	Handler   *extensionpkg.HostAPIHandler
	Manager   *extensionpkg.Manager
	Sessions  *session.Manager
	Instances []bridgepkg.BridgeInstance
}

// NewHarness starts the reusable adapter conformance harness.
func NewHarness(t testing.TB, cfg HarnessConfig) *Harness {
	t.Helper()

	if strings.TrimSpace(cfg.ExtensionDir) == "" {
		t.Fatal("extensiontest: HarnessConfig.ExtensionDir is required")
	}

	now := resolveHarnessStartTime(cfg.StartTime)
	homePaths := newHarnessHomePaths(t)
	markers := NewMarkerPaths(filepath.Join(t.TempDir(), "markers"))
	configureHarnessMarkers(t, markers, cfg)
	workspace := defaultResolvedWorkspace(filepath.Join(t.TempDir(), "workspace"), now)
	if err := os.MkdirAll(workspace.RootDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", workspace.RootDir, err)
	}
	workspaces := &staticWorkspaceResolver{resolved: workspace}

	globalDB := openHarnessGlobalDB(t, homePaths, &workspace)
	manifest, extensionRegistry, extensionName := installHarnessExtension(t, globalDB, cfg)

	bridgeRegistry := bridgepkg.NewRegistry(globalDB, bridgepkg.WithNow(func() time.Time { return now }))
	instances, managedRuntime := createHarnessManagedInstances(
		t,
		bridgeRegistry,
		&workspace,
		extensionName,
		cfg,
	)
	handler, manager, broker, observer, sessions := buildHarnessRuntime(
		t,
		cfg,
		homePaths,
		globalDB,
		workspaces,
		bridgeRegistry,
		extensionRegistry,
		manifest,
		extensionName,
		instances,
		managedRuntime,
		now,
		markers,
	)

	harness := &Harness{
		HomePaths: homePaths,
		Markers:   markers,
		Observer:  observer,
		Bridges:   bridgeRegistry,
		Broker:    broker,
		Handler:   handler,
		Manager:   manager,
		Sessions:  sessions,
		Instances: append([]bridgepkg.BridgeInstance(nil), instances...),
	}

	t.Cleanup(func() {
		harness.stopSessions(t)
		harness.Broker.Close()
		if err := harness.Manager.Stop(aghtestutil.Context(t)); err != nil {
			t.Fatalf("manager.Stop() error = %v", err)
		}
	})

	return harness
}

func resolveHarnessStartTime(startTime time.Time) time.Time {
	now := startTime.UTC()
	if now.IsZero() {
		now = time.Date(2026, 4, 11, 4, 0, 0, 0, time.UTC)
	}
	return now
}

func newHarnessHomePaths(t testing.TB) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

func configureHarnessMarkers(t testing.TB, markers MarkerPaths, cfg HarnessConfig) {
	t.Helper()

	for key, value := range markers.Env() {
		if key == EnvCrashOncePath {
			continue
		}
		t.Setenv(key, value)
	}
	if cfg.CrashOnceOnFirstDelivery {
		t.Setenv(EnvCrashOncePath, markers.CrashOnce)
	}
	for key, value := range cfg.ExtraEnv {
		t.Setenv(key, value)
	}
}

func openHarnessGlobalDB(
	t testing.TB,
	homePaths aghconfig.HomePaths,
	workspace *workspacepkg.ResolvedWorkspace,
) *globaldb.GlobalDB {
	t.Helper()

	resolvedWorkspace := mustHarnessWorkspace(t, workspace)
	globalDB, err := globaldb.OpenGlobalDB(aghtestutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("globaldb.OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(aghtestutil.Context(t)); err != nil {
			t.Fatalf("globaldb.Close() error = %v", err)
		}
	})
	if err := globalDB.InsertWorkspace(aghtestutil.Context(t), resolvedWorkspace.Workspace); err != nil {
		t.Fatalf("globalDB.InsertWorkspace() error = %v", err)
	}
	return globalDB
}

func mustHarnessWorkspace(
	t testing.TB,
	workspace *workspacepkg.ResolvedWorkspace,
) *workspacepkg.ResolvedWorkspace {
	t.Helper()

	if workspace == nil {
		t.Fatal("workspace = nil")
	}
	return workspace
}

func harnessWorkspaceID(workspace *workspacepkg.ResolvedWorkspace) string {
	if workspace == nil {
		return ""
	}
	if workspaceID := strings.TrimSpace(workspace.WorkspaceID); workspaceID != "" {
		return workspaceID
	}
	return strings.TrimSpace(workspace.ID)
}

func installHarnessExtension(
	t testing.TB,
	globalDB *globaldb.GlobalDB,
	cfg HarnessConfig,
) (*extensionpkg.Manifest, *extensionpkg.Registry, string) {
	t.Helper()

	manifest, err := extensionpkg.LoadManifest(cfg.ExtensionDir)
	if err != nil {
		t.Fatalf("extension.LoadManifest() error = %v", err)
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(cfg.ExtensionDir)
	if err != nil {
		t.Fatalf("extension.ComputeDirectoryChecksum() error = %v", err)
	}
	extensionRegistry := extensionpkg.NewRegistry(globalDB.DB())
	if err := extensionRegistry.Install(manifest, cfg.ExtensionDir, checksum); err != nil {
		t.Fatalf("extensionRegistry.Install() error = %v", err)
	}

	extensionName := strings.TrimSpace(cfg.ExtensionName)
	if extensionName == "" {
		extensionName = manifest.Name
	}
	return manifest, extensionRegistry, extensionName
}

func createHarnessManagedInstances(
	t testing.TB,
	bridgeRegistry *bridgepkg.Service,
	workspace *workspacepkg.ResolvedWorkspace,
	extensionName string,
	cfg HarnessConfig,
) ([]bridgepkg.BridgeInstance, []subprocess.InitializeBridgeManagedInstance) {
	t.Helper()

	managedConfigs := cfg.ManagedInstances
	if len(managedConfigs) == 0 {
		routingPolicy := cfg.RoutingPolicy
		if routingPolicy == (bridgepkg.RoutingPolicy{}) {
			routingPolicy = bridgepkg.RoutingPolicy{IncludePeer: true}
		}
		managedConfigs = []ManagedInstanceConfig{{
			ID:            bridgeAdapterHarnessBrgTelegramReferenceValue,
			DisplayName:   firstNonEmpty(cfg.DisplayName, "Telegram Reference"),
			RoutingPolicy: routingPolicy,
			BoundSecrets:  cloneBoundSecrets(cfg.BoundSecrets),
		}}
	}

	instances := make([]bridgepkg.BridgeInstance, 0, len(managedConfigs))
	managedRuntime := make([]subprocess.InitializeBridgeManagedInstance, 0, len(managedConfigs))
	for _, managedCfg := range managedConfigs {
		createReq, err := harnessCreateInstanceRequest(cfg, workspace, extensionName, len(instances)+1, managedCfg)
		if err != nil {
			t.Fatalf("harnessCreateInstanceRequest(%q) error = %v", managedCfg.ID, err)
		}
		instance, err := bridgeRegistry.CreateInstance(aghtestutil.Context(t), createReq)
		if err != nil {
			t.Fatalf("bridgeRegistry.CreateInstance(%q) error = %v", createReq.ID, err)
		}
		instances = append(instances, *instance)
		managedRuntime = append(managedRuntime, subprocess.InitializeBridgeManagedInstance{
			Instance:     *instance,
			BoundSecrets: cloneBoundSecrets(managedCfg.BoundSecrets),
		})
	}
	return instances, managedRuntime
}

func harnessCreateInstanceRequest(
	cfg HarnessConfig,
	workspace *workspacepkg.ResolvedWorkspace,
	extensionName string,
	seq int,
	managedCfg ManagedInstanceConfig,
) (bridgepkg.CreateInstanceRequest, error) {
	if workspace == nil {
		return bridgepkg.CreateInstanceRequest{}, errors.New("workspace is required")
	}
	var providerConfig json.RawMessage
	if managedCfg.ProviderConfig != nil {
		encodedProviderConfig, err := json.Marshal(managedCfg.ProviderConfig)
		if err != nil {
			return bridgepkg.CreateInstanceRequest{}, fmt.Errorf(
				"json.Marshal(provider_config for %q): %w",
				managedCfg.ID,
				err,
			)
		}
		providerConfig = encodedProviderConfig
	}

	createReq := bridgepkg.CreateInstanceRequest{
		ID:             firstNonEmpty(managedCfg.ID, fmt.Sprintf("brg-%d", seq)),
		Scope:          bridgepkg.ScopeWorkspace,
		WorkspaceID:    harnessWorkspaceID(workspace),
		Platform:       firstNonEmpty(cfg.Platform, "telegram"),
		ExtensionName:  extensionName,
		DisplayName:    firstNonEmpty(managedCfg.DisplayName, cfg.DisplayName, "Telegram Reference"),
		Enabled:        true,
		Status:         bridgepkg.BridgeStatusStarting,
		DMPolicy:       managedCfg.DMPolicy,
		RoutingPolicy:  managedCfg.RoutingPolicy,
		ProviderConfig: providerConfig,
	}
	if createReq.RoutingPolicy == (bridgepkg.RoutingPolicy{}) {
		createReq.RoutingPolicy = bridgepkg.RoutingPolicy{IncludePeer: true}
	}
	return createReq, nil
}

func buildHarnessRuntime(
	t testing.TB,
	cfg HarnessConfig,
	homePaths aghconfig.HomePaths,
	globalDB *globaldb.GlobalDB,
	workspaces workspacepkg.RuntimeResolver,
	bridgeRegistry *bridgepkg.Service,
	extensionRegistry *extensionpkg.Registry,
	manifest *extensionpkg.Manifest,
	extensionName string,
	instances []bridgepkg.BridgeInstance,
	managedRuntime []subprocess.InitializeBridgeManagedInstance,
	now time.Time,
	markers MarkerPaths,
) (*extensionpkg.HostAPIHandler, *extensionpkg.Manager, *bridgepkg.Broker, *observepkg.Observer, *session.Manager) {
	t.Helper()

	checker := &extensionpkg.CapabilityChecker{}
	checker.Register(extensionName, extensionpkg.SourceUser, manifest)

	var hostHandler *extensionpkg.HostAPIHandler
	telemetrySink := &deferredBridgeTelemetrySink{}
	hostForwarder := newHarnessHostForwarder(t, markers, func() *extensionpkg.HostAPIHandler {
		return hostHandler
	})

	manager := extensionpkg.NewManager(
		extensionRegistry,
		extensionpkg.WithCapabilityChecker(checker),
		extensionpkg.WithBridgeRuntimeResolver(&stubBridgeRuntimeResolver{
			runtimes: map[string]*subprocess.InitializeBridgeRuntime{
				extensionName: {
					RuntimeVersion:   subprocess.InitializeBridgeRuntimeVersion1,
					Provider:         extensionName,
					Platform:         instances[0].Platform,
					ManagedInstances: cloneManagedRuntime(managedRuntime),
				},
			},
		}),
		extensionpkg.WithBridgeTelemetrySink(telemetrySink),
		extensionpkg.WithHostMethodHandler("bridges/instances/list", hostForwarder("bridges/instances/list")),
		extensionpkg.WithHostMethodHandler("bridges/messages/ingest", hostForwarder("bridges/messages/ingest")),
		extensionpkg.WithHostMethodHandler("bridges/instances/get", hostForwarder("bridges/instances/get")),
		extensionpkg.WithHostMethodHandler(
			"bridges/instances/report_state",
			hostForwarder("bridges/instances/report_state"),
		),
		extensionpkg.WithHealthCheckTimeout(20*time.Millisecond),
		extensionpkg.WithSubprocessSignalGrace(15*time.Millisecond),
	)

	broker := bridgepkg.NewBroker(manager, cfg.BrokerOptions...)
	observer := newHarnessObserver(t, globalDB, homePaths, workspaces, bridgeRegistry, broker, now)
	telemetrySink.observer = observer

	driver := harnessDriver(cfg, now)
	notifier := extensionpkg.NewBridgeDeliveryNotifier(broker, observer)
	sessions := newHarnessSessions(t, homePaths, driver, notifier, workspaces, now)

	hostHandler = extensionpkg.NewHostAPIHandler(
		sessions,
		nil,
		observer,
		skillspkg.NewRegistry(skillspkg.RegistryConfig{}),
		extensionpkg.WithHostAPICapabilityChecker(checker),
		extensionpkg.WithHostAPIWorkspaceResolver(workspaces),
		extensionpkg.WithHostAPIBridgeRegistry(bridgeRegistry),
		extensionpkg.WithHostAPIBridgeDedupStore(globalDB),
		extensionpkg.WithHostAPIDeliveryBroker(broker),
		extensionpkg.WithHostAPINow(func() time.Time { return now }),
		extensionpkg.WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		extensionpkg.WithHostAPIRateLimit(1000, 1000),
	)

	if err := manager.Start(aghtestutil.Context(t)); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	return hostHandler, manager, broker, observer, sessions
}

func newHarnessHostForwarder(
	t testing.TB,
	markers MarkerPaths,
	handler func() *extensionpkg.HostAPIHandler,
) func(string) subprocess.HandlerFunc {
	t.Helper()

	return func(method string) subprocess.HandlerFunc {
		return func(ctx context.Context, params json.RawMessage) (any, error) {
			hostHandler := handler()
			if hostHandler == nil {
				return nil, errors.New("extensiontest: host api handler is not initialized")
			}
			result, err := hostHandler.HandleMethod(method)(ctx, params)
			if method == "bridges/instances/report_state" {
				recordHostStateTransition(t, markers.State, params, result, err)
			}
			return result, err
		}
	}
}

func newHarnessObserver(
	t testing.TB,
	globalDB *globaldb.GlobalDB,
	homePaths aghconfig.HomePaths,
	workspaces workspacepkg.RuntimeResolver,
	bridgeRegistry *bridgepkg.Service,
	broker *bridgepkg.Broker,
	now time.Time,
) *observepkg.Observer {
	t.Helper()

	observer, err := observepkg.New(
		aghtestutil.Context(t),
		observepkg.WithRegistry(globalDB),
		observepkg.WithHomePaths(homePaths),
		observepkg.WithWorkspaceResolver(workspaces),
		observepkg.WithBridgeSource(harnessBridgeSource{service: bridgeRegistry, broker: broker}),
		observepkg.WithNow(func() time.Time { return now }),
		observepkg.WithStartTime(now),
	)
	if err != nil {
		t.Fatalf("observe.New() error = %v", err)
	}
	return observer
}

func harnessDriver(cfg HarnessConfig, now time.Time) session.AgentDriver {
	if cfg.Driver != nil {
		return cfg.Driver
	}
	return NewScriptedPromptDriver(now, []ScriptedPromptEvent{
		{Type: acp.EventTypeAgentMessage, Text: bridgeAdapterHarnessHelloKey},
		{Type: acp.EventTypeDone},
	})
}

func newHarnessSessions(
	t testing.TB,
	homePaths aghconfig.HomePaths,
	driver session.AgentDriver,
	notifier session.Notifier,
	workspaces workspacepkg.RuntimeResolver,
	now time.Time,
) *session.Manager {
	t.Helper()

	sandboxRegistry, err := sandboxlocal.NewRegistry()
	if err != nil {
		t.Fatalf("local.NewRegistry() error = %v", err)
	}
	sessions, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithDriver(driver),
		session.WithNotifier(notifier),
		session.WithWorkspaceResolver(workspaces),
		session.WithStore(func(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
			return sessiondb.OpenSessionDB(ctx, sessionID, path)
		}),
		session.WithNow(func() time.Time { return now }),
		session.WithSessionIDGenerator(sequentialIDGenerator("sess")),
		session.WithTurnIDGenerator(sequentialIDGenerator("turn")),
		session.WithSandboxRegistry(sandboxRegistry),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}
	return sessions
}

// AppendInboundUpdate appends one fake platform update line for the adapter to ingest.
func (h *Harness) AppendInboundUpdate(t testing.TB, update any) {
	t.Helper()
	AppendInboundUpdateMarker(t, h.Markers, update)
}

// WaitForHandshake waits until the adapter writes its initialize marker.
func (h *Harness) WaitForHandshake(t testing.TB, timeout time.Duration) HandshakeRecord {
	t.Helper()
	return WaitForHandshakeMarker(t, h.Markers, timeout)
}

// WaitForStates waits until the state marker file satisfies the predicate.
func (h *Harness) WaitForStates(t testing.TB, timeout time.Duration, predicate func([]StateRecord) bool) []StateRecord {
	t.Helper()
	return WaitForStateMarkers(t, h.Markers, timeout, predicate)
}

// WaitForDeliveries waits until the delivery marker file satisfies the predicate.
func (h *Harness) WaitForDeliveries(
	t testing.TB,
	timeout time.Duration,
	predicate func([]DeliveryRecord) bool,
) []DeliveryRecord {
	t.Helper()
	return WaitForDeliveryMarkers(t, h.Markers, timeout, predicate)
}

// WaitForIngests waits until the ingest marker file satisfies the predicate.
func (h *Harness) WaitForIngests(
	t testing.TB,
	timeout time.Duration,
	predicate func([]IngestRecord) bool,
) []IngestRecord {
	t.Helper()
	return WaitForIngestMarkers(t, h.Markers, timeout, predicate)
}

// ObserveHealth returns the current observer health surface.
func (h *Harness) ObserveHealth(t testing.TB) observepkg.Health {
	t.Helper()
	health, err := h.Observer.Health(aghtestutil.Context(t))
	if err != nil {
		t.Fatalf("observer.Health() error = %v", err)
	}
	return health
}

// QueryBridgeHealth returns the current per-instance bridge health rows.
func (h *Harness) QueryBridgeHealth(t testing.TB) []observepkg.BridgeInstanceHealth {
	t.Helper()
	rows, err := h.Observer.QueryBridgeHealth(aghtestutil.Context(t))
	if err != nil {
		t.Fatalf("observer.QueryBridgeHealth() error = %v", err)
	}
	return rows
}

// Report reads the current collected adapter evidence into one reusable report.
func (h *Harness) Report(t testing.TB) ConformanceReport {
	t.Helper()
	return ReportFromMarkers(t, h.Markers)
}

// AppendInboundUpdateMarker appends one fake platform update line for the
// reference adapter marker contract.
func AppendInboundUpdateMarker(t testing.TB, markers MarkerPaths, update any) {
	t.Helper()
	appendJSONLine(t, markers.Updates, update)
}

// WaitForHandshakeMarker waits until the adapter writes its initialize marker.
func WaitForHandshakeMarker(t testing.TB, markers MarkerPaths, timeout time.Duration) HandshakeRecord {
	t.Helper()
	var record HandshakeRecord
	waitForCondition(t, timeout, "adapter handshake", func() bool {
		loaded, err := readJSONFile[HandshakeRecord](markers.Handshake)
		if err != nil {
			return false
		}
		record = loaded
		return true
	})
	return record
}

// WaitForStateMarkers waits until the state marker file satisfies the predicate.
func WaitForStateMarkers(
	t testing.TB,
	markers MarkerPaths,
	timeout time.Duration,
	predicate func([]StateRecord) bool,
) []StateRecord {
	t.Helper()
	return waitForJSONLinesCondition(t, markers.State, timeout, "adapter state markers", predicate)
}

// WaitForDeliveryMarkers waits until the delivery marker file satisfies the predicate.
func WaitForDeliveryMarkers(
	t testing.TB,
	markers MarkerPaths,
	timeout time.Duration,
	predicate func([]DeliveryRecord) bool,
) []DeliveryRecord {
	t.Helper()
	return waitForJSONLinesCondition(t, markers.Delivery, timeout, "adapter delivery markers", predicate)
}

// WaitForIngestMarkers waits until the ingest marker file satisfies the predicate.
func WaitForIngestMarkers(
	t testing.TB,
	markers MarkerPaths,
	timeout time.Duration,
	predicate func([]IngestRecord) bool,
) []IngestRecord {
	t.Helper()
	return waitForJSONLinesCondition(t, markers.Ingest, timeout, "adapter ingest markers", predicate)
}

// ReportFromMarkers reads the collected adapter evidence from one marker set.
func ReportFromMarkers(t testing.TB, markers MarkerPaths) ConformanceReport {
	t.Helper()
	report := ConformanceReport{}
	if handshake, err := readJSONFile[HandshakeRecord](markers.Handshake); err == nil {
		report.Handshake = &handshake
	}
	if ownership, err := readJSONFile[OwnershipRecord](markers.Ownership); err == nil {
		report.Ownership = &ownership
	}
	if states, err := readJSONLinesFile[StateRecord](markers.State); err == nil {
		report.States = states
	}
	if deliveries, err := readJSONLinesFile[DeliveryRecord](markers.Delivery); err == nil {
		report.Deliveries = deliveries
	}
	if ingests, err := readJSONLinesFile[IngestRecord](markers.Ingest); err == nil {
		report.Ingests = ingests
	}
	return report
}

func (h *Harness) stopSessions(t testing.TB) {
	t.Helper()
	for _, info := range h.Sessions.List() {
		if info == nil {
			continue
		}
		if err := h.Sessions.Stop(aghtestutil.Context(t), info.ID); err != nil {
			t.Fatalf("Sessions.Stop(%q) error = %v", info.ID, err)
		}
	}
}

type harnessBridgeSource struct {
	service *bridgepkg.Service
	broker  *bridgepkg.Broker
}

type deferredBridgeTelemetrySink struct {
	observer *observepkg.Observer
}

func (s *deferredBridgeTelemetrySink) RecordBridgeAuthFailure(bridgeInstanceID string) {
	if s == nil || s.observer == nil {
		return
	}
	s.observer.RecordBridgeAuthFailure(bridgeInstanceID)
}

func (s *deferredBridgeTelemetrySink) RecordBridgeRuntimeIssue(
	bridgeInstanceID string,
	status bridgepkg.BridgeStatus,
	message string,
) {
	if s == nil || s.observer == nil {
		return
	}
	s.observer.RecordBridgeRuntimeIssue(bridgeInstanceID, status, message)
}

func (s *deferredBridgeTelemetrySink) ClearBridgeRuntimeIssue(bridgeInstanceID string) {
	if s == nil || s.observer == nil {
		return
	}
	s.observer.ClearBridgeRuntimeIssue(bridgeInstanceID)
}

func (s harnessBridgeSource) ListInstances(ctx context.Context) ([]bridgepkg.BridgeInstance, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListInstances(ctx)
}

func (s harnessBridgeSource) ListRoutes(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListRoutes(ctx, bridgeInstanceID)
}

func (s harnessBridgeSource) DeliveryMetrics() map[string]bridgepkg.BridgeDeliveryMetrics {
	if s.broker == nil {
		return nil
	}
	return s.broker.DeliveryMetrics()
}

type staticWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
}

func (r *staticWorkspaceResolver) Resolve(_ context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
	if r == nil {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
	}
	trimmed := strings.TrimSpace(idOrPath)
	if trimmed == "" {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
	}
	if trimmed == r.resolved.ID || trimmed == r.resolved.WorkspaceID || trimmed == r.resolved.RootDir {
		return r.resolved, nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r *staticWorkspaceResolver) ResolveOrRegister(
	_ context.Context,
	path string,
) (workspacepkg.ResolvedWorkspace, error) {
	if r == nil {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
	}
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == r.resolved.RootDir || trimmed == r.resolved.WorkspaceID || trimmed == r.resolved.ID {
		return r.resolved, nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

type stubBridgeRuntimeResolver struct {
	runtimes map[string]*subprocess.InitializeBridgeRuntime
}

func (r *stubBridgeRuntimeResolver) ResolveBridgeRuntime(
	_ context.Context,
	extensionName string,
) (*subprocess.InitializeBridgeRuntime, error) {
	if r == nil || r.runtimes == nil {
		return nil, nil
	}
	return subprocess.CloneInitializeBridgeRuntime(r.runtimes[strings.TrimSpace(extensionName)]), nil
}

func defaultResolvedWorkspace(root string, now time.Time) workspacepkg.ResolvedWorkspace {
	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:           "ws-bridge-adapter",
			RootDir:      root,
			Name:         "bridge-adapter-workspace",
			DefaultAgent: bridgeAdapterHarnessCoderKey,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		WorkspaceID: "ws-bridge-adapter",
		Config: aghconfig.Config{
			Defaults: aghconfig.DefaultsConfig{Agent: bridgeAdapterHarnessCoderKey},
			Providers: map[string]aghconfig.ProviderConfig{
				"fake": {Command: "fake-agent"},
			},
			Permissions: aghconfig.PermissionsConfig{Mode: aghconfig.PermissionModeApproveAll},
		},
		Agents: []aghconfig.AgentDef{{
			Name:        bridgeAdapterHarnessCoderKey,
			Provider:    "fake",
			Permissions: string(aghconfig.PermissionModeApproveAll),
			Prompt:      "You are a reliable coder.",
		}},
		ResolvedAt: now,
	}
}

func cloneBoundSecrets(src []subprocess.InitializeBridgeBoundSecret) []subprocess.InitializeBridgeBoundSecret {
	if len(src) == 0 {
		return nil
	}
	return append([]subprocess.InitializeBridgeBoundSecret(nil), src...)
}

func cloneManagedRuntime(
	src []subprocess.InitializeBridgeManagedInstance,
) []subprocess.InitializeBridgeManagedInstance {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]subprocess.InitializeBridgeManagedInstance, 0, len(src))
	for _, managed := range src {
		item := managed
		item.BoundSecrets = cloneBoundSecrets(item.BoundSecrets)
		cloned = append(cloned, item)
	}
	return cloned
}

func sequentialIDGenerator(prefix string) session.IDGenerator {
	var counter atomic.Int64
	return func() string {
		return fmt.Sprintf("%s-%d", prefix, counter.Add(1))
	}
}

func readJSONFile[T any](path string) (T, error) {
	var item T
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return item, err
	}
	err = json.Unmarshal(payload, &item)
	return item, err
}

func readJSONLinesFile[T any](path string) ([]T, error) {
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, nil
	}
	lines := bytes.Split(payload, []byte{'\n'})
	items := make([]T, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func waitForJSONLinesCondition[T any](
	t testing.TB,
	path string,
	timeout time.Duration,
	label string,
	predicate func([]T) bool,
) []T {
	t.Helper()

	var records []T
	waitForCondition(t, timeout, label, func() bool {
		loaded, err := readJSONLinesFile[T](path)
		if err != nil {
			return false
		}
		records = loaded
		return predicate(records)
	})
	return records
}

func appendJSONLine(t testing.TB, path string, value any) {
	t.Helper()
	target := strings.TrimSpace(path)
	if target == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(target), err)
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("os.OpenFile(%q) error = %v", target, err)
	}
	defer func() {
		_ = file.Close()
	}()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		t.Fatalf("encoder.Encode(%q) error = %v", target, err)
	}
}

func recordHostStateTransition(
	t testing.TB,
	path string,
	params json.RawMessage,
	result any,
	callErr error,
) {
	t.Helper()

	record := StateRecord{}

	var request extensioncontract.BridgesInstancesReportStateParams
	if err := json.Unmarshal(params, &request); err == nil {
		record.BridgeInstanceID = strings.TrimSpace(request.BridgeInstanceID)
		record.Status = request.Status.Normalize()
		record.Instance = bridgepkg.BridgeInstance{
			ID:          record.BridgeInstanceID,
			Status:      request.Status.Normalize(),
			Degradation: cloneBridgeDegradation(request.Degradation),
		}
	}

	switch typed := result.(type) {
	case *bridgepkg.BridgeInstance:
		if typed != nil {
			record.Instance = copyBridgeInstance(*typed)
		}
	case bridgepkg.BridgeInstance:
		record.Instance = copyBridgeInstance(typed)
	}

	if record.BridgeInstanceID == "" {
		record.BridgeInstanceID = strings.TrimSpace(record.Instance.ID)
	}
	if record.Status == "" {
		record.Status = record.Instance.Status.Normalize()
	}
	if callErr != nil {
		record.Error = callErr.Error()
	}

	appendJSONLine(t, path, record)
}

func cloneBridgeDegradation(degradation *bridgepkg.BridgeDegradation) *bridgepkg.BridgeDegradation {
	if degradation == nil {
		return nil
	}
	cloned := *degradation
	return &cloned
}

func copyBridgeInstance(instance bridgepkg.BridgeInstance) bridgepkg.BridgeInstance {
	copied := instance
	copied.ProviderConfig = append([]byte(nil), instance.ProviderConfig...)
	copied.DeliveryDefaults = append([]byte(nil), instance.DeliveryDefaults...)
	copied.Degradation = cloneBridgeDegradation(instance.Degradation)
	return copied
}

func waitForCondition(t testing.TB, timeout time.Duration, label string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s did not satisfy condition before timeout", label)
}

func normalizeEventType(eventType string) string {
	return strings.ToLower(strings.TrimSpace(eventType))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
