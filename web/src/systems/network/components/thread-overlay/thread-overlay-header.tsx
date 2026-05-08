import { ArrowUpRight, X } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";

import { Button } from "@agh/ui";

import type { NetworkThreadDetail } from "../../types";

export interface ThreadOverlayHeaderProps {
  channel: string;
  threadId: string;
  detail: NetworkThreadDetail | null;
}

function buildParticipantLabel(detail: NetworkThreadDetail | null): string {
  if (!detail) {
    return "";
  }
  const participants = detail.participant_count ?? 0;
  if (participants <= 0) {
    return "no peers";
  }
  return participants === 1 ? "1 peer" : `${participants} peers`;
}

export function ThreadOverlayHeader({ channel, threadId, detail }: ThreadOverlayHeaderProps) {
  const navigate = useNavigate();
  const title = detail?.title ?? "Thread";
  const participantLabel = buildParticipantLabel(detail);

  return (
    <header
      className="flex items-start gap-2 border-b border-(--color-divider) px-4 py-3"
      data-testid="network-thread-overlay-header"
    >
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <h2 className="truncate text-item-title font-semibold text-(--color-text-primary)">
          {title}
        </h2>
        {participantLabel ? (
          <p className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
            {participantLabel}
          </p>
        ) : null}
        <Button
          aria-label="Open thread in main pane"
          className="self-start text-xs text-(--color-text-secondary)"
          data-testid="network-thread-overlay-open-main"
          onClick={() => {
            void navigate({
              params: { channel, threadId },
              search: { view: "full" },
              to: "/network/$channel/threads/$threadId",
            });
          }}
          size="sm"
          type="button"
          variant="ghost"
        >
          <ArrowUpRight aria-hidden="true" className="size-3.5" />
          Open in main
        </Button>
      </div>
      <Button
        aria-label="Close thread overlay"
        data-testid="network-thread-overlay-close"
        onClick={() => {
          void navigate({
            params: { channel },
            to: "/network/$channel/threads",
          });
        }}
        size="icon-sm"
        type="button"
        variant="ghost"
      >
        <X aria-hidden="true" className="size-4" />
      </Button>
    </header>
  );
}
