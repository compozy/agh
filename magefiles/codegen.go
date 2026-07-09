//go:build mage

package main

import (
	"context"

	"github.com/compozy/agh/internal/codegen/openapits"
)

func Codegen() error {
	if err := DaytonaSidecars(); err != nil {
		return err
	}
	if err := runCommandInDir(context.Background(), ".", "go", "run", "./cmd/agh-codegen", "all"); err != nil {
		return err
	}
	artifacts, err := availableWebOpenAPIArtifacts()
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if err := openapits.Generate(context.Background(), artifact); err != nil {
			return err
		}
	}
	return SyncDesignMD()
}

func CodegenCheck() error {
	if err := daytonaSidecarsCheckStamped(); err != nil {
		return err
	}
	if err := runCommandInDir(context.Background(), ".", "go", "run", "./cmd/agh-codegen", "check"); err != nil {
		return err
	}
	artifacts, err := availableWebOpenAPIArtifacts()
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if err := openapits.Check(context.Background(), artifact); err != nil {
			return err
		}
	}
	return SyncDesignMDCheck()
}

// SyncDesignMD refreshes generated DESIGN.md token frontmatter and tables.

// SyncDesignMD refreshes generated DESIGN.md token frontmatter and tables.
func SyncDesignMD() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", designSyncScriptPath, "--write")
}

// SyncDesignMDCheck verifies generated DESIGN.md token frontmatter and tables.

// SyncDesignMDCheck verifies generated DESIGN.md token frontmatter and tables.
func SyncDesignMDCheck() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", designSyncScriptPath, "--check")
}

// BunLint runs the monorepo-wide lint script (oxfmt + oxlint over every workspace).
