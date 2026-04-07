// Package marketplace defines the pluggable skill marketplace contract shared by
// CLI commands and registry backends.
package marketplace

import "context"

// Registry defines the marketplace backend contract.
type Registry interface {
	Search(ctx context.Context, query string, opts SearchOpts) ([]SkillListing, error)
	Download(ctx context.Context, slug string) (*SkillArchive, error)
	Info(ctx context.Context, slug string) (*SkillDetail, error)
}
