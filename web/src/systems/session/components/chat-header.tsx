import { useCallback, useState } from "react";
import { ChevronRight, Loader2, Play, Square, Trash2 } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  MonoBadge,
  StatusDot,
  type StatusDotTone,
} from "@agh/ui";

import { cn } from "@/lib/utils";
import { SessionActivityInline } from "./runtime-activity-notice";
import type { SessionPayload, SessionState } from "../types";

export interface ChatHeaderProps {
  session: SessionPayload;
  onDelete: () => void;
  onStop: () => void;
  onResume: () => void;
  workspaceName?: string;
  isDeleting?: boolean;
  isStopping?: boolean;
  isResuming?: boolean;
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

export function ChatHeader({
  session,
  onDelete,
  onStop,
  onResume,
  workspaceName,
  isDeleting = false,
  isStopping = false,
  isResuming = false,
}: ChatHeaderProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const isActive = session.state === "active" || session.state === "starting";
  const isStopped = session.state === "stopped";
  const signal = STATE_SIGNAL[session.state] ?? { tone: "neutral" };
  const controlsBusy = isStopping || isResuming || isDeleting;
  const providerLabel = session.provider?.trim();

  const handleConfirmDelete = useCallback(() => {
    setDeleteDialogOpen(false);
    onDelete();
  }, [onDelete]);

  return (
    <>
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

          {providerLabel ? (
            <>
              <ChevronRight
                aria-hidden="true"
                className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]"
              />
              <MonoBadge
                tone="accent"
                className="shrink-0 normal-case"
                data-testid="session-provider-badge"
              >
                {providerLabel}
              </MonoBadge>
            </>
          ) : null}

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

          <SessionActivityInline activity={session.activity} />
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={() => setDeleteDialogOpen(true)}
            disabled={controlsBusy}
            data-testid="delete-button"
            aria-label="Delete session"
          >
            {isDeleting ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Trash2 className="size-3.5" />
            )}
          </Button>

          {isActive ? (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              onClick={onStop}
              disabled={controlsBusy && !isStopping}
              data-testid="stop-button"
              aria-label="Stop session"
            >
              {isStopping ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Square className="size-3.5" />
              )}
            </Button>
          ) : null}
          {isStopped ? (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              onClick={onResume}
              disabled={controlsBusy && !isResuming}
              data-testid="resume-button"
              aria-label="Resume session"
            >
              {isResuming ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Play className="size-3.5" />
              )}
            </Button>
          ) : null}
        </div>
      </div>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent
          showCloseButton={!isDeleting}
          className="max-w-md"
          data-testid="delete-dialog"
        >
          <DialogHeader>
            <DialogTitle>Delete session</DialogTitle>
            <DialogDescription>
              This permanently removes <strong>{session.name?.trim() || session.id}</strong>,
              including its transcript and history, and removes it from the session list.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={isDeleting}
              data-testid="delete-dialog-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={isDeleting}
              data-testid="delete-dialog-confirm"
            >
              {isDeleting ? (
                <>
                  <Loader2 className="size-3.5 animate-spin" />
                  Deleting
                </>
              ) : (
                <>
                  <Trash2 className="size-3.5" />
                  Delete session
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
