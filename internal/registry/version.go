package registry

import (
	"strconv"
	"strings"
)

// VersionIsNewer reports whether latest is a valid semantic version newer than
// current. Invalid version strings never produce an update signal.
func VersionIsNewer(current string, latest string) bool {
	normalizedCurrent := normalizeVersion(current)
	normalizedLatest := normalizeVersion(latest)
	if normalizedLatest == "" {
		return false
	}

	latestVersion, latestOK := parseSemanticVersion(normalizedLatest)
	if !latestOK {
		return false
	}
	if normalizedCurrent == "" {
		return true
	}

	currentVersion, currentOK := parseSemanticVersion(normalizedCurrent)
	if !currentOK {
		return false
	}

	return compareSemanticVersions(currentVersion, latestVersion) < 0
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
