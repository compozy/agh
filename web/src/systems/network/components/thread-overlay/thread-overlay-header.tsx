import { useNavigate } from "@tanstack/react-router";
import { ArrowUpRight, X } from "lucide-react";

import { Button, DetailHeader, Eyebrow } from "@agh/ui";

import type { NetworkThreadDetail } from "../../types";

export interface ThreadOverlayHeaderProps {
  workspaceId: string;
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

export function ThreadOverlayHeader({
  workspaceId,
  channel,
  threadId,
  detail,
}: ThreadOverlayHeaderProps) {
  const navigate = useNavigate();
  const title = detail?.title ?? "Thread";
  const participantLabel = buildParticipantLabel(detail);

  return (
    <DetailHeader
      actions={
        <Button
          aria-label="Close thread overlay"
          data-testid="network-thread-overlay-close"
          onClick={() => {
            if (workspaceId) {
              void navigate({
                params: { workspaceId, channel },
                to: "/network/$workspaceId/$channel/threads",
              });
            }
          }}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <X aria-hidden="true" className="size-4" />
        </Button>
      }
      className="px-4 py-3"
      data-testid="network-thread-overlay-header"
      meta={
        <>
          {participantLabel ? <Eyebrow>{participantLabel}</Eyebrow> : null}
          <Button
            aria-label="Open thread in main pane"
            className="h-7 text-xs text-muted"
            data-testid="network-thread-overlay-open-main"
            onClick={() => {
              if (workspaceId) {
                void navigate({
                  params: { workspaceId, channel, threadId },
                  search: { view: "full" },
                  to: "/network/$workspaceId/$channel/threads/$threadId",
                });
              }
            }}
            size="sm"
            type="button"
            variant="ghost"
          >
            <ArrowUpRight aria-hidden="true" className="size-3" />
            Open in main
          </Button>
        </>
      }
      title={<span className="truncate">{title}</span>}
    />
  );
}
