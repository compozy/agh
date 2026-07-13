package globaldb

import (
	"path/filepath"
	"strings"
)

func sessionsDirForDatabasePath(path string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cleanPath), "sessions")
}
