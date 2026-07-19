import { Plus } from "lucide-react";

import { Button, ListingPage, ListingToolbar, PageHead, PillGroup } from "@agh/ui";

import { useListingSearchShortcut } from "@/hooks/use-listing-search-shortcut";
import { MCPServerEditor } from "@/systems/settings";
import { MarketplaceDegradedNotice } from "./marketplace-degraded-notice";
import { MarketplaceKindResults } from "./marketplace-kind-results";
import { useMarketplaceActionController } from "./use-marketplace-action-controller";
import { MARKETPLACE_SCOPE_ICONS, marketplaceKindConfig } from "../lib/marketplace-kind-config";
import { useMarketplaceKindPage } from "../hooks/use-marketplace-kind-page";
import { useMarketplaceMCPEditor } from "../hooks/use-marketplace-mcp-editor";
import type { MarketplaceKind } from "../types";
import type { MarketplaceKindSearch } from "../lib/marketplace-kind-search";

interface MarketplaceKindPageProps {
  kind: MarketplaceKind;
  search: MarketplaceKindSearch;
}

function MarketplaceKindPage({ kind, search }: MarketplaceKindPageProps) {
  const searchInputRef = useListingSearchShortcut();
  const page = useMarketplaceKindPage(kind, search);
  const mcpEditor = useMarketplaceMCPEditor({
    enabled: kind === "mcp",
    scope: page.mcpConfigScope,
    servers: page.mcpEditorServers,
    workspaceId: page.workspaceId,
  });
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
              {page.marketplaceTotalExact ? "in the marketplace" : "loaded from the marketplace"}
            </span>
            <PageHead.MetaDot />
            <span>
              <span className="font-mono text-[11px] tabular-nums text-muted">
                {page.installedCount}
              </span>{" "}
              {config.installedNoun}
            </span>
            {!page.isLoading && page.updatesAvailable > 0 ? (
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
          <div className="flex flex-wrap items-center gap-2">
            {kind === "mcp" && page.scope === "installed" ? (
              <PillGroup
                aria-label="New MCP server scope"
                data-testid="marketplace-mcp-config-scope"
                items={[
                  {
                    value: "workspace" as const,
                    label: "Workspace",
                    testId: "marketplace-mcp-config-scope-workspace",
                  },
                  {
                    value: "global" as const,
                    label: "Global",
                    testId: "marketplace-mcp-config-scope-global",
                  },
                ]}
                onChange={page.setMCPConfigScope}
                value={page.mcpConfigScope}
              />
            ) : null}
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
            {kind === "mcp" && page.scope === "installed" ? (
              <Button
                data-testid="marketplace-mcp-add"
                disabled={page.mcpConfigScope === "workspace" && !page.workspaceId}
                onClick={mcpEditor.openCreate}
                size="sm"
                type="button"
              >
                <Plus aria-hidden="true" className="size-3" />
                Add MCP server
              </Button>
            ) : null}
          </div>
        </ListingToolbar.Trailing>
      </ListingToolbar>

      {page.error &&
      !page.marketplaceContinuationError &&
      ((page.scope === "market" && page.marketEntries.length > 0) ||
        (page.scope === "installed" && page.installedItems.length > 0)) ? (
        <MarketplaceDegradedNotice
          hasItems
          label={config.label}
          onRetry={() => page.refetch()}
          scope="kind"
        />
      ) : null}

      <MarketplaceKindResults
        actions={actions}
        config={config}
        kind={kind}
        onEditMCP={mcpEditor.openEdit}
        page={page}
      />

      {actions.dialogs}
      {mcpEditor.editorProps ? <MCPServerEditor {...mcpEditor.editorProps} /> : null}
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
