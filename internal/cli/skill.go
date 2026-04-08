package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/skills"
	skillbundled "github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/pedronauck/agh/internal/skills/marketplace"
	"github.com/pedronauck/agh/internal/skills/marketplace/clawhub"
	"github.com/pedronauck/agh/internal/store/globaldb"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
	"github.com/spf13/cobra"
)

const (
	defaultSkillName            = "new-skill"
	skillMarkdownFileName       = "SKILL.md"
	defaultMarketplaceRegistry  = "clawhub"
	defaultMarketplaceSearchLim = 20
)

var (
	skillXMLAttributeReplacer = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;")
	skillXMLTextReplacer      = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;")
	validSkillNamePattern     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	validSkillSlugPattern     = regexp.MustCompile(`^@[^/\s]+/[^/\s]+$`)
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

type skillInstallItem struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Version  string `json:"version,omitempty"`
	Registry string `json:"registry"`
	Path     string `json:"path"`
	Hash     string `json:"hash"`
	Status   string `json:"status"`
}

type skillRemoveItem struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

type skillUpdateItem struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Path           string `json:"path"`
	Status         string `json:"status"`
}

type installedMarketplaceSkill struct {
	Name       string
	Dir        string
	FilePath   string
	Provenance skills.Provenance
}

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
	cmd.Flags().StringVar(&sourceFilter, "source", "", "Filter by source: bundled, user, additional, or workspace")
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

func newSkillSearchCommand(deps commandDeps) *cobra.Command {
	limit := defaultMarketplaceSearchLim

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search marketplace skills",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("cli: search limit must be positive: %d", limit)
			}

			_, registry, _, err := loadMarketplaceRegistry(deps)
			if err != nil {
				return err
			}

			results, err := registry.Search(cmd.Context(), args[0], marketplace.SearchOpts{Limit: limit})
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := normalizeSkillSlug(args[0])
			if err != nil {
				return err
			}

			runtime, registry, registryName, err := loadMarketplaceRegistry(deps)
			if err != nil {
				return err
			}

			item, err := installMarketplaceSkill(cmd.Context(), runtime, registry, registryName, slug, false)
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
		Args:  cobra.ExactArgs(1),
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

	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update installed marketplace skills",
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
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, registry, registryName, err := loadMarketplaceRegistry(deps)
			if err != nil {
				return err
			}

			items, err := updateMarketplaceSkills(cmd.Context(), runtime, registry, registryName, args, updateAll)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, skillUpdateBundle(items))
		},
	}
	cmd.Flags().BoolVar(&updateAll, "all", false, "Update every installed marketplace skill")
	return cmd
}

func loadMarketplaceRegistry(deps commandDeps) (runtimeContext, marketplace.Registry, string, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return runtimeContext{}, nil, "", err
	}

	registryCfg := runtime.Config.Skills.Marketplace
	registryName := strings.ToLower(strings.TrimSpace(registryCfg.Registry))
	if registryName == "" {
		registryName = defaultMarketplaceRegistry
	}

	switch registryName {
	case defaultMarketplaceRegistry:
		return runtime, clawhub.NewClient(registryCfg.BaseURL), registryName, nil
	default:
		return runtimeContext{}, nil, "", fmt.Errorf("cli: unsupported marketplace registry %q", registryCfg.Registry)
	}
}

func normalizeSkillSlug(slug string) (string, error) {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return "", errors.New("skill slug is required")
	}
	if !validSkillSlugPattern.MatchString(trimmed) {
		return "", errors.New(`skill slug must match "@author/name"`)
	}
	return trimmed, nil
}

func installMarketplaceSkill(
	ctx context.Context,
	runtime runtimeContext,
	registry marketplace.Registry,
	registryName string,
	slug string,
	replaceExisting bool,
) (skillInstallItem, error) {
	if err := os.MkdirAll(runtime.HomePaths.SkillsDir, 0o755); err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: create skills directory %q: %w", runtime.HomePaths.SkillsDir, err)
	}

	archive, err := registry.Download(ctx, slug)
	if err != nil {
		return skillInstallItem{}, err
	}
	if archive == nil {
		return skillInstallItem{}, fmt.Errorf("cli: marketplace download returned no archive for %q", slug)
	}
	if archive.Data == nil {
		return skillInstallItem{}, fmt.Errorf("cli: marketplace download returned no archive stream for %q", slug)
	}
	defer func() {
		_ = archive.Data.Close()
	}()

	tempRoot, err := os.MkdirTemp(runtime.HomePaths.SkillsDir, ".agh-skill-install-*")
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: create temporary install directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempRoot)
	}()

	if err := extractMarketplaceArchive(archive.Data, tempRoot); err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: extract skill archive for %q: %w", slug, err)
	}

	skillFile, err := locateExtractedSkillFile(tempRoot)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: locate extracted skill for %q: %w", slug, err)
	}

	parsedSkill, err := skills.ParseSkillFile(skillFile)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: parse extracted skill for %q: %w", slug, err)
	}

	if critical := criticalWarnings(skills.VerifyContent(parsedSkill.Content)); len(critical) > 0 {
		return skillInstallItem{}, fmt.Errorf(
			"cli: install blocked for %q due to critical verification findings: %s",
			slug,
			strings.Join(critical, "; "),
		)
	}

	skillBytes, err := os.ReadFile(parsedSkill.FilePath)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: read extracted skill file %q: %w", parsedSkill.FilePath, err)
	}
	hash := skills.ComputeHash(skillBytes)

	version := firstNonEmpty(archive.Version, parsedSkill.Meta.Version)
	targetDir, err := pathWithinRoot(runtime.HomePaths.SkillsDir, parsedSkill.Meta.Name)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: resolve install path for %q: %w", parsedSkill.Meta.Name, err)
	}

	if err := skills.WriteSidecar(parsedSkill.Dir, skills.Provenance{
		Hash:        hash,
		Registry:    registryName,
		Slug:        slug,
		Version:     version,
		InstalledAt: time.Now().UTC(),
	}); err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: write provenance for %q: %w", slug, err)
	}

	if err := moveInstalledSkillDir(parsedSkill.Dir, targetDir, replaceExisting); err != nil {
		return skillInstallItem{}, err
	}

	return skillInstallItem{
		Name:     parsedSkill.Meta.Name,
		Slug:     slug,
		Version:  version,
		Registry: registryName,
		Path:     targetDir,
		Hash:     hash,
		Status:   "installed",
	}, nil
}

func updateMarketplaceSkills(
	ctx context.Context,
	runtime runtimeContext,
	registry marketplace.Registry,
	registryName string,
	args []string,
	updateAll bool,
) ([]skillUpdateItem, error) {
	if updateAll {
		installedSkills, err := listInstalledMarketplaceSkills(runtime.HomePaths.SkillsDir)
		if err != nil {
			return nil, err
		}

		items := make([]skillUpdateItem, 0, len(installedSkills))
		for _, installed := range installedSkills {
			item, err := updateMarketplaceSkill(ctx, runtime, registry, registryName, installed)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	}

	name, err := normalizeSkillName(args[0])
	if err != nil {
		return nil, err
	}

	installed, err := findInstalledMarketplaceSkill(runtime.HomePaths.SkillsDir, name)
	if err != nil {
		return nil, err
	}

	item, err := updateMarketplaceSkill(ctx, runtime, registry, registryName, installed)
	if err != nil {
		return nil, err
	}
	return []skillUpdateItem{item}, nil
}

func updateMarketplaceSkill(
	ctx context.Context,
	runtime runtimeContext,
	registry marketplace.Registry,
	registryName string,
	installed installedMarketplaceSkill,
) (skillUpdateItem, error) {
	slug := strings.TrimSpace(installed.Provenance.Slug)
	if slug == "" {
		return skillUpdateItem{}, fmt.Errorf("cli: marketplace skill %q is missing registry slug metadata", installed.Name)
	}

	detail, err := registry.Info(ctx, slug)
	if err != nil {
		return skillUpdateItem{}, err
	}

	currentVersion := strings.TrimSpace(installed.Provenance.Version)
	latestVersion := ""
	if detail != nil {
		latestVersion = strings.TrimSpace(detail.Version)
	}
	if !versionIsNewer(currentVersion, latestVersion) {
		return skillUpdateItem{
			Name:           installed.Name,
			Slug:           slug,
			CurrentVersion: currentVersion,
			LatestVersion:  firstNonEmpty(latestVersion, currentVersion),
			Path:           installed.Dir,
			Status:         "already up to date",
		}, nil
	}

	installedItem, err := installMarketplaceSkill(ctx, runtime, registry, registryName, slug, true)
	if err != nil {
		return skillUpdateItem{}, err
	}

	return skillUpdateItem{
		Name:           installedItem.Name,
		Slug:           slug,
		CurrentVersion: currentVersion,
		LatestVersion:  firstNonEmpty(installedItem.Version, latestVersion),
		Path:           installedItem.Path,
		Status:         "updated",
	}, nil
}

func removeMarketplaceSkill(skillsDir string, name string) (skillRemoveItem, error) {
	installed, err := findInstalledMarketplaceSkill(skillsDir, name)
	if err != nil {
		return skillRemoveItem{}, err
	}

	if err := os.RemoveAll(installed.Dir); err != nil {
		return skillRemoveItem{}, fmt.Errorf("cli: remove marketplace skill %q: %w", name, err)
	}

	return skillRemoveItem{
		Name:   installed.Name,
		Slug:   installed.Provenance.Slug,
		Path:   installed.Dir,
		Status: "removed",
	}, nil
}

func findInstalledMarketplaceSkill(skillsDir string, name string) (installedMarketplaceSkill, error) {
	skillDir, err := pathWithinRoot(skillsDir, name)
	if err != nil {
		return installedMarketplaceSkill{}, fmt.Errorf("cli: resolve skill path for %q: %w", name, err)
	}

	info, err := os.Stat(skillDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return installedMarketplaceSkill{}, fmt.Errorf("skill %q not found", name)
		}
		return installedMarketplaceSkill{}, fmt.Errorf("cli: inspect skill directory %q: %w", skillDir, err)
	}
	if !info.IsDir() {
		return installedMarketplaceSkill{}, fmt.Errorf("skill %q is not a directory", name)
	}

	hasSidecar, err := skills.HasSidecar(skillDir)
	if err != nil {
		return installedMarketplaceSkill{}, err
	}
	if !hasSidecar {
		return installedMarketplaceSkill{}, fmt.Errorf("skill %q is not a marketplace-installed skill", name)
	}

	return readInstalledMarketplaceSkill(skillDir)
}

func listInstalledMarketplaceSkills(skillsDir string) ([]installedMarketplaceSkill, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []installedMarketplaceSkill{}, nil
		}
		return nil, fmt.Errorf("cli: read installed skills directory %q: %w", skillsDir, err)
	}

	items := make([]installedMarketplaceSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir, err := pathWithinRoot(skillsDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("cli: resolve installed skill path for %q: %w", entry.Name(), err)
		}

		hasSidecar, err := skills.HasSidecar(skillDir)
		if err != nil {
			return nil, err
		}
		if !hasSidecar {
			continue
		}

		item, err := readInstalledMarketplaceSkill(skillDir)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func readInstalledMarketplaceSkill(skillDir string) (installedMarketplaceSkill, error) {
	provenance, err := skills.ReadSidecar(skillDir)
	if err != nil {
		return installedMarketplaceSkill{}, err
	}
	if provenance == nil {
		return installedMarketplaceSkill{}, fmt.Errorf("cli: missing provenance for %q", skillDir)
	}

	skillFile, err := pathWithinRoot(skillDir, skillMarkdownFileName)
	if err != nil {
		return installedMarketplaceSkill{}, fmt.Errorf("cli: resolve skill file in %q: %w", skillDir, err)
	}

	parsedSkill, err := skills.ParseSkillFile(skillFile)
	if err != nil {
		return installedMarketplaceSkill{}, err
	}

	return installedMarketplaceSkill{
		Name:       parsedSkill.Meta.Name,
		Dir:        parsedSkill.Dir,
		FilePath:   parsedSkill.FilePath,
		Provenance: *provenance,
	}, nil
}

func extractMarketplaceArchive(reader io.Reader, destRoot string) error {
	if strings.TrimSpace(destRoot) == "" {
		return errors.New("destination root is required")
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return fmt.Errorf("create destination root %q: %w", destRoot, err)
	}

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		entryName, err := cleanArchiveEntryPath(header.Name)
		if err != nil {
			return err
		}
		targetPath, err := pathWithinRoot(destRoot, filepath.FromSlash(entryName))
		if err != nil {
			return fmt.Errorf("resolve archive entry %q: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("create archive directory %q: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("create archive parent %q: %w", filepath.Dir(targetPath), err)
			}

			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("create archive file %q: %w", targetPath, err)
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				_ = file.Close()
				return fmt.Errorf("write archive file %q: %w", targetPath, err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close archive file %q: %w", targetPath, err)
			}
		default:
			return fmt.Errorf("unsupported archive entry type %d for %q", header.Typeflag, header.Name)
		}
	}
}

func locateExtractedSkillFile(root string) (string, error) {
	var matches []string

	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != skillMarkdownFileName {
			return nil
		}
		matches = append(matches, current)
		if len(matches) > 1 {
			return errors.New("multiple SKILL.md files found in archive")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", errors.New("archive did not contain SKILL.md")
	}
	return matches[0], nil
}

func moveInstalledSkillDir(extractedDir string, targetDir string, replaceExisting bool) error {
	if !replaceExisting {
		if _, err := os.Stat(targetDir); err == nil {
			return fmt.Errorf("skill %q already exists at %s", filepath.Base(targetDir), targetDir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cli: inspect target skill directory %q: %w", targetDir, err)
		}

		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("cli: install skill into %q: %w", targetDir, err)
		}
		return nil
	}

	if _, err := os.Stat(targetDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cli: inspect target skill directory %q: %w", targetDir, err)
		}
		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("cli: install updated skill into %q: %w", targetDir, err)
		}
		return nil
	}

	backupDir := fmt.Sprintf("%s.backup-%d", targetDir, time.Now().UTC().UnixNano())
	if err := os.Rename(targetDir, backupDir); err != nil {
		return fmt.Errorf("cli: stage existing skill backup %q: %w", targetDir, err)
	}

	if err := os.Rename(extractedDir, targetDir); err != nil {
		revertErr := os.Rename(backupDir, targetDir)
		if revertErr != nil {
			return errors.Join(
				fmt.Errorf("cli: install updated skill into %q: %w", targetDir, err),
				fmt.Errorf("cli: restore original skill from %q: %w", backupDir, revertErr),
			)
		}
		return fmt.Errorf("cli: install updated skill into %q: %w", targetDir, err)
	}

	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("cli: remove backup skill directory %q: %w", backupDir, err)
	}
	return nil
}

func cleanArchiveEntryPath(entry string) (string, error) {
	cleaned := path.Clean(strings.TrimSpace(strings.ReplaceAll(entry, "\\", "/")))
	switch {
	case cleaned == ".", cleaned == "":
		return "", errors.New("archive entry path is required")
	case strings.HasPrefix(cleaned, "/"):
		return "", fmt.Errorf("archive entry %q must be relative", entry)
	case cleaned == "..", strings.HasPrefix(cleaned, "../"):
		return "", fmt.Errorf("archive entry %q escapes the extraction root", entry)
	default:
		return cleaned, nil
	}
}

func pathWithinRoot(root string, child string) (string, error) {
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	targetPath := filepath.Join(absRoot, child)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", targetPath, err)
	}
	relative, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q within %q: %w", absTarget, absRoot, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay within the root directory")
	}
	return absTarget, nil
}

func criticalWarnings(warnings []skills.Warning) []string {
	items := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if warning.Severity != skills.SeverityCritical {
			continue
		}
		items = append(items, firstNonEmpty(warning.Message, warning.Pattern))
	}
	return items
}

func versionIsNewer(current string, latest string) bool {
	normalizedCurrent := normalizeVersion(current)
	normalizedLatest := normalizeVersion(latest)
	if normalizedLatest == "" {
		return false
	}
	if normalizedCurrent == "" {
		return true
	}

	currentParts, currentNumeric := parseVersionParts(normalizedCurrent)
	latestParts, latestNumeric := parseVersionParts(normalizedLatest)
	if currentNumeric && latestNumeric {
		for i := 0; i < max(len(currentParts), len(latestParts)); i++ {
			currentPart := versionPartAt(currentParts, i)
			latestPart := versionPartAt(latestParts, i)
			switch {
			case latestPart > currentPart:
				return true
			case latestPart < currentPart:
				return false
			}
		}
		return false
	}

	return normalizedLatest > normalizedCurrent
}

func normalizeVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	return trimmed
}

func parseVersionParts(version string) ([]int, bool) {
	segments := strings.Split(version, ".")
	if len(segments) == 0 {
		return nil, false
	}

	parts := make([]int, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			return nil, false
		}
		value, err := strconv.Atoi(segment)
		if err != nil {
			return nil, false
		}
		parts = append(parts, value)
	}
	return parts, true
}

func versionPartAt(parts []int, index int) int {
	if index < 0 || index >= len(parts) {
		return 0
	}
	return parts[index]
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

	userAgentsDir, err := aghconfig.ResolveUserAgentsSkillsDir(deps.getenv)
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

	resolvedWorkspace, err := resolveSkillWorkspace(ctx, runtime, workspace)
	if err != nil {
		return skillCommandContext{}, err
	}

	skillList, err := registry.ForWorkspace(ctx, resolvedWorkspace)
	if err != nil {
		return skillCommandContext{}, err
	}

	return skillCommandContext{
		workspace: workspace,
		bundledFS: skillbundled.FS(),
		skills:    skillList,
	}, nil
}

func resolveSkillWorkspace(ctx context.Context, runtime runtimeContext, workspaceRoot string) (workspacepkg.ResolvedWorkspace, error) {
	fallback, err := cliResolvedWorkspace(workspaceRoot)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, err
	}

	if strings.TrimSpace(workspaceRoot) == "" {
		return fallback, nil
	}

	if _, err := os.Stat(runtime.HomePaths.DatabaseFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fallback, nil
		}
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("cli: stat workspace database %q: %w", runtime.HomePaths.DatabaseFile, err)
	}

	resolved, err := resolveRegisteredSkillWorkspace(ctx, runtime, workspaceRoot)
	if err != nil {
		if errors.Is(err, workspacepkg.ErrWorkspaceNotFound) {
			return fallback, nil
		}
		return workspacepkg.ResolvedWorkspace{}, err
	}

	return resolved, nil
}

func resolveRegisteredSkillWorkspace(ctx context.Context, runtime runtimeContext, workspaceRoot string) (resolved workspacepkg.ResolvedWorkspace, err error) {
	globalDB, err := globaldb.OpenGlobalDB(ctx, runtime.HomePaths.DatabaseFile)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("cli: open workspace database %q: %w", runtime.HomePaths.DatabaseFile, err)
	}
	defer func() {
		if closeErr := globalDB.Close(ctx); closeErr != nil {
			closeErr = fmt.Errorf("cli: close workspace database %q: %w", runtime.HomePaths.DatabaseFile, closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()

	resolver, err := workspacepkg.NewResolver(
		globalDB,
		workspacepkg.WithHomePaths(runtime.HomePaths),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(runtime.HomePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("cli: create workspace resolver: %w", err)
	}

	resolved, err = resolver.Resolve(ctx, workspaceRoot)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("cli: resolve workspace %q: %w", workspaceRoot, err)
	}

	return resolved, nil
}

func cliResolvedWorkspace(root string) (workspacepkg.ResolvedWorkspace, error) {
	workspaceRoot := strings.TrimSpace(root)
	if workspaceRoot == "" {
		return workspacepkg.ResolvedWorkspace{}, nil
	}

	skillRoots, err := os.ReadDir(filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.SkillsDirName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{RootDir: workspaceRoot},
			}, nil
		}
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("cli: read workspace skills %q: %w", workspaceRoot, err)
	}

	skillPaths := make([]workspacepkg.SkillPath, 0, len(skillRoots))
	for _, entry := range skillRoots {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.SkillsDirName, entry.Name())
		skillFile := filepath.Join(skillDir, skillMarkdownFileName)
		if _, err := os.Stat(skillFile); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("cli: inspect workspace skill %q: %w", skillFile, err)
		}

		skillPaths = append(skillPaths, workspacepkg.SkillPath{
			Dir:    skillDir,
			Source: "workspace",
		})
	}

	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{RootDir: workspaceRoot},
		Skills:    skillPaths,
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
	case "bundled", "marketplace", "user", "additional", "workspace":
		return filter, nil
	case "agents", ".agents":
		return "additional", nil
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
	case skills.SourceMarketplace:
		return "marketplace"
	case skills.SourceUser:
		return "user"
	case skills.SourceAdditional:
		return "additional"
	case skills.SourceWorkspace:
		return "workspace"
	default:
		return "unknown"
	}
}

func skillSearchBundle(items []marketplace.SkillListing) outputBundle {
	return listBundle(
		items,
		items,
		"Marketplace Skills",
		[]string{"Slug", "Name", "Description", "Author", "Version", "Downloads"},
		"skills",
		[]string{"slug", "name", "description", "author", "version", "downloads"},
		func(item marketplace.SkillListing) []string {
			return []string{
				stringOrDash(item.Slug),
				stringOrDash(item.Name),
				stringOrDash(item.Description),
				stringOrDash(item.Author),
				stringOrDash(item.Version),
				strconv.Itoa(item.Downloads),
			}
		},
		func(item marketplace.SkillListing) []string {
			return []string{
				item.Slug,
				item.Name,
				item.Description,
				item.Author,
				item.Version,
				strconv.Itoa(item.Downloads),
			}
		},
	)
}

func skillListBundle(items []skillListItem) outputBundle {
	return listBundle(
		items,
		items,
		"Skills",
		[]string{"Name", "Description", "Source", "Enabled"},
		"skills",
		[]string{"name", "description", "source", "enabled"},
		func(item skillListItem) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Description),
				stringOrDash(item.Source),
				strconv.FormatBool(item.Enabled),
			}
		},
		func(item skillListItem) []string {
			return []string{
				item.Name,
				item.Description,
				item.Source,
				strconv.FormatBool(item.Enabled),
			}
		},
	)
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

func skillInstallBundle(item skillInstallItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Skill Install", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Slug", Value: stringOrDash(item.Slug)},
				{Label: "Version", Value: stringOrDash(item.Version)},
				{Label: "Registry", Value: stringOrDash(item.Registry)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Hash", Value: stringOrDash(item.Hash)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("skill_install", []string{"name", "slug", "version", "registry", "path", "hash", "status"}, []string{
				item.Name,
				item.Slug,
				item.Version,
				item.Registry,
				item.Path,
				item.Hash,
				item.Status,
			}), nil
		},
	}
}

func skillRemoveBundle(item skillRemoveItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Skill Remove", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Slug", Value: stringOrDash(item.Slug)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("skill_remove", []string{"name", "slug", "path", "status"}, []string{
				item.Name,
				item.Slug,
				item.Path,
				item.Status,
			}), nil
		},
	}
}

func skillUpdateBundle(items []skillUpdateItem) outputBundle {
	return listBundle(
		items,
		items,
		"Skill Updates",
		[]string{"Name", "Slug", "Current", "Latest", "Path", "Status"},
		"skill_updates",
		[]string{"name", "slug", "current_version", "latest_version", "path", "status"},
		func(item skillUpdateItem) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Slug),
				stringOrDash(item.CurrentVersion),
				stringOrDash(item.LatestVersion),
				stringOrDash(item.Path),
				stringOrDash(item.Status),
			}
		},
		func(item skillUpdateItem) []string {
			return []string{
				item.Name,
				item.Slug,
				item.CurrentVersion,
				item.LatestVersion,
				item.Path,
				item.Status,
			}
		},
	)
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
