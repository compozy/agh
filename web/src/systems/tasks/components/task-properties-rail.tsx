import { Link } from "@tanstack/react-router";
import { Search } from "lucide-react";
import type { ReactNode } from "react";

import { Button, MonoId, OwnerAvatar, Pill, PropertyRow, Time } from "@agh/ui";

import {
  computeElapsed,
  ownerAvatarKindFor,
  taskOwnerLabel,
  taskRunStatusLabel,
  taskRunStatusTone,
} from "../lib/task-formatters";
import { taskAttemptsUsed } from "../lib/task-command-state";
import type { TaskDetailView, TaskExecutionProfile, TaskRun } from "../types";
import { TaskAutoEnqueueSwitch, TaskPriorityEditor } from "./task-rail-editors";

export interface TaskPropertiesRailProps {
  detail: TaskDetailView;
  runs: readonly TaskRun[];
  profile?: TaskExecutionProfile | null;
  onEditSetup: () => void;
  onInspect: () => void;
  onApprove: () => void;
  onReject: () => void;
  approvalPending?: { approve?: boolean; reject?: boolean };
}

function RailSection({
  label,
  action,
  children,
}: {
  label: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="border-t border-line-soft px-4 py-3.5 first:border-t-0">
      <header className="mb-2 flex items-center justify-between gap-2">
        <span className="eyebrow text-subtle">{label}</span>
        {action}
      </header>
      {children}
    </section>
  );
}

function workerSummary(profile?: TaskExecutionProfile | null): string | null {
  const worker = profile?.worker;
  if (!worker) return null;
  if (worker.agent_name) return worker.agent_name;
  if (worker.allowed_agent_names && worker.allowed_agent_names.length > 0) {
    const [first, ...rest] = worker.allowed_agent_names;
    return rest.length > 0 ? `${first} +${rest.length}` : (first ?? null);
  }
  return worker.mode === "inherit" ? "Inherited" : null;
}

function modelSummary(profile?: TaskExecutionProfile | null): string | null {
  const worker = profile?.worker;
  if (!worker?.model) return null;
  return worker.provider ? `${worker.provider} · ${worker.model}` : worker.model;
}

function sandboxSummary(profile?: TaskExecutionProfile | null): string | null {
  const sandbox = profile?.sandbox;
  if (!sandbox) return null;
  if (sandbox.mode === "ref") return sandbox.sandbox_ref ?? null;
  if (sandbox.mode === "none") return "None";
  return "Inherited";
}

function networkChannel(profile?: TaskExecutionProfile | null): string | null {
  const participation = profile?.network_participation;
  if (participation && participation.mode === "live" && "channel_id" in participation) {
    return participation.channel_id ?? null;
  }
  return null;
}

/**
 * 320px properties rail (§4.4): tier a/b fields only, grouped, with the sole
 * operator entry point (Inspect) in the footer. No lease, heartbeat, claim
 * hash, or seq here — those stay behind Inspect.
 */
export function TaskPropertiesRail({
  detail,
  runs,
  profile,
  onEditSetup,
  onInspect,
  onApprove,
  onReject,
  approvalPending = {},
}: TaskPropertiesRailProps) {
  const record = detail.task;
  const activeRun = detail.summary?.active_run ?? null;
  const owner = record.owner ?? null;
  const ownerName = owner ? taskOwnerLabel(owner) : "Unassigned";
  const attemptsMax = record.max_attempts ?? null;
  const attemptsLabel = attemptsMax
    ? `${taskAttemptsUsed(runs, activeRun)} of ${attemptsMax}`
    : String(taskAttemptsUsed(runs, activeRun));
  const worker = workerSummary(profile);
  const model = modelSummary(profile);
  const sandbox = sandboxSummary(profile);
  const channel = networkChannel(profile);
  const stuckRun = activeRun?.status === "needs_attention" ? activeRun : null;
  const lastFailedRun =
    !activeRun && record.status === "failed"
      ? [...runs].filter(run => run.status === "failed").sort((a, b) => b.attempt - a.attempt)[0]
      : null;

  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="tasks-detail-rail"
    >
      {record.approval_state === "pending" ? (
        <RailSection label="Approval">
          <PropertyRow label="State">
            <Pill.Dot tone="info" />
            Pending
          </PropertyRow>
          <PropertyRow label="Requested by">{record.created_by?.ref ?? "unknown"}</PropertyRow>
          <div className="mt-2 flex items-center justify-end gap-2">
            <Button
              data-testid="tasks-rail-reject"
              disabled={approvalPending.reject}
              onClick={onReject}
              size="sm"
              type="button"
              variant="ghost"
            >
              Reject
            </Button>
            <Button
              data-testid="tasks-rail-approve"
              disabled={approvalPending.approve}
              onClick={onApprove}
              size="sm"
              type="button"
              variant="neutral"
            >
              Approve
            </Button>
          </div>
        </RailSection>
      ) : null}

      {stuckRun ? (
        <RailSection label="Current run">
          <PropertyRow label="Status">
            <Pill.Dot tone={taskRunStatusTone(stuckRun.status)} />
            {taskRunStatusLabel(stuckRun.status)}
          </PropertyRow>
          {stuckRun.claimed_by?.ref ? (
            <PropertyRow label="Claimed by">
              <OwnerAvatar
                name={stuckRun.claimed_by.ref}
                ownerId={stuckRun.claimed_by.ref}
                ownerKind={ownerAvatarKindFor(stuckRun.claimed_by.kind)}
                size="sm"
              />
              <span className="truncate">{stuckRun.claimed_by.ref}</span>
            </PropertyRow>
          ) : null}
          {stuckRun.heartbeat_at ? (
            <PropertyRow label="Last heartbeat">
              <Time iso={stuckRun.heartbeat_at} mode="relative" />
            </PropertyRow>
          ) : null}
          <PropertyRow label="Run id" mono>
            <MonoId value={stuckRun.id} />
          </PropertyRow>
        </RailSection>
      ) : null}

      {lastFailedRun ? (
        <RailSection label="Last run">
          <PropertyRow label="Status">
            <Pill.Dot tone="danger" />
            Failed
          </PropertyRow>
          {lastFailedRun.ended_at ? (
            <PropertyRow label="Ended">
              <Time iso={lastFailedRun.ended_at} mode="relative" />
            </PropertyRow>
          ) : null}
          <PropertyRow label="Duration" mono>
            {computeElapsed(lastFailedRun) ?? "—"}
          </PropertyRow>
          <PropertyRow label="Run id" mono>
            <MonoId value={lastFailedRun.id} />
          </PropertyRow>
        </RailSection>
      ) : null}

      <RailSection label="Properties">
        <PropertyRow editor={<TaskPriorityEditor task={record} />} label="Priority" />
        <PropertyRow label="Owner">
          {owner ? (
            <>
              <OwnerAvatar
                name={ownerName}
                ownerId={owner.ref ?? ownerName}
                ownerKind={ownerAvatarKindFor(owner.kind)}
                size="sm"
              />
              <span className="truncate">{ownerName}</span>
            </>
          ) : (
            <span className="text-muted">Unassigned</span>
          )}
        </PropertyRow>
        {record.workspace_id ? (
          <PropertyRow label="Workspace">{record.workspace_id}</PropertyRow>
        ) : null}
        {record.parent_task_id ? (
          <PropertyRow
            editor={
              <Link
                className="inline-flex min-w-0 items-center rounded-sm px-1.5 py-0.5 text-small-body font-medium text-fg hover:bg-row-hover focus-visible:outline-none focus-visible:shadow-focus-ring"
                data-testid="tasks-rail-parent"
                params={{ id: record.parent_task_id }}
                to="/tasks/$id"
              >
                <span className="truncate">Open parent task</span>
              </Link>
            }
            label="Parent task"
          />
        ) : null}
      </RailSection>

      <RailSection
        action={
          <Button
            className="-mr-1.5 h-auto px-1.5 py-0.5 text-eyebrow font-medium text-muted"
            data-testid="tasks-rail-edit-setup"
            onClick={onEditSetup}
            size="sm"
            type="button"
            variant="ghost"
          >
            Edit setup
          </Button>
        }
        label="Execution"
      >
        {worker ? <PropertyRow label="Agent">{worker}</PropertyRow> : null}
        {model ? (
          <PropertyRow label="Model" mono>
            {model}
          </PropertyRow>
        ) : null}
        {sandbox ? <PropertyRow label="Sandbox">{sandbox}</PropertyRow> : null}
        <PropertyRow label="Attempts">{attemptsLabel}</PropertyRow>
        <PropertyRow editor={<TaskAutoEnqueueSwitch task={record} />} label="Auto-enqueue" />
        {channel ? (
          <PropertyRow label="Channel" mono>
            {channel}
          </PropertyRow>
        ) : null}
      </RailSection>

      <RailSection label="Activity">
        <PropertyRow label="Created">
          <Time iso={record.created_at} mode="relative" />
        </PropertyRow>
        <PropertyRow label="Updated">
          <Time iso={record.updated_at} mode="relative" />
        </PropertyRow>
        {record.closed_at ? (
          <PropertyRow label="Closed">
            <Time iso={record.closed_at} mode="relative" />
          </PropertyRow>
        ) : null}
        <PropertyRow label="Created by">{record.created_by?.ref ?? "unknown"}</PropertyRow>
        <PropertyRow label="Task id" mono>
          <MonoId value={record.id} />
        </PropertyRow>
        {activeRun ? (
          <PropertyRow label="Run id" mono>
            <MonoId value={activeRun.id} />
          </PropertyRow>
        ) : null}
      </RailSection>

      <footer className="flex items-center justify-between gap-2 border-t border-line-soft px-3 py-2.5">
        <Button
          data-testid="tasks-rail-inspect"
          onClick={onInspect}
          size="sm"
          type="button"
          variant="ghost"
        >
          <Search aria-hidden="true" className="size-3" />
          Inspect
        </Button>
        <span className="truncate font-mono text-micro text-faint">
          agh task inspect {record.id}
        </span>
      </footer>
    </div>
  );
}
