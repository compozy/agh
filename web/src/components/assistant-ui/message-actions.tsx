import { useAuiState } from "@assistant-ui/react";

import { cn } from "@/lib/utils";
import {
  formatMessageTimestamp,
  formatMessageTimestampFull,
} from "@/systems/session/lib/format-timestamp";
import { Button, CopyIconButton } from "@agh/ui";
import { useSessionComposerPrefill } from "./hooks/use-session-composer-prefill";
import { deriveMessageActions } from "./message-actions.logic";

// Reveal-on-hover/focus toolbar (`opacity-0 → group-hover:opacity-100`, mapped
// to AGH's neutral ramp + `--duration-slow`). `focus-within` reveals it for
// keyboard users; `pointer-events` gate keeps the hidden row from intercepting
// clicks over the message body.
const REVEAL_CLASS_NAME = cn(
  "flex items-center gap-2 text-small-body text-muted tabular-nums",
  "opacity-0 pointer-events-none transition-opacity duration-slow motion-reduce:transition-none",
  "group-hover/message:opacity-100 group-hover/message:pointer-events-auto",
  "focus-within:opacity-100 focus-within:pointer-events-auto"
);

export interface MessageActionsProps {
  /** `start` aligns the row under a flat assistant message; `end` under the right-aligned user bubble. */
  align: "start" | "end";
  copyLabel: string;
  testId: string;
  goalPrefill?: boolean;
}

export function MessageActions({
  align,
  copyLabel,
  testId,
  goalPrefill = false,
}: MessageActionsProps) {
  const setComposerText = useSessionComposerPrefill();
  const message = useAuiState(
    state => state.message as { content?: unknown; status?: { type?: string } }
  );
  const { source, timestampMs, visible } = deriveMessageActions(message);

  if (!visible) {
    return null;
  }

  const timestamp =
    timestampMs !== null ? (
      <time
        data-testid={`${testId}-timestamp`}
        dateTime={new Date(timestampMs).toISOString()}
        title={formatMessageTimestampFull(timestampMs)}
        className="text-subtle tabular-nums"
      >
        {formatMessageTimestamp(timestampMs)}
      </time>
    ) : null;

  const copy = (
    <CopyIconButton
      value={source}
      copyLabel={copyLabel}
      copiedLabel="Copied"
      className="text-muted hover:text-fg"
      data-testid={`${testId}-copy`}
    />
  );
  const useAsGoal =
    goalPrefill && setComposerText ? (
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className="px-1 text-muted hover:text-fg"
        data-testid={`${testId}-goal-prefill`}
        onClick={() => setComposerText(`/goal ${source}`)}
      >
        Use as Goal
      </Button>
    ) : null;

  return (
    <div
      data-testid={testId}
      className={cn(REVEAL_CLASS_NAME, align === "end" ? "justify-end" : "justify-start")}
    >
      {align === "end" ? (
        <>
          {timestamp}
          {copy}
        </>
      ) : (
        <>
          {copy}
          {useAsGoal}
          {timestamp}
        </>
      )}
    </div>
  );
}
