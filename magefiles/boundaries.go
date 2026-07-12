//go:build mage

package main

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Boundaries verifies that package import rules are not violated.
// Rules: no package may import daemon/, api/httpapi/, api/udsapi/, or cli/.
func Boundaries() error {
	forbidden := []struct {
		importer string
		imported string
	}{
		{"internal/config", "internal/daemon"},
		{"internal/acp", "internal/daemon"},
		{"internal/session", "internal/daemon"},
		{"internal/store", "internal/daemon"},
		{"internal/observe", "internal/daemon"},
		{"internal/events", "internal/daemon"},
		{"internal/doctor", "internal/daemon"},
		{"internal/providers", "internal/daemon"},
		{"internal/loop", "internal/daemon"},
		{"internal/diagnosticcontract", "internal/daemon"},
		{"internal/config", "internal/api/httpapi"},
		{"internal/acp", "internal/api/httpapi"},
		{"internal/session", "internal/api/httpapi"},
		{"internal/store", "internal/api/httpapi"},
		{"internal/observe", "internal/api/httpapi"},
		{"internal/events", "internal/api/httpapi"},
		{"internal/doctor", "internal/api/httpapi"},
		{"internal/providers", "internal/api/httpapi"},
		{"internal/loop", "internal/api/httpapi"},
		{"internal/diagnosticcontract", "internal/api/httpapi"},
		{"internal/config", "internal/api/udsapi"},
		{"internal/acp", "internal/api/udsapi"},
		{"internal/session", "internal/api/udsapi"},
		{"internal/store", "internal/api/udsapi"},
		{"internal/observe", "internal/api/udsapi"},
		{"internal/events", "internal/api/udsapi"},
		{"internal/doctor", "internal/api/udsapi"},
		{"internal/providers", "internal/api/udsapi"},
		{"internal/loop", "internal/api/udsapi"},
		{"internal/diagnosticcontract", "internal/api/udsapi"},
		{"internal/config", "internal/cli"},
		{"internal/acp", "internal/cli"},
		{"internal/session", "internal/cli"},
		{"internal/store", "internal/cli"},
		{"internal/observe", "internal/cli"},
		{"internal/events", "internal/cli"},
		{"internal/doctor", "internal/cli"},
		{"internal/providers", "internal/cli"},
		{"internal/loop", "internal/cli"},
		{"internal/diagnosticcontract", "internal/cli"},
		{"internal/providers", "internal/session"},
		{"internal/providers", "internal/acp"},
		{"internal/providers", "internal/api/core"},
		{"internal/api/contract", "internal/daemon"},
		{"internal/api/contract", "internal/api/httpapi"},
		{"internal/api/contract", "internal/api/udsapi"},
		{"internal/api/contract", "internal/cli"},
		{"internal/diagnosticcontract", "internal/api/contract"},
		{"internal/diagnosticcontract", "internal/api/core"},
		{"internal/events", "internal/api/contract"},
		{"internal/events", "internal/api/core"},
		{"internal/loop", "internal/api/contract"},
		{"internal/loop", "internal/api/core"},
		{"internal/loop", "internal/loop/goal"},
		{"internal/automation", "internal/loop"},
		{"internal/automation", "internal/loop/dsl"},
		{"internal/api/core", "internal/daemon"},
		{"internal/api/core", "internal/api/httpapi"},
		{"internal/api/core", "internal/api/udsapi"},
		{"internal/api/core", "internal/cli"},
		{"internal/api/httpapi", "internal/daemon"},
		{"internal/api/httpapi", "internal/api/udsapi"},
		{"internal/api/httpapi", "internal/cli"},
		{"internal/api/udsapi", "internal/daemon"},
		{"internal/api/udsapi", "internal/api/httpapi"},
		{"internal/api/udsapi", "internal/cli"},
		{"internal/modelcatalog", "internal/daemon"},
		{"internal/modelcatalog", "internal/api/contract"},
		{"internal/modelcatalog", "internal/api/core"},
		{"internal/modelcatalog", "internal/api/httpapi"},
		{"internal/modelcatalog", "internal/api/udsapi"},
		{"internal/modelcatalog", "internal/cli"},
		{"internal/memory/contract", "internal/memory/controller"},
		{"internal/memory/contract", "internal/memory/recall"},
		{"internal/memory/contract", "internal/memory/extractor"},
		{"internal/memory/contract", "internal/memory/provider/local"},
		{"internal/memory/contract", "internal/store/workspacedb"},
		{"internal/memory/controller", "internal/daemon"},
		{"internal/memory/controller", "internal/api/httpapi"},
		{"internal/memory/controller", "internal/api/udsapi"},
		{"internal/memory/controller", "internal/cli"},
		{"internal/memory/recall", "internal/daemon"},
		{"internal/memory/recall", "internal/api/httpapi"},
		{"internal/memory/recall", "internal/api/udsapi"},
		{"internal/memory/recall", "internal/cli"},
		{"internal/memory/extractor", "internal/daemon"},
		{"internal/memory/extractor", "internal/api/httpapi"},
		{"internal/memory/extractor", "internal/api/udsapi"},
		{"internal/memory/extractor", "internal/cli"},
		{"internal/memory/provider/local", "internal/daemon"},
		{"internal/memory/provider/local", "internal/api/httpapi"},
		{"internal/memory/provider/local", "internal/api/udsapi"},
		{"internal/memory/provider/local", "internal/cli"},
		{"internal/sessions/ledger", "internal/daemon"},
		{"internal/sessions/ledger", "internal/api/httpapi"},
		{"internal/sessions/ledger", "internal/api/udsapi"},
		{"internal/sessions/ledger", "internal/cli"},
		{"internal/store/workspacedb", "internal/daemon"},
		{"internal/store/workspacedb", "internal/api/httpapi"},
		{"internal/store/workspacedb", "internal/api/udsapi"},
		{"internal/store/workspacedb", "internal/cli"},
	}

	violations := 0
	for _, rule := range forbidden {
		importerDir := rule.importer
		if _, err := os.Stat(importerDir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect boundary importer %q: %w", importerDir, err)
		}
		importPath := "github.com/compozy/agh/" + rule.imported
		files, err := filesImporting(importerDir, importPath)
		if err != nil {
			return fmt.Errorf("check whether %q imports %q: %w", importerDir, importPath, err)
		}
		if len(files) > 0 {
			fmt.Printf("VIOLATION: %s imports %s\n", rule.importer, rule.imported)
			for _, file := range files {
				fmt.Printf("  %s\n", file)
			}
			violations++
		}
	}

	leafRules := []struct {
		importer string
		allowed  map[string]struct{}
	}{
		{importer: "internal/redact", allowed: map[string]struct{}{}},
		{
			importer: "internal/toolmeta",
			allowed: map[string]struct{}{
				"github.com/compozy/agh/internal/redact": {},
			},
		},
	}
	for _, rule := range leafRules {
		files, err := productionFilesImportingInternalExcept(rule.importer, rule.allowed)
		if err != nil {
			return fmt.Errorf("inspect leaf boundary importer %q: %w", rule.importer, err)
		}
		if len(files) == 0 {
			continue
		}
		fmt.Printf("VIOLATION: %s imports a forbidden internal package\n", rule.importer)
		for _, file := range files {
			fmt.Printf("  %s\n", file)
		}
		violations++
	}

	if violations > 0 {
		return fmt.Errorf("found %d boundary violations", violations)
	}
	fmt.Println("OK: all package boundaries respected")
	return nil
}

func productionFilesImportingInternalExcept(
	root string,
	allowed map[string]struct{},
) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	const internalPrefix = "github.com/compozy/agh/internal/"
	fset := token.NewFileSet()
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse Go imports in %q: %w", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("decode Go import in %q: %w", path, err)
			}
			if !strings.HasPrefix(importPath, internalPrefix) {
				continue
			}
			if _, ok := allowed[importPath]; ok {
				continue
			}
			files = append(files, path)
			break
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func filesImporting(root string, target string) ([]string, error) {
	fset := token.NewFileSet()
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		parsed, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse Go imports in %q: %w", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("decode Go import in %q: %w", path, err)
			}
			if importPath == target {
				files = append(files, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
