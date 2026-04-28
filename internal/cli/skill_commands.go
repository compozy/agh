package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage local AgentSkills",
	}

	cmd.AddCommand(newSkillListCommand(deps))
	cmd.AddCommand(newSkillViewCommand(deps))
	cmd.AddCommand(newSkillInfoCommand(deps))
	cmd.AddCommand(newSkillSearchCommand(deps))
	cmd.AddCommand(newSkillInstallCommand(deps))
	cmd.AddCommand(newSkillRemoveCommand(deps))
	cmd.AddCommand(newSkillUpdateCommand(deps))
	cmd.AddCommand(newSkillCreateCommand(deps))
	return cmd
}

func newSkillListCommand(deps commandDeps) *cobra.Command {
	var sourceFilter string
	var workspace string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List locally available skills",
		Example: `  # List every skill visible in the current workspace
  agh skill list

  # Show only bundled skills
  agh skill list --source bundled`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspaceRef, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			if workspaceRef != "" {
				client, err := clientFromDeps(deps)
				if err != nil {
					return err
				}
				records, err := client.ListSkills(cmd.Context(), SkillQuery{Workspace: workspaceRef})
				if err != nil {
					return err
				}
				items, err := skillListItemsFromRecords(records, sourceFilter)
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, skillListBundle(items))
			}

			ctx, err := loadSkillCommandContext(cmd.Context(), deps)
			if err != nil {
				return err
			}

			items, err := skillListItems(ctx.skills, sourceFilter)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, skillListBundle(items))
		},
	}
	cmd.Flags().StringVar(
		&sourceFilter,
		"source",
		"",
		"Filter by source: bundled, marketplace, user, additional (or agents/.agents), or workspace",
	)
	cmd.Flags().StringVar(
		&workspace,
		"workspace",
		"",
		"Resolve daemon-managed skills from a workspace id, name, or path",
	)
	return cmd
}

func newSkillViewCommand(deps commandDeps) *cobra.Command {
	var filePath string
	var workspace string

	cmd := &cobra.Command{
		Use:   "view <name>",
		Short: "Read a skill or one of its resource files",
		Example: `  # Render a skill as the XML block injected into agents
  agh skill view code-review

  # Read a resource file inside a skill directory
  agh skill view code-review --file references/checklist.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillViewCommand(cmd, deps, args[0], filePath)
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Relative file path inside the skill directory")
	cmd.Flags().StringVar(
		&workspace,
		"workspace",
		"",
		"Resolve the daemon-managed skill from a workspace id, name, or path",
	)
	return cmd
}

func runSkillViewCommand(cmd *cobra.Command, deps commandDeps, name string, filePath string) error {
	skillName := strings.TrimSpace(name)
	if skillName == "" {
		return errors.New("skill name is required")
	}
	workspaceRef, err := commandWorkspaceFlag(cmd)
	if err != nil {
		return err
	}
	if workspaceRef != "" {
		return runDaemonSkillViewCommand(cmd, deps, skillName, filePath, workspaceRef)
	}
	return runLocalSkillViewCommand(cmd, deps, skillName, filePath)
}

func runDaemonSkillViewCommand(
	cmd *cobra.Command,
	deps commandDeps,
	name string,
	filePath string,
	workspaceRef string,
) error {
	if strings.TrimSpace(filePath) != "" {
		return fmt.Errorf("skill view --workspace does not support --file")
	}
	client, err := clientFromDeps(deps)
	if err != nil {
		return err
	}
	record, err := client.GetSkill(cmd.Context(), name, SkillQuery{Workspace: workspaceRef})
	if err != nil {
		return err
	}
	content, err := client.GetSkillContent(cmd.Context(), name, SkillQuery{Workspace: workspaceRef})
	if err != nil {
		return err
	}
	rendered, err := renderSkillXML(&skills.Skill{Meta: skills.SkillMeta{Name: record.Name}}, content, nil)
	if err != nil {
		return err
	}
	item := skillViewItem{
		Name:    record.Name,
		Source:  record.Source,
		Path:    record.Dir,
		Content: rendered,
	}
	return writeCommandOutput(cmd, skillViewBundle(item, rendered))
}

func runLocalSkillViewCommand(cmd *cobra.Command, deps commandDeps, name string, filePath string) error {
	ctx, err := loadSkillCommandContext(cmd.Context(), deps)
	if err != nil {
		return err
	}
	skill, err := findSkillByName(ctx.skills, name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(filePath) != "" {
		return runLocalSkillResourceViewCommand(cmd, ctx, skill, filePath)
	}
	return runLocalSkillXMLViewCommand(cmd, ctx, skill)
}

func runLocalSkillResourceViewCommand(
	cmd *cobra.Command,
	ctx skillCommandContext,
	skill *skills.Skill,
	filePath string,
) error {
	content, err := readSkillResource(skill, ctx.bundledFS, filePath)
	if err != nil {
		return err
	}
	item := skillViewItem{
		Name:    skill.Meta.Name,
		Source:  skillSourceLabel(skill.Source),
		Path:    skill.FilePath,
		File:    strings.TrimSpace(filePath),
		Content: content,
	}
	return writeCommandOutput(cmd, skillViewBundle(item, content))
}

func runLocalSkillXMLViewCommand(cmd *cobra.Command, ctx skillCommandContext, skill *skills.Skill) error {
	resources, err := listSkillResources(skill, ctx.bundledFS)
	if err != nil {
		return err
	}
	content, err := ctx.registry.LoadContent(cmd.Context(), skill)
	if err != nil {
		return err
	}
	rendered, err := renderSkillXML(skill, content, resources)
	if err != nil {
		return err
	}
	item := skillViewItem{
		Name:      skill.Meta.Name,
		Source:    skillSourceLabel(skill.Source),
		Path:      skill.FilePath,
		Content:   rendered,
		Resources: resources,
	}
	return writeCommandOutput(cmd, skillViewBundle(item, rendered))
}

func newSkillInfoCommand(deps commandDeps) *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed metadata for one skill",
		Example: `  # Inspect a skill's metadata and resource list
  agh skill info code-review`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRef, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			if workspaceRef != "" {
				client, err := clientFromDeps(deps)
				if err != nil {
					return err
				}
				record, err := client.GetSkill(cmd.Context(), args[0], SkillQuery{Workspace: workspaceRef})
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, skillInfoBundle(skillInfoItemFromRecord(record)))
			}

			ctx, err := loadSkillCommandContext(cmd.Context(), deps)
			if err != nil {
				return err
			}

			skill, err := findSkillByName(ctx.skills, args[0])
			if err != nil {
				return err
			}

			resources, err := listSkillResources(skill, ctx.bundledFS)
			if err != nil {
				return err
			}

			item := skillInfoItem{
				Name:        skill.Meta.Name,
				Description: skill.Meta.Description,
				Version:     skill.Meta.Version,
				Source:      skillSourceLabel(skill.Source),
				Path:        skill.FilePath,
				Enabled:     skill.Enabled,
				Metadata:    cloneMetadata(skill.Meta.Metadata),
				Resources:   resources,
			}

			return writeCommandOutput(cmd, skillInfoBundle(item))
		},
	}
	cmd.Flags().StringVar(
		&workspace,
		"workspace",
		"",
		"Resolve the daemon-managed skill from a workspace id, name, or path",
	)
	return cmd
}

func newSkillCreateCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Scaffold a new workspace skill",
		Example: `  # Create .agh/skills/api-review/SKILL.md in the current workspace
  agh skill create api-review`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := defaultSkillName
			if len(args) == 1 {
				name = args[0]
			}

			skillName, err := normalizeSkillName(name)
			if err != nil {
				return err
			}

			workspace, err := resolveCLIWorkspaceRoot(deps)
			if err != nil {
				return err
			}

			skillDir := filepath.Join(workspace, aghconfig.DirName, aghconfig.SkillsDirName, skillName)
			if _, err := os.Stat(skillDir); err == nil {
				return fmt.Errorf("skill %q already exists at %s", skillName, skillDir)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("cli: inspect skill directory %q: %w", skillDir, err)
			}

			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				return fmt.Errorf("cli: create skill directory %q: %w", skillDir, err)
			}

			skillFilePath := filepath.Join(skillDir, skillMarkdownFileName)
			content := defaultSkillTemplate(skillName)
			if err := os.WriteFile(skillFilePath, []byte(content), 0o600); err != nil {
				return fmt.Errorf("cli: write skill template %q: %w", skillFilePath, err)
			}

			if _, err := skills.ParseSkillFile(skillFilePath); err != nil {
				return fmt.Errorf("cli: validate generated skill %q: %w", skillFilePath, err)
			}

			return writeCommandOutput(cmd, skillCreateBundle(skillCreateItem{
				Name:   skillName,
				Path:   skillDir,
				File:   skillFilePath,
				Source: "workspace",
				Status: "created",
			}))
		},
	}
}

func newSkillSearchCommand(deps commandDeps) *cobra.Command {
	limit := defaultMarketplaceSearchLim

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search marketplace skills",
		Example: `  # Search the configured marketplace
  agh skill search "code review"

  # Limit marketplace results
  agh skill search testing --limit 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := searchMarketplaceSkills(cmd.Context(), deps, args[0], limit)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, skillSearchBundle(results))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", defaultMarketplaceSearchLim, "Maximum number of marketplace results to return")
	return cmd
}

func newSkillInstallCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "install <slug>",
		Short: "Install a marketplace skill",
		Example: `  # Install the latest marketplace version of a skill
  agh skill install @acme/code-review`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			slug, err := normalizeSkillSlug(args[0])
			if err != nil {
				return err
			}

			runtime, registry, err := loadSkillRegistry(deps)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, registry.Close())
			}()

			item, err := installMarketplaceSkill(cmd.Context(), runtime, registry, slug, "", "", deps.now)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, skillInstallBundle(item))
		},
	}
}

func newSkillRemoveCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed marketplace skill",
		Example: `  # Remove a marketplace-installed skill by local name
  agh skill remove code-review`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := normalizeSkillName(args[0])
			if err != nil {
				return err
			}

			runtime, err := loadRuntimeContext(deps)
			if err != nil {
				return err
			}

			item, err := removeMarketplaceSkill(runtime.HomePaths.SkillsDir, name)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, skillRemoveBundle(item))
		},
	}
}

func newSkillUpdateCommand(deps commandDeps) *cobra.Command {
	updateAll := false
	checkOnly := false

	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Check for or install updates for marketplace skills",
		Example: `  # Check whether one installed skill has an update
  agh skill update code-review --check

  # Update every marketplace-installed skill
  agh skill update --all`,
		Args: func(_ *cobra.Command, args []string) error {
			if updateAll && len(args) > 0 {
				return errors.New("cli: update accepts either a skill name or --all, not both")
			}
			if !updateAll && len(args) != 1 {
				return errors.New("cli: update requires a skill name unless --all is set")
			}
			if len(args) == 1 {
				_, err := normalizeSkillName(args[0])
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			runtime, registry, err := loadSkillRegistry(deps)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, registry.Close())
			}()

			items, err := updateMarketplaceSkills(
				cmd.Context(),
				runtime,
				registry,
				args,
				updateAll,
				checkOnly,
				deps.now,
			)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, skillUpdateBundle(items))
		},
	}
	cmd.Flags().BoolVar(&updateAll, "all", false, "Update every installed marketplace skill")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing them")
	return cmd
}
