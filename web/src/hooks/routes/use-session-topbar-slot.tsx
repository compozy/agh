import { Eraser, Play, Square, Trash2 } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Spinner,
  TopbarOverflowIcon,
  useTopbarSlot,
} from "@agh/ui";

import {
  getSessionDisplayTitle,
  isUserControllableSession,
  type SessionPayload,
} from "@/systems/session";

interface UseSessionTopbarSlotInput {
  session: SessionPayload;
  isDeleting: boolean;
  isStopping: boolean;
  isResuming: boolean;
  isClearing: boolean;
  canClear: boolean;
  onDelete: () => void;
  onStop: () => void;
  onResume: () => void;
  onClear: () => void;
}

/**
 * Composes the session detail-route topbar slot — persisted session identity as
 * the leaf breadcrumb, and the lifecycle controls (stop/attach + overflow for
 * clear/delete) as the actions/overflow slots. Status identity (badge · agent ·
 * provider) is body chrome: `SessionStatusLine` in the session head band.
 */
export function useSessionTopbarSlot({
  session,
  isDeleting,
  isStopping,
  isResuming,
  isClearing,
  canClear,
  onDelete,
  onStop,
  onResume,
  onClear,
}: UseSessionTopbarSlotInput): void {
  const isActive = session.state === "active" || session.state === "starting";
  const isAttachable = session.attachable === true;
  const canResume = isAttachable && isUserControllableSession(session);
  const controlsBusy = isStopping || isResuming || isDeleting;

  useTopbarSlot({
    crumb: getSessionDisplayTitle(session),
    actions: (
      <div className="flex shrink-0 items-center gap-1" data-testid="session-topbar-actions">
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
            {isStopping ? <Spinner className="size-3" /> : <Square className="size-3" />}
          </Button>
        ) : null}
        {canResume ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onResume}
            disabled={controlsBusy && !isResuming}
            data-testid="resume-button"
            aria-label="Attach session"
          >
            {isResuming ? <Spinner className="size-3" /> : <Play className="size-3" />}
          </Button>
        ) : null}
      </div>
    ),
    overflow: (
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="More actions"
          data-testid="session-topbar-overflow"
          render={<Button type="button" variant="ghost" size="icon-sm" />}
        >
          <TopbarOverflowIcon aria-hidden="true" className="size-3" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" data-testid="session-topbar-overflow-menu">
          <DropdownMenuItem
            data-testid="composer-clear-button"
            disabled={!canClear || isClearing}
            onClick={onClear}
          >
            {isClearing ? <Spinner className="size-3" /> : <Eraser className="size-3" />}
            Clear conversation
          </DropdownMenuItem>
          <DropdownMenuItem
            data-testid="delete-button"
            disabled={controlsBusy}
            onClick={onDelete}
            variant="destructive"
          >
            {isDeleting ? <Spinner className="size-3" /> : <Trash2 className="size-3" />}
            Delete session
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  });
}
