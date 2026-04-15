import type { FormEvent } from "react";

import { PillButton } from "@/components/design-system";
import { Button } from "@agh/ui";

import {
  AutomationCheckbox,
  AutomationField,
  AutomationFormSection,
  AutomationInput,
  AutomationTextarea,
} from "./automation-form-primitives";
import { retryDraftForStrategy } from "../lib/automation-drafts";
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
  const retry = retryDraftForStrategy(draft.retry?.strategy ?? "none", draft.retry ?? undefined);
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
      className="flex max-h-[min(84vh,960px)] flex-col"
      data-testid="automation-job-form"
      onSubmit={handleSubmit}
    >
      <div className="border-b border-[color:var(--color-divider)] px-6 py-5">
        <div className="max-w-2xl space-y-1 pr-12">
          <h2 className="text-lg font-semibold text-[color:var(--color-text-primary)]">
            {mode === "create" ? "Create Job" : "Edit Job"}
          </h2>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            Scheduled jobs dispatch prompts to agents on a time-based cadence.
          </p>
        </div>
      </div>

      <div className="flex-1 space-y-5 overflow-y-auto px-6 py-5">
        <AutomationFormSection
          description="Name the job, choose the agent, and define the prompt it should execute."
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
            <span className="text-sm font-medium text-[color:var(--color-text-primary)]">
              Scope
            </span>
            <div className="flex flex-wrap items-center gap-2">
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
          description="Choose the cadence and execution time shape for this job."
          title="Schedule"
        >
          <div className="space-y-2">
            <span className="text-sm font-medium text-[color:var(--color-text-primary)]">Mode</span>
            <div className="flex flex-wrap items-center gap-2">
              <PillButton
                active={schedule.mode === "cron"}
                data-testid="job-schedule-mode-cron"
                onClick={() =>
                  onChange({
                    ...draft,
                    schedule: { mode: "cron", expr: schedule.expr ?? "0 9 * * *" },
                  })
                }
              >
                CRON
              </PillButton>
              <PillButton
                active={schedule.mode === "every"}
                data-testid="job-schedule-mode-every"
                onClick={() =>
                  onChange({
                    ...draft,
                    schedule: { mode: "every", interval: schedule.interval ?? "30m" },
                  })
                }
              >
                EVERY
              </PillButton>
              <PillButton
                active={schedule.mode === "at"}
                data-testid="job-schedule-mode-at"
                onClick={() =>
                  onChange({
                    ...draft,
                    schedule: { mode: "at", time: schedule.time ?? new Date().toISOString() },
                  })
                }
              >
                AT
              </PillButton>
            </div>
          </div>
          <AutomationField
            hint={
              schedule.mode === "cron"
                ? "Standard cron expression"
                : schedule.mode === "every"
                  ? "Duration such as 30m or 4h"
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
        </AutomationFormSection>

        <AutomationFormSection
          description="Retry and fire limits protect the daemon from noisy automation."
          title="Governance"
        >
          <div className="grid gap-4 md:grid-cols-3">
            <AutomationField label="Retry policy">
              <div className="flex flex-wrap items-center gap-2">
                <PillButton
                  active={retry.strategy === "none"}
                  data-testid="job-retry-strategy-none"
                  onClick={() =>
                    onChange({ ...draft, retry: retryDraftForStrategy("none", retry) })
                  }
                  size="dense"
                >
                  NONE
                </PillButton>
                <PillButton
                  active={retry.strategy === "backoff"}
                  data-testid="job-retry-strategy-backoff"
                  onClick={() =>
                    onChange({ ...draft, retry: retryDraftForStrategy("backoff", retry) })
                  }
                  size="dense"
                >
                  BACKOFF
                </PillButton>
              </div>
            </AutomationField>
            <AutomationField label="Max retries">
              <AutomationInput
                data-testid="job-retry-max"
                disabled={retry.strategy !== "backoff"}
                min={0}
                onChange={event =>
                  onChange({
                    ...draft,
                    retry: {
                      ...retryDraftForStrategy("backoff", retry),
                      max_retries: Number(event.target.value || "0"),
                    },
                  })
                }
                type="number"
                value={retry.strategy === "backoff" ? retry.max_retries : 0}
              />
            </AutomationField>
            <AutomationField label="Base delay">
              <AutomationInput
                data-testid="job-retry-delay"
                disabled={retry.strategy !== "backoff"}
                onChange={event =>
                  onChange({
                    ...draft,
                    retry: {
                      ...retryDraftForStrategy("backoff", retry),
                      base_delay: event.target.value,
                    },
                  })
                }
                placeholder="2s"
                value={retry.strategy === "backoff" ? retry.base_delay : ""}
              />
            </AutomationField>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <AutomationField label="Max fires">
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
            <AutomationField label="Window">
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
            description="Disabled jobs stay visible but never dispatch on their schedule."
            label={mode === "create" ? "Enabled on create" : "Enabled"}
            onCheckedChange={checked => onChange({ ...draft, enabled: checked })}
          />
        </AutomationFormSection>
      </div>

      <div className="flex items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-6 py-4">
        <Button
          className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
          onClick={onCancel}
          size="lg"
          type="button"
          variant="outline"
        >
          Cancel
        </Button>
        <Button
          data-testid="submit-job-form"
          disabled={!canSubmit || isPending}
          className="min-w-32"
          size="lg"
          type="submit"
        >
          {isPending ? "Saving..." : mode === "create" ? "Create Job" : "Save Changes"}
        </Button>
      </div>
    </form>
  );
}
