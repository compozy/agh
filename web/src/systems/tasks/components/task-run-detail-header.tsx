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
  PageHeader,
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
      className="flex flex-col border-b border-(--color-divider)"
      data-testid="task-run-detail-header"
    >
      <PageHeader
        breadcrumb={
          <Breadcrumb data-testid="task-run-detail-breadcrumb">
            <BreadcrumbList className="text-(--color-text-label)">
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
                <BreadcrumbPage className="text-(--color-text-secondary)">
                  {record.id}
                </BreadcrumbPage>
              </BreadcrumbItem>
            </BreadcrumbList>
          </Breadcrumb>
        }
        icon={Play}
        meta={
          <div className="flex shrink-0 flex-wrap items-center gap-2">
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
        }
        title={
          <span className="flex min-w-0 items-center gap-2">
            <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
            <span
              className="flex min-w-0 items-center gap-1.5 text-item-title font-semibold text-(--color-text-primary)"
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
        }
        statusRow={
          <div
            className="flex flex-wrap items-center gap-2 text-small-body text-(--color-text-secondary)"
            data-testid="task-run-detail-meta"
          >
            <span>attempt {record.attempt}</span>
            {record.session_id ? (
              <span className="font-mono">session {record.session_id}</span>
            ) : null}
            {record.claimed_by?.ref ? <span>claimed by {record.claimed_by.ref}</span> : null}
            {record.started_at ? (
              <span>started {formatRelativeTime(record.started_at)}</span>
            ) : null}
          </div>
        }
      />
    </header>
  );
}
