package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/skills"
	skillbundled "github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/spf13/cobra"
)

const (
	defaultSkillName      = "new-skill"
	skillMarkdownFileName = "SKILL.md"
)

var (
	skillXMLAttributeReplacer = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;")
	skillXMLTextReplacer      = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;")
	validSkillNamePattern     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

type skillCommandContext struct {
	workspace string
	bundledFS fs.FS
	skills    []*skills.Skill
}

type skillListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Enabled     bool   `json:"enabled"`
}

type skillViewItem struct {
	Name      string   `json:"name"`
	Source    string   `json:"source"`
	Path      string   `json:"path"`
	File      string   `json:"file,omitempty"`
	Content   string   `json:"content"`
	Resources []string `json:"resources,omitempty"`
}

type skillInfoItem struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version,omitempty"`
	Source      string         `json:"source"`
	Path        string         `json:"path"`
	Enabled     bool           `json:"enabled"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Resources   []string       `json:"resources,omitempty"`
}

type skillCreateItem struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	File   string `json:"file"`
	Source string `json:"source"`
	Status string `json:"status"`
}

func newSkillCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage local AgentSkills",
	}

	cmd.AddCommand(newSkillListCommand(deps))
	cmd.AddCommand(newSkillViewCommand(deps))
	cmd.AddCommand(newSkillInfoCommand(deps))
	cmd.AddCommand(newSkillCreateCommand(deps))
	return cmd
}

func newSkillListCommand(deps commandDeps) *cobra.Command {
	var sourceFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List locally available skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
	cmd.Flags().StringVar(&sourceFilter, "source", "", "Filter by source: bundled, user, agents, or workspace")
	return cmd
}

func newSkillViewCommand(deps commandDeps) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "view <name>",
		Short: "Read a skill or one of its resource files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadSkillCommandContext(cmd.Context(), deps)
			if err != nil {
				return err
			}

			skill, err := findSkillByName(ctx.skills, args[0])
			if err != nil {
				return err
			}

			if strings.TrimSpace(filePath) != "" {
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

			resources, err := listSkillResources(skill, ctx.bundledFS)
			if err != nil {
				return err
			}

			rendered, err := renderSkillXML(skill, resources)
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
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Relative file path inside the skill directory")
	return cmd
}

func newSkillInfoCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed metadata for one skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
}

func newSkillCreateCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Scaffold a new workspace skill",
		Args:  cobra.MaximumNArgs(1),
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
			if err := os.WriteFile(skillFilePath, []byte(content), 0o644); err != nil {
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

func loadSkillCommandContext(ctx context.Context, deps commandDeps) (skillCommandContext, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return skillCommandContext{}, err
	}

	workspace, err := resolveCLIWorkspaceRoot(deps)
	if err != nil {
		return skillCommandContext{}, err
	}

	userAgentsDir, err := cliUserAgentsSkillsDir(deps)
	if err != nil {
		return skillCommandContext{}, err
	}

	registry := skills.NewRegistry(skills.RegistryConfig{
		BundledFS:      skillbundled.FS(),
		UserSkillsDir:  runtime.HomePaths.SkillsDir,
		UserAgentsDir:  userAgentsDir,
		DisabledSkills: append([]string(nil), runtime.Config.Skills.DisabledSkills...),
	})
	if err := registry.LoadAll(ctx); err != nil {
		return skillCommandContext{}, err
	}

	skillList, err := registry.ForWorkspace(ctx, workspace)
	if err != nil {
		return skillCommandContext{}, err
	}

	return skillCommandContext{
		workspace: workspace,
		bundledFS: skillbundled.FS(),
		skills:    skillList,
	}, nil
}

func cliUserAgentsSkillsDir(deps commandDeps) (string, error) {
	if deps.getenv != nil {
		if home := strings.TrimSpace(deps.getenv("HOME")); home != "" {
			absHome, err := filepath.Abs(home)
			if err != nil {
				return "", fmt.Errorf("cli: resolve HOME for user agent skills: %w", err)
			}
			return filepath.Join(absHome, ".agents", "skills"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cli: resolve user home for agent skills: %w", err)
	}

	absHome, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("cli: resolve user home for agent skills: %w", err)
	}

	return filepath.Join(absHome, ".agents", "skills"), nil
}

func resolveCLIWorkspaceRoot(deps commandDeps) (string, error) {
	workspace, err := currentWorkingDirectory(deps)
	if err != nil {
		return "", err
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("cli: resolve workspace root %q: %w", workspace, err)
	}
	return absWorkspace, nil
}

func skillListItems(allSkills []*skills.Skill, sourceFilter string) ([]skillListItem, error) {
	filter, err := normalizeSkillSourceFilter(sourceFilter)
	if err != nil {
		return nil, err
	}

	items := make([]skillListItem, 0, len(allSkills))
	for _, skill := range allSkills {
		if skill == nil {
			continue
		}

		source := skillSourceLabel(skill.Source)
		if filter != "" && source != filter {
			continue
		}

		items = append(items, skillListItem{
			Name:        skill.Meta.Name,
			Description: skill.Meta.Description,
			Source:      source,
			Enabled:     skill.Enabled,
		})
	}

	return items, nil
}

func normalizeSkillSourceFilter(sourceFilter string) (string, error) {
	filter := strings.ToLower(strings.TrimSpace(sourceFilter))
	switch filter {
	case "":
		return "", nil
	case "bundled", "user", "agents", "workspace":
		return filter, nil
	case ".agents":
		return "agents", nil
	default:
		return "", fmt.Errorf("cli: invalid skill source %q", sourceFilter)
	}
}

func findSkillByName(allSkills []*skills.Skill, name string) (*skills.Skill, error) {
	skillName := strings.TrimSpace(name)
	if skillName == "" {
		return nil, errors.New("skill name is required")
	}

	for _, skill := range allSkills {
		if skill == nil {
			continue
		}
		if skill.Meta.Name == skillName {
			return skill, nil
		}
	}

	return nil, fmt.Errorf("skill %q not found", skillName)
}

func listSkillResources(skill *skills.Skill, bundledFS fs.FS) ([]string, error) {
	if skill == nil {
		return nil, errors.New("skill is required")
	}

	resources := make([]string, 0)
	switch skill.Source {
	case skills.SourceBundled:
		if bundledFS == nil {
			return nil, errors.New("bundled skills filesystem is required")
		}

		root := strings.TrimSpace(skill.Dir)
		if root == "" {
			return []string{}, nil
		}

		err := fs.WalkDir(bundledFS, root, func(resourcePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}

			relative := strings.TrimPrefix(resourcePath, root+"/")
			if resourcePath == root {
				relative = skillMarkdownFileName
			}
			if relative == skillMarkdownFileName {
				return nil
			}

			resources = append(resources, relative)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("cli: list bundled skill resources for %q: %w", skill.Meta.Name, err)
		}
	default:
		root := strings.TrimSpace(skill.Dir)
		if root == "" {
			return []string{}, nil
		}

		err := filepath.WalkDir(root, func(resourcePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}

			relative, err := filepath.Rel(root, resourcePath)
			if err != nil {
				return err
			}
			if filepath.Clean(relative) == skillMarkdownFileName {
				return nil
			}

			resources = append(resources, filepath.ToSlash(relative))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("cli: list skill resources for %q: %w", skill.Meta.Name, err)
		}
	}

	sort.Strings(resources)
	return resources, nil
}

func readSkillResource(skill *skills.Skill, bundledFS fs.FS, relativePath string) (string, error) {
	if skill == nil {
		return "", errors.New("skill is required")
	}

	switch skill.Source {
	case skills.SourceBundled:
		if bundledFS == nil {
			return "", errors.New("bundled skills filesystem is required")
		}

		cleanPath, err := cleanBundledSkillRelativePath(relativePath)
		if err != nil {
			return "", err
		}
		root := strings.TrimSpace(skill.Dir)
		if root == "" {
			return "", errors.New("skill directory is required")
		}

		content, err := fs.ReadFile(bundledFS, path.Join(root, cleanPath))
		if err != nil {
			return "", fmt.Errorf("cli: read bundled skill file %q: %w", cleanPath, err)
		}
		return string(content), nil
	default:
		cleanPath, err := cleanFilesystemSkillRelativePath(relativePath)
		if err != nil {
			return "", err
		}

		root := strings.TrimSpace(skill.Dir)
		if root == "" {
			return "", errors.New("skill directory is required")
		}

		targetPath := filepath.Join(root, cleanPath)
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return "", fmt.Errorf("cli: resolve skill directory %q: %w", root, err)
		}
		resolvedRoot, err := filepath.EvalSymlinks(absRoot)
		if err != nil {
			return "", fmt.Errorf("cli: resolve skill directory %q: %w", absRoot, err)
		}
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return "", fmt.Errorf("cli: resolve skill file %q: %w", targetPath, err)
		}
		resolvedTarget, err := filepath.EvalSymlinks(absTarget)
		if err != nil {
			return "", fmt.Errorf("cli: resolve skill file %q: %w", absTarget, err)
		}

		relativeToRoot, err := filepath.Rel(resolvedRoot, resolvedTarget)
		if err != nil {
			return "", fmt.Errorf("cli: resolve skill file %q within %q: %w", resolvedTarget, resolvedRoot, err)
		}
		if relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
			return "", errors.New("skill file path must stay within the skill directory")
		}

		content, err := os.ReadFile(resolvedTarget)
		if err != nil {
			return "", fmt.Errorf("cli: read skill file %q: %w", cleanPath, err)
		}
		return string(content), nil
	}
}

func cleanBundledSkillRelativePath(relativePath string) (string, error) {
	cleaned := path.Clean(strings.TrimSpace(strings.ReplaceAll(relativePath, "\\", "/")))
	switch {
	case cleaned == ".", cleaned == "":
		return "", errors.New("skill file path is required")
	case strings.HasPrefix(cleaned, "/"):
		return "", errors.New("skill file path must be relative")
	case cleaned == "..", strings.HasPrefix(cleaned, "../"):
		return "", errors.New("skill file path must stay within the skill directory")
	default:
		return cleaned, nil
	}
}

func cleanFilesystemSkillRelativePath(relativePath string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(relativePath))
	switch {
	case cleaned == ".", cleaned == "":
		return "", errors.New("skill file path is required")
	case filepath.IsAbs(cleaned):
		return "", errors.New("skill file path must be relative")
	case cleaned == "..", strings.HasPrefix(cleaned, ".."+string(filepath.Separator)):
		return "", errors.New("skill file path must stay within the skill directory")
	default:
		return cleaned, nil
	}
}

func renderSkillXML(skill *skills.Skill, resources []string) (string, error) {
	if skill == nil {
		return "", errors.New("skill is required")
	}

	var builder strings.Builder
	builder.WriteString(`<skill_content name="`)
	builder.WriteString(skillXMLAttributeReplacer.Replace(skill.Meta.Name))
	builder.WriteString(`">`)
	builder.WriteString("\n")
	builder.WriteString(skill.Content)
	if !strings.HasSuffix(skill.Content, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("\n<skill_resources>\n")
	for _, resource := range resources {
		builder.WriteString("  <file>")
		builder.WriteString(skillXMLTextReplacer.Replace(resource))
		builder.WriteString("</file>\n")
	}
	builder.WriteString("</skill_resources>\n")
	builder.WriteString("</skill_content>")
	return builder.String(), nil
}

func normalizeSkillName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", errors.New("skill name is required")
	case trimmed == ".", trimmed == "..":
		return "", errors.New("skill name must not be a relative path segment")
	case filepath.IsAbs(trimmed):
		return "", errors.New("skill name must be relative")
	case strings.Contains(trimmed, "/"), strings.Contains(trimmed, `\`):
		return "", errors.New("skill name must not include path separators")
	case !validSkillNamePattern.MatchString(trimmed):
		return "", errors.New("skill name must contain only letters, numbers, dots, underscores, and hyphens")
	default:
		return trimmed, nil
	}
}

func defaultSkillTemplate(name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = defaultSkillName
	}

	return fmt.Sprintf(`---
name: %q
description: Describe when to use this skill.
---

# %s

Describe the workflow, constraints, and expected outcome for this skill.
`, trimmedName, titleizeSkillName(trimmedName))
}

func titleizeSkillName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return "New Skill"
	}

	titled := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}

		lower := strings.ToLower(part)
		titled = append(titled, strings.ToUpper(lower[:1])+lower[1:])
	}
	if len(titled) == 0 {
		return "New Skill"
	}
	return strings.Join(titled, " ")
}

func skillSourceLabel(source skills.SkillSource) string {
	switch source {
	case skills.SourceBundled:
		return "bundled"
	case skills.SourceUser:
		return "user"
	case skills.SourceAgents:
		return "agents"
	case skills.SourceWorkspace:
		return "workspace"
	default:
		return "unknown"
	}
}

func skillListBundle(items []skillListItem) outputBundle {
	return outputBundle{
		jsonValue: items,
		human: func() (string, error) {
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					stringOrDash(item.Name),
					stringOrDash(item.Description),
					stringOrDash(item.Source),
					strconv.FormatBool(item.Enabled),
				})
			}
			return renderHumanTable("Skills", []string{"Name", "Description", "Source", "Enabled"}, rows), nil
		},
		toon: func() (string, error) {
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					item.Name,
					item.Description,
					item.Source,
					strconv.FormatBool(item.Enabled),
				})
			}
			return renderToonArray("skills", []string{"name", "description", "source", "enabled"}, rows), nil
		},
	}
}

func skillViewBundle(item skillViewItem, rendered string) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return rendered, nil
		},
		toon: func() (string, error) {
			return rendered, nil
		},
	}
}

func skillInfoBundle(item skillInfoItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			base := renderHumanSection("Skill", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Description", Value: stringOrDash(item.Description)},
				{Label: "Version", Value: stringOrDash(item.Version)},
				{Label: "Source", Value: stringOrDash(item.Source)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Enabled", Value: strconv.FormatBool(item.Enabled)},
			})

			metadataRows := make([][]string, 0, len(item.Metadata))
			for _, entry := range sortedSkillMetadataEntries(item.Metadata) {
				metadataRows = append(metadataRows, []string{entry.Label, entry.Value})
			}
			metadata := renderHumanTable("Metadata", []string{"Key", "Value"}, metadataRows)

			resourceRows := make([][]string, 0, len(item.Resources))
			for _, resource := range item.Resources {
				resourceRows = append(resourceRows, []string{resource})
			}
			resources := renderHumanTable("Resources", []string{"Path"}, resourceRows)

			return renderHumanBlocks(base, metadata, resources), nil
		},
		toon: func() (string, error) {
			metadataRows := make([][]string, 0, len(item.Metadata))
			for _, entry := range sortedSkillMetadataEntries(item.Metadata) {
				metadataRows = append(metadataRows, []string{entry.Label, entry.Value})
			}

			resourceRows := make([][]string, 0, len(item.Resources))
			for _, resource := range item.Resources {
				resourceRows = append(resourceRows, []string{resource})
			}

			return renderHumanBlocks(
				renderToonObject("skill", []string{"name", "description", "version", "source", "path", "enabled"}, []string{
					item.Name,
					item.Description,
					item.Version,
					item.Source,
					item.Path,
					strconv.FormatBool(item.Enabled),
				}),
				renderToonArray("metadata", []string{"key", "value"}, metadataRows),
				renderToonArray("resources", []string{"path"}, resourceRows),
			), nil
		},
	}
}

func skillCreateBundle(item skillCreateItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Skill", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Source", Value: stringOrDash(item.Source)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "File", Value: stringOrDash(item.File)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("skill", []string{"name", "source", "path", "file", "status"}, []string{
				item.Name,
				item.Source,
				item.Path,
				item.File,
				item.Status,
			}), nil
		},
	}
}

func sortedSkillMetadataEntries(metadata map[string]any) []keyValue {
	if len(metadata) == 0 {
		return nil
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]keyValue, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, keyValue{
			Label: key,
			Value: formatSkillMetadataValue(metadata[key]),
		})
	}
	return entries
}

func formatSkillMetadataValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return compactJSON(payload)
	}
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	clone := make(map[string]any, len(metadata))
	for key, value := range metadata {
		clone[key] = value
	}
	return clone
}
