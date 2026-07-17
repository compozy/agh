package settings

import (
	"context"
	"strings"

	"github.com/compozy/agh/internal/marketplace"
)

func (s *service) notifyMCPCatalogInstalled(ctx context.Context, entryID string) {
	if s == nil || s.marketplaceInstallEvents == nil {
		return
	}
	s.marketplaceInstallEvents.NotifyInstall(context.WithoutCancel(ctx), marketplace.InstallOutcome{
		Kind:       marketplace.KindMCP,
		EntryID:    strings.TrimSpace(entryID),
		Outcome:    marketplace.InstallOutcomeSucceeded,
		PolicyGate: marketplace.InstallPolicyGatePassed,
	})
}
