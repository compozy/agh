package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/skills"
	skillbundled "github.com/pedronauck/agh/skills"
	"github.com/spf13/cobra"
)

const (
	agentLocalSkillSource = "agent-local"
)

const (
	additionalSkillSource  = "additional"
	bundledSkillSource     = "bundled"
	marketplaceSkillSource = "marketplace"
	userSkillSource        = "user"
)

type skillCommandScope struct {
	query     SkillQuery
	useDaemon bool
}

func loadSkillCommandContext(ctx context.Context, deps commandDeps, agentName string) (skillCommandContext, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return skillCommandContext{}, fmt.Errorf("cli: load skill runtime context: %w", err)
	}

	registry := skills.NewRegistry(skills.RegistryConfig{
		BundledFS:      skillbundled.FS(),
		UserAgentsDir:  runtime.HomePaths.AgentsDir,
		UserSkillsDir:  runtime.HomePaths.SkillsDir,
		DisabledSkills: append([]string(nil), runtime.Config.Skills.DisabledSkills...),
	})
	if err := registry.LoadAll(ctx); err != nil {
		return skillCommandContext{}, fmt.Errorf("cli: load skill registry: %w", err)
	}

	var skillList []*skills.Skill
	if strings.TrimSpace(agentName) != "" {
		skillList, err = registry.ForAgent(ctx, nil, agentName)
		if err != nil {
			return skillCommandContext{}, fmt.Errorf("cli: load agent skills: %w", err)
		}
	} else {
		skillList = registry.List()
	}

	return skillCommandContext{
		bundledFS: skillbundled.FS(),
		registry:  registry,
		skills:    skillList,
	}, nil
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

func resolveSkillCommandScope(
	ctx context.Context,
	cmd *cobra.Command,
	deps commandDeps,
	originRef string,
) (skillCommandScope, error) {
	workspaceRef, err := commandWorkspaceFlag(cmd)
	if err != nil {
		return skillCommandScope{}, err
	}
	agentRef, err := commandSkillAgentFlag(cmd)
	if err != nil {
		return skillCommandScope{}, err
	}

	scope := skillCommandScope{
		query: SkillQuery{
			Workspace: workspaceRef,
			ForAgent:  agentRef,
		},
		useDaemon: workspaceRef != "",
	}
	if scope.useDaemon {
		return scope, nil
	}
	if agentRef != "" {
		return scope, nil
	}

	credentials := agentCredentialsFromEnv(deps)
	if strings.TrimSpace(credentials.SessionID) == "" && strings.TrimSpace(credentials.AgentName) == "" {
		return scope, nil
	}

	client, err := clientFromDeps(deps)
	if err != nil {
		return skillCommandScope{}, err
	}
	caller, err := resolveAgentCallerFromEnv(ctx, deps, client, "", originRef)
	if err != nil {
		return skillCommandScope{}, err
	}

	scope.query = SkillQuery{
		Workspace: firstNonEmpty(
			strings.TrimSpace(caller.Session.WorkspaceID),
			strings.TrimSpace(caller.Session.WorkspacePath),
		),
		ForAgent: strings.TrimSpace(caller.Session.AgentName),
	}
	scope.useDaemon = scope.query.Workspace != "" || scope.query.ForAgent != ""
	return scope, nil
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

func skillListItemsFromRecords(records []SkillRecord, sourceFilter string) ([]skillListItem, error) {
	filter, err := normalizeSkillSourceFilter(sourceFilter)
	if err != nil {
		return nil, err
	}

	items := make([]skillListItem, 0, len(records))
	for _, record := range records {
		source := strings.TrimSpace(record.Source)
		if filter != "" && source != filter {
			continue
		}

		items = append(items, skillListItem{
			Name:        record.Name,
			Description: record.Description,
			Source:      source,
			Enabled:     record.Enabled,
		})
	}

	return items, nil
}

func skillInfoItemFromRecord(record SkillRecord) skillInfoItem {
	return skillInfoItem{
		Name:        record.Name,
		Description: record.Description,
		Version:     record.Version,
		Source:      strings.TrimSpace(record.Source),
		Path:        record.Dir,
		Enabled:     record.Enabled,
		Metadata:    cloneMetadata(record.Metadata),
	}
}

func commandWorkspaceFlag(cmd *cobra.Command) (string, error) {
	workspace, err := cmd.Flags().GetString("workspace")
	if err != nil {
		return "", fmt.Errorf("read workspace flag: %w", err)
	}
	trimmed := strings.TrimSpace(workspace)
	if cmd.Flags().Changed("workspace") && trimmed == "" {
		return "", fmt.Errorf("workspace flag cannot be empty")
	}
	return trimmed, nil
}

func commandSkillAgentFlag(cmd *cobra.Command) (string, error) {
	agentName, err := cmd.Flags().GetString("for-agent")
	if err != nil {
		return "", fmt.Errorf("read for-agent flag: %w", err)
	}
	trimmed := strings.TrimSpace(agentName)
	if cmd.Flags().Changed("for-agent") && trimmed == "" {
		return "", fmt.Errorf("for-agent flag cannot be empty")
	}
	if trimmed != "" {
		if err := aghconfig.ValidateAgentName(trimmed); err != nil {
			return "", err
		}
	}
	return trimmed, nil
}

func normalizeSkillSourceFilter(sourceFilter string) (string, error) {
	filter := strings.ToLower(strings.TrimSpace(sourceFilter))
	switch filter {
	case "":
		return "", nil
	case bundledSkillSource,
		marketplaceSkillSource,
		userSkillSource,
		additionalSkillSource,
		workspaceSkillSource,
		agentLocalSkillSource:
		return filter, nil
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

func renderSkillXML(skill *skills.Skill, content string, resources []string) (string, error) {
	if skill == nil {
		return "", errors.New("skill is required")
	}

	var builder strings.Builder
	builder.WriteString(`<skill_content name="`)
	builder.WriteString(skillXMLAttributeReplacer.Replace(skill.Meta.Name))
	builder.WriteString(`">`)
	builder.WriteString("\n")
	builder.WriteString(skillXMLTextReplacer.Replace(content))
	if !strings.HasSuffix(content, "\n") {
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
		return bundledSkillSource
	case skills.SourceMarketplace:
		return marketplaceSkillSource
	case skills.SourceUser:
		return userSkillSource
	case skills.SourceAdditional:
		return additionalSkillSource
	case skills.SourceWorkspace:
		return workspaceSkillSource
	case skills.SourceAgentLocal:
		return agentLocalSkillSource
	default:
		return "unknown"
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
	maps.Copy(clone, metadata)
	return clone
}
