import { AlertCircle, Compass, Loader2, User2 } from "lucide-react";
import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";

import { Empty, useTopbarSlot } from "@agh/ui";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  AgentInfoPanel,
  AgentPageActions,
  AgentPageStatusPill,
  AgentSessionsList,
  AgentStatsGrid,
} from "@/systems/agent";
import { useAgentDetailPage } from "@/hooks/routes/use-agent-detail-page";

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

function AgentDetailContent({ name }: AgentDetailContentProps) {
  const page = useAgentDetailPage(name);
  const sessions = page.sessions;

  useTopbarSlot({
    count: sessions.length,
    tabs: page.agent ? <AgentPageStatusPill sessions={sessions} /> : undefined,
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
        <Loader2 className="size-5 animate-spin text-(--subtle)" />
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
            <button
              type="button"
              onClick={page.onGoHome}
              className="inline-flex items-center gap-2 rounded-md border border-(--line) px-3 py-1.5 text-xs text-(--muted) transition-colors hover:border-accent hover:text-(--fg)"
              data-testid="agent-detail-go-home"
            >
              <Compass className="size-3.5" />
              Go home
            </button>
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
          {hasResolvedSessions ? <AgentStatsGrid sessions={sessions} /> : null}
          <AgentSessionsList
            agentName={name}
            sessions={sessions}
            isLoading={page.sessionsLoading}
            isError={page.sessionsError}
          />
        </div>
      </div>
      <AgentInfoPanel agent={page.agent} />
    </div>
  );
}
