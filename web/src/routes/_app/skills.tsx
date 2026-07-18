import { AlertCircle, RefreshCw, Wrench } from "lucide-react";
import { Link, Outlet, createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  PageHead,
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
import { SkillListFilters, SkillListPanel } from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";
import { preloadSkillsRoute } from "./-skill-preload";

function validateSkillsSearch(search: Record<string, unknown>): SkillsRouteSearch {
  return {
    enabled: parseSkillEnabledFilter(search.enabled),
    q: normalizeListingSearchValue(search.q),
    source: parseSkillSourceFilter(search.source),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/skills")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Skills", to: "/skills" } },
  }),
  validateSearch: validateSkillsSearch,
  loader: ({ context }) => preloadSkillsRoute(context.queryClient),
  component: SkillsPage,
});

function SkillsPage() {
  const page = useSkillsPage(Route.useSearch());
  const { activeWorkspace } = useActiveWorkspace();

  // Publish null while a child route is mounted: a non-null slot from this
  // parent would steal the detail route's publish (layout effects run child
  // first, parent last in the same commit).
  useTopbarSlot(
    page.hasChildMatch
      ? null
      : {
          actions: (
            <div className="flex items-center gap-2" data-testid="skills-topbar-actions">
              <Button
                data-testid="skills-browse-marketplace"
                render={<Link search={{ kind: "skills" }} to="/marketplace" />}
                nativeButton={false}
                size="sm"
                variant="ghost"
              >
                Browse marketplace
              </Button>
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
            </div>
          ),
        }
  );

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
        className="flex min-h-0 flex-1 items-center justify-center py-10"
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
      <PageHead
        count={page.skillCount}
        countTestId="skills-page-count"
        data-testid="skills-page-head"
        icon={Wrench}
        meta={
          <>
            <span>Installed skills available to agents in this workspace.</span>
            <PageHead.MetaDot />
            <span>{workspaceLabel}</span>
          </>
        }
        title="Skills"
      />

      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search installed skills"
            data-testid="skill-search-input"
            onChange={page.setSearchQuery}
            placeholder="Search skills"
            value={page.searchQuery}
          />
          <ListingToolbar.Filters>
            <SkillListFilters
              enabledFilter={page.enabledFilter}
              onEnabledFilterChange={page.setEnabledFilter}
              onSourceFilterChange={page.setSourceFilter}
              sourceFilter={page.sourceFilter}
            />
          </ListingToolbar.Filters>
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
        </ListingToolbar.Trailing>
      </ListingToolbar>

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
    </ListingPage>
  );
}
