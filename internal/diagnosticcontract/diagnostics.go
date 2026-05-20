package diagnosticcontract

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// DiagnosticItem is the canonical actionable-diagnostic wire shape.
type DiagnosticItem struct {
	ID               string         `json:"id"`
	Code             string         `json:"code"`
	Severity         string         `json:"severity"`
	Category         string         `json:"category"`
	Title            string         `json:"title"`
	Message          string         `json:"message"`
	SuggestedCommand string         `json:"suggested_command,omitempty"`
	DocURL           string         `json:"doc_url,omitempty"`
	DataFreshness    string         `json:"data_freshness"`
	Evidence         map[string]any `json:"evidence,omitempty"`
}

const (
	SeverityOK       = "ok"
	SeverityInfo     = "info"
	SeverityWarn     = "warn"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

const (
	FreshnessLive    = "live"
	FreshnessOffline = "offline"
	FreshnessStale   = "stale"
)

const (
	CategoryProvider   = "provider"
	CategoryDaemon     = "daemon"
	CategoryConfig     = "config"
	CategoryVault      = "vault"
	CategoryMCP        = "mcp"
	CategoryBridge     = "bridge"
	CategoryExtension  = "extension"
	CategorySession    = "session"
	CategoryTask       = "task"
	CategoryHome       = "home"
	CategorySecrets    = "secrets"
	CategoryMigrations = "migrations"
	CategoryNetwork    = "network"
)

const (
	CodeBinaryVersionMismatch         = "binary_version_mismatch"
	CodeBridgeHealthUnavailable       = "bridge_health_unavailable"
	CodeBridgeNotFound                = "bridge_not_found"
	CodeBridgeNotificationSuppressed  = "bridge_notification_suppressed"
	CodeBridgeTargetUnavailable       = "bridge_target_unavailable"
	CodeBulkTooLarge                  = "bulk_too_large"
	CodeBundleConsentRequired         = "bundle_consent_required"
	CodeBundlePartialFailure          = "bundle_partial_failure"
	CodeBundleSizeExceeded            = "bundle_size_exceeded"
	CodeConfigActiveSessionsBlock     = "config_active_sessions_block"
	CodeConfigApplyUnsupported        = "config_apply_unsupported"
	CodeConfigDriftPresent            = "config_drift_present"
	CodeConfigDriftStale              = "config_drift_stale"
	CodeConfigInvalid                 = "config_invalid"
	CodeConfigPartialFailure          = "config_partial_failure"
	CodeConfigReloadTimeout           = "config_reload_timeout"
	CodeConfigRestartRequired         = "config_restart_required"
	CodeConfigValidateFailed          = "config_validate_failed"
	CodeCursorConflict                = "cursor_conflict"
	CodeDaemonHealthUnavailable       = "daemon_health_unavailable"
	CodeDaemonStateSuspect            = "daemon_state_suspect"
	CodeDaemonUnavailable             = "daemon_unavailable"
	CodeDiskWriteFailed               = "disk_write_failed"
	CodeExtensionBlockedByBundle      = "extension_blocked_by_bundle"
	CodeExtensionChecksumUnverified   = "extension_checksum_unverified"
	CodeExtensionInstallFailed        = "extension_install_failed"
	CodeExtensionInUse                = "extension_in_use"
	CodeExtensionNotFound             = "extension_not_found"
	CodeFlagNotApplicable             = "flag_not_applicable"
	CodeForbiddenOperatorAction       = "forbidden_operator_action"
	CodeForceOpRateLimited            = "force_op_rate_limited"
	CodeForceOpRequiresReason         = "force_op_requires_reason"
	CodeHomeDiskSpaceCritical         = "home_disk_space_critical"
	CodeHomeDiskSpaceLow              = "home_disk_space_low"
	CodeHomePathMissing               = "home_path_missing"
	CodeHomePermsWrong                = "home_perms_wrong"
	CodeIDFormatUnknown               = "id_format_unknown"
	CodeIdentityLookupUnavailable     = "identity_lookup_unavailable"
	CodeIdentityMismatch              = "identity_mismatch"
	CodeIdentityRequired              = "identity_required"
	CodeIdentityStale                 = "identity_stale"
	CodeIdentityUnauthorized          = "identity_unauthorized"
	CodeMarketplaceUnavailable        = "marketplace_unavailable"
	CodeMCPAuthRequired               = "mcp_auth_required"
	CodeMCPServerUnavailable          = "mcp_server_unavailable"
	CodeMigrationsPending             = "migrations_pending"
	CodePresetBuiltinProtected        = "preset_builtin_protected"
	CodePresetDuplicateName           = "preset_duplicate_name"
	CodePresetFilterInvalid           = "preset_filter_invalid"
	CodePresetNotFound                = "preset_not_found"
	CodeProbeFailed                   = "probe_failed"
	CodeProbeTimeout                  = "probe_timeout"
	CodeProviderClassificationUnknown = "provider_classification_unknown"
	CodeProviderCLIMissing            = "provider_cli_missing"
	CodeProviderCredentialUnresolved  = "provider_credential_unresolved"
	CodeProviderLoginRequiresLocalTTY = "provider_login_requires_local_tty"
	CodeProviderNotAuthenticated      = "provider_not_authenticated"
	CodeProviderNotInstalled          = "provider_not_installed"
	CodeProviderPermissionDenied      = "provider_permission_denied"
	CodeProviderRateLimited           = "provider_rate_limited"
	CodeProviderTransientFailure      = "provider_transient_failure"
	CodeRetryChainTooDeep             = "retry_chain_too_deep"
	CodeSchedulerPaused               = "scheduler_paused"
	// #nosec G101 -- diagnostic code label, not credential material.
	CodeSecretsPermsWrong      = "secrets_perms_wrong"
	CodeSessionBusy            = "session_busy"
	CodeSessionLocked          = "session_locked"
	CodeSessionQueueFull       = "session_queue_full"
	CodeSessionResumeAmbiguous = "session_resume_ambiguous"
	CodeSkillNotFound          = "skill_not_found"
	CodeSocketPathUnwritable   = "socket_path_unwritable"
	CodeTargetAmbiguous        = "target_ambiguous"
	CodeTargetUnknown          = "target_unknown"
	CodeTaskRunAlreadyTerminal = "task_run_already_terminal"
	CodeTaskRunCrashed         = "task_run_crashed"
	CodeTaskRunNotReleasable   = "task_run_not_releasable"
	CodeTaskRunOrphan          = "task_run_orphan"
	CodeTaskRunStaleLease      = "task_run_stale_lease"
	CodeTaskRunStillActive     = "task_run_still_active"
	CodeTaskRunStranded        = "task_run_stranded"
	CodeTaskRunStuck           = "task_run_stuck"
	CodeUnknownActorFormat     = "unknown_actor_format"
	CodeUnknownComponent       = "unknown_component"
	CodeVaultRefUnresolved     = "vault_ref_unresolved"
)

// DiagnosticCodeSpec records the canonical owner metadata for one code.
type DiagnosticCodeSpec struct {
	Code     string
	Category string
}

var diagnosticCodeSpecs = []DiagnosticCodeSpec{
	{Code: CodeBinaryVersionMismatch, Category: CategoryHome},
	{Code: CodeBridgeHealthUnavailable, Category: CategoryBridge},
	{Code: CodeBridgeNotFound, Category: CategoryBridge},
	{Code: CodeBridgeNotificationSuppressed, Category: CategoryBridge},
	{Code: CodeBridgeTargetUnavailable, Category: CategoryBridge},
	{Code: CodeBulkTooLarge, Category: CategoryTask},
	{Code: CodeBundleConsentRequired, Category: CategoryDaemon},
	{Code: CodeBundlePartialFailure, Category: CategoryDaemon},
	{Code: CodeBundleSizeExceeded, Category: CategoryDaemon},
	{Code: CodeConfigActiveSessionsBlock, Category: CategoryConfig},
	{Code: CodeConfigApplyUnsupported, Category: CategoryConfig},
	{Code: CodeConfigDriftPresent, Category: CategoryConfig},
	{Code: CodeConfigDriftStale, Category: CategoryConfig},
	{Code: CodeConfigInvalid, Category: CategoryConfig},
	{Code: CodeConfigPartialFailure, Category: CategoryConfig},
	{Code: CodeConfigReloadTimeout, Category: CategoryConfig},
	{Code: CodeConfigRestartRequired, Category: CategoryConfig},
	{Code: CodeConfigValidateFailed, Category: CategoryConfig},
	{Code: CodeCursorConflict, Category: CategoryDaemon},
	{Code: CodeDaemonHealthUnavailable, Category: CategoryDaemon},
	{Code: CodeDaemonStateSuspect, Category: CategoryDaemon},
	{Code: CodeDaemonUnavailable, Category: CategoryDaemon},
	{Code: CodeDiskWriteFailed, Category: CategoryDaemon},
	{Code: CodeExtensionBlockedByBundle, Category: CategoryExtension},
	{Code: CodeExtensionChecksumUnverified, Category: CategoryExtension},
	{Code: CodeExtensionInstallFailed, Category: CategoryExtension},
	{Code: CodeExtensionInUse, Category: CategoryExtension},
	{Code: CodeExtensionNotFound, Category: CategoryExtension},
	{Code: CodeFlagNotApplicable, Category: CategoryDaemon},
	{Code: CodeForbiddenOperatorAction, Category: CategoryTask},
	{Code: CodeForceOpRateLimited, Category: CategoryTask},
	{Code: CodeForceOpRequiresReason, Category: CategoryTask},
	{Code: CodeHomeDiskSpaceCritical, Category: CategoryHome},
	{Code: CodeHomeDiskSpaceLow, Category: CategoryHome},
	{Code: CodeHomePathMissing, Category: CategoryHome},
	{Code: CodeHomePermsWrong, Category: CategoryHome},
	{Code: CodeIDFormatUnknown, Category: CategoryTask},
	{Code: CodeIdentityLookupUnavailable, Category: CategorySession},
	{Code: CodeIdentityMismatch, Category: CategorySession},
	{Code: CodeIdentityRequired, Category: CategorySession},
	{Code: CodeIdentityStale, Category: CategorySession},
	{Code: CodeIdentityUnauthorized, Category: CategorySession},
	{Code: CodeMarketplaceUnavailable, Category: CategoryExtension},
	{Code: CodeMCPAuthRequired, Category: CategoryMCP},
	{Code: CodeMCPServerUnavailable, Category: CategoryMCP},
	{Code: CodeMigrationsPending, Category: CategoryMigrations},
	{Code: CodePresetBuiltinProtected, Category: CategoryBridge},
	{Code: CodePresetDuplicateName, Category: CategoryBridge},
	{Code: CodePresetFilterInvalid, Category: CategoryBridge},
	{Code: CodePresetNotFound, Category: CategoryBridge},
	{Code: CodeProbeFailed, Category: CategoryDaemon},
	{Code: CodeProbeTimeout, Category: CategoryDaemon},
	{Code: CodeProviderClassificationUnknown, Category: CategoryProvider},
	{Code: CodeProviderCLIMissing, Category: CategoryProvider},
	{Code: CodeProviderCredentialUnresolved, Category: CategoryProvider},
	{Code: CodeProviderLoginRequiresLocalTTY, Category: CategoryProvider},
	{Code: CodeProviderNotAuthenticated, Category: CategoryProvider},
	{Code: CodeProviderNotInstalled, Category: CategoryProvider},
	{Code: CodeProviderPermissionDenied, Category: CategoryProvider},
	{Code: CodeProviderRateLimited, Category: CategoryProvider},
	{Code: CodeProviderTransientFailure, Category: CategoryProvider},
	{Code: CodeRetryChainTooDeep, Category: CategoryTask},
	{Code: CodeSchedulerPaused, Category: CategoryTask},
	{Code: CodeSecretsPermsWrong, Category: CategorySecrets},
	{Code: CodeSessionBusy, Category: CategorySession},
	{Code: CodeSessionLocked, Category: CategorySession},
	{Code: CodeSessionQueueFull, Category: CategorySession},
	{Code: CodeSessionResumeAmbiguous, Category: CategorySession},
	{Code: CodeSkillNotFound, Category: CategoryExtension},
	{Code: CodeSocketPathUnwritable, Category: CategoryDaemon},
	{Code: CodeTargetAmbiguous, Category: CategoryBridge},
	{Code: CodeTargetUnknown, Category: CategoryBridge},
	{Code: CodeTaskRunAlreadyTerminal, Category: CategoryTask},
	{Code: CodeTaskRunCrashed, Category: CategoryTask},
	{Code: CodeTaskRunNotReleasable, Category: CategoryTask},
	{Code: CodeTaskRunOrphan, Category: CategoryTask},
	{Code: CodeTaskRunStaleLease, Category: CategoryTask},
	{Code: CodeTaskRunStillActive, Category: CategoryTask},
	{Code: CodeTaskRunStranded, Category: CategoryTask},
	{Code: CodeTaskRunStuck, Category: CategoryTask},
	{Code: CodeUnknownActorFormat, Category: CategoryDaemon},
	{Code: CodeUnknownComponent, Category: CategoryDaemon},
	{Code: CodeVaultRefUnresolved, Category: CategoryVault},
}

var (
	diagnosticSeveritySet  = setFromValues(SeverityOK, SeverityInfo, SeverityWarn, SeverityError, SeverityCritical)
	diagnosticFreshnessSet = setFromValues(FreshnessLive, FreshnessOffline, FreshnessStale)
	diagnosticCategorySet  = setFromValues(
		CategoryProvider,
		CategoryDaemon,
		CategoryConfig,
		CategoryVault,
		CategoryMCP,
		CategoryBridge,
		CategoryExtension,
		CategorySession,
		CategoryTask,
		CategoryHome,
		CategorySecrets,
		CategoryMigrations,
		CategoryNetwork,
	)
	diagnosticCodeCategoryMap = categoryMapFromCodeSpecs(diagnosticCodeSpecs)
)

func setFromValues(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func categoryMapFromCodeSpecs(specs []DiagnosticCodeSpec) map[string]string {
	out := make(map[string]string, len(specs))
	for _, spec := range specs {
		out[spec.Code] = spec.Category
	}
	return out
}

// DiagnosticCodeSpecs returns the sorted canonical diagnostic code registry.
func DiagnosticCodeSpecs() []DiagnosticCodeSpec {
	specs := append([]DiagnosticCodeSpec(nil), diagnosticCodeSpecs...)
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Code < specs[j].Code
	})
	return specs
}

// DiagnosticCodes returns all registered deterministic diagnostic codes.
func DiagnosticCodes() []string {
	codes := make([]string, 0, len(diagnosticCodeSpecs))
	for _, spec := range diagnosticCodeSpecs {
		codes = append(codes, spec.Code)
	}
	sort.Strings(codes)
	return codes
}

// IsDiagnosticCode reports whether code belongs to the canonical registry.
func IsDiagnosticCode(code string) bool {
	_, ok := DiagnosticCodeCategory(code)
	return ok
}

// DiagnosticCodeCategory returns the canonical category for code.
func DiagnosticCodeCategory(code string) (string, bool) {
	category, ok := diagnosticCodeCategoryMap[strings.TrimSpace(code)]
	return category, ok
}

// IsDiagnosticSeverity reports whether severity belongs to the closed set.
func IsDiagnosticSeverity(severity string) bool {
	_, ok := diagnosticSeveritySet[strings.TrimSpace(severity)]
	return ok
}

// IsDiagnosticFreshness reports whether freshness belongs to the closed set.
func IsDiagnosticFreshness(freshness string) bool {
	_, ok := diagnosticFreshnessSet[strings.TrimSpace(freshness)]
	return ok
}

// IsDiagnosticCategory reports whether category belongs to the closed set.
func IsDiagnosticCategory(category string) bool {
	_, ok := diagnosticCategorySet[strings.TrimSpace(category)]
	return ok
}

// ValidateDiagnosticItem checks the public contract without mutating it.
func ValidateDiagnosticItem(item DiagnosticItem) error {
	var errs []error
	if strings.TrimSpace(item.ID) == "" {
		errs = append(errs, errors.New("id is required"))
	}
	if category, ok := DiagnosticCodeCategory(item.Code); !ok {
		errs = append(errs, fmt.Errorf("unknown diagnostic code %q", item.Code))
	} else if strings.TrimSpace(item.Category) != category {
		errs = append(errs, fmt.Errorf(
			"diagnostic code %q category = %q, want %q",
			item.Code,
			item.Category,
			category,
		))
	}
	if !IsDiagnosticSeverity(item.Severity) {
		errs = append(errs, fmt.Errorf("unknown diagnostic severity %q", item.Severity))
	}
	if !IsDiagnosticCategory(item.Category) {
		errs = append(errs, fmt.Errorf("unknown diagnostic category %q", item.Category))
	}
	if !IsDiagnosticFreshness(item.DataFreshness) {
		errs = append(errs, fmt.Errorf("unknown diagnostic freshness %q", item.DataFreshness))
	}
	if strings.TrimSpace(item.Title) == "" {
		errs = append(errs, errors.New("title is required"))
	}
	if strings.TrimSpace(item.Message) == "" {
		errs = append(errs, errors.New("message is required"))
	}
	return errors.Join(errs...)
}

// ValidateDiagnosticRegistry checks duplicate codes and unknown categories.
func ValidateDiagnosticRegistry() error {
	seen := make(map[string]struct{}, len(diagnosticCodeSpecs))
	var errs []error
	for _, spec := range diagnosticCodeSpecs {
		code := strings.TrimSpace(spec.Code)
		switch code {
		case "":
			errs = append(errs, errors.New("diagnostic code is empty"))
		default:
			if _, exists := seen[code]; exists {
				errs = append(errs, fmt.Errorf("duplicate diagnostic code %q", code))
			}
		}
		seen[code] = struct{}{}
		if !IsDiagnosticCategory(spec.Category) {
			errs = append(errs, fmt.Errorf("diagnostic code %q has unknown category %q", code, spec.Category))
		}
	}
	return errors.Join(errs...)
}
