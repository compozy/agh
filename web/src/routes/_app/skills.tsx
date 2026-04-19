import { AlertCircle, Loader2, Wrench } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty, PageHeader, SplitPane, Tabs, TabsList, TabsTrigger } from "@agh/ui";
import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { MarketplaceView, SkillDetailPanel, SkillListPanel } from "@/systems/skill";

export const Route = createFileRoute("/_app/skills")({
  component: SkillsPage,
});

function SkillsPage() {
  const page = useSkillsPage();

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="skills-loading">
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
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
        count={page.skillCount}
        controls={controls}
        icon={() => <Wrench className="size-3.5" data-testid="skills-shell-icon" />}
        title={<span data-testid="skills-shell-title">Skills</span>}
      />
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
          installUnavailableReason="Marketplace install is not implemented yet"
          isInstalling={false}
          skills={page.skills}
        />
      )}
    </div>
  );
}
