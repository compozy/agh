package acp

import (
	"context"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/compozy/agh/internal/config"
)

func (d *Driver) applySessionMode(
	ctx context.Context,
	process *AgentProcess,
	permissions aghconfig.PermissionMode,
) error {
	if ctx == nil || process == nil || process.conn == nil {
		return nil
	}

	modeID := preferredSessionMode(process.CapsSnapshot().SupportedModes, permissions, process.toolGateway != nil)
	if modeID == "" {
		return nil
	}

	_, err := acpsdk.SendRequest[acpsdk.SetSessionModeResponse](
		process.conn,
		ctx,
		acpsdk.AgentMethodSessionSetMode,
		acpsdk.SetSessionModeRequest{
			SessionId: acpsdk.SessionId(process.SessionID),
			ModeId:    acpsdk.SessionModeId(modeID),
		},
	)
	return err
}

func (d *Driver) applySessionModel(ctx context.Context, process *AgentProcess, preferredModel string) error {
	if ctx == nil || process == nil || process.conn == nil {
		return nil
	}
	modelID := strings.TrimSpace(preferredModel)
	if modelID == "" {
		return nil
	}

	caps := process.CapsSnapshot()
	if option, ok := findModelConfigOption(caps.ConfigOptions); ok {
		if !configOptionAllowsValue(option, modelID) {
			return newNegotiationError(
				NegotiationCodeModelUnavailable,
				"model",
				modelID,
				option.ID,
				configOptionChoices(option),
				nil,
			)
		}
		if err := d.applySessionConfigOption(ctx, process, option.ID, modelID); err != nil {
			return newNegotiationError(
				NegotiationCodeModelUnavailable,
				"model",
				modelID,
				option.ID,
				configOptionChoices(option),
				err,
			)
		}
		return nil
	}

	return newNegotiationError(
		NegotiationCodeModelUnavailable,
		"model",
		modelID,
		"",
		nil,
		errModelConfigOptionRequired,
	)
}

func (d *Driver) applySessionReasoningEffort(ctx context.Context, process *AgentProcess, effort string) error {
	if ctx == nil || process == nil || process.conn == nil {
		return nil
	}
	effortID := strings.TrimSpace(effort)
	if effortID == "" {
		return nil
	}

	caps := process.CapsSnapshot()
	option, ok := findReasoningConfigOption(caps.ConfigOptions)
	if !ok {
		return newNegotiationError(
			NegotiationCodeReasoningOptionMissing,
			"reasoning effort",
			effortID,
			"",
			nil,
			nil,
		)
	}
	if !configOptionAllowsValue(option, effortID) {
		return newNegotiationError(
			NegotiationCodeReasoningEffortUnsupported,
			"reasoning effort",
			effortID,
			option.ID,
			configOptionChoices(option),
			nil,
		)
	}
	if err := d.applySessionConfigOption(ctx, process, option.ID, effortID); err != nil {
		return newNegotiationError(
			NegotiationCodeReasoningEffortUnsupported,
			"reasoning effort",
			effortID,
			option.ID,
			configOptionChoices(option),
			err,
		)
	}
	return nil
}

func (d *Driver) applySessionConfigOption(
	ctx context.Context,
	process *AgentProcess,
	optionID string,
	valueID string,
) error {
	response, err := acpsdk.SendRequest[acpsdk.SetSessionConfigOptionResponse](
		process.conn,
		ctx,
		acpsdk.AgentMethodSessionSetConfigOption,
		acpsdk.SetSessionConfigOptionRequest{
			ValueId: &acpsdk.SetSessionConfigOptionValueId{
				SessionId: acpsdk.SessionId(process.SessionID),
				ConfigId:  acpsdk.SessionConfigId(strings.TrimSpace(optionID)),
				Value:     acpsdk.SessionConfigValueId(strings.TrimSpace(valueID)),
			},
		},
	)
	if err != nil {
		return err
	}
	process.setConfigOptions(sessionConfigOptionsFromSDK(response.ConfigOptions))
	return nil
}

func preferredSessionMode(
	supported []string,
	permissions aghconfig.PermissionMode,
	toolGatewayEnabled bool,
) string {
	if len(supported) == 0 {
		return ""
	}

	lookup := make(map[string]string, len(supported))
	for _, mode := range supported {
		trimmed := strings.TrimSpace(mode)
		if trimmed == "" {
			continue
		}
		lookup[strings.ToLower(trimmed)] = trimmed
	}

	if toolGatewayEnabled {
		for _, candidate := range permissionGatewayModeCandidates() {
			if matched, ok := lookup[strings.ToLower(candidate)]; ok {
				return matched
			}
		}
	}

	for _, candidate := range sessionModeCandidates(permissions) {
		if matched, ok := lookup[strings.ToLower(candidate)]; ok {
			return matched
		}
	}
	return ""
}

func permissionGatewayModeCandidates() []string {
	return []string{
		clientDefaultKey,
		"ask",
	}
}

func sessionModeCandidates(permissions aghconfig.PermissionMode) []string {
	switch permissions {
	case aghconfig.PermissionModeApproveAll:
		return []string{
			"full-access",
			"full_access",
			"bypassPermissions",
			"bypass_permissions",
			"auto",
			"acceptEdits",
		}
	case aghconfig.PermissionModeApproveReads, aghconfig.PermissionModeDenyAll:
		return []string{
			"read-only",
			"read_only",
			"readOnly",
			"plan",
			"ask",
		}
	default:
		return nil
	}
}
