import { AlertCircle, Wrench } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Empty,
  PillGroup,
  Spinner,
  SplitPane,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { type SkillsRouteSearch, useSkillsPage } from "@/hooks/routes/use-skills-page";
import { MarketplaceView, SkillDetailPanel, SkillListPanel } from "@/systems/skill";

function normalizeSearchValue(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function validateSkillsSearch(search: Record<string, unknown>): SkillsRouteSearch {
  return {
    content: normalizeSearchValue(search.content),
    q: normalizeSearchValue(search.q),
    skill: normalizeSearchValue(search.skill),
    tab: search.tab === "installed" || search.tab === "marketplace" ? search.tab : undefined,
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

  useTopbarSlot({
    count: page.activeTab === "marketplace" ? page.marketplaceSkillCount : page.skillCount,
    tabs: (
      <PillGroup<SkillsTabValue>
        aria-label="Skills tab"
        data-testid="skills-tabs"
        items={TAB_ITEMS}
        onChange={value => page.setActiveTab(value)}
        size="sm"
        value={page.activeTab as SkillsTabValue}
      />
    ),
  });

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="skills-loading">
        <Spinner aria-hidden="true" className="size-5 text-(--subtle)" />
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

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="skills-shell">
      {page.backgroundError ? (
        <div className="border-b border-(--line) px-6 py-3">
          <Alert data-testid="skills-background-error" variant="warning">
            <AlertCircle aria-hidden="true" className="size-4" />
            <AlertTitle>Showing cached skills</AlertTitle>
            <AlertDescription>
              {page.backgroundError.message ??
                "The latest skill refresh failed. Existing data remains available."}
            </AlertDescription>
          </Alert>
        </div>
      ) : null}
      {page.activeTab === "installed" ? (
        <SplitPane
          data-testid="skills-split-pane"
          detail={
            <SkillDetailPanel
              content={page.selectedSkillContent}
              contentError={page.contentError}
              error={page.detailError}
              isActionPending={page.isActionPending}
              isContentLoading={page.isContentLoading}
              isLoading={page.isLoadingDetail}
              onDisable={page.handleDisable}
              onEnable={page.handleEnable}
              onRetryContent={page.handleRetryContent}
              onViewContent={page.handleViewContent}
              skill={page.selectedSkill}
            />
          }
          list={
            <SkillListPanel
              onSearchChange={page.setSearchQuery}
              onSelectSkill={page.setSelectedSkillName}
              searchQuery={page.searchQuery}
              selectedSkillName={page.effectiveSelectedName}
              skills={page.skills}
            />
          }
        />
      ) : (
        <MarketplaceView
          installedSkillNames={page.installedSkillNames}
          installUnavailableReason="The daemon API only exposes metadata for already installed marketplace skills here. Remote marketplace search and install are not available in this view yet."
          isInstalling={false}
          onSearchChange={page.setSearchQuery}
          searchQuery={page.searchQuery}
          skills={page.marketplaceSkills}
        />
      )}
    </div>
  );
}
