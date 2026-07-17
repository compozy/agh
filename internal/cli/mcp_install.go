package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	mcpOAuthValueInputFlag    = "oauth-client-secret-value"
	mcpOAuthVaultRefInputFlag = "oauth-client-secret-vault-ref"
)

func newMCPInstallCommand(deps commandDeps) *cobra.Command {
	var (
		name                      string
		scope                     string
		workspaceID               string
		values                    []string
		vaultRefs                 []string
		oauthClientSecretValue    string
		oauthClientSecretVaultRef string
	)
	cmd := &cobra.Command{
		Use:   installCommandKey + " <entry>",
		Short: "Install a curated MCP server",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := mcpInstallEnvInputs(values, vaultRefs)
			if err != nil {
				return err
			}
			oauthClientSecret, err := mcpInstallOAuthClientSecretInput(
				cmd.Flags().Changed(mcpOAuthValueInputFlag),
				oauthClientSecretValue,
				cmd.Flags().Changed(mcpOAuthVaultRefInputFlag),
				oauthClientSecretVaultRef,
			)
			if err != nil {
				return err
			}
			request := InstallSettingsMCPServerRequest{
				EntryID:     strings.TrimSpace(args[0]),
				Name:        strings.TrimSpace(name),
				Scope:       contract.SettingsWorkspaceScopeKind(strings.TrimSpace(scope)),
				WorkspaceID: strings.TrimSpace(workspaceID),
				Values: &contract.SettingsMCPCatalogInstallValuesPayload{
					Env:               env,
					OAuthClientSecret: oauthClientSecret,
				},
			}
			if err := validateMCPInstallScope(request.Scope, request.WorkspaceID); err != nil {
				return err
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.InstallSettingsMCPServer(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, mcpInstallBundle(&response))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Override the installed MCP server name")
	cmd.Flags().StringVar(
		&scope,
		mcpScopeKey,
		string(contract.SettingsWorkspaceScopeGlobal),
		"Install scope: global or workspace",
	)
	cmd.Flags().StringVar(&workspaceID, "workspace", "", "Workspace ID for workspace scope")
	cmd.Flags().StringArrayVar(&values, "set", nil, "Set a feed field as KEY=VALUE")
	cmd.Flags().StringArrayVar(&vaultRefs, "vault-ref", nil, "Bind a feed field as KEY=vault:mcp/...")
	cmd.Flags().StringVar(
		&oauthClientSecretValue,
		mcpOAuthValueInputFlag,
		"",
		"Set the write-only OAuth client secret",
	)
	cmd.Flags().StringVar(
		&oauthClientSecretVaultRef,
		mcpOAuthVaultRefInputFlag,
		"",
		"Bind the OAuth client secret to an existing vault:mcp/... ref",
	)
	return cmd
}

func mcpInstallOAuthClientSecretInput(
	valueSet bool,
	value string,
	vaultRefSet bool,
	vaultRef string,
) (*contract.SettingsMCPSecretInputPayload, error) {
	if valueSet && vaultRefSet {
		return nil, errors.New("cli: OAuth client secret is assigned more than once")
	}
	if valueSet {
		if strings.TrimSpace(value) == "" {
			return nil, errors.New("cli: --oauth-client-secret-value requires a non-blank value")
		}
		return &contract.SettingsMCPSecretInputPayload{Value: value}, nil
	}
	if vaultRefSet {
		trimmedRef := strings.TrimSpace(vaultRef)
		if trimmedRef == "" {
			return nil, errors.New("cli: --oauth-client-secret-vault-ref requires a non-blank value")
		}
		return &contract.SettingsMCPSecretInputPayload{VaultRef: trimmedRef}, nil
	}
	return nil, nil
}

func mcpInstallEnvInputs(
	values []string,
	vaultRefs []string,
) (map[string]contract.SettingsMCPSecretInputPayload, error) {
	inputs := make(map[string]contract.SettingsMCPSecretInputPayload, len(values)+len(vaultRefs))
	for _, assignment := range values {
		key, value, err := parseMCPInstallAssignment("--set", assignment)
		if err != nil {
			return nil, err
		}
		if _, exists := inputs[key]; exists {
			return nil, fmt.Errorf("cli: MCP install field %q is assigned more than once", key)
		}
		inputs[key] = contract.SettingsMCPSecretInputPayload{Value: value}
	}
	for _, assignment := range vaultRefs {
		key, ref, err := parseMCPInstallAssignment("--vault-ref", assignment)
		if err != nil {
			return nil, err
		}
		if _, exists := inputs[key]; exists {
			return nil, fmt.Errorf("cli: MCP install field %q is assigned more than once", key)
		}
		inputs[key] = contract.SettingsMCPSecretInputPayload{VaultRef: strings.TrimSpace(ref)}
	}
	if len(inputs) == 0 {
		return nil, nil
	}
	return inputs, nil
}

func parseMCPInstallAssignment(flag string, assignment string) (string, string, error) {
	key, value, ok := strings.Cut(assignment, "=")
	key = strings.TrimSpace(key)
	if !ok || key == "" || strings.TrimSpace(value) == "" {
		return "", "", fmt.Errorf("cli: %s requires KEY=VALUE", flag)
	}
	return key, value, nil
}

func validateMCPInstallScope(scope contract.SettingsWorkspaceScopeKind, workspaceID string) error {
	switch scope {
	case contract.SettingsWorkspaceScopeGlobal:
		if strings.TrimSpace(workspaceID) != "" {
			return errors.New("cli: --workspace requires --scope workspace")
		}
	case contract.SettingsWorkspaceScopeWorkspace:
		if strings.TrimSpace(workspaceID) == "" {
			return errors.New("cli: --scope workspace requires --workspace")
		}
	default:
		return fmt.Errorf("cli: unsupported MCP install scope %q", scope)
	}
	return nil
}

func mcpInstallBundle(response *InstallSettingsMCPServerRecord) outputBundle {
	return outputBundle{
		jsonValue: response,
		human: func() (string, error) {
			refs := make([]string, 0, len(response.MCPServer.SecretEnv))
			for key, ref := range response.MCPServer.SecretEnv {
				refs = append(refs, key+"="+ref)
			}
			sort.Strings(refs)
			return renderHumanSection("MCP Install", []keyValue{
				{Label: automationNameValue, Value: response.MCPServer.Name},
				{Label: "Transport", Value: response.MCPServer.Transport},
				{Label: mcpScopeValue, Value: string(response.MCPServer.Scope)},
				{Label: "Catalog Entry", Value: response.MCPServer.CatalogEntry},
				{Label: "Catalog Version", Value: response.MCPServer.CatalogVersion},
				{Label: "Secret Refs", Value: stringOrDash(strings.Join(refs, ", "))},
				{Label: cliAppliedValue, Value: fmt.Sprintf("%t", response.Apply.Applied)},
				{Label: cliLifecycleValue, Value: string(response.Apply.Lifecycle)},
				{Label: "Apply Record", Value: stringOrDash(response.Apply.ApplyRecordID)},
				{Label: cliActiveGenerationValue, Value: fmt.Sprintf("%d", response.Apply.ActiveGeneration)},
				{Label: cliNextActionValue, Value: string(response.Apply.NextAction)},
				{Label: "Next Step", Value: string(response.NextStep)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"mcp_install",
				[]string{
					automationNameKey,
					"transport",
					mcpScopeKey,
					mcpWorkspaceIDKey,
					"catalog_entry",
					"catalog_version",
					cliAppliedKey,
					cliLifecycleKey,
					cliApplyRecordIDKey,
					cliActiveGenerationKey,
					cliNextActionKey,
					"next_step",
				},
				[]string{
					response.MCPServer.Name,
					response.MCPServer.Transport,
					string(response.MCPServer.Scope),
					response.MCPServer.WorkspaceID,
					response.MCPServer.CatalogEntry,
					response.MCPServer.CatalogVersion,
					fmt.Sprintf("%t", response.Apply.Applied),
					string(response.Apply.Lifecycle),
					response.Apply.ApplyRecordID,
					fmt.Sprintf("%d", response.Apply.ActiveGeneration),
					string(response.Apply.NextAction),
					string(response.NextStep),
				},
			), nil
		},
	}
}
