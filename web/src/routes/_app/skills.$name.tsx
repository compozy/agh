import { AlertCircle, Wrench } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  type SkillDetailRouteSearch,
  useSkillDetailPage,
} from "@/hooks/routes/use-skill-detail-page";
import { SkillDetailPanel } from "@/systems/skill";

function normalizeSearchValue(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function validateSkillDetailSearch(search: Record<string, unknown>): SkillDetailRouteSearch {
  return {
    content: normalizeSearchValue(search.content),
  };
}

export const Route = createFileRoute("/_app/skills/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.name, icon: Wrench },
  }),
  validateSearch: validateSkillDetailSearch,
  component: SkillDetailRoute,
});

function SkillDetailRoute() {
  const { name } = Route.useParams();
  const page = useSkillDetailPage(name, Route.useSearch());

  if (page.workspaceId === "") {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="skill-detail-no-workspace"
      >
        <Empty
          className="max-w-md"
          description="Select a workspace to inspect this skill."
          icon={AlertCircle}
          title="No workspace selected"
        />
      </div>
    );
  }

  if (page.isLoadingDetail && !page.selectedSkill) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="skill-detail-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  return (
    <SkillDetailPanel
      content={page.selectedSkillContent}
      contentError={page.contentError}
      error={page.detailError}
      isActionPending={page.isActionPending}
      isContentLoading={page.isContentLoading}
      isLoading={page.isLoadingDetail && !page.selectedSkill}
      isShadowsLoading={page.isLoadingShadows}
      onBack={page.handleBack}
      onDisable={() => page.handleDisable()}
      onEnable={() => page.handleEnable()}
      onRetryContent={page.handleRetryContent}
      onViewContent={() => page.handleViewContent()}
      shadows={page.selectedSkillShadows}
      shadowsError={page.shadowsError}
      skill={page.selectedSkill}
    />
  );
}
