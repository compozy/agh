import { Plus, RefreshCw, Settings2 } from "lucide-react";

import { Button, Pill, cn } from "@agh/ui";

import type { SessionPayload } from "@/systems/session";
import type { AgentPayload } from "../types";

export interface AgentPageStatusPillProps {
  sessions: SessionPayload[];
}

/**
 * Pill that surfaces whether any of an agent's sessions are active. Routes
 * compose it into the topbar slot or detail body.
 */
export function AgentPageStatusPill({ sessions }: AgentPageStatusPillProps) {
  const activeCount = sessions.filter(session => session.state === "active").length;
  const status =
    activeCount > 0
      ? { label: "ACTIVE", tone: "success" as const }
      : { label: "IDLE", tone: "neutral" as const };
  return (
    <Pill mono tone={status.tone} data-testid="agent-page-header-status">
      {status.label}
    </Pill>
  );
}

export interface AgentPageActionsProps {
  agent: AgentPayload;
  isRefreshing: boolean;
  onRefresh: () => void;
  onConfigure: () => void;
  onNewSession: () => void;
  isCreatingSession: boolean;
  newSessionDisabled: boolean;
}

/**
 * Right-side action cluster for the agent detail route. Routes push it into
 * the topbar `actions` slot. Pre-P4 the cluster lived in `<PageHeader meta>`.
 */
export function AgentPageActions({
  isRefreshing,
  onRefresh,
  onConfigure,
  onNewSession,
  isCreatingSession,
  newSessionDisabled,
}: AgentPageActionsProps) {
  return (
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
        <RefreshCw aria-hidden="true" className={cn("size-3", isRefreshing && "animate-spin")} />
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
        <Settings2 aria-hidden="true" className="size-3" />
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
        <Plus aria-hidden="true" className="size-3" />
        New session
      </Button>
    </div>
  );
}
