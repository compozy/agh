package bundles

import "strings"

func activationSpecDrift(storedHash string, currentHash string) bool {
	return strings.TrimSpace(storedHash) != strings.TrimSpace(currentHash)
}
