package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const cliAgentDefinitionFileName = "AGENT.md"

func newAgentCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Inspect AGH agent definitions",
	}

	cmd.AddCommand(newAgentCreateCommand(deps))
	cmd.AddCommand(newAgentListCommand(deps))
	cmd.AddCommand(newAgentInfoCommand(deps))
	cmd.AddCommand(newAgentSoulCommand(deps))
	cmd.AddCommand(newAgentHeartbeatCommand(deps))
	return cmd
}

type agentCreateFlags struct {
	workspace    string
	provider     string
	command      string
	model        string
	prompt       string
	promptFile   string
	tools        []string
	toolsets     []string
	denyTools    []string
	permissions  string
	categoryPath []string
	force        bool
}

type agentDefinitionFrontmatter struct {
	Name         string   `yaml:"name"`
	Provider     string   `yaml:"provider,omitempty"`
	Command      string   `yaml:"command,omitempty"`
	Model        string   `yaml:"model,omitempty"`
	Tools        []string `yaml:"tools,omitempty"`
	Toolsets     []string `yaml:"toolsets,omitempty"`
	DenyTools    []string `yaml:"deny_tools,omitempty"`
	Permissions  string   `yaml:"permissions,omitempty"`
	CategoryPath []string `yaml:"category_path,omitempty"`
}

func newAgentCreateCommand(deps commandDeps) *cobra.Command {
	var flags agentCreateFlags
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a global or workspace-local AGENT.md definition",
		Example: `  # Create a workspace-local agent definition
  agh agent create pricing_strategist \
    --workspace ~/dev/ad8 \
    --provider claude \
    --model claude-sonnet-4-6 \
    --prompt "You own pricing strategy." \
    -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			flags.workspace = workspace
			agent, err := createAgentDefinition(cmd, deps, args[0], flags)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentBundle(agentRecordFromDefinition(agent)))
		},
	}
	cmd.Flags().String("workspace", "", "Workspace id, name, or path to create the agent under")
	cmd.Flags().StringVar(&flags.provider, "provider", "", "Provider name for sessions using this agent")
	cmd.Flags().StringVar(&flags.command, "command", "", "Optional provider command override")
	cmd.Flags().StringVar(&flags.model, "model", "", "Optional provider model")
	cmd.Flags().StringVar(&flags.prompt, "prompt", "", "Agent system prompt body")
	cmd.Flags().StringVar(&flags.promptFile, "prompt-file", "", "Read the agent system prompt body from a file")
	cmd.Flags().StringArrayVar(&flags.tools, "tool", nil, "Allowed tool pattern (repeatable)")
	cmd.Flags().StringArrayVar(&flags.toolsets, "toolset", nil, "Allowed toolset reference (repeatable)")
	cmd.Flags().StringArrayVar(&flags.denyTools, "deny-tool", nil, "Denied tool pattern (repeatable)")
	cmd.Flags().StringVar(&flags.permissions, "permissions", "", "Optional permission mode")
	cmd.Flags().StringArrayVar(&flags.categoryPath, "category", nil, "Agent category path segment (repeatable)")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Overwrite an existing AGENT.md definition")
	return cmd
}

func createAgentDefinition(
	cmd *cobra.Command,
	deps commandDeps,
	name string,
	flags agentCreateFlags,
) (aghconfig.AgentDef, error) {
	agentName := aghconfig.NormalizeAgentName(name)
	if err := aghconfig.ValidateAgentName(agentName); err != nil {
		return aghconfig.AgentDef{}, err
	}
	prompt, err := agentCreatePrompt(flags)
	if err != nil {
		return aghconfig.AgentDef{}, err
	}
	contents, agent, err := renderAgentDefinition(agentName, prompt, flags)
	if err != nil {
		return aghconfig.AgentDef{}, err
	}
	agentsDir, err := resolveAgentCreateDirectory(cmd, deps, flags.workspace)
	if err != nil {
		return aghconfig.AgentDef{}, err
	}
	path := filepath.Join(agentsDir, agentName, cliAgentDefinitionFileName)
	if err := writeAgentDefinitionFile(path, contents, flags.force); err != nil {
		return aghconfig.AgentDef{}, err
	}
	agent.SourcePath = filepath.Clean(path)
	return agent, nil
}

func agentCreatePrompt(flags agentCreateFlags) (string, error) {
	prompt := strings.TrimSpace(flags.prompt)
	promptFile := strings.TrimSpace(flags.promptFile)
	if prompt != "" && promptFile != "" {
		return "", errors.New("cli: use either --prompt or --prompt-file, not both")
	}
	if promptFile != "" {
		contents, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("cli: read prompt file %q: %w", promptFile, err)
		}
		prompt = strings.TrimSpace(string(contents))
	}
	if prompt == "" {
		return "", errors.New("cli: --prompt or --prompt-file is required")
	}
	return prompt, nil
}

func renderAgentDefinition(
	agentName string,
	prompt string,
	flags agentCreateFlags,
) (string, aghconfig.AgentDef, error) {
	metadata := agentDefinitionFrontmatter{
		Name:         agentName,
		Provider:     strings.TrimSpace(flags.provider),
		Command:      strings.TrimSpace(flags.command),
		Model:        strings.TrimSpace(flags.model),
		Tools:        trimSpawnAtoms(flags.tools),
		Toolsets:     trimSpawnAtoms(flags.toolsets),
		DenyTools:    trimSpawnAtoms(flags.denyTools),
		Permissions:  strings.TrimSpace(flags.permissions),
		CategoryPath: trimSpawnAtoms(flags.categoryPath),
	}
	frontmatter, err := yaml.Marshal(metadata)
	if err != nil {
		return "", aghconfig.AgentDef{}, fmt.Errorf("cli: render agent frontmatter: %w", err)
	}
	contents := fmt.Sprintf("---\n%s---\n\n%s\n", string(frontmatter), prompt)
	agent, err := aghconfig.ParseAgentDef([]byte(contents))
	if err != nil {
		return "", aghconfig.AgentDef{}, fmt.Errorf("cli: validate generated agent definition: %w", err)
	}
	if agent.Name != agentName {
		return "", aghconfig.AgentDef{}, fmt.Errorf(
			"cli: generated agent name %q does not match %q",
			agent.Name,
			agentName,
		)
	}
	return contents, agent, nil
}

func resolveAgentCreateDirectory(cmd *cobra.Command, deps commandDeps, workspaceRef string) (string, error) {
	if strings.TrimSpace(workspaceRef) == "" {
		homePaths, err := deps.resolveHome()
		if err != nil {
			return "", err
		}
		if deps.ensureHome != nil {
			if err := deps.ensureHome(homePaths); err != nil {
				return "", err
			}
		}
		return homePaths.AgentsDir, nil
	}

	client, err := clientFromDeps(deps)
	if err != nil {
		return "", err
	}
	detail, err := client.GetWorkspace(cmd.Context(), workspaceRef)
	if err != nil {
		return "", err
	}
	rootDir := strings.TrimSpace(detail.Workspace.RootDir)
	if rootDir == "" {
		return "", errors.New("cli: resolved workspace root_dir is empty")
	}
	return filepath.Join(rootDir, aghconfig.DirName, aghconfig.AgentsDirName), nil
}

func writeAgentDefinitionFile(path string, contents string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("cli: agent definition already exists at %s (use --force to overwrite)", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cli: inspect agent definition %q: %w", path, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cli: create agent directory %q: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("cli: write agent definition %q: %w", path, err)
	}
	return nil
}

func agentRecordFromDefinition(agent aghconfig.AgentDef) AgentRecord {
	return AgentRecord{
		Name:         agent.Name,
		Provider:     agent.Provider,
		Command:      agent.Command,
		Model:        agent.Model,
		Tools:        agent.Tools,
		Toolsets:     agent.Toolsets,
		DenyTools:    agent.DenyTools,
		Permissions:  agent.Permissions,
		CategoryPath: agent.CategoryPath,
		Prompt:       agent.Prompt,
	}
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
