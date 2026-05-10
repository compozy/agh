import { Link } from "@tanstack/react-router";
import { ArrowUpRight, Play } from "lucide-react";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
  Button,
  Pill,
} from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import { formatRelativeTime, taskRunStatusTone, taskStatusSignal } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

export interface TaskRunDetailHeaderProps {
  run: TaskRunDetailView;
  onCancelRun?: () => void;
  isCancelPending?: boolean;
}

function formatElapsed(startedAt?: string | null, endedAt?: string | null): string | null {
  if (!startedAt) {
    return null;
  }

  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) {
    return null;
  }

  const end = endedAt ? Date.parse(endedAt) : Date.now();
  if (Number.isNaN(end)) {
    return null;
  }

  const delta = Math.max(0, end - start);
  const totalSeconds = Math.floor(delta / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }

  return `${seconds}s`;
}

function normalizeSessionText(value?: string | null): string {
  return typeof value === "string" ? value.trim() : "";
}

export function TaskRunDetailHeader({
  run,
  onCancelRun,
  isCancelPending = false,
}: TaskRunDetailHeaderProps) {
  const record = run.run;
  const task = run.task;
  const session = run.session;
  const identifier = task.identifier ?? task.id;
  const canCancel =
    record.status === "queued" ||
    record.status === "claimed" ||
    record.status === "starting" ||
    record.status === "running";
  const elapsed = formatElapsed(record.started_at, record.ended_at);
  const signal = taskStatusSignal(record.status);
  const linkedSessionID = normalizeSessionText(session?.session_id ?? record.session_id);
  const linkedSessionAgent = normalizeSessionText(session?.agent_name);

  return (
    <header
      data-slot="page-header"
      className="flex min-h-11 flex-col gap-2 border-b border-(--line) px-4 py-2.5"
      data-testid="task-run-detail-header"
    >
      <div
        data-slot="page-header-breadcrumb"
        className="min-w-0 font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
      >
        <Breadcrumb data-testid="task-run-detail-breadcrumb">
          <BreadcrumbList className="text-(--muted)">
            <BreadcrumbItem>
              <BreadcrumbLink
                data-testid="task-run-detail-breadcrumb-tasks"
                render={<Link to="/tasks" />}
              >
                Tasks
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink
                data-testid="task-run-detail-breadcrumb-task"
                render={<Link params={{ id: task.id }} to="/tasks/$id" />}
              >
                {identifier}
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage className="text-(--muted)">{record.id}</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <div
        data-slot="page-header-main"
        className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3"
      >
        <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
          <span
            aria-hidden="true"
            data-slot="page-header-icon"
            className="inline-flex size-6 shrink-0 items-center justify-center rounded-(--radius-sm) bg-(--elevated) text-(--accent)"
          >
            <Play className="size-3.5" />
          </span>
          <h1 className="truncate text-[22px] font-medium tracking-[-0.026em] text-(--fg-strong)">
            <span className="flex min-w-0 items-center gap-2">
              <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
              <span
                className="flex min-w-0 items-center gap-1.5 text-item-title font-medium text-(--fg)"
                data-testid="task-run-detail-title"
              >
                Run{" "}
                <Pill mono data-testid="task-run-detail-run-id">
                  {record.id}
                </Pill>
              </span>
              <Pill tone={pillToneFromLegacyTone(taskRunStatusTone(record.status))}>
                {record.status}
              </Pill>
            </span>
          </h1>
        </div>
        <div
          data-slot="page-header-meta"
          className="ml-auto flex shrink-0 flex-wrap items-center gap-2 text-[13px] text-(--muted)"
        >
          {elapsed ? (
            <Pill mono data-testid="task-run-detail-duration">
              {elapsed}
            </Pill>
          ) : null}
          {linkedSessionID && linkedSessionAgent ? (
            <Pill.Link
              data-testid="task-run-detail-open-session"
              render={
                <Link
                  params={{ name: linkedSessionAgent, id: linkedSessionID }}
                  to="/agents/$name/sessions/$id"
                />
              }
            >
              Open session
              <ArrowUpRight className="size-3" />
            </Pill.Link>
          ) : linkedSessionID ? (
            <Pill.Link
              data-testid="task-run-detail-open-session"
              render={<Link params={{ id: linkedSessionID }} to="/session/$id" />}
            >
              Open session
              <ArrowUpRight className="size-3" />
            </Pill.Link>
          ) : null}
          {canCancel && onCancelRun ? (
            <Button
              data-testid="task-run-detail-cancel"
              disabled={isCancelPending}
              onClick={onCancelRun}
              size="sm"
              type="button"
              variant="outline"
            >
              Kill run
            </Button>
          ) : null}
        </div>
      </div>
      <div
        data-slot="page-header-status-row"
        className="flex flex-wrap items-center gap-x-4 gap-y-2 text-small-body text-(--muted)"
        data-testid="task-run-detail-meta"
      >
        <span>attempt {record.attempt}</span>
        {record.session_id ? <span className="font-mono">session {record.session_id}</span> : null}
        {record.claimed_by?.ref ? <span>claimed by {record.claimed_by.ref}</span> : null}
        {record.started_at ? <span>started {formatRelativeTime(record.started_at)}</span> : null}
      </div>
    </header>
  );
}
