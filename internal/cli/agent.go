package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newAgentCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Inspect AGH agent definitions",
	}

	cmd.AddCommand(newAgentListCommand(deps))
	cmd.AddCommand(newAgentInfoCommand(deps))
	cmd.AddCommand(newAgentSoulCommand(deps))
	cmd.AddCommand(newAgentHeartbeatCommand(deps))
	return cmd
}

func newAgentListCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed agent definitions",
		Example: `  # Show every agent definition available to the daemon
  agh agent list

  # Show agents resolved for a workspace
  agh agent list --workspace ~/dev/ai/acme-startup

  # Emit the same list as JSON
  agh agent list -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := agentQueryFromCommand(cmd)
			if err != nil {
				return err
			}
			agents, err := client.ListAgents(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentListBundle(agents))
		},
	}
	cmd.Flags().String("workspace", "", "Resolve agents from a workspace id, name, or path")
	return cmd
}

func newAgentInfoCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show one agent definition",
		Example: `  # Inspect the default bootstrap agent
  agh agent info general

  # Inspect a workspace-local agent
  agh agent info reviewer --workspace ~/dev/ai/acme-startup

  # Inspect an agent definition as JSON
  agh agent info reviewer -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := agentQueryFromCommand(cmd)
			if err != nil {
				return err
			}
			agent, err := client.GetAgent(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentBundle(agent))
		},
	}
	cmd.Flags().String("workspace", "", "Resolve the agent from a workspace id, name, or path")
	return cmd
}

func agentQueryFromCommand(cmd *cobra.Command) (AgentQuery, error) {
	workspace, err := commandWorkspaceFlag(cmd)
	if err != nil {
		return AgentQuery{}, err
	}
	return AgentQuery{Workspace: workspace}, nil
}

func agentListBundle(items []AgentRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Agents",
		[]string{"Name", "Provider", "Model", "Category", "Tools", "Permissions"},
		"agents",
		[]string{"name", "provider", "model", "category", "tool_count", "permissions"},
		func(item AgentRecord) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Provider),
				stringOrDash(item.Model),
				stringOrDash(agentCategoryLabel(item.CategoryPath)),
				strconv.Itoa(len(item.Tools)),
				stringOrDash(item.Permissions),
			}
		},
		func(item AgentRecord) []string {
			return []string{
				item.Name,
				item.Provider,
				item.Model,
				agentCategoryLabel(item.CategoryPath),
				strconv.Itoa(len(item.Tools)),
				item.Permissions,
			}
		},
	)
}

func agentBundle(item AgentRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			base := renderHumanSection("Agent", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Provider", Value: stringOrDash(item.Provider)},
				{Label: "Command", Value: stringOrDash(item.Command)},
				{Label: "Model", Value: stringOrDash(item.Model)},
				{Label: "Category", Value: stringOrDash(agentCategoryLabel(item.CategoryPath))},
				{Label: "Tools", Value: stringOrDash(strings.Join(item.Tools, ", "))},
				{Label: "Permissions", Value: stringOrDash(item.Permissions)},
			})

			servers := make([][]string, 0, len(item.MCPServers))
			for _, server := range item.MCPServers {
				servers = append(servers, []string{
					stringOrDash(server.Name),
					stringOrDash(server.Command),
					stringOrDash(strings.Join(server.Args, " ")),
				})
			}
			mcp := renderHumanTable("MCP Servers", []string{"Name", "Command", "Args"}, servers)
			prompt := renderHumanSection("Prompt", []keyValue{{Label: "Body", Value: stringOrDash(item.Prompt)}})
			return renderHumanBlocks(base, mcp, prompt), nil
		},
		toon: func() (string, error) {
			// Detail output emits tool names; list output keeps the table dense with tool_count.
			return renderToonObject("agent", []string{
				"name", "provider", "command", "model", "category", "tools", "permissions", "prompt",
			}, []string{
				item.Name,
				item.Provider,
				item.Command,
				item.Model,
				agentCategoryLabel(item.CategoryPath),
				strings.Join(item.Tools, "|"),
				item.Permissions,
				item.Prompt,
			}), nil
		},
	}
}

func agentCategoryLabel(path []string) string {
	if len(path) == 0 {
		return ""
	}
	// AGENT.md category paths render as a single space-delimited CLI path.
	return strings.Join(path, " / ")
}
