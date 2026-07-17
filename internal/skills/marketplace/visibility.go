package marketplace

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/skills"
)

// VerifyInstallVisible verifies that a completed marketplace install is visible
// through AGH skill discovery with the marketplace provenance that agents rely on.
func VerifyInstallVisible(resolver SkillResolver, result InstallResult) error {
	if resolver == nil {
		return classifiedf(
			ErrUnavailable,
			"installed marketplace skill %q cannot be verified because skill discovery is not configured",
			result.Name,
		)
	}
	skill, ok := resolver.Get(result.Name)
	if !ok {
		return classifiedf(
			ErrUnavailable,
			"installed marketplace skill %q is not visible after skill discovery; inspect %s and retry the install",
			result.Name,
			result.Path,
		)
	}
	if skill.Source != skills.SourceMarketplace {
		return classifiedf(
			ErrUnavailable,
			"installed marketplace skill %q resolved as %s%s after skill discovery; "+
				"remove or disable that declaration and retry the install",
			result.Name,
			skills.SkillSourceName(skill.Source),
			installVisibilityLocation(skill),
		)
	}
	if skill.Provenance == nil {
		return classifiedf(
			ErrUnavailable,
			"installed marketplace skill %q is missing provenance after skill discovery; inspect %s and retry the install",
			result.Name,
			result.Path,
		)
	}
	if strings.TrimSpace(skill.Provenance.Slug) != strings.TrimSpace(result.Slug) {
		return classifiedf(
			ErrUnavailable,
			"installed marketplace skill %q resolved slug %q after skill discovery, want %q; "+
				"remove the conflicting skill and retry the install",
			result.Name,
			skill.Provenance.Slug,
			result.Slug,
		)
	}
	if !skill.Enabled {
		return classifiedf(
			ErrUnavailable,
			"installed marketplace skill %q is visible but disabled after skill discovery; "+
				"enable the skill and retry discovery",
			result.Name,
		)
	}
	return nil
}

func installVisibilityLocation(skill *skills.Skill) string {
	if skill == nil {
		return ""
	}
	path := firstNonEmpty(skill.FilePath, skill.Dir)
	if path == "" {
		return ""
	}
	return fmt.Sprintf(" from %s", path)
}
