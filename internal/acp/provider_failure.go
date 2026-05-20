package acp

import (
	"errors"
	"fmt"
	execpkg "os/exec"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
)

// ProviderFailureKind classifies provider-facing failures into the recovery
// branches operators and agents need at the CLI/API boundary.
type ProviderFailureKind string

const (
	ProviderFailureMissingCLI       ProviderFailureKind = "missing_cli"
	ProviderFailureUnauthenticated  ProviderFailureKind = "unauthenticated"
	ProviderFailureInvalidModel     ProviderFailureKind = "invalid_model"
	ProviderFailureModelUnavailable ProviderFailureKind = "model_unavailable"
	ProviderFailurePermissionDenied ProviderFailureKind = "permission_denied"
	ProviderFailureRateLimited      ProviderFailureKind = "rate_limited"
	ProviderFailureTransient        ProviderFailureKind = "transient"
)

// ProviderFailureAction is the stable next action paired with a provider
// failure classification.
type ProviderFailureAction string

const (
	ProviderFailureActionInstallCLI        ProviderFailureAction = "install_cli"
	ProviderFailureActionLogin             ProviderFailureAction = "login"
	ProviderFailureActionChangeModel       ProviderFailureAction = "change_model"
	ProviderFailureActionRequestPermission ProviderFailureAction = "request_permission"
	ProviderFailureActionWait              ProviderFailureAction = "wait"
	ProviderFailureActionRetry             ProviderFailureAction = "retry"
)

// ProviderFailureDiagnostic is the typed provider-specific recovery metadata
// embedded into redacted session failure summaries.
type ProviderFailureDiagnostic struct {
	Kind     ProviderFailureKind
	Action   ProviderFailureAction
	Guidance string
}

type providerFailurePattern struct {
	kind     ProviderFailureKind
	action   ProviderFailureAction
	guidance string
	needles  []string
}

var providerFailurePatterns = []providerFailurePattern{
	{
		kind:     ProviderFailureMissingCLI,
		action:   ProviderFailureActionInstallCLI,
		guidance: "install the provider CLI and retry",
		needles: []string{
			"executable file not found",
			"executable not found",
			"command not found",
		},
	},
	{
		kind:     ProviderFailureRateLimited,
		action:   ProviderFailureActionWait,
		guidance: "wait for the provider quota or rate-limit window, then retry",
		needles: []string{
			"429",
			"rate_limit",
			"rate limit",
			"too many requests",
			"quota exceeded",
			"insufficient_quota",
		},
	},
	{
		kind:     ProviderFailureUnauthenticated,
		action:   ProviderFailureActionLogin,
		guidance: "run provider auth login for this provider",
		needles: []string{
			"401",
			"unauthorized",
			"authentication required",
			"authentication failed",
			"not logged in",
			"login required",
			"invalid api key",
			"missing api key",
			"no api key",
			"expired token",
		},
	},
	{
		kind:     ProviderFailurePermissionDenied,
		action:   ProviderFailureActionRequestPermission,
		guidance: "request provider or model access before retrying",
		needles: []string{
			"403",
			"forbidden",
			"permission denied",
			"access denied",
			"not entitled",
			"entitlement",
			"does not have access",
			"do not have access",
			"not allowed to access",
		},
	},
	{
		kind:     ProviderFailureInvalidModel,
		action:   ProviderFailureActionChangeModel,
		guidance: "choose a model configured for this provider",
		needles: []string{
			"invalid model",
			"unknown model",
			"model not found",
			"model does not exist",
			"unsupported model",
			"invalid model id",
		},
	},
	{
		kind:     ProviderFailureModelUnavailable,
		action:   ProviderFailureActionChangeModel,
		guidance: "choose an available model or refresh the provider model catalog",
		needles: []string{
			"model unavailable",
			"model is unavailable",
			"model not available",
			"model is not available",
			"not available for this provider",
			"not available in your region",
		},
	},
	{
		kind:     ProviderFailureTransient,
		action:   ProviderFailureActionRetry,
		guidance: "retry after the provider recovers",
		needles: []string{
			"500",
			"502",
			"503",
			"504",
			"529",
			"overloaded",
			"temporarily unavailable",
			"server error",
			"connection reset",
			"jsondecodeerror",
		},
	},
}

// ProviderFailureDiagnosticFromError returns stable provider recovery metadata
// for known ACP/native-provider failure signals.
func ProviderFailureDiagnosticFromError(err error) (ProviderFailureDiagnostic, bool) {
	if err == nil {
		return ProviderFailureDiagnostic{}, false
	}
	if errors.Is(err, execpkg.ErrNotFound) {
		return providerFailureDiagnostic(
			ProviderFailureMissingCLI,
			ProviderFailureActionInstallCLI,
			"install the provider CLI and retry",
		), true
	}
	if reqErr, ok := errors.AsType[*acpsdk.RequestError](err); ok {
		if diagnostic, matched := providerFailureDiagnosticFromRequestError(reqErr); matched {
			return diagnostic, true
		}
	}
	return providerFailureDiagnosticFromText(err.Error())
}

func providerFailureDiagnosticFromRequestError(
	reqErr *acpsdk.RequestError,
) (ProviderFailureDiagnostic, bool) {
	if reqErr == nil {
		return ProviderFailureDiagnostic{}, false
	}
	if reqErr.Code == -32000 {
		return providerFailureDiagnostic(
			ProviderFailureUnauthenticated,
			ProviderFailureActionLogin,
			"run provider auth login for this provider",
		), true
	}
	return providerFailureDiagnosticFromText(requestErrorDiagnosticText(reqErr))
}

func providerFailureDiagnosticFromText(text string) (ProviderFailureDiagnostic, bool) {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return ProviderFailureDiagnostic{}, false
	}
	for _, pattern := range providerFailurePatterns {
		if !providerTextContainsAny(normalized, pattern.needles...) {
			continue
		}
		return providerFailureDiagnostic(pattern.kind, pattern.action, pattern.guidance), true
	}
	return ProviderFailureDiagnostic{}, false
}

func providerFailureDiagnostic(
	kind ProviderFailureKind,
	action ProviderFailureAction,
	guidance string,
) ProviderFailureDiagnostic {
	return ProviderFailureDiagnostic{
		Kind:     kind,
		Action:   action,
		Guidance: strings.TrimSpace(guidance),
	}
}

func providerFailureDiagnosticSummary(err error, summary string) string {
	diagnostic, ok := ProviderFailureDiagnosticFromError(err)
	if !ok {
		return summary
	}
	return diagnostic.Summary(summary)
}

// Summary attaches stable recovery metadata to the redacted failure text.
func (d ProviderFailureDiagnostic) Summary(summary string) string {
	trimmed := strings.TrimSpace(summary)
	if providerTextContainsAny(strings.ToLower(trimmed), "provider_failure_kind=") {
		return trimmed
	}
	metadata := fmt.Sprintf(
		"provider_failure_kind=%s; next_action=%s",
		strings.TrimSpace(string(d.Kind)),
		strings.TrimSpace(string(d.Action)),
	)
	if guidance := strings.TrimSpace(d.Guidance); guidance != "" {
		metadata += "; guidance=" + guidance
	}
	if trimmed == "" {
		return metadata
	}
	return trimmed + "; " + metadata
}

func providerTextContainsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
