package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newWorkspaceCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage registered workspaces",
	}

	cmd.AddCommand(newWorkspaceAddCommand(deps))
	cmd.AddCommand(newWorkspaceListCommand(deps))
	cmd.AddCommand(newWorkspaceInfoCommand(deps))
	cmd.AddCommand(newWorkspaceEditCommand(deps))
	cmd.AddCommand(newWorkspaceRemoveCommand(deps))
	return cmd
}

func newWorkspaceAddCommand(deps commandDeps) *cobra.Command {
	var (
		name           string
		addDirs        []string
		defaultAgent   string
		environmentRef string
	)

	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Register a workspace",
		Example: `  # Register a workspace with a stable name
  agh workspace add "$PWD" --name checkout-api

  # Include an additional directory and set a workspace default agent
  agh workspace add "$PWD" --name platform --add-dir "$PWD/docs" --default-agent architect`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			workspace, err := client.CreateWorkspace(cmd.Context(), WorkspaceCreateRequest{
				RootDir:        strings.TrimSpace(args[0]),
				Name:           strings.TrimSpace(name),
				AddDirs:        trimmedUniqueStrings(addDirs),
				DefaultAgent:   strings.TrimSpace(defaultAgent),
				EnvironmentRef: strings.TrimSpace(environmentRef),
			})
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, workspaceRecordBundle(workspace))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Optional workspace name")
	cmd.Flags().
		StringArrayVar(&addDirs, "add-dir", nil, "Additional directory to include (repeatable)")
	cmd.Flags().
		StringVar(&defaultAgent, "default-agent", "", "Default agent override for this workspace")
	cmd.Flags().
		StringVar(&environmentRef, "environment", "", "Environment profile override for this workspace")
	return cmd
}

func newWorkspaceListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered workspaces",
		Example: `  # Show every registered workspace
  agh workspace list

  # Return workspace records as JSON
  agh workspace list --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			workspaces, err := client.ListWorkspaces(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, workspaceListBundle(workspaces))
		},
	}
}

func newWorkspaceInfoCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "info <name-or-id>",
		Short: "Show one workspace with resolved details",
		Example: `  # Show workspace paths, agents, and skills by name
  agh workspace info checkout-api

  # Emit resolved workspace details as JSON
  agh workspace info ws_1234 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			detail, err := client.GetWorkspace(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, workspaceDetailBundle(detail))
		},
	}
}

func workspaceEditFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("name") ||
		cmd.Flags().Changed("add-dir") ||
		cmd.Flags().Changed("remove-dir") ||
		cmd.Flags().Changed("default-agent") ||
		cmd.Flags().Changed("environment")
}

func newWorkspaceEditCommand(deps commandDeps) *cobra.Command {
	var (
		name           string
		addDirs        []string
		removeDirs     []string
		defaultAgent   string
		environmentRef string
	)

	cmd := &cobra.Command{
		Use:   "edit <name-or-id>",
		Short: "Edit a registered workspace",
		Example: `  # Rename a workspace
  agh workspace edit checkout-api --name checkout

  # Clear the workspace default agent
  agh workspace edit checkout-api --default-agent ""`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			if !workspaceEditFlagsChanged(cmd) {
				return errors.New("cli: at least one edit flag is required")
			}

			detail, err := client.GetWorkspace(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			request := WorkspaceUpdateRequest{}
			if cmd.Flags().Changed("name") {
				trimmedName := strings.TrimSpace(name)
				if trimmedName == "" {
					return errors.New("cli: --name cannot be empty")
				}
				request.Name = &trimmedName
			}
			if cmd.Flags().Changed("add-dir") || cmd.Flags().Changed("remove-dir") {
				mergedDirs, err := mergeWorkspaceAddDirs(
					detail.Workspace.AddDirs,
					addDirs,
					removeDirs,
				)
				if err != nil {
					return err
				}
				request.AddDirs = &mergedDirs
			}
			if cmd.Flags().Changed("default-agent") {
				trimmedDefaultAgent := strings.TrimSpace(defaultAgent)
				request.DefaultAgent = &trimmedDefaultAgent
			}
			if cmd.Flags().Changed("environment") {
				trimmedEnvironment := strings.TrimSpace(environmentRef)
				request.EnvironmentRef = &trimmedEnvironment
			}

			updated, err := client.UpdateWorkspace(cmd.Context(), detail.Workspace.ID, request)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, workspaceRecordBundle(updated))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Rename the workspace")
	cmd.Flags().
		StringArrayVar(&addDirs, "add-dir", nil, "Additional directory to include (repeatable)")
	cmd.Flags().
		StringArrayVar(&removeDirs, "remove-dir", nil, "Additional directory to remove (repeatable)")
	cmd.Flags().
		StringVar(&defaultAgent, "default-agent", "", "Override the workspace default agent (set empty to clear)")
	cmd.Flags().
		StringVar(&environmentRef, "environment", "", "Override the workspace environment profile (set empty to clear)")
	return cmd
}

func newWorkspaceRemoveCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name-or-id>",
		Short: "Remove a workspace registration",
		Example: `  # Remove a workspace registration by name
  agh workspace remove checkout-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			detail, err := client.GetWorkspace(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := client.DeleteWorkspace(cmd.Context(), detail.Workspace.ID); err != nil {
				return err
			}

			return writeCommandOutput(cmd, workspaceRecordBundle(detail.Workspace))
		},
	}
}

func workspaceRecordBundle(item WorkspaceRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Workspace", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Root", Value: stringOrDash(item.RootDir)},
				{Label: "Additional Dirs", Value: stringOrDash(strings.Join(item.AddDirs, ", "))},
				{Label: "Default Agent", Value: stringOrDash(item.DefaultAgent)},
				{Label: "Environment", Value: stringOrDash(item.EnvironmentRef)},
				{Label: "Created", Value: stringOrDash(formatTime(item.CreatedAt))},
				{Label: "Updated", Value: stringOrDash(formatTime(item.UpdatedAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("workspace", []string{
				"id", "name", "root_dir", "add_dirs", "default_agent", "environment_ref", "created_at", "updated_at",
			}, []string{
				item.ID,
				item.Name,
				item.RootDir,
				strings.Join(item.AddDirs, "|"),
				item.DefaultAgent,
				item.EnvironmentRef,
				formatTime(item.CreatedAt),
				formatTime(item.UpdatedAt),
			}), nil
		},
	}
}

func workspaceListBundle(items []WorkspaceRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Workspaces",
		[]string{"ID", "Name", "Root", "Add Dirs", "Default Agent", "Environment", "Updated"},
		"workspaces",
		[]string{"id", "name", "root_dir", "add_dir_count", "default_agent", "environment_ref", "updated_at"},
		func(item WorkspaceRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.Name),
				stringOrDash(item.RootDir),
				strconv.Itoa(len(item.AddDirs)),
				stringOrDash(item.DefaultAgent),
				stringOrDash(item.EnvironmentRef),
				stringOrDash(formatTime(item.UpdatedAt)),
			}
		},
		func(item WorkspaceRecord) []string {
			return []string{
				item.ID,
				item.Name,
				item.RootDir,
				strconv.Itoa(len(item.AddDirs)),
				item.DefaultAgent,
				item.EnvironmentRef,
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func workspaceDetailBundle(detail WorkspaceDetailRecord) outputBundle {
	return outputBundle{
		jsonValue: detail,
		human:     func() (string, error) { return renderWorkspaceDetailHuman(detail) },
		toon:      func() (string, error) { return renderWorkspaceDetailToon(detail) },
	}
}

func renderWorkspaceDetailHuman(detail WorkspaceDetailRecord) (string, error) {
	workspaceBlock, err := workspaceRecordBundle(detail.Workspace).human()
	if err != nil {
		return "", err
	}

	return renderHumanBlocks(
		workspaceBlock,
		renderHumanTable(
			"Sessions",
			[]string{"ID", "Name", "Agent", "State", "Workspace", "Updated"},
			workspaceSessionRows(detail.Sessions, true),
		),
		renderHumanTable(
			"Agents",
			[]string{"Name", "Provider", "Model", "Permissions"},
			workspaceAgentRows(detail.Agents, true),
		),
		renderHumanTable(
			"Skills",
			[]string{"Name", "Source", "Directory"},
			workspaceSkillRows(detail.Skills, true),
		),
	), nil
}

func renderWorkspaceDetailToon(detail WorkspaceDetailRecord) (string, error) {
	workspaceBlock, err := workspaceRecordBundle(detail.Workspace).toon()
	if err != nil {
		return "", err
	}

	return renderHumanBlocks(
		workspaceBlock,
		renderToonArray(
			"sessions",
			[]string{"id", "name", "agent_name", "state", "workspace", "updated_at"},
			workspaceSessionRows(detail.Sessions, false),
		),
		renderToonArray(
			"agents",
			[]string{"name", "provider", "model", "permissions"},
			workspaceAgentRows(detail.Agents, false),
		),
		renderToonArray(
			"skills",
			[]string{"name", "source", "dir"},
			workspaceSkillRows(detail.Skills, false),
		),
	), nil
}

func workspaceSessionRows(items []SessionRecord, human bool) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := []string{
			item.ID,
			item.Name,
			item.AgentName,
			string(item.State),
			displaySessionWorkspace(item),
			formatTime(item.UpdatedAt),
		}
		if human {
			row = []string{
				stringOrDash(item.ID),
				stringOrDash(item.Name),
				stringOrDash(item.AgentName),
				stringOrDash(string(item.State)),
				stringOrDash(displaySessionWorkspace(item)),
				stringOrDash(formatTime(item.UpdatedAt)),
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func workspaceAgentRows(items []AgentRecord, human bool) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := []string{item.Name, item.Provider, item.Model, item.Permissions}
		if human {
			row = []string{
				stringOrDash(item.Name),
				stringOrDash(item.Provider),
				stringOrDash(item.Model),
				stringOrDash(item.Permissions),
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func workspaceSkillRows(items []WorkspaceSkillRecord, human bool) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := []string{item.Name, item.Source, item.Dir}
		if human {
			row = []string{
				stringOrDash(item.Name),
				stringOrDash(item.Source),
				stringOrDash(item.Dir),
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func mergeWorkspaceAddDirs(existing []string, add []string, remove []string) ([]string, error) {
	addDirs := trimmedUniqueStrings(add)
	removeDirs := trimmedUniqueStrings(remove)

	removeSet := make(map[string]struct{}, len(removeDirs))
	for _, dir := range removeDirs {
		removeSet[dir] = struct{}{}
	}
	for _, dir := range addDirs {
		if _, exists := removeSet[dir]; exists {
			return nil, fmt.Errorf("cli: cannot add and remove the same directory: %s", dir)
		}
	}

	merged := make([]string, 0, len(existing)+len(addDirs))
	seen := make(map[string]struct{}, len(existing)+len(addDirs))
	for _, dir := range trimmedUniqueStrings(existing) {
		if _, removed := removeSet[dir]; removed {
			continue
		}
		if _, exists := seen[dir]; exists {
			continue
		}
		seen[dir] = struct{}{}
		merged = append(merged, dir)
	}
	for _, dir := range addDirs {
		if _, exists := seen[dir]; exists {
			continue
		}
		seen[dir] = struct{}{}
		merged = append(merged, dir)
	}
	return merged, nil
}

func trimmedUniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
