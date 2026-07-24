import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Button, Panel, Section, StatusDot, Time } from "@agh/ui";

import { useApproveTask, useRejectTask } from "@/systems/tasks";

import type { HomeAttention, HomeAttentionItem } from "../types";

export interface HomeAttentionZoneProps {
  attention: HomeAttention;
}

type ResolvedKind = "approved" | "rejected";

interface HomeAttentionRowProps {
  item: HomeAttentionItem;
  resolved: ResolvedKind | undefined;
  onApprove: (taskId: string) => void;
  onReject: (taskId: string) => void;
  isMutating: boolean;
}

function attentionDotTone(kind: string): "warning" | "danger" {
  return kind === "failure" ? "danger" : "warning";
}

function attentionSentence(item: HomeAttentionItem): string {
  switch (item.kind) {
    case "approval":
      return "is waiting for your approval";
    case "failure":
      return item.detail ? `failed — ${item.detail}` : "failed";
    default:
      return item.detail ? `needs attention — ${item.detail}` : "needs your attention";
  }
}

function HomeAttentionRow({
  item,
  resolved,
  onApprove,
  onReject,
  isMutating,
}: HomeAttentionRowProps) {
  if (resolved) {
    return (
      <div
        className="grid grid-cols-[14px_minmax(0,1fr)_auto] items-center gap-3 bg-success-tint px-4 py-3"
        data-resolved={resolved}
        data-slot="home-attention-row"
      >
        <StatusDot label="Resolved" tone="faint" />
        <span className="truncate text-small-body text-muted">
          {resolved === "approved" ? "Approved — " : "Rejected — "}
          <span className="font-medium text-fg-strong">{item.title}</span>
          {resolved === "approved" ? " is starting" : " will not run"}
        </span>
        <span className="font-mono text-mono-id tabular-nums text-subtle">now</span>
      </div>
    );
  }

  return (
    <div
      className="grid grid-cols-[14px_minmax(0,1fr)_auto_auto] items-center gap-3 px-4 py-3 transition-colors duration-base hover:bg-row-hover max-[760px]:grid-cols-[14px_minmax(0,1fr)_auto]"
      data-slot="home-attention-row"
    >
      <StatusDot label={item.kind} tone={attentionDotTone(item.kind)} />
      <span className="truncate text-small-body text-muted max-[760px]:whitespace-normal">
        <span className="font-medium text-fg-strong">{item.title}</span> {attentionSentence(item)}
      </span>
      <span className="font-mono text-mono-id tabular-nums text-subtle max-[760px]:hidden">
        <Time iso={item.occurred_at} />
      </span>
      <span className="flex items-center gap-1.5">
        {item.actions.includes("approve") && item.task_id ? (
          <Button
            disabled={isMutating}
            onClick={() => onApprove(item.task_id ?? "")}
            size="sm"
            variant="primary"
          >
            Approve
          </Button>
        ) : null}
        {item.actions.includes("reject") && item.task_id ? (
          <Button
            disabled={isMutating}
            onClick={() => onReject(item.task_id ?? "")}
            size="sm"
            variant="ghost"
          >
            Reject
          </Button>
        ) : null}
        <Button
          render={<Link params={{ id: item.task_id ?? "" }} to="/tasks/$id" />}
          size="sm"
          variant="ghost"
        >
          Open
        </Button>
      </span>
    </div>
  );
}

/**
 * Zone 1 — everything currently waiting on the user, with the daemon-accepted
 * verbs inline. Approve/reject resolve the row optimistically; the overview
 * refetch reconciles the counters.
 */
export function HomeAttentionZone({ attention }: HomeAttentionZoneProps) {
  const approveTask = useApproveTask();
  const rejectTask = useRejectTask();
  const [resolvedById, setResolvedById] = useState<Record<string, ResolvedKind>>({});

  if (attention.total === 0 && attention.items.length === 0) {
    return (
      <Section
        count={0}
        label="Needs you"
        right={
          <Button
            render={<Link search={{ mode: "inbox" }} to="/tasks" />}
            size="sm"
            variant="ghost"
          >
            Open inbox
            <ChevronRight aria-hidden="true" />
          </Button>
        }
      >
        <Panel bodyClassName="px-4 py-3.5">
          <p className="text-small-body text-subtle">Nothing needs you right now.</p>
        </Panel>
      </Section>
    );
  }

  const resolveRow = (taskId: string, kind: ResolvedKind) => {
    setResolvedById(current => ({ ...current, [taskId]: kind }));
  };

  return (
    <Section
      count={attention.total}
      label="Needs you"
      right={
        <Button render={<Link search={{ mode: "inbox" }} to="/tasks" />} size="sm" variant="ghost">
          Open inbox
          <ChevronRight aria-hidden="true" />
        </Button>
      }
    >
      <Panel bodyClassName="p-0" className="overflow-hidden">
        <div className="divide-y divide-line-soft">
          {attention.items.map(item => (
            <HomeAttentionRow
              isMutating={approveTask.isPending || rejectTask.isPending}
              item={item}
              key={`${item.kind}:${item.task_id ?? item.title}`}
              onApprove={taskId => {
                approveTask.mutate(
                  { id: taskId },
                  { onSuccess: () => resolveRow(taskId, "approved") }
                );
              }}
              onReject={taskId => {
                rejectTask.mutate(
                  { id: taskId },
                  { onSuccess: () => resolveRow(taskId, "rejected") }
                );
              }}
              resolved={item.task_id ? resolvedById[item.task_id] : undefined}
            />
          ))}
        </div>
      </Panel>
    </Section>
  );
}
