import { useState } from "react";
import { AlertCircle, Lock, Pencil, Play, Search } from "lucide-react";
import {
  Button,
  cn,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Empty,
  Eyebrow,
  Metric,
  PAGE_CONTENT_GUTTER,
  PageHead,
  Pill,
  Section,
  Spinner,
  TopbarOverflowIcon,
  useTopbarSlot,
  type MetricTone,
} from "@agh/ui";

import {
  automationScopeLabel,
  automationSourceLabel,
  automationStatusTone,
  describeSchedule,
  formatDate,
  formatDateTime,
  formatRelativeTime,
} from "../lib/automation-formatters";
import { automationTargetLabel, projectAutomationTarget } from "../lib/automation-target";
import type {
  AutomationJob,
  AutomationRun,
  AutomationRunStatus,
  AutomationTrigger,
} from "../types";
import { AutomationRunHistory } from "./automation-run-history";
import { AutomationDeleteAction } from "./automation-delete-action";
import {
  AutomationTargetSection,
  GovernanceSection,
  PromptSection,
  TriggerHookSection,
} from "./automation-detail-sections";

interface AutomationDetailPanelProps {
  error: Error | null;
  state: {
    isDeleting: boolean;
    isLoading: boolean;
    isTogglePending: boolean;
    isTriggerPending: boolean;
  };
  item: AutomationJob | AutomationTrigger | undefined;
  kind: "jobs" | "triggers";
  onDelete: () => void | Promise<void>;
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
          label="Runs shown"
          subtext="recent window"
          value={metrics.runsValue}
        />
        <Metric
          data-testid="automation-job-metric-success-rate"
          label="Recent success"
          subtext="shown terminal runs"
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
              {scheduler.catch_up_policy ?? "skip"}
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

export function AutomationDetailPanel({
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
        className="flex min-h-0 flex-1 items-center justify-center py-10"
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
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center py-10"
        data-testid="automation-detail-empty"
      >
        <Empty
          className="max-w-md"
          description={`This ${kind === "jobs" ? "job" : "trigger"} is no longer available.`}
          icon={Search}
          title={kind === "jobs" ? "Job unavailable" : "Trigger unavailable"}
        />
      </div>
    );
  }

  return (
    <AutomationDetailLoadedPanel
      isDeleting={isDeleting}
      isTogglePending={isTogglePending}
      isTriggerPending={isTriggerPending}
      item={item}
      kind={kind}
      onDelete={onDelete}
      onEdit={onEdit}
      onToggleEnabled={onToggleEnabled}
      onTriggerNow={onTriggerNow}
      runs={runs}
      runsError={runsError}
      runsLoading={runsLoading}
    />
  );
}

interface AutomationDetailLoadedPanelProps {
  isDeleting: boolean;
  isTogglePending: boolean;
  isTriggerPending: boolean;
  item: AutomationJob | AutomationTrigger;
  kind: "jobs" | "triggers";
  onDelete: () => void | Promise<void>;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onTriggerNow?: () => void;
  runs: AutomationRun[];
  runsError: Error | null;
  runsLoading: boolean;
}

function AutomationDetailLoadedPanel({
  isDeleting,
  isTogglePending,
  isTriggerPending,
  item,
  kind,
  onDelete,
  onEdit,
  onToggleEnabled,
  onTriggerNow,
  runs,
  runsError,
  runsLoading,
}: AutomationDetailLoadedPanelProps) {
  const isJob = kind === "jobs";
  const isDynamic = item.source === "dynamic";
  const job = isJob ? (item as AutomationJob) : null;
  const trigger = !isJob ? (item as AutomationTrigger) : null;
  const target = projectAutomationTarget(item);
  const enabledTone = automationStatusTone(item.enabled ? "enabled" : "disabled");
  const [deleteOpen, setDeleteOpen] = useState(false);
  const showRunNow = isJob && Boolean(onTriggerNow);
  // When Edit/Run claim the action row, Enable/Disable (+ Delete) move to overflow.
  const showOverflow = isDynamic || showRunNow;
  // Route chrome §07: secondary → primary → overflow (Delete / Enable|Disable).
  const detailActions = (
    <div className="flex items-center gap-2" data-testid="automation-detail-actions">
      {isDynamic ? (
        <Button
          data-testid="edit-automation-btn"
          onClick={onEdit}
          size="sm"
          type="button"
          variant="neutral"
        >
          <Pencil className="size-3" />
          Edit
        </Button>
      ) : null}
      {showRunNow ? (
        <Button
          data-testid="trigger-job-btn"
          disabled={isTriggerPending}
          onClick={onTriggerNow}
          size="sm"
          type="button"
        >
          <Play className="size-3" />
          {isTriggerPending ? "Queuing..." : "Run now"}
        </Button>
      ) : null}
      {!showOverflow ? (
        <Button
          data-testid="toggle-automation-btn"
          disabled={isTogglePending}
          onClick={() => onToggleEnabled(!item.enabled)}
          size="sm"
          type="button"
          variant={item.enabled ? "neutral" : "default"}
        >
          {isTogglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
        </Button>
      ) : null}
    </div>
  );
  const detailOverflow = showOverflow ? (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="More actions"
        data-testid="automation-detail-overflow"
        render={<Button type="button" variant="ghost" size="icon-sm" />}
      >
        <TopbarOverflowIcon aria-hidden="true" className="size-3" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="automation-detail-overflow-menu">
        <DropdownMenuItem
          data-testid="toggle-automation-btn"
          disabled={isTogglePending}
          onClick={() => onToggleEnabled(!item.enabled)}
        >
          {isTogglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
        </DropdownMenuItem>
        {isDynamic ? (
          <DropdownMenuItem
            data-testid="delete-automation-btn"
            onClick={() => setDeleteOpen(true)}
            variant="destructive"
          >
            Delete {isJob ? "job" : "trigger"}
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  ) : undefined;
  const detailMeta = (
    <span data-testid="automation-detail-meta">
      {`${automationTargetLabel(target)} · Scope: ${automationScopeLabel(item.scope)} · Updated ${formatDate(item.updated_at)}`}
    </span>
  );
  const detailPills = (
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
      {item.source === "config" ? <Lock aria-hidden="true" className="size-3 text-subtle" /> : null}
    </>
  );

  // Single publisher: crumb + actions/overflow must share one slot push (store is last-writer).
  useTopbarSlot({
    actions: detailActions,
    crumb: item.name,
    overflow: detailOverflow,
  });

  return (
    <section
      className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col overflow-hidden")}
      data-testid="automation-detail-panel"
    >
      {isDynamic ? (
        <AutomationDeleteAction
          hideTrigger
          isPending={isDeleting}
          kind={kind}
          name={item.name}
          onConfirm={onDelete}
          onOpenChange={setDeleteOpen}
          open={deleteOpen}
        />
      ) : null}
      <div className="pt-5">
        <PageHead
          data-testid="automation-detail-header"
          title={item.name}
          variant="detail"
          meta={detailMeta}
          pills={detailPills}
        />
      </div>

      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto py-5">
        {!isDynamic ? (
          <div className="flex items-start gap-2 rounded-md border border-dashed border-line px-4 py-3 text-xs text-muted">
            <Lock aria-hidden="true" className="mt-0.5 size-3 shrink-0 text-subtle" />
            <p>
              This automation is defined in configuration files. Only the enabled state can be
              toggled from the UI.
            </p>
          </div>
        ) : null}

        {job ? <JobScheduleSection job={job} /> : null}
        {job ? <JobStatsSection job={job} runs={runs} /> : null}
        {job ? <JobSchedulerSection job={job} /> : null}
        {trigger ? (
          <TriggerHookSection target={target.kind === "agent" ? target : null} trigger={trigger} />
        ) : null}

        {target.kind === "loop" ? (
          <AutomationTargetSection showInputMapping={!isJob} target={target} />
        ) : null}
        {target.kind === "agent" ? (
          <PromptSection isTrigger={!isJob} prompt={target.prompt} />
        ) : null}
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
