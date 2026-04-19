import { ChevronRight, Play, Square } from "lucide-react";

import { Button, MonoBadge, StatusDot, type StatusDotTone } from "@agh/ui";

import { cn } from "@/lib/utils";
import type { SessionPayload, SessionState } from "../types";

export interface ChatHeaderProps {
  session: SessionPayload;
  onStop: () => void;
  onResume: () => void;
  workspaceName?: string;
}

interface StateSignal {
  tone: StatusDotTone;
  pulse?: boolean;
}

const STATE_SIGNAL: Record<SessionState, StateSignal> = {
  active: { tone: "success" },
  starting: { tone: "warning", pulse: true },
  stopping: { tone: "warning" },
  stopped: { tone: "neutral" },
};

export function ChatHeader({ session, onStop, onResume, workspaceName }: ChatHeaderProps) {
  const isActive = session.state === "active" || session.state === "starting";
  const isStopped = session.state === "stopped";
  const signal = STATE_SIGNAL[session.state] ?? { tone: "neutral" };

  return (
    <div
      className={cn(
        "flex h-12 items-center justify-between gap-3 border-b px-4",
        "border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]/90 backdrop-blur"
      )}
      data-testid="chat-header"
    >
      <div
        className="flex min-w-0 items-center gap-2 overflow-hidden"
        data-testid="chat-breadcrumb"
      >
        <StatusDot
          size="md"
          tone={signal.tone}
          pulse={signal.pulse}
          data-testid="agent-status-dot"
          aria-label={`Session state: ${session.state}`}
        />
        <span className="sr-only">{`Session state: ${session.state}`}</span>
        <span className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
          {session.agent_name}
        </span>

        <ChevronRight
          aria-hidden="true"
          className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]"
        />

        <span
          className="truncate text-[13px] text-[color:var(--color-text-secondary)]"
          data-testid="session-name"
        >
          {session.name?.trim() || session.id}
        </span>

        {workspaceName ? (
          <>
            <ChevronRight
              aria-hidden="true"
              className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]"
            />
            <MonoBadge
              tone="default"
              className="shrink-0 normal-case"
              data-testid="session-workspace-badge"
            >
              {workspaceName}
            </MonoBadge>
          </>
        ) : null}
      </div>

      <div className="flex shrink-0 items-center gap-1">
        {isActive ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onStop}
            data-testid="stop-button"
            aria-label="Stop session"
          >
            <Square className="size-3.5" />
          </Button>
        ) : null}
        {isStopped ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onResume}
            data-testid="resume-button"
            aria-label="Resume session"
          >
            <Play className="size-3.5" />
          </Button>
        ) : null}
      </div>
    </div>
  );
}
