import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";
import { AlertCircle, Compass, User2 } from "lucide-react";

import {
  Button,
  Empty,
  LaneTabs,
  Skeleton,
  TabsContent,
  useTopbarSlot,
  type LaneTabsItem,
} from "@agh/ui";

import { useAgentDetailPage } from "@/hooks/routes/use-agent-detail-page";
import {
  AgentConfigurationTab,
  AgentDiagnosticsBanner,
  AgentInstructionsTab,
  AgentOverviewTab,
  AgentPageActions,
  AgentPageMeta,
  AgentPageStatusPill,
  AgentSessionsTab,
  useAgentInstructionsTab,
  validateAgentDetailSearch,
  type AgentDetailTab,
  type AgentInstructionFile,
  type AgentPayload,
} from "@/systems/agent";
import type { SessionPayload } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadAgentDetailRoute } from "./-app-preload";

export const Route = createFileRoute("/_app/agents/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: {
      title: params.name,
      icon: User2,
    },
  }),
  validateSearch: validateAgentDetailSearch,
  loader: ({ context, location, params }) =>
    location.pathname.split("/").filter(Boolean).length === 2
      ? preloadAgentDetailRoute(context.queryClient, params.name)
      : Promise.resolve(),
  component: AgentDetailPage,
});

interface AgentInstructionsSectionProps {
  agent: AgentPayload;
  file: AgentInstructionFile;
  onFileChange: (file: AgentInstructionFile) => void;
  onEditAgentPrompt: () => void;
  workspaceId: string | null;
  sessions: SessionPayload[];
  onNewSession: () => void;
}

function AgentInstructionsSection({
  agent,
  file,
  onFileChange,
  onEditAgentPrompt,
  workspaceId,
  sessions,
  onNewSession,
}: AgentInstructionsSectionProps) {
  const viewModel = useAgentInstructionsTab({ agent, file, workspaceId, sessions });
  return (
    <AgentInstructionsTab
      agent={agent}
      file={file}
      onFileChange={onFileChange}
      onEditAgentPrompt={onEditAgentPrompt}
      viewModel={viewModel}
      onNewSession={onNewSession}
    />
  );
}

function AgentDetailPage() {
  const { name } = Route.useParams();
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;

  if (hasChildMatch) {
    return <Outlet />;
  }

  return <AgentDetailContent name={name} />;
}

interface AgentDetailContentProps {
  name: string;
}

function AgentDetailContent({ name }: AgentDetailContentProps) {
  const rawSearch = Route.useSearch();
  const page = useAgentDetailPage(name, rawSearch);
  const search = page.search;
  const { activeWorkspaceId } = useActiveWorkspace();

  const tabItems: LaneTabsItem<AgentDetailTab>[] = [
    { value: "overview", label: "Overview", testId: "agent-tab-overview" },
    { value: "instructions", label: "Instructions", testId: "agent-tab-instructions" },
    { value: "configuration", label: "Configuration", testId: "agent-tab-configuration" },
    {
      value: "sessions",
      label: "Sessions",
      count: page.sessionsTotal,
      liveLabel: page.activeSessionsTotal > 0 ? "Live" : undefined,
      testId: "agent-tab-sessions",
    },
  ];

  useTopbarSlot({
    back: page.onBackToAgents,
    backLabel: "Agents",
    tabs: page.agent ? (
      <div className="flex min-w-0 items-center gap-2">
        {!page.sessionsLoading && !page.sessionsError ? (
          <AgentPageStatusPill activeCount={page.activeSessionsTotal} />
        ) : null}
        <AgentPageMeta agent={page.agent} />
      </div>
    ) : undefined,
    actions: page.agent ? (
      <AgentPageActions
        onEditSettings={() => page.onEditSettings()}
        onNewSession={page.onNewSession}
        onDuplicate={page.onDuplicate}
        onDelete={page.onDelete}
        isCreatingSession={page.isCreatingForAgent}
        newSessionDisabled={page.newSessionDisabled}
      />
    ) : undefined,
  });

  if (page.agentLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 flex-col gap-4 px-6 py-5"
        data-testid="agent-detail-loading"
      >
        <Skeleton className="h-10 w-64" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-20 rounded-md" />
          ))}
        </div>
        <Skeleton className="h-48 rounded-md" />
      </div>
    );
  }

  if (page.agentError || !page.agent) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-8">
        <Empty
          icon={AlertCircle}
          title="Agent not found"
          description={
            page.agentError?.message ?? `No agent named "${name}" was found in this workspace.`
          }
          action={
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={page.onBackToAgents}
              data-testid="agent-detail-back-agents"
            >
              <Compass className="size-3" />
              Back to agents
            </Button>
          }
          data-testid="agent-detail-not-found"
        />
      </div>
    );
  }

  return (
    <div
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-testid="agent-detail-page"
    >
      {page.deleteDialog}
      <LaneTabs
        ariaLabel="Agent detail panels"
        className="min-h-0 flex-1 gap-0"
        items={tabItems}
        listClassName="w-full px-6"
        onChange={page.setTab}
        value={search.tab}
        data-testid="agent-detail-tabs"
      >
        <div
          className="flex min-h-0 flex-1 flex-col overflow-y-auto px-6 py-5"
          data-testid="agent-detail-body"
        >
          {page.agent.diagnostics && page.agent.diagnostics.length > 0 ? (
            <AgentDiagnosticsBanner diagnostics={page.agent.diagnostics} />
          ) : null}

          <TabsContent value="overview" className="flex flex-col gap-6">
            <AgentOverviewTab
              agent={page.agent}
              sessions={page.sessions}
              sessionsTotal={page.sessionsTotal}
              activeSessionsTotal={page.activeSessionsTotal}
              resumableSessionsTotal={page.resumableSessionsTotal}
              lastSessionActivityAt={page.lastSessionActivityAt}
              sessionsLoading={page.sessionsLoading}
              sessionsError={page.sessionsError}
              onViewAllSessions={() => page.setTab("sessions")}
            />
          </TabsContent>

          <TabsContent value="instructions" className="flex flex-col gap-6">
            <AgentInstructionsSection
              agent={page.agent}
              file={search.file}
              onFileChange={page.setFile}
              onEditAgentPrompt={() => page.onEditSettings("instructions")}
              workspaceId={activeWorkspaceId}
              sessions={page.sessions}
              onNewSession={page.onNewSession}
            />
          </TabsContent>

          <TabsContent value="configuration" className="flex flex-col gap-6">
            <AgentConfigurationTab
              agent={page.agent}
              onEditSection={section => page.onEditSettings(section)}
            />
          </TabsContent>

          <TabsContent value="sessions" className="flex flex-col gap-6">
            <AgentSessionsTab
              agentName={name}
              sessions={page.sessions}
              total={page.sessionsTotal}
              active={page.activeSessionsTotal}
              resumable={page.resumableSessionsTotal}
              lastActivityAt={page.lastSessionActivityAt}
              status={page.sessionsLoading ? "loading" : page.sessionsError ? "error" : "ready"}
              paginationStatus={
                page.isLoadingMoreSessions
                  ? "loading"
                  : page.hasMoreSessions
                    ? "available"
                    : undefined
              }
              onLoadMore={page.onLoadMoreSessions}
              filter={search.filter}
              onFilterChange={page.setFilter}
              onNewSession={page.onNewSession}
              onClearFilter={() => page.setFilter("all")}
            />
          </TabsContent>
        </div>
      </LaneTabs>
    </div>
  );
}
