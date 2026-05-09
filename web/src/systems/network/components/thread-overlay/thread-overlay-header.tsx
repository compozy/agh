import { ArrowUpRight, X } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";

import { Button, Eyebrow, PageHeader } from "@agh/ui";

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
    <PageHeader
      className="px-4 py-3"
      data-testid="network-thread-overlay-header"
      title={title}
      statusRow={
        <>
          {participantLabel ? <Eyebrow weight="medium">{participantLabel}</Eyebrow> : null}
          <Button
            aria-label="Open thread in main pane"
            className="h-7 text-xs text-(--color-text-secondary)"
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
        </>
      }
      controls={
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
      }
    />
  );
}
