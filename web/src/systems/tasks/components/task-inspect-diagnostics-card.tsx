import { Activity, AlertCircle, CheckCircle2, Clock, PauseCircle, Terminal } from "lucide-react";
import type { ReactNode } from "react";

import { BlockLoading, Empty, MonoId, Pill, PillDot, Section, Time, type PillTone } from "@agh/ui";

import { taskRunStatusLabel, taskRunStatusTone } from "../lib/task-formatters";
import type { TaskInspectView } from "../types";

export interface TaskInspectDiagnosticsCardProps {
  inspect: TaskInspectView | null;
  isLoading?: boolean;
  errorMessage?: string | null;
  label?: string;
  testId?: string;
}

type TaskInspectDiagnostic = NonNullable<TaskInspectView["diagnostics"]>[number];

const NEXT_ACTION_TONE: Record<string, PillTone> = {
  claim_available: "success",
  waiting_for_session: "info",
  stranded: "warning",
  running: "accent",
  recovery_required: "danger",
  terminal: "neutral",
};

const SEVERITY_TONE: Record<string, PillTone> = {
  ok: "success",
  info: "info",
  warn: "warning",
  error: "danger",
  critical: "danger",
};

function toneForNextAction(nextAction?: string | null): PillTone {
  if (!nextAction) {
    return "neutral";
  }
  return NEXT_ACTION_TONE[nextAction] ?? "neutral";
}

function toneForSeverity(severity?: string | null): PillTone {
  if (!severity) {
    return "neutral";
  }
  return SEVERITY_TONE[severity] ?? "neutral";
}

function labelForValue(value?: string | null): string {
  if (!value) {
    return "--";
  }
  return value.replaceAll("_", " ");
}

function diagnosticEvidenceEntries(
  evidence: TaskInspectDiagnostic["evidence"]
): Array<[string, string]> {
  if (!evidence || typeof evidence !== "object") {
    return [];
  }

  return Object.entries(evidence)
    .flatMap(([key, value]) => {
      if (typeof value === "string") {
        return value.trim() ? ([[key, value]] satisfies Array<[string, string]>) : [];
      }
      if (typeof value === "number" || typeof value === "boolean") {
        return [[key, String(value)]] satisfies Array<[string, string]>;
      }
      return [];
    })
    .slice(0, 4);
}

function SummaryItem({
  label,
  children,
  testId,
}: {
  label: string;
  children: ReactNode;
  testId: string;
}) {
  return (
    <div
      className="flex min-h-16 flex-col justify-between rounded-md border border-line-soft bg-input-fill px-3 py-2.5"
      data-testid={testId}
    >
      <span className="text-eyebrow font-medium tracking-eyebrow text-subtle">{label}</span>
      <div className="mt-2 min-w-0 text-small-body text-fg-strong">{children}</div>
    </div>
  );
}

function InspectSummary({ inspect, testId }: { inspect: TaskInspectView; testId: string }) {
  const run = inspect.current_run ?? null;
  const session = inspect.bound_session ?? null;
  const schedulerTone = inspect.scheduler.paused ? "warning" : "success";
  const schedulerLabel = inspect.scheduler.paused ? "paused" : "active";

  return (
    <div className="grid gap-3 md:grid-cols-4" data-testid={`${testId}-summary`}>
      <SummaryItem label="Next action" testId={`${testId}-next-action`}>
        <Pill tone={toneForNextAction(inspect.next_action)}>
          <PillDot />
          {labelForValue(inspect.next_action)}
        </Pill>
      </SummaryItem>
      <SummaryItem label="Current run" testId={`${testId}-current-run`}>
        {run ? (
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <MonoId size="sm" value={run.run_id} />
            <Pill tone={taskRunStatusTone(run.status)}>{taskRunStatusLabel(run.status)}</Pill>
          </span>
        ) : (
          <span className="text-muted">No current run</span>
        )}
      </SummaryItem>
      <SummaryItem label="Bound session" testId={`${testId}-bound-session`}>
        {session ? (
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <MonoId size="sm" value={session.session_id} />
            {session.state ? <Pill tone="info">{session.state}</Pill> : null}
          </span>
        ) : (
          <span className="text-muted">No bound session</span>
        )}
      </SummaryItem>
      <SummaryItem label="Scheduler" testId={`${testId}-scheduler`}>
        <Pill tone={schedulerTone}>
          {inspect.scheduler.paused ? (
            <PauseCircle className="size-3" />
          ) : (
            <Activity className="size-3" />
          )}
          {schedulerLabel}
        </Pill>
      </SummaryItem>
    </div>
  );
}

function DiagnosticRow({
  diagnostic,
  testId,
}: {
  diagnostic: TaskInspectDiagnostic;
  testId: string;
}) {
  const tone = toneForSeverity(diagnostic.severity);
  const evidenceEntries = diagnosticEvidenceEntries(diagnostic.evidence);

  return (
    <article className="rounded-md border border-line-soft bg-input-fill p-3" data-testid={testId}>
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="flex min-w-0 flex-wrap items-center gap-1.5">
            <Pill tone={tone}>
              <PillDot />
              {labelForValue(diagnostic.severity)}
            </Pill>
            <Pill mono tone="neutral">
              {diagnostic.code}
            </Pill>
            {diagnostic.data_freshness ? (
              <Pill tone="info">{labelForValue(diagnostic.data_freshness)}</Pill>
            ) : null}
          </div>
          <h3 className="text-card-title font-medium text-fg-strong">{diagnostic.title}</h3>
          <p className="max-w-4xl text-small-body leading-relaxed text-muted">
            {diagnostic.message}
          </p>
        </div>
        <MonoId size="sm" value={diagnostic.id} />
      </div>

      {diagnostic.suggested_command ? (
        <div
          className="mt-3 flex min-w-0 items-start gap-2 rounded-sm border border-line-soft bg-canvas-soft px-2.5 py-2"
          data-testid={`${testId}-command`}
        >
          <Terminal className="mt-0.5 size-3.5 shrink-0 text-accent" aria-hidden="true" />
          <code className="min-w-0 break-all font-mono text-mono-id tracking-mono-id text-fg">
            {diagnostic.suggested_command}
          </code>
        </div>
      ) : null}

      {evidenceEntries.length > 0 ? (
        <dl className="mt-3 grid gap-2 md:grid-cols-2" data-testid={`${testId}-evidence`}>
          {evidenceEntries.map(([key, value]) => (
            <div className="min-w-0" key={key}>
              <dt className="text-eyebrow font-medium tracking-eyebrow text-subtle">
                {labelForValue(key)}
              </dt>
              <dd className="mt-1 truncate font-mono text-mono-id tracking-mono-id text-muted">
                {value}
              </dd>
            </div>
          ))}
        </dl>
      ) : null}
    </article>
  );
}

export function TaskInspectDiagnosticsCard({
  inspect,
  isLoading = false,
  errorMessage = null,
  label = "Inspect diagnostics",
  testId = "task-inspect-diagnostics-card",
}: TaskInspectDiagnosticsCardProps) {
  if (isLoading && !inspect) {
    return (
      <Section
        aria-label={label}
        bodyClassName="gap-4"
        className="w-full gap-4"
        data-testid={`${testId}-loading-section`}
        icon={Clock}
        label={label}
      >
        <BlockLoading
          data-testid={`${testId}-loading`}
          label="Loading inspect snapshot"
          size="sm"
          surface="bare"
        />
      </Section>
    );
  }

  if (errorMessage && !inspect) {
    return (
      <Section
        aria-label={label}
        bodyClassName="gap-4"
        className="w-full gap-4"
        data-testid={testId}
        icon={AlertCircle}
        label={label}
      >
        <Empty
          data-testid={`${testId}-error`}
          description={errorMessage}
          icon={AlertCircle}
          title="Unable to load diagnostics"
        />
      </Section>
    );
  }

  if (!inspect) {
    return (
      <Section
        aria-label={label}
        bodyClassName="gap-4"
        className="w-full gap-4"
        data-testid={testId}
        icon={Clock}
        label={label}
      >
        <Empty
          data-testid={`${testId}-empty`}
          description="No inspect snapshot is available for this task state."
          icon={Clock}
          title="No inspect snapshot"
        />
      </Section>
    );
  }

  const diagnostics = inspect.diagnostics ?? [];

  return (
    <Section
      aria-label={label}
      bodyClassName="gap-4"
      className="w-full gap-4"
      count={diagnostics.length}
      data-testid={testId}
      icon={diagnostics.length > 0 ? AlertCircle : CheckCircle2}
      label={label}
      right={
        <span className="inline-flex items-center gap-1.5 text-small-body text-subtle">
          as of <Time iso={inspect.as_of} mode="relative" />
        </span>
      }
    >
      <InspectSummary inspect={inspect} testId={testId} />

      {diagnostics.length === 0 ? (
        <Empty
          data-testid={`${testId}-no-diagnostics`}
          description="The inspect snapshot did not find recovery diagnostics for the current run."
          icon={CheckCircle2}
          title="No diagnostics"
        />
      ) : (
        <div className="flex flex-col gap-3" data-testid={`${testId}-items`}>
          {diagnostics.map(diagnostic => (
            <DiagnosticRow
              diagnostic={diagnostic}
              key={diagnostic.id}
              testId={`${testId}-item-${diagnostic.code}`}
            />
          ))}
        </div>
      )}
    </Section>
  );
}
