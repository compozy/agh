// Package agentidentity resolves daemon-validated caller identity for
// agent-facing CLI and UDS operations.
package agentidentity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	// EnvSessionID is the daemon-issued session identifier visible inside agent sessions.
	EnvSessionID = "AGH_SESSION_ID"
	// EnvAgent is the daemon-issued agent name visible inside agent sessions.
	EnvAgent = "AGH_AGENT"

	// HeaderSessionID carries EnvSessionID over the local UDS HTTP transport.
	HeaderSessionID = "X-AGH-Session-ID"
	// HeaderAgent carries EnvAgent over the local UDS HTTP transport.
	HeaderAgent = "X-AGH-Agent"
	// HeaderWorkspaceID optionally narrows an agent request to the caller workspace.
	HeaderWorkspaceID = "X-AGH-Workspace-ID"
)

const (
	agentCommandFailedMessage = "agent command failed"

	// ExitOK reports successful agent command execution.
	ExitOK = 0
	// ExitIdentityRequired reports missing caller identity input.
	ExitIdentityRequired = 64
	// ExitIdentityInvalid reports stale or mismatched caller identity.
	ExitIdentityInvalid = 65
	// ExitUnauthorized reports a caller identity that is valid but not allowed for the requested scope.
	ExitUnauthorized = 77
	// ExitUnavailable reports daemon lookup or validation infrastructure failure.
	ExitUnavailable = 69
)

var (
	// ErrIdentityRequired reports missing required agent caller sandbox.
	ErrIdentityRequired = errors.New("agent identity required")
	// ErrIdentityStale reports a missing, unknown, stopped, or otherwise inactive session identity.
	ErrIdentityStale = errors.New("agent identity stale")
	// ErrIdentityMismatch reports env/header identity that does not match the daemon session record.
	ErrIdentityMismatch = errors.New("agent identity mismatch")
	// ErrIdentityUnauthorized reports a validated identity that is not allowed for the requested scope.
	ErrIdentityUnauthorized = errors.New("agent identity unauthorized")
	// ErrIdentityLookupUnavailable reports validation infrastructure that is not available.
	ErrIdentityLookupUnavailable = errors.New("agent identity lookup unavailable")
)

// Credentials carries untrusted caller identity hints from env or transport headers.
type Credentials struct {
	SessionID   string
	AgentName   string
	WorkspaceID string
}

// SessionSnapshot is the daemon-authoritative session subset needed for identity validation.
type SessionSnapshot struct {
	ID            string
	Name          string
	AgentName     string
	Provider      string
	Model         string
	WorkspaceID   string
	WorkspacePath string
	Channel       string
	Type          session.Type
	Lineage       *store.SessionLineage
	State         session.State
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SessionLookup loads a daemon-authoritative session snapshot by session ID.
type SessionLookup func(context.Context, string) (SessionSnapshot, error)

// ResolveOptions configures agent caller resolution.
type ResolveOptions struct {
	Credentials         Credentials
	Lookup              SessionLookup
	ExpectedWorkspaceID string
	OriginKind          taskpkg.OriginKind
	OriginRef           string
}

// Caller is a validated agent-session caller and its task-domain actor context.
type Caller struct {
	Credentials Credentials
	Session     SessionSnapshot
	Actor       taskpkg.ActorContext
}

// Error carries a stable machine-readable identity failure code with an actionable message.
type Error struct {
	Code    string
	Message string
	Action  string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Action) == "" {
		return strings.TrimSpace(e.Message)
	}
	return strings.TrimSpace(e.Message) + ": " + strings.TrimSpace(e.Action)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ErrorPayload is the stable machine-readable CLI error shape for agent namespaces.
type ErrorPayload struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Action   string `json:"action"`
	ExitCode int    `json:"exit_code"`
}

// Resolve validates untrusted caller credentials against the daemon session lookup.
func Resolve(ctx context.Context, opts ResolveOptions) (Caller, error) {
	creds := normalizeCredentials(opts.Credentials)
	if err := validateResolveInputs(ctx, opts.Lookup, creds); err != nil {
		return Caller{}, err
	}
	snapshot, err := lookupSessionSnapshot(ctx, opts.Lookup, creds)
	if err != nil {
		return Caller{}, err
	}
	if err := validateWorkspace(snapshot, opts.ExpectedWorkspaceID, creds.WorkspaceID); err != nil {
		return Caller{}, err
	}
	actor, err := deriveActorContext(snapshot.ID, opts.OriginKind, opts.OriginRef)
	if err != nil {
		return Caller{}, fmt.Errorf("agent identity: derive actor context: %w", err)
	}

	return Caller{
		Credentials: creds,
		Session:     snapshot,
		Actor:       actor,
	}, nil
}

func validateResolveInputs(ctx context.Context, lookup SessionLookup, creds Credentials) error {
	if ctx == nil {
		return identityError(
			ErrIdentityLookupUnavailable,
			"identity_lookup_unavailable",
			"agent identity cannot be validated",
			"retry after the daemon is reachable",
		)
	}
	if creds.SessionID == "" {
		return identityError(
			ErrIdentityRequired,
			"identity_required",
			EnvSessionID+" is required for agent commands",
			"run this command from an AGH-managed agent session",
		)
	}
	if creds.AgentName == "" {
		return identityError(
			ErrIdentityRequired,
			"identity_required",
			EnvAgent+" is required for agent commands",
			"run this command from an AGH-managed agent session",
		)
	}
	if lookup == nil {
		return identityError(
			ErrIdentityLookupUnavailable,
			"identity_lookup_unavailable",
			"agent identity cannot be validated",
			"retry after the daemon is reachable",
		)
	}
	return nil
}

func lookupSessionSnapshot(ctx context.Context, lookup SessionLookup, creds Credentials) (SessionSnapshot, error) {
	snapshot, err := lookup(ctx, creds.SessionID)
	if err != nil {
		if errors.Is(err, ErrIdentityLookupUnavailable) ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return SessionSnapshot{}, identityError(
				ErrIdentityLookupUnavailable,
				"identity_lookup_unavailable",
				"agent identity cannot be validated",
				"retry after the daemon is reachable",
			)
		}
		return SessionSnapshot{}, identityError(
			ErrIdentityStale,
			"identity_stale",
			"agent session identity is not known to the daemon",
			"start or resume the AGH session, then retry",
		)
	}
	snapshot = normalizeSessionSnapshot(snapshot)
	if snapshot.ID == "" || snapshot.State != session.StateActive {
		return SessionSnapshot{}, identityError(
			ErrIdentityStale,
			"identity_stale",
			"agent session identity is not active",
			"start or resume the AGH session, then retry",
		)
	}
	if snapshot.ID != creds.SessionID {
		return SessionSnapshot{}, identityError(
			ErrIdentityMismatch,
			"identity_mismatch",
			"agent session lookup returned a different session",
			"clear stale AGH identity environment variables and retry",
		)
	}
	if snapshot.AgentName != creds.AgentName {
		return SessionSnapshot{}, identityError(
			ErrIdentityMismatch,
			"identity_mismatch",
			EnvAgent+" does not match the daemon session agent",
			"clear stale AGH identity environment variables and retry",
		)
	}
	return snapshot, nil
}

func validateWorkspace(snapshot SessionSnapshot, expectedWorkspaceID string, credentialsWorkspaceID string) error {
	expectedWorkspaceID = strings.TrimSpace(expectedWorkspaceID)
	if expectedWorkspaceID == "" {
		expectedWorkspaceID = credentialsWorkspaceID
	}
	if expectedWorkspaceID == "" || snapshot.WorkspaceID == expectedWorkspaceID {
		return nil
	}
	return identityError(
		ErrIdentityUnauthorized,
		"identity_unauthorized",
		"agent session does not belong to the requested workspace",
		"use the session workspace or start a session in the requested workspace",
	)
}

func deriveActorContext(
	sessionID string,
	originKind taskpkg.OriginKind,
	originRef string,
) (taskpkg.ActorContext, error) {
	originKind = originKind.Normalize()
	if originKind == "" {
		originKind = taskpkg.OriginKindAgentSession
	}
	originRef = strings.TrimSpace(originRef)
	if originRef == "" {
		originRef = originRefForKind(originKind)
	}
	return taskpkg.DeriveAgentSessionActorContextForOrigin(sessionID, originKind, originRef)
}

// SessionSnapshotFromInfo converts the runtime session read model into a validation snapshot.
func SessionSnapshotFromInfo(info *session.Info) SessionSnapshot {
	if info == nil {
		return SessionSnapshot{}
	}
	return SessionSnapshot{
		ID:            info.ID,
		Name:          info.Name,
		AgentName:     info.AgentName,
		Provider:      info.Provider,
		Model:         info.Model,
		WorkspaceID:   info.WorkspaceID,
		WorkspacePath: info.Workspace,
		Channel:       info.Channel,
		Type:          info.Type,
		Lineage:       store.CloneSessionLineage(info.Lineage),
		State:         info.State,
		CreatedAt:     info.CreatedAt,
		UpdatedAt:     info.UpdatedAt,
	}
}

// ErrorPayloadFor returns the stable machine-readable error payload for agent CLI output.
func ErrorPayloadFor(err error) ErrorPayload {
	payload := ErrorPayload{
		Code:     "agent_error",
		Message:  strings.TrimSpace(errorString(err)),
		Action:   "inspect the daemon error and retry",
		ExitCode: ExitCodeForError(err),
	}
	var identityErr *Error
	if errors.As(err, &identityErr) && identityErr != nil {
		payload.Code = strings.TrimSpace(identityErr.Code)
		payload.Message = strings.TrimSpace(identityErr.Message)
		payload.Action = strings.TrimSpace(identityErr.Action)
	}
	if payload.Code == "" {
		payload.Code = "agent_error"
	}
	if payload.Message == "" {
		payload.Message = agentCommandFailedMessage
	}
	if payload.Action == "" {
		payload.Action = "inspect the daemon error and retry"
	}
	return payload
}

// MarshalErrorJSON renders a stable JSON error object for agent CLI commands.
func MarshalErrorJSON(err error) ([]byte, error) {
	return json.Marshal(struct {
		Error ErrorPayload `json:"error"`
	}{Error: ErrorPayloadFor(err)})
}

// MarshalErrorJSONL renders one stable JSONL error frame for agent CLI streaming commands.
func MarshalErrorJSONL(err error) ([]byte, error) {
	return json.Marshal(struct {
		Type  string       `json:"type"`
		Error ErrorPayload `json:"error"`
	}{
		Type:  "error",
		Error: ErrorPayloadFor(err),
	})
}

// ExitCodeForError maps agent identity and command errors to deterministic CLI exit codes.
func ExitCodeForError(err error) int {
	switch {
	case err == nil:
		return ExitOK
	case errors.Is(err, ErrIdentityRequired):
		return ExitIdentityRequired
	case errors.Is(err, ErrIdentityLookupUnavailable):
		return ExitUnavailable
	case errors.Is(err, ErrIdentityMismatch), errors.Is(err, ErrIdentityStale):
		return ExitIdentityInvalid
	case errors.Is(err, ErrIdentityUnauthorized):
		return ExitUnauthorized
	default:
		return ExitUnavailable
	}
}

func normalizeCredentials(creds Credentials) Credentials {
	return Credentials{
		SessionID:   strings.TrimSpace(creds.SessionID),
		AgentName:   strings.TrimSpace(creds.AgentName),
		WorkspaceID: strings.TrimSpace(creds.WorkspaceID),
	}
}

func normalizeSessionSnapshot(snapshot SessionSnapshot) SessionSnapshot {
	snapshot.ID = strings.TrimSpace(snapshot.ID)
	snapshot.Name = strings.TrimSpace(snapshot.Name)
	snapshot.AgentName = strings.TrimSpace(snapshot.AgentName)
	snapshot.Provider = strings.TrimSpace(snapshot.Provider)
	snapshot.Model = strings.TrimSpace(snapshot.Model)
	snapshot.WorkspaceID = strings.TrimSpace(snapshot.WorkspaceID)
	snapshot.WorkspacePath = strings.TrimSpace(snapshot.WorkspacePath)
	snapshot.Channel = strings.TrimSpace(snapshot.Channel)
	return snapshot
}

func originRefForKind(kind taskpkg.OriginKind) string {
	switch kind.Normalize() {
	case taskpkg.OriginKindCLI:
		return "agent.cli"
	case taskpkg.OriginKindUDS:
		return "agent.uds"
	default:
		return "agent.session"
	}
}

func identityError(err error, code string, message string, action string) error {
	return &Error{
		Code:    code,
		Message: strings.TrimSpace(message),
		Action:  strings.TrimSpace(action),
		Err:     err,
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
