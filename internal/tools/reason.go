package tools

// ReasonCode is a deterministic machine-readable reason.
type ReasonCode string

const (
	// ReasonIDEmpty reports an empty id.
	ReasonIDEmpty ReasonCode = "id_empty"
	// ReasonIDEmptySegment reports a missing namespace segment.
	ReasonIDEmptySegment ReasonCode = "id_empty_segment"
	// ReasonIDInvalidFormat reports a grammar violation.
	ReasonIDInvalidFormat ReasonCode = "id_invalid_format"
	// ReasonIDReservedConflict reports ambiguous reserved separator usage.
	ReasonIDReservedConflict ReasonCode = "reserved_conflict"
	// ReasonReservedNamespace reports an extension or external source claiming a reserved namespace.
	ReasonReservedNamespace ReasonCode = "reserved_namespace"
	// ReasonIDTooLong reports an id over the provider-safe limit.
	ReasonIDTooLong ReasonCode = "id_too_long"
	// ReasonDependencyMissing reports a missing dependency.
	ReasonDependencyMissing ReasonCode = "dependency_missing"
	// ReasonBackendUnhealthy reports an unhealthy backend.
	ReasonBackendUnhealthy ReasonCode = "backend_unhealthy"
	// ReasonBackendNotExecutable reports a descriptor without an executable backend.
	ReasonBackendNotExecutable ReasonCode = "backend_not_executable"
	// ReasonExtensionInactive reports an inactive extension.
	ReasonExtensionInactive ReasonCode = "extension_inactive"
	// ReasonExtensionRuntimeMismatch reports a manifest/runtime mismatch.
	ReasonExtensionRuntimeMismatch ReasonCode = "extension_runtime_mismatch"
	// ReasonExtensionCapabilityMissing reports a missing extension capability.
	ReasonExtensionCapabilityMissing ReasonCode = "extension_capability_missing"
	// ReasonRuntimeDescriptorMissing reports a missing runtime descriptor.
	ReasonRuntimeDescriptorMissing ReasonCode = "runtime_descriptor_missing"
	// ReasonRuntimeDescriptorMismatch reports a runtime descriptor mismatch.
	ReasonRuntimeDescriptorMismatch ReasonCode = "runtime_descriptor_mismatch"
	// ReasonHandlerMissing reports a missing extension handler.
	ReasonHandlerMissing ReasonCode = "handler_missing"
	// ReasonMCPUnreachable reports an unreachable MCP server.
	ReasonMCPUnreachable ReasonCode = "mcp_unreachable"
	// ReasonMCPAuthUnconfigured reports missing MCP auth configuration.
	ReasonMCPAuthUnconfigured ReasonCode = "mcp_auth_unconfigured"
	// ReasonMCPAuthRequired reports that MCP login is required.
	ReasonMCPAuthRequired ReasonCode = "mcp_auth_required"
	// ReasonMCPAuthExpired reports expired MCP auth.
	ReasonMCPAuthExpired ReasonCode = "mcp_auth_expired"
	// ReasonMCPAuthInvalid reports invalid MCP auth.
	ReasonMCPAuthInvalid ReasonCode = "mcp_auth_invalid"
	// ReasonMCPAuthRefreshFailed reports failed MCP auth refresh.
	ReasonMCPAuthRefreshFailed ReasonCode = "mcp_auth_refresh_failed"
	// ReasonSourceDisabled reports a disabled source.
	ReasonSourceDisabled ReasonCode = "source_disabled"
	// ReasonPolicyDenied reports a policy denial.
	ReasonPolicyDenied ReasonCode = "policy_denied"
	// ReasonVisibilityDenied reports a descriptor hidden from a scoped projection.
	ReasonVisibilityDenied ReasonCode = "visibility_denied"
	// ReasonApprovalRequired reports required approval.
	ReasonApprovalRequired ReasonCode = "approval_required"
	// ReasonApprovalUnreachable reports no available approval channel.
	ReasonApprovalUnreachable ReasonCode = "approval_unreachable"
	// ReasonApprovalTimedOut reports approval timeout.
	ReasonApprovalTimedOut ReasonCode = "approval_timed_out"
	// ReasonApprovalCanceled reports approval cancellation.
	ReasonApprovalCanceled ReasonCode = "approval_canceled"
	// ReasonApprovalTokenMissing reports a missing local approval token.
	ReasonApprovalTokenMissing ReasonCode = "approval_token_missing"
	// ReasonApprovalTokenExpired reports an expired local approval token.
	ReasonApprovalTokenExpired ReasonCode = "approval_token_expired"
	// ReasonApprovalTokenMismatch reports a local approval token binding mismatch.
	ReasonApprovalTokenMismatch ReasonCode = "approval_token_mismatch"
	// ReasonApprovalTokenReplayed reports a replayed local approval token.
	ReasonApprovalTokenReplayed ReasonCode = "approval_token_replayed"
	// ReasonSessionDenied reports session lineage denial.
	ReasonSessionDenied ReasonCode = "session_denied"
	// ReasonHookDenied reports hook denial.
	ReasonHookDenied ReasonCode = "hook_denied"
	// ReasonSchemaInvalid reports invalid JSON schema.
	ReasonSchemaInvalid ReasonCode = "schema_invalid"
	// ReasonConflictedID reports a canonical id conflict.
	ReasonConflictedID ReasonCode = "conflicted_id"
	// ReasonConflictedSanitizedName reports an external-name sanitization conflict.
	ReasonConflictedSanitizedName ReasonCode = "conflicted_sanitized_name"
	// ReasonResultBudgetExceeded reports a result budget violation.
	ReasonResultBudgetExceeded ReasonCode = "result_budget_exceeded"
	// ReasonCallCanceled reports dispatch cancellation.
	ReasonCallCanceled ReasonCode = "call_canceled"
	// ReasonCallTimedOut reports dispatch deadline expiration.
	ReasonCallTimedOut ReasonCode = "call_timed_out"
	// ReasonSecretMetadata reports sensitive metadata in a public envelope.
	ReasonSecretMetadata ReasonCode = "secret_metadata"
	// ReasonToolsetUnknown reports a policy reference to an unknown toolset.
	ReasonToolsetUnknown ReasonCode = "toolset_unknown"
	// ReasonToolsetCycle reports recursive toolset membership.
	ReasonToolsetCycle ReasonCode = "toolset_cycle"
	// ReasonToolUnknown reports a policy reference to an unknown tool.
	ReasonToolUnknown ReasonCode = "tool_unknown"
)

var validReasonCodes = map[ReasonCode]struct{}{
	ReasonIDEmpty:                    {},
	ReasonIDEmptySegment:             {},
	ReasonIDInvalidFormat:            {},
	ReasonIDReservedConflict:         {},
	ReasonReservedNamespace:          {},
	ReasonIDTooLong:                  {},
	ReasonDependencyMissing:          {},
	ReasonBackendUnhealthy:           {},
	ReasonBackendNotExecutable:       {},
	ReasonExtensionInactive:          {},
	ReasonExtensionRuntimeMismatch:   {},
	ReasonExtensionCapabilityMissing: {},
	ReasonRuntimeDescriptorMissing:   {},
	ReasonRuntimeDescriptorMismatch:  {},
	ReasonHandlerMissing:             {},
	ReasonMCPUnreachable:             {},
	ReasonMCPAuthUnconfigured:        {},
	ReasonMCPAuthRequired:            {},
	ReasonMCPAuthExpired:             {},
	ReasonMCPAuthInvalid:             {},
	ReasonMCPAuthRefreshFailed:       {},
	ReasonSourceDisabled:             {},
	ReasonPolicyDenied:               {},
	ReasonVisibilityDenied:           {},
	ReasonApprovalRequired:           {},
	ReasonApprovalUnreachable:        {},
	ReasonApprovalTimedOut:           {},
	ReasonApprovalCanceled:           {},
	ReasonApprovalTokenMissing:       {},
	ReasonApprovalTokenExpired:       {},
	ReasonApprovalTokenMismatch:      {},
	ReasonApprovalTokenReplayed:      {},
	ReasonSessionDenied:              {},
	ReasonHookDenied:                 {},
	ReasonSchemaInvalid:              {},
	ReasonConflictedID:               {},
	ReasonConflictedSanitizedName:    {},
	ReasonResultBudgetExceeded:       {},
	ReasonCallCanceled:               {},
	ReasonCallTimedOut:               {},
	ReasonSecretMetadata:             {},
	ReasonToolsetUnknown:             {},
	ReasonToolsetCycle:               {},
	ReasonToolUnknown:                {},
}

// Validate ensures the reason code is documented.
func (r ReasonCode) Validate(field string) error {
	if _, ok := validReasonCodes[r]; ok {
		return nil
	}
	return NewValidationError(field, ReasonPolicyDenied, "unsupported reason code")
}
