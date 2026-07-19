import { Outlet, useChildMatches } from "@tanstack/react-router";
import { AlertCircle, Compass } from "lucide-react";

import {
  Button,
  cn,
  Empty,
  LaneTabs,
  PAGE_CONTENT_GUTTER,
  Skeleton,
  TabsContent,
  useTopbarSlot,
  type LaneTabsItem,
} from "@agh/ui";

import { useAgentDetailPage } from "@/hooks/routes/use-agent-detail-page";
import {
  AgentConfigurationTab,
  AgentDetailHeader,
  AgentDiagnosticsBanner,
  AgentInstructionsTab,
  AgentOverviewTab,
  AgentPageActions,
  AgentPageOverflow,
  AgentRuntimeControl,
  AgentSessionsTab,
  useAgentInstructionsTab,
  type AgentDetailSearch,
  type AgentDetailTab,
  type AgentInstructionFile,
  type AgentPayload,
} from "@/systems/agent";
import type { SessionPayload } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

const SETTINGS_ROUTE_ID = "/_app/agents/$name/settings";
const SESSION_ROUTE_ID = "/_app/agents/$name/sessions/$id";

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

export function AgentDetailPage({ name, rawSearch }: AgentDetailContentProps) {
  const childMatches = useChildMatches();
  const isSettingsChild = childMatches.some(match => match.routeId === SETTINGS_ROUTE_ID);
  const isSessionChild = childMatches.some(match => match.routeId === SESSION_ROUTE_ID);

  // Session child remains a full-page replacement. Settings mounts as an overlay
  // over the detail shell so deep links keep entity context behind the scrim.
  if (isSessionChild) {
    return <Outlet />;
  }

  return (
    <>
      <AgentDetailContent name={name} rawSearch={rawSearch} />
      {isSettingsChild ? <Outlet /> : null}
    </>
  );
}

interface AgentDetailContentProps {
  name: string;
  rawSearch: AgentDetailSearch;
}

function AgentDetailContent({ name, rawSearch }: AgentDetailContentProps) {
  const page = useAgentDetailPage(name, rawSearch);
  const search = page.search;
  const { activeWorkspaceId } = useActiveWorkspace();

  const metricsReady = !page.metricsLoading && !page.metricsUnavailable;
  const tabItems: LaneTabsItem<AgentDetailTab>[] = [
    { value: "overview", label: "Overview", testId: "agent-tab-overview" },
    { value: "instructions", label: "Instructions", testId: "agent-tab-instructions" },
    { value: "configuration", label: "Configuration", testId: "agent-tab-configuration" },
    {
      value: "sessions",
      label: "Sessions",
      // Omit count when metrics are loading/unavailable — never show an inferred zero.
      ...(metricsReady ? { count: page.sessionsTotal } : {}),
      liveLabel: metricsReady && page.activeSessionsTotal > 0 ? "Live" : undefined,
      testId: "agent-tab-sessions",
    },
  ];

  // Topbar carries route actions only; the provider·model·reasoning selector
  // lives on the PageHead's trailing edge (OpenDesign agent-detail dh__actions).
  useTopbarSlot(
    page.agent
      ? {
          actions: (
            <AgentPageActions
              onEditSettings={() => page.onEditSettings()}
              onNewSession={page.onNewSession}
              isCreatingSession={page.isCreatingForAgent}
              newSessionDisabled={page.newSessionDisabled}
            />
          ),
          overflow: <AgentPageOverflow onDelete={page.onDelete} onDuplicate={page.onDuplicate} />,
        }
      : null
  );

  if (page.agentLoading) {
    return (
      <div
        className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col gap-4 py-5")}
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
      <div className={cn(PAGE_CONTENT_GUTTER, "flex flex-1 items-center justify-center py-8")}>
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
      <div className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col")}>
        {page.agent.diagnostics && page.agent.diagnostics.length > 0 ? (
          <div className="shrink-0 pt-5" data-testid="agent-diagnostics-slot">
            <AgentDiagnosticsBanner diagnostics={page.agent.diagnostics} />
          </div>
        ) : null}
        <AgentDetailHeader
          agent={page.agent}
          activeCount={page.activeSessionsTotal}
          metricsUnavailable={page.metricsUnavailable || page.metricsLoading}
          actions={<AgentRuntimeControl agent={page.agent} workspaceId={activeWorkspaceId} />}
        />
        <LaneTabs
          ariaLabel="Agent detail panels"
          className="min-h-0 flex-1 gap-0"
          items={tabItems}
          listClassName="w-full"
          onChange={page.setTab}
          value={search.tab}
          data-testid="agent-detail-tabs"
        >
          <div
            className="flex min-h-0 flex-1 flex-col overflow-y-auto py-5"
            data-testid="agent-detail-body"
          >
            <TabsContent value="overview" className="flex flex-col gap-6">
              <AgentOverviewTab
                agent={page.agent}
                sessions={page.sessions}
                sessionsTotal={page.sessionsTotal}
                activeSessionsTotal={page.activeSessionsTotal}
                failedSessionsTotal={page.failedSessionsTotal}
                runtimeSeconds={page.runtimeSeconds}
                metricsUnavailable={page.metricsUnavailable}
                metricsLoading={page.metricsLoading}
                lastSessionActivityAt={page.lastSessionActivityAt}
                sessionsLoading={page.sessionsLoading}
                sessionsError={page.sessionsError}
                onEditRuntime={() => page.onEditSettings("runtime")}
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
                failed={page.failedSessionsTotal}
                runtimeSeconds={page.runtimeSeconds}
                metricsUnavailable={page.metricsUnavailable}
                metricsLoading={page.metricsLoading}
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
    </div>
  );
}
