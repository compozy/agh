import {
  ArrowRight,
  Bot,
  CalendarDays,
  Clock3,
  Loader2,
  Lock,
  Pencil,
  Play,
  RefreshCw,
  Search,
  Trash2,
  Zap,
} from "lucide-react";
import type { ComponentType } from "react";

import { Pill } from "@/components/design-system";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

import { AutomationRunHistory } from "./automation-run-history";
import {
  automationScopeLabel,
  automationSemanticTone,
  automationSourceLabel,
  automationSourceTone,
  describeFireLimit,
  describeRetry,
  describeSchedule,
  formatDate,
  formatDateTime,
  formatRelativeTime,
} from "../lib/automation-formatters";
import type { AutomationJob, AutomationRun, AutomationTrigger } from "../types";

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
  isDeleting: boolean;
  isLoading: boolean;
  isTogglePending: boolean;
  isTriggerPending: boolean;
  item: AutomationJob | AutomationTrigger | undefined;
  kind: "jobs" | "triggers";
  onDelete: () => void;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onTriggerNow: () => void;
  runs: AutomationRun[];
  runsError: Error | null;
  runsLoading: boolean;
}

function AutomationTag({
  children,
  tone,
}: {
  children: string;
  tone: "amber" | "danger" | "green" | "neutral" | "violet";
}) {
  return (
    <Pill className="border-none" emphasis="strong" kind="state" tone={tone}>
      {children}
    </Pill>
  );
}

function SectionEyebrow({ children }: { children: string }) {
  return (
    <p className="font-mono text-[0.66rem] font-semibold tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
      {children}
    </p>
  );
}

function MetaChip({
  children,
  icon,
}: {
  children: string;
  icon?: ComponentType<{ className?: string }>;
}) {
  const Icon = icon;

  return (
    <span className="inline-flex items-center gap-2 rounded-full bg-[color:var(--color-neutral-tint)] px-3 py-1.5 text-sm text-[color:var(--color-text-secondary)]">
      {Icon ? <Icon className="size-3.5 text-[color:var(--color-text-tertiary)]" /> : null}
      {children}
    </span>
  );
}

function EmptyState({
  actionLabel,
  description,
  icon,
  onAction,
  title,
}: AutomationDetailEmptyState) {
  const Icon = icon === "jobs" ? Clock3 : icon === "triggers" ? Zap : Search;

  return (
    <div
      className="flex flex-1 items-center justify-center px-8 py-8"
      data-testid="automation-detail-empty"
    >
      <Empty className="border-none bg-transparent p-0">
        <EmptyHeader className="gap-4">
          <EmptyMedia
            className="size-18 rounded-3xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-label)] [&_svg:not([class*='size-'])]:size-8"
            variant="icon"
          >
            <Icon />
          </EmptyMedia>
          <div className="space-y-2">
            <EmptyTitle className="text-xl font-medium text-[color:var(--color-text-primary)]">
              {title}
            </EmptyTitle>
            <EmptyDescription className="max-w-md text-sm leading-6 text-[color:var(--color-text-secondary)]">
              {description}
            </EmptyDescription>
          </div>
        </EmptyHeader>
        {actionLabel && onAction ? (
          <EmptyContent className="pt-1">
            <Button
              className="border-[color:var(--color-accent)] bg-transparent text-[color:var(--color-accent)] hover:bg-[color:var(--color-accent-tint)] hover:text-[color:var(--color-accent)]"
              onClick={onAction}
              size="lg"
              type="button"
              variant="outline"
            >
              <span className="font-mono text-[0.8rem] leading-none">+</span>
              {actionLabel}
            </Button>
          </EmptyContent>
        ) : null}
      </Empty>
    </div>
  );
}

function JobScheduleCard({ job }: { job: AutomationJob }) {
  const mode = job.schedule ? (job.schedule.mode ?? "cron") : "manual";
  const ScheduleIcon = mode === "every" ? RefreshCw : mode === "at" ? CalendarDays : Clock3;
  const scheduleValue =
    mode === "cron"
      ? (job.schedule?.expr ?? "Cron schedule")
      : mode === "every"
        ? (job.schedule?.interval ?? "Interval")
        : mode === "at"
          ? formatDate(job.schedule?.time)
          : describeSchedule(job.schedule);

  return (
    <section className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-5">
      <SectionEyebrow>SCHEDULE</SectionEyebrow>
      <div className="mt-4 flex flex-col gap-5 xl:flex-row xl:items-center xl:justify-between">
        <div className="flex min-w-0 items-center gap-5">
          <div className="flex size-14 shrink-0 flex-col items-center justify-center rounded-xl bg-[color:var(--color-surface-elevated)] text-[color:var(--color-accent)]">
            <ScheduleIcon className="size-5" />
          </div>
          <div className="min-w-0">
            <p className="font-mono text-[0.68rem] tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
              {mode}
            </p>
            <p className="mt-2 font-mono text-3xl tracking-[-0.04em] text-[color:var(--color-text-primary)]">
              {scheduleValue}
            </p>
            <p className="mt-2 text-base text-[color:var(--color-text-secondary)]">
              {describeSchedule(job.schedule)}
            </p>
          </div>
        </div>

        <div className="text-left xl:text-right">
          <p className="font-mono text-[0.68rem] tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
            Next run
          </p>
          <p className="mt-2 text-3xl font-semibold tracking-[-0.03em] text-[color:var(--color-accent)]">
            {formatRelativeTime(job.next_run)}
          </p>
          <p className="mt-2 text-sm text-[color:var(--color-text-secondary)]">
            {formatDateTime(job.next_run)}
          </p>
        </div>
      </div>
    </section>
  );
}

function TriggerActivationCard({ trigger }: { trigger: AutomationTrigger }) {
  const matches = Object.entries(trigger.filter ?? {});

  return (
    <section className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-5">
      <SectionEyebrow>ACTIVATION</SectionEyebrow>
      <div className="mt-5 flex flex-col gap-4 xl:flex-row xl:items-center">
        <div className="space-y-3">
          <p className="font-mono text-[0.66rem] tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
            When
          </p>
          <span className="inline-flex min-h-10 items-center rounded-xl border border-[color:var(--color-info)] bg-[color:var(--color-info-tint)] px-4 py-2 font-mono text-[0.8rem] text-[color:var(--color-info)]">
            {trigger.event}
          </span>
        </div>

        <ArrowRight className="hidden size-4 shrink-0 text-[color:var(--color-text-tertiary)] xl:block" />

        <div className="space-y-3">
          <p className="font-mono text-[0.66rem] tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
            Matches
          </p>
          <div className="flex flex-wrap items-center gap-2">
            {matches.length > 0 ? (
              matches.map(([key, value]) => (
                <span
                  className="inline-flex min-h-9 items-center rounded-xl bg-[color:var(--color-surface-elevated)] px-3 py-2 font-mono text-[0.76rem] text-[color:var(--color-text-secondary)]"
                  key={`${key}-${value}`}
                >
                  {`${key} ${value}`}
                </span>
              ))
            ) : (
              <span className="inline-flex min-h-9 items-center rounded-xl bg-[color:var(--color-surface-elevated)] px-3 py-2 text-sm text-[color:var(--color-text-secondary)]">
                No filters
              </span>
            )}
          </div>
        </div>

        <ArrowRight className="hidden size-4 shrink-0 text-[color:var(--color-text-tertiary)] xl:block" />

        <div className="space-y-3">
          <p className="font-mono text-[0.66rem] tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
            Dispatches to
          </p>
          <div className="inline-flex items-center gap-2 rounded-xl bg-[color:var(--color-surface-elevated)] px-4 py-3 text-[color:var(--color-text-primary)]">
            <Bot className="size-4 text-[color:var(--color-text-label)]" />
            <span className="text-base font-medium">{trigger.agent_name}</span>
          </div>
        </div>
      </div>
    </section>
  );
}

export function AutomationDetailPanel({
  emptyState,
  error,
  isDeleting,
  isLoading,
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
}: AutomationDetailPanelProps) {
  if (isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="automation-detail-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-6 text-sm text-[color:var(--color-danger)]"
        data-testid="automation-detail-error"
      >
        Failed to load automation details
      </div>
    );
  }

  if (!item) {
    if (emptyState) {
      return <EmptyState {...emptyState} />;
    }

    return (
      <div
        className="flex flex-1 items-center justify-center px-6 text-sm text-[color:var(--color-text-secondary)]"
        data-testid="automation-detail-empty"
      >
        Select an automation item to inspect its configuration and run history.
      </div>
    );
  }

  const isJob = kind === "jobs";
  const isDynamic = item.source === "dynamic";
  const trigger = isJob ? null : (item as AutomationTrigger);
  const job = isJob ? (item as AutomationJob) : null;

  return (
    <section
      className="flex flex-1 flex-col overflow-y-auto px-6 py-5"
      data-testid="automation-detail-panel"
    >
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-[1.75rem] font-semibold tracking-[-0.03em] text-[color:var(--color-text-primary)]">
                {item.name}
              </h2>
              <AutomationTag tone={automationSemanticTone(item.enabled ? "enabled" : "disabled")}>
                {item.enabled ? "ENABLED" : "DISABLED"}
              </AutomationTag>
              <AutomationTag tone={automationSourceTone(item.source)}>
                {automationSourceLabel(item.source)}
              </AutomationTag>
              {item.source === "config" ? (
                <Lock className="size-4 text-[color:var(--color-text-label)]" />
              ) : null}
            </div>

            <p className="text-sm text-[color:var(--color-text-secondary)]">
              {`Agent: ${item.agent_name} · Scope: ${item.scope} · Updated ${formatDate(item.updated_at)}`}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button
              className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
              data-testid="toggle-automation-btn"
              disabled={isTogglePending}
              onClick={() => onToggleEnabled(!item.enabled)}
              size="lg"
              type="button"
              variant="outline"
            >
              {isTogglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
            </Button>
            {isJob ? (
              <Button
                data-testid="trigger-job-btn"
                disabled={isTriggerPending}
                onClick={onTriggerNow}
                size="lg"
                type="button"
              >
                <Play className="size-4" />
                {isTriggerPending ? "Queuing..." : "Run now"}
              </Button>
            ) : null}
            {isDynamic ? (
              <Button
                className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
                data-testid="edit-automation-btn"
                onClick={onEdit}
                size="lg"
                type="button"
                variant="outline"
              >
                <Pencil className="size-4" />
                Edit
              </Button>
            ) : null}
            {isDynamic ? (
              <Button
                className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-danger)] hover:bg-[color:var(--color-danger-tint)] hover:text-[color:var(--color-danger)]"
                data-testid="delete-automation-btn"
                disabled={isDeleting}
                onClick={onDelete}
                size="lg"
                type="button"
                variant="outline"
              >
                <Trash2 className="size-4" />
                {isDeleting ? "Deleting..." : "Delete"}
              </Button>
            ) : null}
          </div>
        </div>

        {!isDynamic ? (
          <div className="flex items-start gap-3 rounded-xl border border-dashed border-[color:var(--color-divider)] px-4 py-3 text-sm text-[color:var(--color-text-secondary)]">
            <Lock className="mt-0.5 size-4 shrink-0 text-[color:var(--color-text-label)]" />
            <p>
              This automation is defined in configuration files. Only the enabled state can be
              toggled from the UI.
            </p>
          </div>
        ) : null}

        {job ? <JobScheduleCard job={job} /> : null}
        {trigger ? <TriggerActivationCard trigger={trigger} /> : null}

        <section className="space-y-4">
          <div className="flex items-center gap-2">
            <SectionEyebrow>{isJob ? "PROMPT" : "PROMPT TEMPLATE"}</SectionEyebrow>
            {!isJob ? <AutomationTag tone="violet">GO TEMPLATE</AutomationTag> : null}
          </div>
          <pre className="whitespace-pre-wrap rounded-xl bg-[color:var(--color-surface)] px-4 py-4 font-mono text-sm leading-7 text-[color:var(--color-text-secondary)]">
            {item.prompt}
          </pre>
          <div className="flex flex-wrap items-center gap-3">
            <MetaChip icon={RefreshCw}>{describeRetry(item.retry)}</MetaChip>
            <MetaChip icon={Zap}>{describeFireLimit(item.fire_limit)}</MetaChip>
            <MetaChip>{automationScopeLabel(item.scope)}</MetaChip>
          </div>
        </section>

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

        {trigger?.webhook_id ? (
          <div className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4">
            <div className="grid gap-3 md:grid-cols-3">
              <div>
                <SectionEyebrow>Event</SectionEyebrow>
                <p className="mt-2 text-sm text-[color:var(--color-text-primary)]">
                  {trigger.event}
                </p>
              </div>
              <div>
                <SectionEyebrow>Endpoint</SectionEyebrow>
                <p className="mt-2 text-sm text-[color:var(--color-text-primary)]">
                  {trigger.endpoint_slug ?? "Unavailable"}
                </p>
              </div>
              <div>
                <SectionEyebrow>Webhook id</SectionEyebrow>
                <p className="mt-2 text-sm text-[color:var(--color-text-primary)]">
                  {trigger.webhook_id}
                </p>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </section>
  );
}
