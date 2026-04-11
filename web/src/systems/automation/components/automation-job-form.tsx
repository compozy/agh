import type { FormEvent } from "react";

import { PillButton } from "@/components/design-system";

import {
  AutomationCheckbox,
  AutomationField,
  AutomationFormSection,
  AutomationInput,
  AutomationSelect,
  AutomationTextarea,
} from "./automation-form-primitives";
import type { CreateAutomationJobRequest } from "../types";

interface AutomationJobFormProps {
  activeWorkspaceId?: string | null;
  draft: CreateAutomationJobRequest;
  isPending: boolean;
  mode: "create" | "edit";
  onCancel: () => void;
  onChange: (draft: CreateAutomationJobRequest) => void;
  onSubmit: () => void;
}

function currentSchedule(draft: CreateAutomationJobRequest) {
  return draft.schedule ?? { mode: "cron" as const, expr: "0 9 * * *" };
}

export function AutomationJobForm({
  activeWorkspaceId,
  draft,
  isPending,
  mode,
  onCancel,
  onChange,
  onSubmit,
}: AutomationJobFormProps) {
  const schedule = currentSchedule(draft);
  const canSubmit =
    draft.name.trim() !== "" &&
    draft.agent_name.trim() !== "" &&
    draft.prompt.trim() !== "" &&
    (draft.scope === "global" || Boolean(draft.workspace_id)) &&
    ((schedule.mode === "cron" && Boolean(schedule.expr?.trim())) ||
      (schedule.mode === "every" && Boolean(schedule.interval?.trim())) ||
      (schedule.mode === "at" && Boolean(schedule.time?.trim())));

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canSubmit || isPending) {
      return;
    }
    onSubmit();
  };

  return (
    <form
      className="flex flex-1 flex-col gap-5 overflow-y-auto p-6"
      data-testid="automation-job-form"
      onSubmit={handleSubmit}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h2 className="text-base font-semibold text-[color:var(--color-text-primary)]">
            {mode === "create" ? "Create job" : "Edit job"}
          </h2>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            Scheduled automations dispatch an agent prompt on a time-based cadence.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="inline-flex h-9 items-center rounded-lg border border-[color:var(--color-divider)] px-4 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
            onClick={onCancel}
            type="button"
          >
            Cancel
          </button>
          <button
            className="inline-flex h-9 items-center rounded-lg bg-[color:var(--color-accent)] px-4 text-sm font-medium text-[color:var(--color-accent-ink)] transition-colors hover:bg-[color:var(--color-accent-hover)] disabled:cursor-not-allowed disabled:bg-[color:var(--color-disabled)] disabled:text-[color:var(--color-text-tertiary)]"
            data-testid="submit-job-form"
            disabled={!canSubmit || isPending}
            type="submit"
          >
            {isPending ? "Saving..." : mode === "create" ? "Create job" : "Save changes"}
          </button>
        </div>
      </div>

      <AutomationFormSection
        description="Name the job, pick the target agent, and define the prompt it should execute."
        title="Core"
      >
        <div className="grid gap-4 md:grid-cols-2">
          <AutomationField label="Name">
            <AutomationInput
              data-testid="job-name-input"
              onChange={event => onChange({ ...draft, name: event.target.value })}
              placeholder="daily-standup"
              value={draft.name}
            />
          </AutomationField>
          <AutomationField label="Agent">
            <AutomationInput
              data-testid="job-agent-input"
              onChange={event => onChange({ ...draft, agent_name: event.target.value })}
              placeholder="coder"
              value={draft.agent_name}
            />
          </AutomationField>
        </div>
        <AutomationField label="Prompt">
          <AutomationTextarea
            data-testid="job-prompt-input"
            onChange={event => onChange({ ...draft, prompt: event.target.value })}
            placeholder="Summarize the latest commits and open review actions."
            value={draft.prompt}
          />
        </AutomationField>
        <div className="space-y-2">
          <span className="text-sm font-medium text-[color:var(--color-text-primary)]">Scope</span>
          <div className="flex items-center gap-2">
            <PillButton
              active={draft.scope === "global"}
              data-testid="job-scope-global"
              onClick={() =>
                onChange({
                  ...draft,
                  scope: "global",
                  workspace_id: undefined,
                })
              }
            >
              GLOBAL
            </PillButton>
            <PillButton
              active={draft.scope === "workspace"}
              data-testid="job-scope-workspace"
              onClick={() =>
                onChange({
                  ...draft,
                  scope: "workspace",
                  workspace_id: activeWorkspaceId ?? draft.workspace_id,
                })
              }
            >
              WORKSPACE
            </PillButton>
          </div>
          {draft.scope === "workspace" ? (
            <p className="text-xs text-[color:var(--color-text-secondary)]">
              {draft.workspace_id
                ? `Bound to workspace ${draft.workspace_id}.`
                : "Select an active workspace before saving a workspace-scoped job."}
            </p>
          ) : null}
        </div>
      </AutomationFormSection>

      <AutomationFormSection
        description="Choose how often the automation should fire."
        title="Schedule"
      >
        <div className="grid gap-4 md:grid-cols-2">
          <AutomationField label="Mode">
            <AutomationSelect
              data-testid="job-schedule-mode"
              onChange={event => {
                const mode = event.target.value as CreateAutomationJobRequest["schedule"]["mode"];
                onChange({
                  ...draft,
                  schedule:
                    mode === "cron"
                      ? { mode, expr: schedule.expr ?? "0 9 * * *" }
                      : mode === "every"
                        ? { mode, interval: schedule.interval ?? "30m" }
                        : { mode, time: schedule.time ?? new Date().toISOString() },
                });
              }}
              value={schedule.mode}
            >
              <option value="cron">Cron</option>
              <option value="every">Every interval</option>
              <option value="at">One-shot</option>
            </AutomationSelect>
          </AutomationField>
          <AutomationField
            hint={
              schedule.mode === "cron"
                ? "e.g. 0 9 * * *"
                : schedule.mode === "every"
                  ? "e.g. 30m"
                  : "UTC timestamp"
            }
            label={
              schedule.mode === "cron"
                ? "Cron expression"
                : schedule.mode === "every"
                  ? "Interval"
                  : "Run at"
            }
          >
            {schedule.mode === "cron" ? (
              <AutomationInput
                data-testid="job-schedule-expr"
                onChange={event =>
                  onChange({
                    ...draft,
                    schedule: { mode: "cron", expr: event.target.value },
                  })
                }
                placeholder="0 9 * * *"
                value={schedule.expr ?? ""}
              />
            ) : schedule.mode === "every" ? (
              <AutomationInput
                data-testid="job-schedule-interval"
                onChange={event =>
                  onChange({
                    ...draft,
                    schedule: { mode: "every", interval: event.target.value },
                  })
                }
                placeholder="30m"
                value={schedule.interval ?? ""}
              />
            ) : (
              <AutomationInput
                data-testid="job-schedule-time"
                onChange={event =>
                  onChange({
                    ...draft,
                    schedule: { mode: "at", time: event.target.value },
                  })
                }
                placeholder="2026-04-15T15:00:00Z"
                value={schedule.time ?? ""}
              />
            )}
          </AutomationField>
        </div>
      </AutomationFormSection>

      <AutomationFormSection
        description="Retries and fire limits protect the daemon from noisy automations."
        title="Governance"
      >
        <div className="grid gap-4 md:grid-cols-3">
          <AutomationField label="Retry strategy">
            <AutomationSelect
              data-testid="job-retry-strategy"
              onChange={event =>
                onChange({
                  ...draft,
                  retry: {
                    ...(draft.retry ?? { base_delay: "2s", max_retries: 3 }),
                    strategy: event.target.value as "none" | "backoff",
                  },
                })
              }
              value={draft.retry?.strategy ?? "none"}
            >
              <option value="none">None</option>
              <option value="backoff">Backoff</option>
            </AutomationSelect>
          </AutomationField>
          <AutomationField label="Max retries">
            <AutomationInput
              data-testid="job-retry-max"
              min={0}
              onChange={event =>
                onChange({
                  ...draft,
                  retry: {
                    ...(draft.retry ?? { strategy: "none", base_delay: "2s" }),
                    max_retries: Number(event.target.value || "0"),
                  },
                })
              }
              type="number"
              value={draft.retry?.max_retries ?? 3}
            />
          </AutomationField>
          <AutomationField label="Base delay">
            <AutomationInput
              data-testid="job-retry-delay"
              onChange={event =>
                onChange({
                  ...draft,
                  retry: {
                    ...(draft.retry ?? { strategy: "none", max_retries: 3 }),
                    base_delay: event.target.value,
                  },
                })
              }
              placeholder="2s"
              value={draft.retry?.base_delay ?? "2s"}
            />
          </AutomationField>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <AutomationField label="Fire limit">
            <AutomationInput
              data-testid="job-fire-limit-max"
              min={1}
              onChange={event =>
                onChange({
                  ...draft,
                  fire_limit: {
                    ...(draft.fire_limit ?? { window: "1h" }),
                    max: Number(event.target.value || "1"),
                  },
                })
              }
              type="number"
              value={draft.fire_limit?.max ?? 12}
            />
          </AutomationField>
          <AutomationField label="Limit window">
            <AutomationInput
              data-testid="job-fire-limit-window"
              onChange={event =>
                onChange({
                  ...draft,
                  fire_limit: {
                    ...(draft.fire_limit ?? { max: 12 }),
                    window: event.target.value,
                  },
                })
              }
              placeholder="1h"
              value={draft.fire_limit?.window ?? "1h"}
            />
          </AutomationField>
        </div>
        <AutomationCheckbox
          checked={draft.enabled ?? true}
          description="Disabled jobs stay visible in the dashboard but will not fire until re-enabled."
          label="Job enabled"
          onCheckedChange={checked => onChange({ ...draft, enabled: checked })}
          testId="job-enabled-toggle"
        />
      </AutomationFormSection>
    </form>
  );
}
