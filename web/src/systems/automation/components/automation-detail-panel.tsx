import { AlertCircle, Bot, Clock3, Lock, Pencil, Play, Search, Trash2, Zap } from "lucide-react";

import {
  Button,
  CodeBlock,
  DetailHeader,
  Empty,
  Eyebrow,
  KindChip,
  Metric,
  Pill,
  Section,
  Spinner,
  type MetricTone,
} from "@agh/ui";

import { AutomationRunHistory } from "./automation-run-history";
import {
  automationScopeLabel,
  automationSourceLabel,
  automationStatusTone,
  describeFireLimit,
  describeRetry,
  describeSchedule,
  formatDate,
  formatDateTime,
  formatRelativeTime,
} from "../lib/automation-formatters";
import type {
  AutomationJob,
  AutomationRun,
  AutomationRunStatus,
  AutomationTrigger,
} from "../types";

export interface AutomationDetailEmptyState {
  actionLabel?: string;
  description: string;
  icon: "jobs" | "search" | "triggers";
  onAction?: () => void;
  title: string;
}

interface AutomationDetailPanelProps {
  emptyState?: AutomationDetailEmptyState | null;
  error: Error | null;
  state: {
    isDeleting: boolean;
    isLoading: boolean;
    isTogglePending: boolean;
    isTriggerPending: boolean;
  };
  item: AutomationJob | AutomationTrigger | undefined;
  kind: "jobs" | "triggers";
  onDelete: () => void;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onTriggerNow?: () => void;
  runs: AutomationRun[];
  runsError: Error | null;
  runsLoading: boolean;
}

interface JobMetricsCopy {
  successRateValue: string;
  successRateTone: MetricTone;
  lastRunValue: string;
  lastRunSubtext?: string;
  runsValue: string;
  nextRunValue: string;
  nextRunSubtext?: string;
}

const TERMINAL_STATUSES: ReadonlySet<AutomationRunStatus> = new Set([
  "completed",
  "failed",
  "canceled",
]);

function computeJobMetrics(runs: AutomationRun[], job: AutomationJob): JobMetricsCopy {
  const terminal = runs.filter(run => TERMINAL_STATUSES.has(run.status));
  const completed = runs.filter(run => run.status === "completed").length;
  const lastCompleted = terminal.find(run => run.status === "completed" || run.status === "failed");

  let successRateValue = "--";
  let successRateTone: MetricTone = "default";
  if (terminal.length > 0) {
    const pct = (completed / terminal.length) * 100;
    successRateValue = `${Math.round(pct)}%`;
    successRateTone = pct >= 90 ? "success" : pct >= 70 ? "default" : "warning";
  }

  const lastRunValue = lastCompleted ? formatRelativeTime(lastCompleted.started_at) : "--";
  const lastRunSubtext = lastCompleted ? formatDateTime(lastCompleted.started_at) : undefined;

  const nextRun = job.scheduler?.next_run_at ?? job.next_run;
  const nextRunValue = formatRelativeTime(nextRun);
  const nextRunSubtext = nextRun ? formatDateTime(nextRun) : undefined;

  return {
    successRateValue,
    successRateTone,
    lastRunValue,
    lastRunSubtext,
    runsValue: String(runs.length),
    nextRunValue,
    nextRunSubtext,
  };
}

function EmptyDetailState({ emptyState }: { emptyState: AutomationDetailEmptyState }) {
  const Icon = emptyState.icon === "jobs" ? Clock3 : emptyState.icon === "triggers" ? Zap : Search;

  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid="automation-detail-empty"
    >
      <Empty
        action={
          emptyState.actionLabel && emptyState.onAction ? (
            <Button onClick={emptyState.onAction} type="button" variant="outline">
              {emptyState.actionLabel}
            </Button>
          ) : undefined
        }
        className="max-w-md"
        description={emptyState.description}
        icon={Icon}
        title={emptyState.title}
      />
    </div>
  );
}

function JobScheduleSection({ job }: { job: AutomationJob }) {
  const mode = job.schedule?.mode ?? "manual";
  const expression =
    job.schedule?.mode === "cron"
      ? (job.schedule.expr ?? "--")
      : job.schedule?.mode === "every"
        ? (job.schedule.interval ?? "--")
        : job.schedule?.mode === "at"
          ? formatDateTime(job.schedule.time)
          : "Manual";

  return (
    <Section
      label="Schedule"
      right={
        <Pill mono tone="accent">
          {mode}
        </Pill>
      }
    >
      <div className="flex flex-wrap items-center gap-3 rounded-md border border-line bg-canvas-soft px-4 py-3">
        <div className="min-w-0 flex-1">
          <p className="font-mono text-item-title text-fg">{expression}</p>
          <p className="mt-1 text-xs text-muted">{describeSchedule(job.schedule)}</p>
        </div>
      </div>
    </Section>
  );
}

function JobStatsSection({ job, runs }: { job: AutomationJob; runs: AutomationRun[] }) {
  const metrics = computeJobMetrics(runs, job);

  return (
    <Section label="Stats">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <Metric
          data-testid="automation-job-metric-runs"
          label="Runs"
          subtext="lifetime executions"
          value={metrics.runsValue}
        />
        <Metric
          data-testid="automation-job-metric-success-rate"
          label="Success rate"
          subtext="terminal runs only"
          tone={metrics.successRateTone}
          value={metrics.successRateValue}
        />
        <Metric
          data-testid="automation-job-metric-last-run"
          label="Last run"
          subtext={metrics.lastRunSubtext}
          value={metrics.lastRunValue}
        />
        <Metric
          data-testid="automation-job-metric-next-run"
          label="Next run"
          subtext={metrics.nextRunSubtext}
          tone="accent"
          value={metrics.nextRunValue}
        />
      </div>
    </Section>
  );
}

function JobSchedulerSection({ job }: { job: AutomationJob }) {
  if (!job.scheduler) {
    return null;
  }

  const scheduler = job.scheduler;
  const registeredTone = scheduler.registered ? "success" : "neutral";

  return (
    <Section
      label="Scheduler"
      right={
        <Pill mono tone={registeredTone}>
          {scheduler.registered ? "REGISTERED" : "IDLE"}
        </Pill>
      }
    >
      <div
        className="grid gap-2 rounded-md border border-line bg-canvas-soft px-4 py-3 md:grid-cols-2"
        data-testid="automation-job-scheduler"
      >
        <div>
          <Eyebrow className="text-muted">Next cursor</Eyebrow>
          <p className="mt-1 text-small-body text-muted">{formatDateTime(scheduler.next_run_at)}</p>
        </div>
        <div>
          <Eyebrow className="text-muted">Last scheduled</Eyebrow>
          <p className="mt-1 text-small-body text-muted">
            {formatDateTime(scheduler.last_scheduled_at)}
          </p>
        </div>
        <div>
          <Eyebrow className="text-muted">Fire ID</Eyebrow>
          <p className="mt-1 break-all font-mono text-xs text-muted">
            {scheduler.last_fire_id || "--"}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <Eyebrow className="text-muted">Catch-up</Eyebrow>
            <p className="mt-1 font-mono text-xs text-muted">
              {scheduler.catch_up_policy ?? "skip_missed"}
            </p>
          </div>
          <div>
            <Eyebrow className="text-muted">Misfires</Eyebrow>
            <p className="mt-1 font-mono text-xs text-muted">{scheduler.misfire_count ?? 0}</p>
          </div>
        </div>
      </div>
    </Section>
  );
}

function TriggerHookSection({ trigger }: { trigger: AutomationTrigger }) {
  const filters = Object.entries(trigger.filter ?? {});

  return (
    <Section label="Hook" right={<KindChip kind={trigger.event} />}>
      <div className="space-y-3 rounded-md border border-line bg-canvas-soft px-4 py-3">
        <div className="flex flex-wrap items-center gap-2">
          <Eyebrow className="text-muted">Event</Eyebrow>
          <Pill mono tone="info">
            {trigger.event}
          </Pill>
        </div>
        <div className="space-y-1.5">
          <Eyebrow className="text-muted">Filters</Eyebrow>
          {filters.length === 0 ? (
            <p className="text-xs text-subtle">No filters</p>
          ) : (
            <div className="flex flex-wrap items-center gap-1.5">
              {filters.map(([key, value]) => (
                <Pill mono key={`${key}=${value}`} tone="neutral">
                  {`${key}=${value}`}
                </Pill>
              ))}
            </div>
          )}
        </div>
        {trigger.event === "webhook" ? (
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <Eyebrow className="text-muted">Endpoint</Eyebrow>
              <p className="mt-1 font-mono text-small-body text-fg">
                {trigger.endpoint_slug ?? "--"}
              </p>
            </div>
            <div>
              <Eyebrow className="text-muted">Webhook id</Eyebrow>
              <p className="mt-1 font-mono text-small-body text-fg">{trigger.webhook_id ?? "--"}</p>
            </div>
          </div>
        ) : null}
        <div className="flex flex-wrap items-center gap-2">
          <Bot className="size-3.5 text-subtle" />
          <span className="text-small-body text-muted">Dispatches to</span>
          <Pill mono tone="neutral">
            {trigger.agent_name}
          </Pill>
        </div>
      </div>
    </Section>
  );
}

function PromptSection({ isTrigger, prompt }: { isTrigger: boolean; prompt: string }) {
  return (
    <Section
      label={isTrigger ? "Prompt template" : "Prompt"}
      right={
        isTrigger ? (
          <Pill mono tone="info">
            GO TEMPLATE
          </Pill>
        ) : undefined
      }
    >
      <CodeBlock code={prompt} copyable={false} showPrompt={false} />
    </Section>
  );
}

function GovernanceSection({ item }: { item: AutomationJob | AutomationTrigger }) {
  return (
    <Section label="Governance">
      <div className="grid gap-2 rounded-md border border-line bg-canvas-soft px-4 py-3 md:grid-cols-2">
        <div>
          <Eyebrow className="text-muted">Retry</Eyebrow>
          <p className="mt-1 text-small-body text-muted">{describeRetry(item.retry)}</p>
        </div>
        <div>
          <Eyebrow className="text-muted">Fire limit</Eyebrow>
          <p className="mt-1 text-small-body text-muted">{describeFireLimit(item.fire_limit)}</p>
        </div>
      </div>
    </Section>
  );
}

export function AutomationDetailPanel({
  emptyState,
  error,
  state,
  item,
  kind,
  onDelete,
  onEdit,
  onToggleEnabled,
  onTriggerNow,
  runs,
  runsError,
  runsLoading,
}: AutomationDetailPanelProps) {
  const { isDeleting, isLoading, isTogglePending, isTriggerPending } = state;
  if (isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="automation-detail-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center p-6"
        data-testid="automation-detail-error"
      >
        <Empty
          className="max-w-md"
          description={error.message ?? "Failed to load automation details"}
          icon={AlertCircle}
          title="Unable to load details"
        />
      </div>
    );
  }

  if (!item) {
    if (emptyState) {
      return <EmptyDetailState emptyState={emptyState} />;
    }

    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center p-6"
        data-testid="automation-detail-empty"
      >
        <Empty
          className="max-w-md"
          description="Select an automation item to inspect its configuration and run history."
          icon={Search}
          title="Select an automation"
        />
      </div>
    );
  }

  const isJob = kind === "jobs";
  const isDynamic = item.source === "dynamic";
  const job = isJob ? (item as AutomationJob) : null;
  const trigger = !isJob ? (item as AutomationTrigger) : null;
  const enabledTone = automationStatusTone(item.enabled ? "enabled" : "disabled");

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="automation-detail-panel"
    >
      <DetailHeader
        actions={
          <>
            <Button
              data-testid="toggle-automation-btn"
              disabled={isTogglePending}
              onClick={() => onToggleEnabled(!item.enabled)}
              size="sm"
              type="button"
              variant="outline"
            >
              {isTogglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
            </Button>
            {isJob && onTriggerNow ? (
              <Button
                data-testid="trigger-job-btn"
                disabled={isTriggerPending}
                onClick={onTriggerNow}
                size="sm"
                type="button"
              >
                <Play className="size-3.5" />
                {isTriggerPending ? "Queuing..." : "Run now"}
              </Button>
            ) : null}
            {isDynamic ? (
              <Button
                data-testid="edit-automation-btn"
                onClick={onEdit}
                size="sm"
                type="button"
                variant="outline"
              >
                <Pencil className="size-3.5" />
                Edit
              </Button>
            ) : null}
            {isDynamic ? (
              <Button
                data-testid="delete-automation-btn"
                disabled={isDeleting}
                onClick={onDelete}
                size="sm"
                type="button"
                variant="destructive"
              >
                <Trash2 className="size-3.5" />
                {isDeleting ? "Deleting..." : "Delete"}
              </Button>
            ) : null}
          </>
        }
        data-testid="automation-detail-header"
        meta={
          <span data-testid="automation-detail-meta">
            {`Agent: ${item.agent_name} · Scope: ${automationScopeLabel(item.scope)} · Updated ${formatDate(item.updated_at)}`}
          </span>
        }
        pills={
          <>
            <span className="flex items-center gap-1.5">
              <Pill.Dot tone={enabledTone} />
              <Pill mono tone={enabledTone}>
                {item.enabled ? "ENABLED" : "DISABLED"}
              </Pill>
            </span>
            <Pill mono tone={item.source === "dynamic" ? "info" : "neutral"}>
              {automationSourceLabel(item.source)}
            </Pill>
            {item.source === "config" ? (
              <Lock aria-hidden="true" className="size-3.5 text-subtle" />
            ) : null}
          </>
        }
        title={item.name}
      />

      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-5">
        {!isDynamic ? (
          <div className="flex items-start gap-2 rounded-md border border-dashed border-line px-4 py-3 text-xs text-muted">
            <Lock aria-hidden="true" className="mt-0.5 size-3.5 shrink-0 text-subtle" />
            <p>
              This automation is defined in configuration files. Only the enabled state can be
              toggled from the UI.
            </p>
          </div>
        ) : null}

        {job ? <JobScheduleSection job={job} /> : null}
        {job ? <JobStatsSection job={job} runs={runs} /> : null}
        {job ? <JobSchedulerSection job={job} /> : null}
        {trigger ? <TriggerHookSection trigger={trigger} /> : null}

        <PromptSection isTrigger={!isJob} prompt={item.prompt} />
        <GovernanceSection item={item} />

        <AutomationRunHistory
          emptyDescription={
            isJob
              ? "Runs will appear here after the first scheduled or manual execution."
              : "Runs will appear here after the first matching activation."
          }
          emptyTitle="No runs recorded yet"
          error={runsError}
          isLoading={runsLoading}
          runs={runs}
          title="Runs"
        />
      </div>
    </section>
  );
}
