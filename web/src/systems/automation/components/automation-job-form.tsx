import { Bot, Box, CalendarCheck, Check, Clock, Info, Repeat } from "lucide-react";

import {
  Button,
  DialogFooter,
  Field,
  FieldLabel,
  Input,
  PillGroup,
  type PillGroupItem,
} from "@agh/ui";

import { useAutomationJobForm } from "../hooks/use-automation-job-form";
import type { WorkspaceOption } from "../lib/trigger-preview";
import type {
  AutomationFireLimit,
  AutomationRetry,
  AutomationScheduleMode,
  CreateAutomationJobRequest,
} from "../types";
import { AgentRunStep } from "./job-form/agent-run-step";
import { CronBuilder } from "./job-form/cron-builder";
import { OutputMode } from "./job-form/output-mode";
import { JobPreview } from "./job-form/preview/job-preview";
import { ReliabilitySection } from "./job-form/reliability-section";
import { ScheduleAt } from "./job-form/schedule-at";
import { ScheduleEvery } from "./job-form/schedule-every";
import { ScopeStep } from "./job-form/scope-step";
import { TaskRunStep } from "./job-form/task-run-step";
import { FlowStep } from "./trigger-form/flow-step";

interface AutomationJobFormProps {
  activeWorkspaceId?: string | null;
  draft: CreateAutomationJobRequest;
  isPending: boolean;
  mode: "create" | "edit";
  onCancel: () => void;
  onChange: (draft: CreateAutomationJobRequest) => void;
  onSubmit: () => void;
  /** Workspaces selectable for a workspace-scoped job. */
  workspaces?: ReadonlyArray<WorkspaceOption>;
  /** Known agent names; falls back to a free-text agent input when empty. */
  agents?: string[];
}

const EMPTY_AGENTS: string[] = [];

const SCHEDULE_MODE_ITEMS: PillGroupItem<AutomationScheduleMode>[] = [
  {
    value: "cron",
    label: (
      <span className="flex items-center gap-1.5">
        <Repeat aria-hidden="true" className="size-3" />
        Repeat on cron
      </span>
    ),
    testId: "job-schedule-mode-cron",
  },
  {
    value: "every",
    label: (
      <span className="flex items-center gap-1.5">
        <Clock aria-hidden="true" className="size-3" />
        Every interval
      </span>
    ),
    testId: "job-schedule-mode-every",
  },
  {
    value: "at",
    label: (
      <span className="flex items-center gap-1.5">
        <CalendarCheck aria-hidden="true" className="size-3" />
        Once at a time
      </span>
    ),
    testId: "job-schedule-mode-at",
  },
];

/** Job-level reliability badge mirroring the artboard's collapsible summary. */
function reliabilityBadge(
  retry: AutomationRetry,
  fireLimit: AutomationFireLimit | undefined,
  enabled: boolean,
  locked: boolean
): string {
  const retryLabel =
    locked || retry.strategy === "none" ? "No retry" : `Backoff ×${retry.max_retries}`;
  const limit = fireLimit ?? { max: 12, window: "1h" };
  return `${retryLabel} · ${limit.max}/${limit.window} · ${enabled ? "enabled" : "disabled"}`;
}

export function AutomationJobForm({
  activeWorkspaceId,
  draft,
  isPending,
  mode,
  onCancel,
  onChange,
  onSubmit,
  workspaces,
  agents = EMPTY_AGENTS,
}: AutomationJobFormProps) {
  const form = useAutomationJobForm({
    activeWorkspaceId,
    draft,
    isPending,
    mode,
    onChange,
    onSubmit,
    workspaces,
    agents,
  });

  return (
    <form
      className="flex min-h-0 flex-col"
      data-testid="automation-job-form"
      onSubmit={form.handleSubmit}
    >
      <div className="grid min-h-0 flex-1 grid-cols-[1fr_var(--width-right-rail-default)] max-lg:grid-cols-1 max-lg:grid-rows-[minmax(0,1fr)_auto]">
        <section className="min-h-0 overflow-y-auto px-6 py-5">
          <Field className="mb-1">
            <FieldLabel htmlFor="job-name">Job name</FieldLabel>
            <Input
              className="font-mono"
              data-testid="job-name-input"
              id="job-name"
              onChange={event => form.onName(event.target.value)}
              placeholder="daily-code-review"
              value={draft.name}
            />
          </Field>

          <div className="mt-5">
            <FlowStep
              active
              icon={Box}
              kicker="For"
              subtitle="Where the job lives. A global job runs against the runtime; a workspace job runs in one project."
              title="A workspace, or the whole runtime"
            >
              <ScopeStep
                onScopeChange={form.onScopeChange}
                onWorkspaceChange={form.onWorkspaceChange}
                scope={draft.scope}
                workspaceId={draft.workspace_id}
                workspaces={form.resolvedWorkspaces}
              />
            </FlowStep>

            <FlowStep
              active
              icon={Bot}
              kicker="Run"
              subtitle="Prompt an agent directly, or hand the work to a durable task that carries its own owner."
              title="What fires on each tick"
            >
              <div className="space-y-4">
                <OutputMode onOutputChange={form.onOutputChange} output={form.output} />
                {form.output === "task" && draft.task ? (
                  <TaskRunStep
                    jobName={draft.name}
                    onOwnerKind={form.onOwnerKind}
                    onOwnerRef={form.onOwnerRef}
                    onTaskChannel={form.onTaskChannel}
                    onTaskDescription={form.onTaskDescription}
                    onTaskTitle={form.onTaskTitle}
                    task={draft.task}
                  />
                ) : (
                  <AgentRunStep
                    agent={draft.agent_name}
                    agents={form.agents}
                    onAgentChange={form.onAgentChange}
                    onPromptChange={form.onPromptChange}
                    prompt={draft.prompt}
                  />
                )}
              </div>
            </FlowStep>

            <FlowStep
              active
              icon={Clock}
              kicker="When"
              last
              subtitle="All times evaluate in UTC, the runtime's automation timezone."
              title="On this schedule"
            >
              <div className="space-y-4">
                <PillGroup
                  aria-label="Schedule mode"
                  items={SCHEDULE_MODE_ITEMS}
                  onChange={form.onScheduleMode}
                  value={form.scheduleMode}
                />
                {form.scheduleMode === "cron" ? (
                  <CronBuilder
                    expr={draft.schedule.expr ?? ""}
                    model={form.cronModel}
                    onDailyTime={form.onDailyTime}
                    onEveryMinutes={form.onEveryMinutes}
                    onExpr={form.onCronExpr}
                    onFrequency={form.onCronFrequency}
                    onHourlyMinute={form.onHourlyMinute}
                    onMonthDay={form.onMonthDay}
                    onMonthlyTime={form.onMonthlyTime}
                    onPreset={form.onCronPreset}
                    onToggleWeekday={form.onToggleWeekday}
                    onWeeklyTime={form.onWeeklyTime}
                    onWeekdayPreset={form.onWeekdayPreset}
                    readout={form.preview.scheduleReadout}
                    valid={form.preview.scheduleValid}
                  />
                ) : null}
                {form.scheduleMode === "every" ? (
                  <ScheduleEvery
                    interval={draft.schedule.interval ?? ""}
                    onInterval={form.onEveryInterval}
                    onPreset={form.onEveryPreset}
                    readout={form.preview.scheduleReadout}
                    valid={form.preview.scheduleValid}
                  />
                ) : null}
                {form.scheduleMode === "at" ? (
                  <ScheduleAt
                    onTime={form.onAtTime}
                    readout={form.preview.scheduleReadout}
                    time={draft.schedule.time ?? ""}
                    valid={form.preview.scheduleValid}
                  />
                ) : null}
              </div>
            </FlowStep>
          </div>

          <ReliabilitySection
            badge={reliabilityBadge(
              form.retry,
              draft.fire_limit ?? undefined,
              draft.enabled ?? true,
              form.output === "task"
            )}
            defaultOpen={form.reliabilityDefaultOpen}
            enabled={draft.enabled ?? true}
            fireLimit={draft.fire_limit ?? undefined}
            locked={form.output === "task"}
            mode={mode}
            onEnabledChange={form.onEnabledChange}
            onFireLimitChange={form.onFireLimitChange}
            onRetryChange={form.onRetryChange}
            retry={form.retry}
          />
        </section>

        <JobPreview preview={form.preview} />
      </div>

      <DialogFooter variant="ruled">
        <div className="flex flex-1 items-center gap-2 text-form-hint text-subtle">
          <Info aria-hidden="true" className="size-3 shrink-0 text-faint" />
          <span>
            Created as a <b className="font-medium text-muted">dynamic</b> job: editable,
            disable-able, and deletable anytime.
          </span>
        </div>
        <Button onClick={onCancel} type="button" variant="outline">
          Cancel
        </Button>
        <Button
          className="min-w-32"
          data-testid="submit-job-form"
          disabled={!form.canSubmit || isPending}
          type="submit"
        >
          {isPending ? (
            "Saving..."
          ) : mode === "create" ? (
            <>
              <Check aria-hidden="true" className="size-4" />
              Create job
            </>
          ) : (
            "Save changes"
          )}
        </Button>
      </DialogFooter>
    </form>
  );
}
