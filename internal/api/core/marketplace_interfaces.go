package core

import (
	"context"

	marketplacepkg "github.com/compozy/agh/internal/marketplace"
	registrypkg "github.com/compozy/agh/internal/registry"
	skillmarketplace "github.com/compozy/agh/internal/skills/marketplace"
)

// MarketplaceCatalogService exposes the daemon-owned curated feed projection.
type MarketplaceCatalogService interface {
	marketplacepkg.Service
	marketplacepkg.SkillInstallResolver
}

// SkillMarketplaceService exposes remote skill marketplace lifecycle operations.
type SkillMarketplaceService interface {
	Search(ctx context.Context, query string, limit int) ([]registrypkg.Listing, error)
	Info(ctx context.Context, slug string) (*registrypkg.Detail, error)
	Install(ctx context.Context, slug string, version string) (skillmarketplace.InstallResult, error)
	Update(ctx context.Context, req skillmarketplace.UpdateRequest) ([]skillmarketplace.UpdateResult, error)
	Remove(ctx context.Context, name string) (skillmarketplace.RemoveResult, error)
}

// InstalledSkillMarketplaceService exposes the global marketplace sidecar projection.
type InstalledSkillMarketplaceService interface {
	ListInstalled(ctx context.Context) ([]skillmarketplace.InstalledSkill, error)
}
