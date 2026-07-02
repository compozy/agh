import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowUpRight, ListTodo, X } from "lucide-react";

import { Button, DetailHeader, Eyebrow, MonoId, Pill } from "@agh/ui";

import { usePromoteNetworkThreadTask } from "../../hooks/use-network-actions";
import type { NetworkThreadDetail, NetworkThreadTaskLink } from "../../types";
import { ThreadSubscriptionControl } from "./thread-subscription-control";

export interface ThreadOverlayHeaderProps {
  workspaceId: string;
  channel: string;
  threadId: string;
  detail: NetworkThreadDetail | null;
  rootMessageId?: string | null;
  selfPeerId?: string | null;
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
  rootMessageId,
  selfPeerId,
}: ThreadOverlayHeaderProps) {
  const navigate = useNavigate();
  const promote = usePromoteNetworkThreadTask();
  const title = detail?.title ?? "Thread";
  const participantLabel = buildParticipantLabel(detail);
  const originMessageId = rootMessageId ?? detail?.root_message_id ?? "";
  const canPromote = originMessageId.trim() !== "";

  const handlePromote = async () => {
    if (!canPromote) {
      return;
    }
    try {
      const result = await promote.mutateAsync({
        workspaceId,
        channel,
        threadId,
        data: {
          origin_message_id: originMessageId,
          title,
        },
      });
      void navigate({ params: { id: result.task.id }, to: "/tasks/$id" });
    } catch {
      // The mutation hook surfaces the toast; keep the operator on the thread.
    }
  };

  return (
    <DetailHeader
      actions={
        <>
          <ThreadSubscriptionControl
            channel={channel}
            peerId={selfPeerId}
            threadId={threadId}
            workspaceId={workspaceId}
          />
          <Button
            aria-label="Promote thread to task"
            data-testid="network-thread-promote-task"
            disabled={!canPromote || promote.isPending}
            onClick={() => void handlePromote()}
            size="sm"
            title="Create a task from this thread without starting a run."
            type="button"
            variant="neutral"
          >
            <ListTodo aria-hidden="true" className="size-3" />
            Promote
          </Button>
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
        </>
      }
      className="px-4 py-3"
      data-testid="network-thread-overlay-header"
      meta={
        <>
          {participantLabel ? <Eyebrow className="text-subtle">{participantLabel}</Eyebrow> : null}
          <Button
            aria-label="Open thread in main pane"
            className="text-muted"
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

export function ThreadTaskLinks({ links }: { links: ReadonlyArray<NetworkThreadTaskLink> }) {
  if (links.length === 0) {
    return null;
  }

  return (
    <div
      aria-label="Linked tasks"
      className="flex flex-wrap items-center gap-2 border-b border-line bg-canvas-soft px-4 py-2"
      data-testid="network-thread-task-links"
    >
      <Eyebrow className="text-subtle">Tasks</Eyebrow>
      {links.map(link => (
        <Pill.Link
          data-testid={`network-thread-task-link-${link.task_id}`}
          key={link.task_id}
          render={<Link params={{ id: link.task_id }} to="/tasks/$id" />}
        >
          <MonoId size="sm" value={link.task_id} />
        </Pill.Link>
      ))}
    </div>
  );
}
