package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/marketplace"
	"github.com/pedronauck/agh/internal/skills/marketplace/clawhub"
)

type installedMarketplaceSkill struct {
	Name       string
	Dir        string
	FilePath   string
	Provenance skills.Provenance
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
	targetDirOverride string,
) (item skillInstallItem, err error) {
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
		err = joinContextError(err, archive.Data.Close(), "cli: close marketplace archive for %q: %w", slug)
	}()

	tempRoot, err := os.MkdirTemp(runtime.HomePaths.SkillsDir, ".agh-skill-install-*")
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: create temporary install directory: %w", err)
	}
	defer func() {
		// Best-effort cleanup; install correctness is determined by the primary result.
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

	content, err := skills.ReadSkillContent(skillFile)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: read extracted skill content for %q: %w", slug, err)
	}

	if critical := criticalWarnings(skills.VerifyContent(content)); len(critical) > 0 {
		return skillInstallItem{}, fmt.Errorf(
			"cli: install blocked for %q due to critical verification findings: %s",
			slug,
			strings.Join(critical, "; "),
		)
	}

	hash, err := skills.ComputeDirectoryHash(parsedSkill.Dir)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: compute extracted skill hash for %q: %w", slug, err)
	}

	version := firstNonEmpty(archive.Version, parsedSkill.Meta.Version)
	targetDir, err := resolveMarketplaceInstallTarget(runtime.HomePaths.SkillsDir, parsedSkill.Meta.Name, targetDirOverride)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: resolve install path for %q: %w", slug, err)
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

	installedItem, err := installMarketplaceSkill(ctx, runtime, registry, registryName, slug, true, installed.Dir)
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

func extractMarketplaceArchive(reader io.Reader, destRoot string) (err error) {
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
		err = joinContextError(err, gzipReader.Close(), "close gzip stream: %w")
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
				writeErr := fmt.Errorf("write archive file %q: %w", targetPath, err)
				if closeErr := file.Close(); closeErr != nil {
					return errors.Join(writeErr, fmt.Errorf("close archive file %q after write failure: %w", targetPath, closeErr))
				}
				return writeErr
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

func pathInsideRoot(root string, target string) (string, error) {
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}

	absTarget, err := filepath.Abs(strings.TrimSpace(target))
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", target, err)
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

func resolveMarketplaceInstallTarget(skillsDir string, parsedName string, targetDirOverride string) (string, error) {
	if trimmedOverride := strings.TrimSpace(targetDirOverride); trimmedOverride != "" {
		return pathInsideRoot(skillsDir, trimmedOverride)
	}
	return pathWithinRoot(skillsDir, parsedName)
}

func joinContextError(base error, extra error, format string, args ...any) error {
	if extra == nil {
		return base
	}

	args = append(args, extra)
	wrapped := fmt.Errorf(format, args...)
	if base == nil {
		return wrapped
	}
	return errors.Join(base, wrapped)
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

	currentVersion, currentOK := parseSemanticVersion(normalizedCurrent)
	latestVersion, latestOK := parseSemanticVersion(normalizedLatest)
	if currentOK && latestOK {
		return compareSemanticVersions(currentVersion, latestVersion) < 0
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

type semanticVersion struct {
	core       []int
	prerelease []string
}

func parseSemanticVersion(version string) (semanticVersion, bool) {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return semanticVersion{}, false
	}

	corePart, _, _ := strings.Cut(trimmed, "+")
	corePart, prereleasePart, hasPrerelease := strings.Cut(corePart, "-")

	core, ok := parseVersionParts(corePart)
	if !ok {
		return semanticVersion{}, false
	}

	parsed := semanticVersion{core: core}
	if !hasPrerelease {
		return parsed, true
	}

	identifiers, ok := parsePrereleaseIdentifiers(prereleasePart)
	if !ok {
		return semanticVersion{}, false
	}
	parsed.prerelease = identifiers
	return parsed, true
}

func parsePrereleaseIdentifiers(value string) ([]string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, false
	}

	parts := strings.Split(trimmed, ".")
	identifiers := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		identifiers = append(identifiers, part)
	}
	return identifiers, true
}

func compareSemanticVersions(current semanticVersion, latest semanticVersion) int {
	for i := 0; i < max(len(current.core), len(latest.core)); i++ {
		currentPart := versionPartAt(current.core, i)
		latestPart := versionPartAt(latest.core, i)
		switch {
		case currentPart < latestPart:
			return -1
		case currentPart > latestPart:
			return 1
		}
	}

	switch {
	case len(current.prerelease) == 0 && len(latest.prerelease) == 0:
		return 0
	case len(current.prerelease) == 0:
		return 1
	case len(latest.prerelease) == 0:
		return -1
	default:
		return comparePrereleaseIdentifiers(current.prerelease, latest.prerelease)
	}
}

func comparePrereleaseIdentifiers(current []string, latest []string) int {
	for i := 0; i < max(len(current), len(latest)); i++ {
		switch {
		case i >= len(current):
			return -1
		case i >= len(latest):
			return 1
		}

		currentID := current[i]
		latestID := latest[i]
		currentNumber, currentNumeric := parseNumericIdentifier(currentID)
		latestNumber, latestNumeric := parseNumericIdentifier(latestID)

		switch {
		case currentNumeric && latestNumeric:
			switch {
			case currentNumber < latestNumber:
				return -1
			case currentNumber > latestNumber:
				return 1
			}
		case currentNumeric:
			return -1
		case latestNumeric:
			return 1
		case currentID < latestID:
			return -1
		case currentID > latestID:
			return 1
		}
	}

	return 0
}

func parseNumericIdentifier(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return number, true
}
