import { Link } from "@tanstack/react-router";

import { CodeBlock, MetadataList, Metric, Pill, Section } from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import { taskRunStatusTone } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

export interface TaskRunIdentityPanelProps {
  run: TaskRunDetailView;
}

function normalizeSessionText(value?: string | null): string {
  return typeof value === "string" ? value.trim() : "";
}

export function TaskRunIdentityPanel({ run }: TaskRunIdentityPanelProps) {
  const record = run.run;
  const session = run.session;
  const linkedSessionID = normalizeSessionText(session?.session_id ?? record.session_id);
  const linkedSessionAgent = normalizeSessionText(session?.agent_name);

  return (
    <Section aria-label="Run identity" data-testid="task-run-detail-identity" label="Run identity">
      <div className="rounded-(--radius-diagram) border border-(--line) bg-(--canvas-soft) px-4 py-3">
        <MetadataList className="gap-y-2">
          <MetadataList.Row
            label="Run ID"
            valueProps={{ "data-testid": "task-run-detail-identity-run" }}
          >
            <Pill mono>{record.id}</Pill>
          </MetadataList.Row>
          <MetadataList.Row label="Status">
            <Pill tone={pillToneFromLegacyTone(taskRunStatusTone(record.status))}>
              {record.status}
            </Pill>
          </MetadataList.Row>
          <MetadataList.Row
            label="Attempt"
            valueProps={{ "data-testid": "task-run-detail-identity-attempt" }}
          >
            {record.attempt}
          </MetadataList.Row>
          {record.idempotency_key ? (
            <MetadataList.Row
              label="Idempotency"
              valueProps={{ "data-testid": "task-run-detail-identity-idempotency" }}
            >
              <Pill mono>{record.idempotency_key}</Pill>
            </MetadataList.Row>
          ) : null}
          {record.claimed_by?.ref ? (
            <MetadataList.Row
              label="Claimed by"
              valueProps={{ "data-testid": "task-run-detail-identity-claimed-by" }}
            >
              {record.claimed_by.ref}
            </MetadataList.Row>
          ) : null}
          <MetadataList.Row label="Session">
            {linkedSessionID && linkedSessionAgent ? (
              <Pill.Link
                data-testid="task-run-detail-session-link"
                render={
                  <Link
                    params={{ name: linkedSessionAgent, id: linkedSessionID }}
                    to="/agents/$name/sessions/$id"
                  />
                }
              >
                {linkedSessionID}
              </Pill.Link>
            ) : linkedSessionID ? (
              <Pill.Link
                data-testid="task-run-detail-session-link"
                render={<Link params={{ id: linkedSessionID }} to="/session/$id" />}
              >
                {linkedSessionID}
              </Pill.Link>
            ) : (
              <span className="text-(--subtle)" data-testid="task-run-detail-session-missing">
                None
              </span>
            )}
          </MetadataList.Row>
        </MetadataList>
      </div>
    </Section>
  );
}

export interface TaskRunProgressPanelProps {
  run: TaskRunDetailView;
}

function formatCount(value?: number | null): string {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "--";
  }

  return value.toLocaleString();
}

function formatElapsed(startedAt?: string | null, endedAt?: string | null): string {
  if (!startedAt) {
    return "--";
  }

  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) {
    return "--";
  }

  const end = endedAt ? Date.parse(endedAt) : Date.now();
  if (Number.isNaN(end)) {
    return "--";
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

function formatCost(value?: number | null, currency?: string | null): string {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "--";
  }

  const formatted = value.toLocaleString(undefined, {
    maximumFractionDigits: 4,
    minimumFractionDigits: 2,
  });

  if (currency) {
    return `${currency} ${formatted}`;
  }

  return formatted;
}

export function TaskRunProgressPanel({ run }: TaskRunProgressPanelProps) {
  const summary = run.summary;
  const record = run.run;
  const elapsed = formatElapsed(record.started_at, record.ended_at);

  return (
    <Section aria-label="Run progress" data-testid="task-run-detail-progress" label="Progress">
      <div className="grid gap-3 sm:grid-cols-2">
        <Metric
          data-testid="task-run-detail-progress-tool-calls"
          label="Tool calls"
          value={formatCount(summary?.tool_call_count)}
        />
        <Metric
          data-testid="task-run-detail-progress-turns"
          label="Turns"
          value={formatCount(summary?.turn_count)}
        />
        <Metric
          data-testid="task-run-detail-progress-input-tokens"
          label="Input tokens"
          value={formatCount(summary?.input_tokens)}
        />
        <Metric
          data-testid="task-run-detail-progress-output-tokens"
          label="Output tokens"
          value={formatCount(summary?.output_tokens)}
        />
        <Metric
          data-testid="task-run-detail-progress-total-tokens"
          label="Total tokens"
          value={formatCount(summary?.total_tokens)}
        />
        <Metric data-testid="task-run-detail-progress-elapsed" label="Elapsed" value={elapsed} />
        <Metric
          className="sm:col-span-2"
          data-testid="task-run-detail-progress-cost"
          label="Cost"
          value={formatCost(summary?.total_cost, summary?.cost_currency)}
        />
      </div>
    </Section>
  );
}

export interface TaskRunActivityPanelProps {
  run: TaskRunDetailView;
}

export function TaskRunActivityPanel({ run }: TaskRunActivityPanelProps) {
  const summary = run.summary;
  const record = run.run;
  const lastEventType = summary?.last_event_type;
  const lastActivityAt = summary?.last_activity_at;
  const error = record.error;
  const result = record.result;

  return (
    <Section aria-label="Run activity" data-testid="task-run-detail-activity" label="Activity">
      <div className="flex flex-col gap-3 rounded-(--radius-diagram) border border-(--line) bg-(--canvas-soft) p-4">
        <MetadataList className="gap-y-2">
          {lastEventType ? (
            <MetadataList.Row label="Last event">
              <Pill mono data-testid="task-run-detail-activity-event">
                {lastEventType}
              </Pill>
            </MetadataList.Row>
          ) : null}
          {lastActivityAt ? (
            <MetadataList.Row
              label="Last activity"
              valueProps={{ "data-testid": "task-run-detail-activity-timestamp" }}
            >
              {lastActivityAt}
            </MetadataList.Row>
          ) : null}
        </MetadataList>
        {error ? (
          <div
            className="rounded-md border border-(--danger) bg-(--danger-tint) px-3 py-2"
            data-testid="task-run-detail-activity-error"
          >
            <p className="text-badge font-semibold text-(--danger)">Error</p>
            <p className="mt-1 text-small-body text-(--danger)">{error}</p>
          </div>
        ) : null}
        {result !== undefined && result !== null ? (
          <div data-testid="task-run-detail-activity-result">
            <p className="mb-2 text-badge font-semibold text-(--muted)">Result</p>
            <CodeBlock
              code={JSON.stringify(result, null, 2)}
              copyable={false}
              language="json"
              showPrompt={false}
            />
          </div>
        ) : null}
      </div>
    </Section>
  );
}
