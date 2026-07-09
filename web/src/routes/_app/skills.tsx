import { AlertCircle, RefreshCw, Store, Wrench } from "lucide-react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  PillGroup,
  Spinner,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  parseSkillEnabledFilter,
  parseSkillSourceFilter,
  type SkillsRouteSearch,
  useSkillsPage,
} from "@/hooks/routes/use-skills-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { MarketplaceView, SkillListFilters, SkillListPanel } from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";

function validateSkillsSearch(search: Record<string, unknown>): SkillsRouteSearch {
  return {
    enabled: parseSkillEnabledFilter(search.enabled),
    q: normalizeListingSearchValue(search.q),
    source: parseSkillSourceFilter(search.source),
    tab: search.tab === "installed" || search.tab === "marketplace" ? search.tab : undefined,
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/skills")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Skills", icon: Wrench },
  }),
  validateSearch: validateSkillsSearch,
  component: SkillsPage,
});

const TAB_ITEMS = [
  { value: "installed", label: "Installed", testId: "tab-installed" },
  { value: "marketplace", label: "Marketplace", testId: "tab-marketplace" },
] as const;

type SkillsTabValue = (typeof TAB_ITEMS)[number]["value"];

function SkillsPage() {
  const page = useSkillsPage(Route.useSearch());
  const { activeWorkspace } = useActiveWorkspace();

  useTopbarSlot({
    count: page.hasChildMatch
      ? undefined
      : page.activeTab === "marketplace"
        ? page.marketplaceListingCount
        : page.skillCount,
    tabs: page.hasChildMatch ? undefined : (
      <PillGroup<SkillsTabValue>
        aria-label="Skills tab"
        data-testid="skills-tabs"
        items={TAB_ITEMS}
        onChange={value => page.setActiveTab(value)}
        size="sm"
        value={page.activeTab as SkillsTabValue}
      />
    ),
    actions: page.hasChildMatch ? undefined : (
      <div className="flex items-center gap-2" data-testid="skills-topbar-actions">
        <Button
          data-testid="skills-refresh"
          onClick={page.handleRefresh}
          size="sm"
          type="button"
          variant="ghost"
        >
          <RefreshCw aria-hidden="true" className="size-3" />
          Refresh
        </Button>
        {page.activeTab === "installed" ? (
          <Button
            data-testid="skills-browse-marketplace"
            onClick={page.browseMarketplace}
            size="sm"
            type="button"
          >
            <Store aria-hidden="true" className="size-3" />
            Browse marketplace
          </Button>
        ) : null}
      </div>
    ),
  });

  if (page.hasChildMatch) {
    return <Outlet />;
  }

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="skills-loading">
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="skills-error"
      >
        <Empty
          className="max-w-md"
          description={page.error.message ?? "Failed to load skills"}
          icon={AlertCircle}
          title="Unable to load skills"
        />
      </div>
    );
  }

  const isMarketplace = page.activeTab === "marketplace";
  const count = isMarketplace ? page.marketplaceListingCount : page.skillCount;
  const workspaceLabel = activeWorkspace?.name ?? activeWorkspace?.id ?? "workspace";

  return (
    <ListingPage
      banner={
        page.backgroundError ? (
          <div className="border-b border-line px-9 py-3">
            <Alert data-testid="skills-background-error" variant="warning">
              <AlertCircle aria-hidden="true" className="size-4" />
              <AlertTitle>Showing cached skills</AlertTitle>
              <AlertDescription>
                {page.backgroundError.message ??
                  "The latest skill refresh failed. Existing data remains available."}
              </AlertDescription>
            </Alert>
          </div>
        ) : null
      }
      data-testid="skills-shell"
    >
      <ListingPage.Head
        count={count}
        countTestId="skills-page-count"
        data-testid="skills-page-head"
        meta={
          <>
            <span>
              {isMarketplace
                ? "Browse and install skills from the marketplace."
                : "Installed skills available to agents in this workspace."}
            </span>
            <ListingPage.MetaDot />
            <span>{workspaceLabel}</span>
          </>
        }
        title="Skills"
      />

      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label={isMarketplace ? "Search marketplace skills" : "Search installed skills"}
            data-testid={isMarketplace ? "marketplace-search-input" : "skill-search-input"}
            onChange={page.setSearchQuery}
            placeholder={isMarketplace ? "Search skills on the marketplace…" : "Search skills"}
            value={page.searchQuery}
          />
          {!isMarketplace ? (
            <ListingToolbar.Filters>
              <SkillListFilters
                enabledFilter={page.enabledFilter}
                onEnabledFilterChange={page.setEnabledFilter}
                onSourceFilterChange={page.setSourceFilter}
                sourceFilter={page.sourceFilter}
              />
            </ListingToolbar.Filters>
          ) : null}
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
        </ListingToolbar.Trailing>
      </ListingToolbar>

      {isMarketplace ? (
        <MarketplaceView
          installedSkillNames={page.installedSkillNames}
          isInstalling={page.isInstalling}
          isRemoving={page.isRemoving}
          isSearchEnabled={page.isMarketplaceSearchEnabled}
          isSearching={page.isMarketplaceSearching}
          isUpdating={page.isUpdating}
          listings={page.marketplaceListings}
          onClearSearch={() => page.setSearchQuery("")}
          onInstall={page.handleInstallMarketplace}
          onRemove={page.handleRemoveMarketplace}
          onUpdate={page.handleUpdateMarketplace}
          searchError={page.marketplaceSearchError}
          view={page.view}
        />
      ) : (
        <div data-testid="skill-list-panel">
          <SkillListPanel
            enabledFilter={page.enabledFilter}
            isActionPending={page.isActionPending}
            onClearFilters={page.clearFilters}
            onDisable={page.handleDisable}
            onEnable={page.handleEnable}
            searchQuery={page.searchQuery}
            sourceFilter={page.sourceFilter}
            skills={page.skills}
            view={page.view}
          />
        </div>
      )}
    </ListingPage>
  );
}
