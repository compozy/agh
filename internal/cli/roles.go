package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const rolesListCommandUse = "list"

func newRolesCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles",
		Short: "Inspect effective background role configuration",
	}
	cmd.AddCommand(newRolesListCommand(deps))
	cmd.AddCommand(newRolesShowCommand(deps))
	return cmd
}

func newRolesListCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   rolesListCommandUse,
		Short: "List effective background roles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query, err := roleQueryFromCommand(cmd)
			if err != nil {
				return err
			}
			roles, err := client.ListRoles(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, roleListBundle(roles))
		},
	}
	cmd.Flags().String("workspace", "", "Resolve roles from a workspace id, name, or path")
	return cmd
}

func newRolesShowCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <role>",
		Short: "Show one effective background role",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query, err := roleQueryFromCommand(cmd)
			if err != nil {
				return err
			}
			role, err := client.GetRole(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, roleBundle(role))
		},
	}
	cmd.Flags().String("workspace", "", "Resolve the role from a workspace id, name, or path")
	return cmd
}

func roleQueryFromCommand(cmd *cobra.Command) (RoleQuery, error) {
	workspace, err := commandWorkspaceFlag(cmd)
	if err != nil {
		return RoleQuery{}, err
	}
	return RoleQuery{Workspace: workspace}, nil
}

func roleListBundle(roles []RoleRecord) outputBundle {
	return listBundle(
		roles,
		roles,
		"Roles",
		[]string{
			roleOutputLabel,
			"Enabled",
			"Resolution",
			agentOutputLabel,
			agentKernelProviderValue,
			agentKernelModelValue,
			"Diagnostics",
		},
		"roles",
		[]string{
			"role",
			"enabled",
			"resolution_mode",
			agentAgentKey,
			cliProviderKey,
			agentModelKey,
			"diagnostic_count",
		},
		roleListValues,
		roleListValues,
	)
}

func roleListValues(role RoleRecord) []string {
	return []string{
		role.Role,
		strconv.FormatBool(role.Enabled),
		string(role.ResolutionMode),
		roleStatusValue(role.Agent),
		roleStatusValue(role.Provider),
		roleStatusValue(role.Model),
		strconv.Itoa(len(role.Diagnostics)),
	}
}

func roleBundle(role RoleRecord) outputBundle {
	return outputBundle{
		jsonValue: role,
		human: func() (string, error) {
			return renderHumanSection(roleOutputLabel, []keyValue{
				{Label: roleOutputLabel, Value: role.Role},
				{Label: "Enabled", Value: strconv.FormatBool(role.Enabled)},
				{Label: "Resolution", Value: string(role.ResolutionMode)},
				{Label: agentOutputLabel, Value: stringOrDash(roleStatusValue(role.Agent))},
				{Label: agentKernelProviderValue, Value: stringOrDash(roleStatusValue(role.Provider))},
				{Label: agentKernelModelValue, Value: stringOrDash(roleStatusValue(role.Model))},
				{Label: "Reasoning Effort", Value: stringOrDash(roleStatusValue(role.ReasoningEffort))},
				{Label: "Timeout", Value: stringOrDash(roleStatusValue(role.Timeout))},
				{Label: "Diagnostics", Value: strconv.Itoa(len(role.Diagnostics))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("role", []string{
				"role",
				"enabled",
				"resolution_mode",
				agentAgentKey,
				cliProviderKey,
				agentModelKey,
				"reasoning_effort",
				"timeout",
			}, []string{
				role.Role,
				strconv.FormatBool(role.Enabled),
				string(role.ResolutionMode),
				roleStatusValue(role.Agent),
				roleStatusValue(role.Provider),
				roleStatusValue(role.Model),
				roleStatusValue(role.ReasoningEffort),
				roleStatusValue(role.Timeout),
			}), nil
		},
	}
}

func roleStatusValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
