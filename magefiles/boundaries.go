//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
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
		if _, err := os.Stat(importerDir); os.IsNotExist(err) {
			continue
		}
		importPath := "github.com/compozy/agh/" + rule.imported
		cmd := exec.Command("grep", "-r", "--include=*.go", "-l", importPath, importerDir)
		out, err := cmd.Output()
		if err != nil {
			continue // grep returns exit 1 when no match — that's good
		}
		if len(strings.TrimSpace(string(out))) > 0 {
			fmt.Printf("VIOLATION: %s imports %s\n", rule.importer, rule.imported)
			for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				fmt.Printf("  %s\n", f)
			}
			violations++
		}
	}

	if violations > 0 {
		return fmt.Errorf("found %d boundary violations", violations)
	}
	fmt.Println("OK: all package boundaries respected")
	return nil
}
