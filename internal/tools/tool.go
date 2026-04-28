// Package tools defines canonical Tool Registry contracts.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Tool is the cold desired-state resource spec for a registry tool.
type Tool struct {
	ID                  ToolID          `json:"id"`
	Backend             BackendRef      `json:"backend"`
	DisplayTitle        string          `json:"display_title,omitempty"`
	Description         string          `json:"description"`
	InputSchema         json.RawMessage `json:"input_schema"`
	OutputSchema        json.RawMessage `json:"output_schema,omitempty"`
	Source              SourceRef       `json:"source"`
	Visibility          Visibility      `json:"visibility"`
	Risk                RiskClass       `json:"risk"`
	ReadOnly            bool            `json:"read_only"`
	Destructive         bool            `json:"destructive"`
	OpenWorld           bool            `json:"open_world"`
	RequiresInteraction bool            `json:"requires_interaction"`
	ConcurrencySafe     bool            `json:"concurrency_safe"`
	MaxResultBytes      int64           `json:"max_result_bytes,omitempty"`
	Toolsets            []ToolsetID     `json:"toolsets,omitempty"`
	Tags                []string        `json:"tags,omitempty"`
	SearchHints         []string        `json:"search_hints,omitempty"`
}

// Descriptor is the normalized runtime metadata used for indexing and dispatch.
type Descriptor struct {
	ID                  ToolID          `json:"id"`
	Backend             BackendRef      `json:"backend"`
	DisplayTitle        string          `json:"display_title,omitempty"`
	Description         string          `json:"description"`
	InputSchema         json.RawMessage `json:"input_schema"`
	OutputSchema        json.RawMessage `json:"output_schema,omitempty"`
	Source              SourceRef       `json:"source"`
	Visibility          Visibility      `json:"visibility"`
	Risk                RiskClass       `json:"risk"`
	ReadOnly            bool            `json:"read_only"`
	Destructive         bool            `json:"destructive"`
	OpenWorld           bool            `json:"open_world"`
	RequiresInteraction bool            `json:"requires_interaction"`
	ConcurrencySafe     bool            `json:"concurrency_safe"`
	MaxResultBytes      int64           `json:"max_result_bytes,omitempty"`
	Toolsets            []ToolsetID     `json:"toolsets,omitempty"`
	Tags                []string        `json:"tags,omitempty"`
	SearchHints         []string        `json:"search_hints,omitempty"`
}

// Descriptor converts a cold resource into the runtime descriptor shape.
func (t Tool) Descriptor() Descriptor {
	return Descriptor{
		ID:                  t.ID,
		Backend:             t.Backend,
		DisplayTitle:        t.DisplayTitle,
		Description:         t.Description,
		InputSchema:         cloneRawMessage(t.InputSchema),
		OutputSchema:        cloneRawMessage(t.OutputSchema),
		Source:              t.Source,
		Visibility:          t.Visibility,
		Risk:                t.Risk,
		ReadOnly:            t.ReadOnly,
		Destructive:         t.Destructive,
		OpenWorld:           t.OpenWorld,
		RequiresInteraction: t.RequiresInteraction,
		ConcurrencySafe:     t.ConcurrencySafe,
		MaxResultBytes:      t.MaxResultBytes,
		Toolsets:            cloneToolsets(t.Toolsets),
		Tags:                cloneStrings(t.Tags),
		SearchHints:         cloneStrings(t.SearchHints),
	}
}

// Tool returns the cold resource shape for a runtime descriptor.
func (d Descriptor) Tool() Tool {
	return Tool{
		ID:                  d.ID,
		Backend:             d.Backend,
		DisplayTitle:        d.DisplayTitle,
		Description:         d.Description,
		InputSchema:         cloneRawMessage(d.InputSchema),
		OutputSchema:        cloneRawMessage(d.OutputSchema),
		Source:              d.Source,
		Visibility:          d.Visibility,
		Risk:                d.Risk,
		ReadOnly:            d.ReadOnly,
		Destructive:         d.Destructive,
		OpenWorld:           d.OpenWorld,
		RequiresInteraction: d.RequiresInteraction,
		ConcurrencySafe:     d.ConcurrencySafe,
		MaxResultBytes:      d.MaxResultBytes,
		Toolsets:            cloneToolsets(d.Toolsets),
		Tags:                cloneStrings(d.Tags),
		SearchHints:         cloneStrings(d.SearchHints),
	}
}

// Validate ensures the descriptor is dispatchable metadata.
func (d Descriptor) Validate() error {
	if err := d.ID.Validate(); err != nil {
		return err
	}
	if err := d.Backend.Validate("backend"); err != nil {
		return err
	}
	if err := d.Source.Validate("source"); err != nil {
		return err
	}
	if err := d.Visibility.Validate("visibility"); err != nil {
		return err
	}
	if err := d.Risk.Validate("risk"); err != nil {
		return err
	}
	if err := ValidateJSONObject("input_schema", d.InputSchema, true); err != nil {
		return err
	}
	if err := ValidateJSONObject("output_schema", d.OutputSchema, false); err != nil {
		return err
	}
	if d.MaxResultBytes < 0 {
		return NewValidationError(
			"max_result_bytes",
			ReasonResultBudgetExceeded,
			"must be greater than or equal to zero",
		)
	}
	if d.ReadOnly && (d.Destructive || d.OpenWorld) {
		return NewValidationError(
			"read_only",
			ReasonPolicyDenied,
			"read-only tools cannot be destructive or open-world",
		)
	}
	for i, toolset := range d.Toolsets {
		if err := toolset.Validate(); err != nil {
			return wrapField(err, fmt.Sprintf("toolsets[%d]", i))
		}
	}
	return nil
}

// Validate ensures the cold resource can normalize into a descriptor.
func (t Tool) Validate() error {
	return t.Descriptor().Validate()
}

// BackendKind identifies the executable backend class.
type BackendKind string

const (
	// BackendNativeGo executes a daemon-compiled Go handler.
	BackendNativeGo BackendKind = "native_go"
	// BackendExtensionHost executes through the extension host subprocess runtime.
	BackendExtensionHost BackendKind = "extension_host"
	// BackendMCP executes through daemon-owned MCP client adapters.
	BackendMCP BackendKind = "mcp"
	// BackendBridge is reserved for a later bridge adapter TechSpec.
	BackendBridge BackendKind = "bridge"
)

// Validate ensures the backend kind is documented.
func (k BackendKind) Validate(field string) error {
	switch k {
	case BackendNativeGo, BackendExtensionHost, BackendMCP, BackendBridge:
		return nil
	default:
		return NewValidationError(field, ReasonBackendNotExecutable, "unsupported backend kind")
	}
}

// BackendRef binds a descriptor to its executable backend.
type BackendRef struct {
	Kind                 BackendKind `json:"kind"`
	ExtensionID          string      `json:"extension_id,omitempty"`
	Handler              string      `json:"handler,omitempty"`
	MCPServer            string      `json:"mcp_server,omitempty"`
	MCPTool              string      `json:"mcp_tool,omitempty"`
	NativeName           string      `json:"native_name,omitempty"`
	RequiresCapabilities []string    `json:"requires_capabilities,omitempty"`
}

// Validate ensures the backend reference has the fields required for its kind.
func (b BackendRef) Validate(field string) error {
	if err := b.Kind.Validate(field + ".kind"); err != nil {
		return err
	}
	switch b.Kind {
	case BackendNativeGo:
		if b.NativeName == "" {
			return NewValidationError(
				field+".native_name",
				ReasonDependencyMissing,
				"native backend requires native_name",
			)
		}
	case BackendExtensionHost:
		if b.ExtensionID == "" {
			return NewValidationError(
				field+".extension_id",
				ReasonExtensionInactive,
				"extension_host backend requires extension_id",
			)
		}
		if b.Handler == "" {
			return NewValidationError(field+".handler", ReasonHandlerMissing, "extension_host backend requires handler")
		}
	case BackendMCP:
		if b.MCPServer == "" {
			return NewValidationError(field+".mcp_server", ReasonMCPUnreachable, "mcp backend requires mcp_server")
		}
		if b.MCPTool == "" {
			return NewValidationError(field+".mcp_tool", ReasonDependencyMissing, "mcp backend requires mcp_tool")
		}
	case BackendBridge:
		return NewValidationError(field+".kind", ReasonBackendNotExecutable, "bridge backend is reserved post-MVP")
	}
	return nil
}

// SourceKind identifies the provenance class for a descriptor.
type SourceKind string

const (
	// SourceBuiltin marks daemon-defined tools.
	SourceBuiltin SourceKind = "builtin"
	// SourceMCP marks tools discovered from MCP servers.
	SourceMCP SourceKind = "mcp"
	// SourceExtension marks tools provided by extensions.
	SourceExtension SourceKind = "extension"
	// SourceDynamic marks future runtime-assembled tools.
	SourceDynamic SourceKind = "dynamic"
)

// ToolSource preserves the public source-name type used by existing resource contracts.
type ToolSource = SourceKind

const (
	// ToolSourceBuiltin marks daemon-defined tools.
	ToolSourceBuiltin = SourceBuiltin
	// ToolSourceMCP marks tools discovered from MCP servers.
	ToolSourceMCP = SourceMCP
	// ToolSourceExtension marks tools provided by extensions.
	ToolSourceExtension = SourceExtension
	// ToolSourceDynamic marks future runtime-assembled tools.
	ToolSourceDynamic = SourceDynamic
)

// String returns the stable source kind text.
func (k SourceKind) String() string {
	return string(k)
}

// Validate ensures the source kind is documented.
func (k SourceKind) Validate(field string) error {
	switch k {
	case SourceBuiltin, SourceMCP, SourceExtension, SourceDynamic:
		return nil
	default:
		return NewValidationError(field, ReasonSourceDisabled, "unsupported source kind")
	}
}

// SourceRef preserves provenance without creating alternate tool identities.
type SourceRef struct {
	Kind            SourceKind `json:"kind"`
	Owner           string     `json:"owner"`
	RawServerName   string     `json:"raw_server_name,omitempty"`
	RawToolName     string     `json:"raw_tool_name,omitempty"`
	ResourceID      string     `json:"resource_id,omitempty"`
	ResourceVersion string     `json:"resource_version,omitempty"`
	WorkspaceID     string     `json:"workspace_id,omitempty"`
	Scope           string     `json:"scope,omitempty"`
}

// Validate ensures source provenance can support deterministic diagnostics.
func (s SourceRef) Validate(field string) error {
	if err := s.Kind.Validate(field + ".kind"); err != nil {
		return err
	}
	if s.Owner == "" {
		return NewValidationError(field+".owner", ReasonSourceDisabled, "source owner is required")
	}
	if s.Kind == SourceMCP && (s.RawServerName == "" || s.RawToolName == "") {
		return NewValidationError(field, ReasonMCPUnreachable, "mcp sources require raw server and tool names")
	}
	return nil
}

// Visibility identifies which surfaces may display a descriptor.
type Visibility string

const (
	// VisibilityInternal limits a tool to daemon-internal use.
	VisibilityInternal Visibility = "internal"
	// VisibilityOperator exposes a tool to operator diagnostics.
	VisibilityOperator Visibility = "operator"
	// VisibilitySession exposes a tool to session-scoped views.
	VisibilitySession Visibility = "session"
	// VisibilityModel exposes a tool to model-visible projections.
	VisibilityModel Visibility = "model"
)

// Validate ensures the visibility is documented.
func (v Visibility) Validate(field string) error {
	switch v {
	case VisibilityInternal, VisibilityOperator, VisibilitySession, VisibilityModel:
		return nil
	default:
		return NewValidationError(field, ReasonPolicyDenied, "unsupported visibility")
	}
}

// RiskClass classifies the safety posture of a tool.
type RiskClass string

const (
	// RiskRead marks read-only local inspection.
	RiskRead RiskClass = "read"
	// RiskMutating marks state-changing behavior.
	RiskMutating RiskClass = "mutating"
	// RiskOpenWorld marks access to arbitrary external state.
	RiskOpenWorld RiskClass = "open_world"
	// RiskDestructive marks destructive or irreversible behavior.
	RiskDestructive RiskClass = "destructive"
)

// Validate ensures the risk class is documented.
func (r RiskClass) Validate(field string) error {
	switch r {
	case RiskRead, RiskMutating, RiskOpenWorld, RiskDestructive:
		return nil
	default:
		return NewValidationError(field, ReasonPolicyDenied, "unsupported risk class")
	}
}

// Registry owns tool discovery and dispatch for all surfaces.
type Registry interface {
	List(ctx context.Context, scope Scope) ([]ToolView, error)
	Search(ctx context.Context, scope Scope, q SearchQuery) ([]ToolView, error)
	Get(ctx context.Context, scope Scope, id ToolID) (ToolView, error)
	Call(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error)
}

// Handle is the executable runtime contract for one tool.
type Handle interface {
	Descriptor() Descriptor
	Availability(ctx context.Context, scope Scope) Availability
	Call(ctx context.Context, req CallRequest) (ToolResult, error)
}

// Provider contributes descriptors and executable handles from one source.
type Provider interface {
	ID() SourceRef
	List(ctx context.Context, scope Scope) ([]Descriptor, error)
	Resolve(ctx context.Context, scope Scope, id ToolID) (Handle, bool, error)
}

// NativeToolFunc is the daemon-compiled function signature for native tools.
type NativeToolFunc func(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error)

// ExtensionToolInvoker invokes out-of-process extension tool providers.
type ExtensionToolInvoker interface {
	ProvideTools(ctx context.Context, extensionID string) ([]ExtensionToolRuntimeDescriptor, error)
	CallTool(ctx context.Context, extensionID string, req ExtensionToolCallRequest) (ToolResult, error)
}

// MCPCallExecutor lists and calls MCP tools without exposing credential material.
type MCPCallExecutor interface {
	ListTools(ctx context.Context, source SourceRef) ([]MCPToolDescriptor, error)
	CallTool(ctx context.Context, source SourceRef, req MCPToolCallRequest) (ToolResult, error)
}

// MCPAuthStatusProvider returns redacted MCP auth status for diagnostics.
type MCPAuthStatusProvider interface {
	Status(ctx context.Context, source SourceRef) (MCPAuthStatus, error)
}

// PolicyEvaluator computes the effective policy decision for a descriptor.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, scope Scope, d Descriptor) (EffectiveToolDecision, error)
}

// ResultLimiter applies descriptor result budgets and redaction policy.
type ResultLimiter interface {
	Apply(ctx context.Context, d Descriptor, result ToolResult) (ToolResult, error)
}

// HookRunner runs typed registry hooks around dispatch.
type HookRunner interface {
	PreCall(ctx context.Context, call CallRequest) (CallRequest, EffectiveToolDecision, error)
	PostCall(ctx context.Context, call CallRequest, result ToolResult) (ToolResult, error)
	PostError(ctx context.Context, call CallRequest, err error) error
}

// Scope identifies the caller context used for projections and dispatch.
type Scope struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	Operator    bool   `json:"operator,omitempty"`
}

// SearchQuery describes a registry search request.
type SearchQuery struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// ToolView is a descriptor plus effective diagnostics for a caller.
type ToolView struct {
	Descriptor   Descriptor            `json:"descriptor"`
	Availability Availability          `json:"availability"`
	Decision     EffectiveToolDecision `json:"decision"`
}

// CallRequest is the canonical dispatch request.
type CallRequest struct {
	ToolID        ToolID          `json:"tool_id"`
	SessionID     string          `json:"session_id,omitempty"`
	WorkspaceID   string          `json:"workspace_id,omitempty"`
	Input         json.RawMessage `json:"input"`
	ApprovalToken string          `json:"approval_token,omitempty"`
}

// ExtensionToolRuntimeDescriptor is the runtime reconciliation proof for an extension tool.
type ExtensionToolRuntimeDescriptor struct {
	ID                 ToolID    `json:"id"`
	Handler            string    `json:"handler"`
	InputSchemaDigest  string    `json:"input_schema_digest"`
	OutputSchemaDigest string    `json:"output_schema_digest,omitempty"`
	ReadOnly           bool      `json:"read_only"`
	Risk               RiskClass `json:"risk"`
	Capabilities       []string  `json:"capabilities,omitempty"`
}

// ExtensionToolCallRequest is the extension host call request.
type ExtensionToolCallRequest struct {
	ToolID    ToolID          `json:"tool_id"`
	Handler   string          `json:"handler"`
	SessionID string          `json:"session_id,omitempty"`
	Input     json.RawMessage `json:"input"`
}

// ExtensionToolCallResponse is the extension host call response.
type ExtensionToolCallResponse struct {
	Result ToolResult `json:"result"`
}

// MCPToolDescriptor describes one externally discovered MCP tool.
type MCPToolDescriptor struct {
	ID          ToolID          `json:"id"`
	RawName     string          `json:"raw_name"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
	ReadOnly    bool            `json:"read_only"`
}

// MCPToolCallRequest is the daemon-owned MCP adapter call request.
type MCPToolCallRequest struct {
	ToolID      ToolID          `json:"tool_id"`
	RawToolName string          `json:"raw_tool_name"`
	Input       json.RawMessage `json:"input"`
}

// MCPToolCallResponse is the daemon-owned MCP adapter call response.
type MCPToolCallResponse struct {
	Result ToolResult `json:"result"`
}

// MCPAuthStatus is a redacted auth diagnostic for external MCP sources.
type MCPAuthStatus struct {
	ServerName   string     `json:"server_name"`
	Status       string     `json:"status"`
	AuthType     string     `json:"auth_type,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Refreshable  bool       `json:"refreshable"`
	TokenPresent bool       `json:"token_present"`
	Diagnostic   string     `json:"diagnostic,omitempty"`
}

// EffectiveToolDecision records the combined policy and availability decision.
type EffectiveToolDecision struct {
	VisibleToOperator    bool         `json:"visible_to_operator"`
	VisibleToSession     bool         `json:"visible_to_session"`
	Callable             bool         `json:"callable"`
	ApprovalRequired     bool         `json:"approval_required"`
	SystemPermissionMode string       `json:"system_permission_mode,omitempty"`
	SessionPolicyResult  string       `json:"session_policy_result,omitempty"`
	AgentPolicyResult    string       `json:"agent_policy_result,omitempty"`
	RegistryPolicyResult string       `json:"registry_policy_result,omitempty"`
	SourcePolicyResult   string       `json:"source_policy_result,omitempty"`
	AvailabilityResult   string       `json:"availability_result,omitempty"`
	HookResult           string       `json:"hook_result,omitempty"`
	ReasonCodes          []ReasonCode `json:"reason_codes,omitempty"`
}

// ValidateProvider rejects nil or malformed providers before registry use.
func ValidateProvider(provider Provider) error {
	if isNilInterface(provider) {
		return NewValidationError("provider", ReasonDependencyMissing, "provider is required")
	}
	source := provider.ID()
	if err := source.Validate("provider.id"); err != nil {
		return err
	}
	return nil
}

// ValidateHandle rejects nil or malformed handles before dispatch.
func ValidateHandle(handle Handle) error {
	if isNilInterface(handle) {
		return NewValidationError("handle", ReasonBackendNotExecutable, "handle is required")
	}
	if err := handle.Descriptor().Validate(); err != nil {
		return err
	}
	return nil
}

func cloneRawMessage(src json.RawMessage) json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), src...)
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	return append([]string(nil), src...)
}

func cloneToolsets(src []ToolsetID) []ToolsetID {
	if len(src) == 0 {
		return nil
	}
	return append([]ToolsetID(nil), src...)
}
