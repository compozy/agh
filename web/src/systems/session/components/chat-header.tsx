import { ChevronRight, Square, Play } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@agh/ui";
import type { SessionPayload } from "../types";

export interface ChatHeaderProps {
  session: SessionPayload;
  onStop: () => void;
  onResume: () => void;
  workspaceName?: string;
}

const STATE_DOT_COLOR: Record<string, string> = {
  active: "bg-[color:var(--color-success)]",
  starting: "bg-[color:var(--color-warning)] animate-pulse",
  stopping: "bg-[color:var(--color-warning)]",
  stopped: "bg-[color:var(--color-text-tertiary)]",
};

export function ChatHeader({ session, onStop, onResume, workspaceName }: ChatHeaderProps) {
  const isActive = session.state === "active" || session.state === "starting";
  const isStopped = session.state === "stopped";

  return (
    <div
      className={cn(
        "flex h-11 items-center justify-between border-b px-4",
        "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
      )}
      data-testid="chat-header"
    >
      <div className="flex items-center gap-1.5 overflow-hidden" data-testid="chat-breadcrumb">
        {/* Agent avatar dot */}
        <span
          className={cn("size-2.5 shrink-0 rounded-full", STATE_DOT_COLOR[session.state])}
          aria-label={`Session state: ${session.state}`}
          data-testid="agent-status-dot"
        />
        <span className="sr-only">{`Session state: ${session.state}`}</span>
        <span className="truncate text-sm font-medium text-[color:var(--color-text-primary)]">
          {session.agent_name}
        </span>

        <ChevronRight className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]" />

        {/* Session name */}
        <span
          className="truncate text-sm text-[color:var(--color-text-secondary)]"
          data-testid="session-name"
        >
          {session.name?.trim() || session.id}
        </span>

        {workspaceName && (
          <>
            <ChevronRight className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]" />
            <span
              className="truncate text-xs text-[color:var(--color-text-tertiary)]"
              data-testid="session-workspace-badge"
            >
              {workspaceName}
            </span>
          </>
        )}
      </div>

      <div className="flex items-center gap-1">
        {isActive && (
          <Button variant="ghost" size="icon-sm" onClick={onStop} data-testid="stop-button">
            <Square className="size-3.5" />
          </Button>
        )}
        {isStopped && (
          <Button variant="ghost" size="icon-sm" onClick={onResume} data-testid="resume-button">
            <Play className="size-3.5" />
          </Button>
        )}
      </div>
    </div>
  );
}
