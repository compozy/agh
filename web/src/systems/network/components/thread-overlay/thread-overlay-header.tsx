import { ArrowUpRight, X } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";

import { Button, Eyebrow } from "@agh/ui";

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
      data-slot="page-header"
      className="flex min-h-11 flex-col gap-2 border-b border-(--line) px-4 py-3"
      data-testid="network-thread-overlay-header"
    >
      <div
        data-slot="page-header-main"
        className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3"
      >
        <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
          <h1 className="truncate text-[22px] font-medium tracking-[-0.026em] text-(--fg-strong)">
            {title}
          </h1>
        </div>
        <div
          data-slot="page-header-controls"
          className="ml-auto flex shrink-0 items-center gap-1.5"
        >
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
        </div>
      </div>
      <div
        data-slot="page-header-status-row"
        className="flex flex-wrap items-center gap-x-4 gap-y-2 text-small-body text-(--muted)"
      >
        {participantLabel ? <Eyebrow weight="medium">{participantLabel}</Eyebrow> : null}
        <Button
          aria-label="Open thread in main pane"
          className="h-7 text-xs text-(--muted)"
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
    </header>
  );
}
