import { AlertCircle, Loader2, Wrench } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Empty,
  PageHeader,
  SplitPane,
  Tabs,
  TabsList,
  TabsTrigger,
} from "@agh/ui";
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
  validateSearch: validateSkillsSearch,
  component: SkillsPage,
});

function SkillsPage() {
  const page = useSkillsPage(Route.useSearch());

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="skills-loading">
        <Loader2 aria-hidden="true" className="size-5 animate-spin text-(--color-text-tertiary)" />
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

  const controls = (
    <Tabs
      aria-label="Skills tab"
      data-testid="skills-tabs"
      onValueChange={value => page.setActiveTab(value as typeof page.activeTab)}
      value={page.activeTab}
    >
      <TabsList className="h-8" variant="default">
        <TabsTrigger data-testid="tab-installed" value="installed">
          Installed
        </TabsTrigger>
        <TabsTrigger data-testid="tab-marketplace" value="marketplace">
          Marketplace
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="skills-shell">
      <PageHeader
        count={page.activeTab === "marketplace" ? page.marketplaceSkillCount : page.skillCount}
        controls={controls}
        icon={() => <Wrench className="size-3.5" data-testid="skills-shell-icon" />}
        title={<span data-testid="skills-shell-title">Skills</span>}
      />
      {page.backgroundError ? (
        <div className="border-b border-(--color-divider) px-6 py-3">
          <Alert
            className="border-(--color-warning)/40"
            data-testid="skills-background-error"
            variant="warning"
          >
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
