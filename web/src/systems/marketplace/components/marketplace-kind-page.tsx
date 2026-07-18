import { AlertCircle, SearchX } from "lucide-react";

import {
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  PageHead,
  PillGroup,
  RouteState,
} from "@agh/ui";

import { useListingSearchShortcut } from "@/hooks/use-listing-search-shortcut";
import { useMarketplaceActionController } from "./use-marketplace-action-controller";
import { MarketplaceCard } from "./marketplace-card";
import { MarketplaceGridSkeleton } from "./marketplace-grid";
import { MarketplaceInstalledCard } from "./marketplace-installed-card";
import { MarketplaceDegradedNotice } from "./marketplace-degraded-notice";
import { MARKETPLACE_SCOPE_ICONS, marketplaceKindConfig } from "../lib/marketplace-kind-config";
import { useMarketplaceKindPage } from "../hooks/use-marketplace-kind-page";
import type { MarketplaceKind } from "../types";
import type { MarketplaceKindSearch } from "../lib/marketplace-kind-search";

interface MarketplaceKindPageProps {
  kind: MarketplaceKind;
  search: MarketplaceKindSearch;
}

function MarketplaceKindPage({ kind, search }: MarketplaceKindPageProps) {
  const searchInputRef = useListingSearchShortcut();
  const page = useMarketplaceKindPage(kind, search);
  const actions = useMarketplaceActionController(page.workspaceId, {
    installedItems: page.installedItems,
    onViewInstalled: () => page.setScope("installed"),
  });
  const config = marketplaceKindConfig(kind);
  const ScopeInstalledIcon = MARKETPLACE_SCOPE_ICONS.installed;
  const ScopeMarketIcon = MARKETPLACE_SCOPE_ICONS.market;
  const headCount = page.scope === "market" ? page.marketplaceTotal : page.installedCount;
  const updatesLabel =
    page.updatesAvailable === 1
      ? "1 update available"
      : `${page.updatesAvailable} updates available`;

  return (
    <ListingPage data-testid={`marketplace-kind-${kind}`}>
      <PageHead
        count={page.isLoading && !page.marketplaceTotal ? "–" : headCount}
        countTestId={`marketplace-kind-count-${kind}`}
        data-testid={`marketplace-kind-head-${kind}`}
        icon={config.icon}
        meta={
          <>
            <span>
              <span className="font-mono text-[11px] tabular-nums text-muted">
                {page.marketplaceTotal}
              </span>{" "}
              in the marketplace
            </span>
            <PageHead.MetaDot />
            <span>
              <span className="font-mono text-[11px] tabular-nums text-muted">
                {page.installedCount}
              </span>{" "}
              {config.installedNoun}
            </span>
            {page.updatesAvailable > 0 ? (
              <>
                <PageHead.MetaDot />
                <span>
                  <span className="font-mono text-[11px] tabular-nums text-muted">
                    {page.updatesAvailable}
                  </span>{" "}
                  {updatesLabel.replace(/^\d+\s/, "")}
                </span>
              </>
            ) : null}
          </>
        }
        title={config.label}
      />

      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label={`Search ${config.label.toLowerCase()}`}
            containerClassName="w-full max-w-105"
            data-testid={`marketplace-kind-search-${kind}`}
            onChange={page.setDraftQuery}
            placeholder={config.searchPlaceholder}
            ref={searchInputRef}
            value={page.draftQuery}
          />
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <PillGroup
            aria-label="Marketplace scope"
            data-testid={`marketplace-scope-${kind}`}
            items={[
              {
                value: "installed" as const,
                testId: `marketplace-scope-installed-${kind}`,
                label: (
                  <span className="inline-flex items-center gap-1.5">
                    <ScopeInstalledIcon aria-hidden="true" className="size-3.5" />
                    <span>Installed</span>
                    <span className="inline-flex h-(--size-pill-group-badge) min-w-(--size-pill-group-badge) items-center justify-center rounded-mono-badge bg-badge-fill px-(--space-pill-group-badge-x) text-pill-group-badge font-medium tabular-nums text-muted">
                      {page.installedCount}
                    </span>
                  </span>
                ),
              },
              {
                value: "market" as const,
                testId: `marketplace-scope-market-${kind}`,
                label: (
                  <span className="inline-flex items-center gap-1.5">
                    <ScopeMarketIcon aria-hidden="true" className="size-3.5" />
                    <span>Marketplace</span>
                  </span>
                ),
              },
            ]}
            onChange={page.setScope}
            value={page.scope}
          />
        </ListingToolbar.Trailing>
      </ListingToolbar>

      {page.error && page.scope === "market" && page.marketEntries.length > 0 ? (
        <MarketplaceDegradedNotice
          hasItems
          label={config.label}
          onRetry={() => page.refetch()}
          scope="kind"
        />
      ) : null}

      {page.isLoading ? (
        <MarketplaceGridSkeleton count={6} />
      ) : page.error &&
        ((page.scope === "market" && page.marketEntries.length === 0) ||
          (page.scope === "installed" && page.installedItems.length === 0)) ? (
        <RouteState
          action={
            <Button onClick={() => page.refetch()} size="sm" type="button">
              Retry
            </Button>
          }
          cause={page.error.message}
          message="No sources responded. Retry, or come back in a moment."
          mode="error"
          title="The marketplace is unreachable"
        />
      ) : page.scope === "market" && page.marketEntries.length === 0 ? (
        page.query ? (
          <Empty
            action={
              <Button onClick={page.clearSearch} size="sm" type="button" variant="outline">
                Clear search
              </Button>
            }
            data-testid={`marketplace-query-empty-${kind}`}
            description={`Nothing matches "${page.query}" in the marketplace.`}
            icon={SearchX}
            title={`No ${config.label.toLowerCase()} match this query`}
          />
        ) : (
          <Empty
            data-testid={`marketplace-empty-${kind}`}
            description={`No ${config.label.toLowerCase()} are available from configured sources.`}
            icon={AlertCircle}
            title={`No ${config.label.toLowerCase()} yet`}
          />
        )
      ) : page.scope === "installed" && page.installedItems.length === 0 ? (
        page.query ? (
          <Empty
            action={
              <Button onClick={page.clearSearch} size="sm" type="button" variant="outline">
                Clear search
              </Button>
            }
            data-testid={`marketplace-installed-query-empty-${kind}`}
            description={`Nothing matches "${page.query}" in your ${config.installedNoun} ${config.label.toLowerCase()}.`}
            icon={SearchX}
            title={`No ${config.label.toLowerCase()} match this query`}
          />
        ) : (
          <Empty
            action={
              <Button
                data-testid={`marketplace-browse-market-${kind}`}
                onClick={() => page.setScope("market")}
                size="sm"
                type="button"
              >
                Browse the marketplace
              </Button>
            }
            data-testid={`marketplace-installed-empty-${kind}`}
            description={
              <>
                {config.teachingEmptyBody} You can also use{" "}
                <code className="rounded-xs border border-line-soft bg-input-fill px-1.5 py-px font-mono text-xs text-fg">
                  {config.cliHint}
                </code>
                .
              </>
            }
            icon={config.icon}
            title={config.teachingEmptyTitle}
          />
        )
      ) : page.scope === "market" ? (
        <div
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
          data-testid="marketplace-grid"
          data-view="cards"
        >
          {page.marketEntries.map(entry => (
            <MarketplaceCard
              entry={entry}
              flashing={actions.isEntryFlashing(entry)}
              key={`${entry.kind}:${entry.entry_id}`}
              onAction={actions.handleAction}
              pending={actions.isEntryPending(entry)}
            />
          ))}
        </div>
      ) : (
        <div
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
          data-testid="marketplace-installed-grid"
          data-view="cards"
        >
          {page.installedItems.map(item => (
            <MarketplaceInstalledCard
              item={item}
              key={`${item.entry.kind}:${item.entry.entry_id}:${item.activationId ?? item.skill?.name ?? item.mcpServer?.name ?? ""}`}
              onAction={actions.handleAction}
              onAuthorize={actions.handleAuthorize}
              onDeactivate={actions.handleDeactivate}
              onRemove={actions.handleRemove}
              onToggleEnabled={actions.handleToggleEnabled}
              onUpdateBundle={actions.handleUpdateBundle}
              pending={actions.isEntryPending(item.entry)}
            />
          ))}
        </div>
      )}

      {actions.dialogs}
      <span aria-live="polite" className="sr-only">
        {page.query
          ? `Search updated · ${page.scope === "market" ? page.marketEntries.length : page.installedItems.length} results`
          : ""}
      </span>
    </ListingPage>
  );
}

export { MarketplaceKindPage };
export type { MarketplaceKindPageProps };
