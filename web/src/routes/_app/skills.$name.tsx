import { AlertCircle } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  type SkillDetailRouteSearch,
  useSkillDetailPage,
} from "@/hooks/routes/use-skill-detail-page";
import { normalizeListingSearchValue } from "@/lib/listing-search";
import { SkillDetailPanel } from "@/systems/skill";
import { preloadSkillDetailRoute } from "./-skill-preload";

function validateSkillDetailSearch(search: Record<string, unknown>): SkillDetailRouteSearch {
  return {
    content: normalizeListingSearchValue(search.content),
  };
}

export const Route = createFileRoute("/_app/skills/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: params.name } },
  }),
  validateSearch: validateSkillDetailSearch,
  loaderDeps: ({ search }) => ({ content: search.content }),
  loader: ({ context, deps, params }) =>
    preloadSkillDetailRoute(context.queryClient, params.name, deps),
  component: SkillDetailRoute,
});

function SkillDetailRoute() {
  const { name } = Route.useParams();
  const page = useSkillDetailPage(name, Route.useSearch());

  if (page.workspaceId === "") {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center py-10"
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
      actionStatus={page.isActionPending ? "pending" : "idle"}
      contentStatus={page.isContentLoading ? "loading" : page.contentError ? "error" : "ready"}
      detailStatus={
        page.isLoadingDetail && !page.selectedSkill
          ? "loading"
          : page.detailError
            ? "error"
            : "ready"
      }
      shadowsStatus={page.isLoadingShadows ? "loading" : page.shadowsError ? "error" : "ready"}
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
