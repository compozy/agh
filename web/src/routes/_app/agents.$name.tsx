import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";
import { AlertCircle, Compass, User2 } from "lucide-react";
import { useMemo, useState } from "react";

import { Button, Empty, PillGroup, Spinner, useTopbarSlot } from "@agh/ui";

import { useAgentDetailPage } from "@/hooks/routes/use-agent-detail-page";
import {
  AgentInfoInspector,
  AgentPageActions,
  AgentPageStatusPill,
  AgentSessionsList,
  AgentStatsGrid,
  splitAgentSessions,
} from "@/systems/agent";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/agents/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.name, icon: User2 },
  }),
  component: AgentDetailPage,
});

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

type AgentSessionView = "normal" | "memory_extraction";

function AgentDetailContent({ name }: AgentDetailContentProps) {
  const page = useAgentDetailPage(name);
  const [sessionView, setSessionView] = useState<AgentSessionView>("normal");
  const { normalSessions, memoryExtractionSessions } = useMemo(
    () => splitAgentSessions(page.sessions),
    [page.sessions]
  );
  const hasMemoryExtractionSessions = memoryExtractionSessions.length > 0;
  const activeSessionView: AgentSessionView = hasMemoryExtractionSessions ? sessionView : "normal";
  const visibleSessions =
    activeSessionView === "memory_extraction" ? memoryExtractionSessions : normalSessions;

  useTopbarSlot({
    count: normalSessions.length,
    tabs: page.agent ? <AgentPageStatusPill sessions={normalSessions} /> : undefined,
    actions: page.agent ? (
      <AgentPageActions
        agent={page.agent}
        isRefreshing={page.isRefreshing}
        onRefresh={page.onRefresh}
        onConfigure={page.onConfigure}
        onNewSession={page.onNewSession}
        isCreatingSession={page.isCreatingForAgent}
        newSessionDisabled={page.newSessionDisabled}
      />
    ) : undefined,
  });

  if (page.agentLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="agent-detail-loading">
        <Spinner className="size-5 text-subtle" />
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
              onClick={page.onGoHome}
              data-testid="agent-detail-go-home"
            >
              <Compass className="size-3" />
              Go home
            </Button>
          }
          data-testid="agent-detail-not-found"
        />
      </div>
    );
  }

  const hasResolvedSessions = !page.sessionsLoading && !page.sessionsError;

  return (
    <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden" data-testid="agent-detail-page">
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <div
          className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-5"
          data-testid="agent-detail-body"
        >
          {hasResolvedSessions ? (
            <div className="flex flex-col gap-3" data-testid="agent-session-summary">
              <AgentStatsGrid sessions={normalSessions} />
              {hasMemoryExtractionSessions ? (
                <PillGroup<AgentSessionView>
                  aria-label="Session view"
                  value={activeSessionView}
                  onChange={setSessionView}
                  size="sm"
                  data-testid="agent-session-view-toggle"
                  items={[
                    {
                      value: "normal",
                      label: "Sessions",
                      badge: normalSessions.length,
                      testId: "agent-session-view-normal",
                    },
                    {
                      value: "memory_extraction",
                      label: "Memory extraction",
                      badge: memoryExtractionSessions.length,
                      testId: "agent-session-view-memory-extraction",
                    },
                  ]}
                />
              ) : null}
            </div>
          ) : null}
          <AgentSessionsList
            agentName={name}
            sessions={visibleSessions}
            isLoading={page.sessionsLoading}
            isError={page.sessionsError}
            emptyTitle={
              activeSessionView === "memory_extraction"
                ? "No memory extraction sessions"
                : undefined
            }
            emptyDescription={
              activeSessionView === "memory_extraction"
                ? `Memory extraction sessions for ${name} appear after recall processing.`
                : undefined
            }
          />
        </div>
      </div>
      <AgentInfoInspector agent={page.agent} />
    </div>
  );
}
