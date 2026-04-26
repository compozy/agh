import { Plus, RefreshCw, Settings2 } from "lucide-react";

import { Button, MonoBadge, PageHeader, Toolbar, cn } from "@agh/ui";

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
  const meta = (
    <div className="flex items-center gap-3 font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
      <span data-testid="agent-page-header-provider">{agent.provider}</span>
      <span aria-hidden="true">·</span>
      <span data-testid="agent-page-header-session-count">
        {sessions.length} {sessions.length === 1 ? "session" : "sessions"}
      </span>
    </div>
  );

  return (
    <div className="flex flex-col" data-testid="agent-page-header">
      <PageHeader
        title={
          <span className="flex items-center gap-2">
            <span
              aria-hidden="true"
              className="inline-flex size-6 shrink-0 items-center justify-center rounded-md bg-[color:var(--color-surface-elevated)] text-[color:var(--color-accent)]"
            >
              <AgentIcon provider={agent.provider} tone="accent" className="size-3.5" />
            </span>
            <span className="truncate">{agent.name}</span>
            <MonoBadge tone={status.tone} data-testid="agent-page-header-status">
              {status.label}
            </MonoBadge>
          </span>
        }
        meta={meta}
      />
      <Toolbar data-testid="agent-page-toolbar">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={isRefreshing}
          data-testid="agent-page-refresh"
        >
          <RefreshCw
            aria-hidden="true"
            className={cn("size-3.5", isRefreshing && "animate-spin")}
          />
          Refresh
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onConfigure}
          data-testid="agent-page-configure"
        >
          <Settings2 aria-hidden="true" className="size-3.5" />
          Configure
        </Button>
        <Button
          type="button"
          variant="default"
          size="sm"
          onClick={onNewSession}
          disabled={newSessionDisabled}
          aria-busy={isCreatingSession}
          data-testid="agent-page-new-session"
          className="ml-auto"
        >
          <Plus aria-hidden="true" className="size-3.5" />
          New session
        </Button>
      </Toolbar>
    </div>
  );
}
