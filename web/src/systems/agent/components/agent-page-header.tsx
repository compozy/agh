import { Plus, RefreshCw, Settings2 } from "lucide-react";

import { Button, Pill, PageHeader, cn } from "@agh/ui";

import { AgentIcon } from "./agent-icon";
import type { AgentPayload } from "../types";
import type { SessionPayload } from "@/systems/session";

export interface AgentPageHeaderProps {
  agent: AgentPayload;
  sessions: SessionPayload[];
  isRefreshing: boolean;
  onRefresh: () => void;
  onConfigure: () => void;
  onNewSession: () => void;
  isCreatingSession: boolean;
  newSessionDisabled: boolean;
}

export function AgentPageHeader({
  agent,
  sessions,
  isRefreshing,
  onRefresh,
  onConfigure,
  onNewSession,
  isCreatingSession,
  newSessionDisabled,
}: AgentPageHeaderProps) {
  const activeCount = sessions.filter(session => session.state === "active").length;
  const status =
    activeCount > 0
      ? { label: "ACTIVE", tone: "success" as const }
      : { label: "IDLE", tone: "neutral" as const };

  return (
    <PageHeader
      data-testid="agent-page-header"
      icon={({ className }) => (
        <AgentIcon provider={agent.provider} tone="accent" className={className} />
      )}
      title={
        <span className="flex items-center gap-2">
          <span className="truncate">{agent.name}</span>
          <Pill mono tone={status.tone} data-testid="agent-page-header-status">
            {status.label}
          </Pill>
        </span>
      }
      count={sessions.length}
      meta={
        <div className="flex items-center gap-2" data-testid="agent-page-toolbar">
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            onClick={onRefresh}
            disabled={isRefreshing}
            aria-label="Refresh"
            title="Refresh"
            data-testid="agent-page-refresh"
          >
            <RefreshCw
              aria-hidden="true"
              className={cn("size-3.5", isRefreshing && "animate-spin")}
            />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            onClick={onConfigure}
            aria-label="Configure"
            title="Configure"
            data-testid="agent-page-configure"
          >
            <Settings2 aria-hidden="true" className="size-3.5" />
          </Button>
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={onNewSession}
            disabled={newSessionDisabled}
            aria-busy={isCreatingSession}
            data-testid="agent-page-new-session"
          >
            <Plus aria-hidden="true" className="size-3.5" />
            New session
          </Button>
        </div>
      }
    />
  );
}
