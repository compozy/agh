import type { FormEvent } from "react";

import {
  Button,
  DialogFooter,
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldTitle,
  Input,
  PillGroup,
  Section,
  Switch,
  Textarea,
} from "@agh/ui";

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
    if (!canSubmit || isPending) return;
    onSubmit();
  };

  return (
    <form
      className="flex max-h-[min(84vh,960px)] flex-col"
      data-testid="automation-job-form"
      onSubmit={handleSubmit}
    >
      <div className="flex-1 space-y-6 overflow-y-auto px-5 py-5">
        <Section label="Core">
          <div className="space-y-4 rounded-md border border-(--color-divider) bg-(--color-surface) p-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="job-name">Name</FieldLabel>
                <Input
                  data-testid="job-name-input"
                  id="job-name"
                  onChange={event => onChange({ ...draft, name: event.target.value })}
                  placeholder="daily-standup"
                  value={draft.name}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="job-agent">Agent</FieldLabel>
                <Input
                  data-testid="job-agent-input"
                  id="job-agent"
                  onChange={event => onChange({ ...draft, agent_name: event.target.value })}
                  placeholder="coder"
                  value={draft.agent_name}
                />
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="job-prompt">Prompt</FieldLabel>
              <Textarea
                data-testid="job-prompt-input"
                id="job-prompt"
                onChange={event => onChange({ ...draft, prompt: event.target.value })}
                placeholder="Summarize the latest commits and open review actions."
                rows={4}
                value={draft.prompt}
              />
            </Field>
            <Field>
              <FieldTitle>Scope</FieldTitle>
              <PillGroup
                aria-label="Scope"
                items={[
                  { value: "global", label: "GLOBAL", testId: "job-scope-global" },
                  { value: "workspace", label: "WORKSPACE", testId: "job-scope-workspace" },
                ]}
                onChange={next => {
                  if (next === "global") {
                    onChange({ ...draft, scope: "global", workspace_id: undefined });
                  } else {
                    onChange({
                      ...draft,
                      scope: "workspace",
                      workspace_id: activeWorkspaceId ?? draft.workspace_id,
                    });
                  }
                }}
                value={draft.scope}
              />
              {draft.scope === "workspace" ? (
                <FieldDescription>
                  {draft.workspace_id
                    ? `Bound to workspace ${draft.workspace_id}.`
                    : "Select an active workspace before saving a workspace-scoped job."}
                </FieldDescription>
              ) : null}
            </Field>
          </div>
        </Section>

        <Section label="Schedule">
          <div className="space-y-4 rounded-md border border-(--color-divider) bg-(--color-surface) p-4">
            <Field>
              <FieldTitle>Mode</FieldTitle>
              <PillGroup
                aria-label="Schedule mode"
                items={[
                  { value: "cron", label: "CRON", testId: "job-schedule-mode-cron" },
                  { value: "every", label: "EVERY", testId: "job-schedule-mode-every" },
                  { value: "at", label: "AT", testId: "job-schedule-mode-at" },
                ]}
                onChange={next => {
                  if (next === "cron") {
                    onChange({
                      ...draft,
                      schedule: { mode: "cron", expr: schedule.expr ?? "0 9 * * *" },
                    });
                  } else if (next === "every") {
                    onChange({
                      ...draft,
                      schedule: { mode: "every", interval: schedule.interval ?? "30m" },
                    });
                  } else {
                    onChange({
                      ...draft,
                      schedule: { mode: "at", time: schedule.time ?? new Date().toISOString() },
                    });
                  }
                }}
                value={schedule.mode}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="job-schedule-input">
                {schedule.mode === "cron"
                  ? "Cron expression"
                  : schedule.mode === "every"
                    ? "Interval"
                    : "Run at"}
              </FieldLabel>
              {schedule.mode === "cron" ? (
                <Input
                  data-testid="job-schedule-expr"
                  id="job-schedule-input"
                  onChange={event =>
                    onChange({ ...draft, schedule: { mode: "cron", expr: event.target.value } })
                  }
                  placeholder="0 9 * * *"
                  value={schedule.expr ?? ""}
                />
              ) : schedule.mode === "every" ? (
                <Input
                  data-testid="job-schedule-interval"
                  id="job-schedule-input"
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
                <Input
                  data-testid="job-schedule-time"
                  id="job-schedule-input"
                  onChange={event =>
                    onChange({ ...draft, schedule: { mode: "at", time: event.target.value } })
                  }
                  placeholder="2026-04-15T15:00:00Z"
                  value={schedule.time ?? ""}
                />
              )}
              <FieldDescription>
                {schedule.mode === "cron"
                  ? "Standard cron expression"
                  : schedule.mode === "every"
                    ? "Duration such as 30m or 4h"
                    : "UTC timestamp"}
              </FieldDescription>
            </Field>
          </div>
        </Section>

        <Section label="Governance">
          <div className="space-y-4 rounded-md border border-(--color-divider) bg-(--color-surface) p-4">
            <div className="grid gap-4 md:grid-cols-3">
              <Field>
                <FieldTitle>Retry policy</FieldTitle>
                <PillGroup
                  aria-label="Retry policy"
                  items={[
                    { value: "none", label: "NONE", testId: "job-retry-strategy-none" },
                    { value: "backoff", label: "BACKOFF", testId: "job-retry-strategy-backoff" },
                  ]}
                  onChange={next =>
                    onChange({ ...draft, retry: retryDraftForStrategy(next, retry) })
                  }
                  size="sm"
                  value={retry.strategy}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="job-retry-max">Max retries</FieldLabel>
                <Input
                  data-testid="job-retry-max"
                  disabled={retry.strategy !== "backoff"}
                  id="job-retry-max"
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
              </Field>
              <Field>
                <FieldLabel htmlFor="job-retry-delay">Base delay</FieldLabel>
                <Input
                  data-testid="job-retry-delay"
                  disabled={retry.strategy !== "backoff"}
                  id="job-retry-delay"
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
              </Field>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="job-fire-limit-max">Max fires</FieldLabel>
                <Input
                  data-testid="job-fire-limit-max"
                  id="job-fire-limit-max"
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
              </Field>
              <Field>
                <FieldLabel htmlFor="job-fire-limit-window">Window</FieldLabel>
                <Input
                  data-testid="job-fire-limit-window"
                  id="job-fire-limit-window"
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
              </Field>
            </div>
            <Field orientation="horizontal">
              <Switch
                checked={draft.enabled ?? true}
                data-testid="job-enabled-toggle"
                onCheckedChange={checked => onChange({ ...draft, enabled: checked })}
              />
              <FieldContent>
                <FieldTitle>{mode === "create" ? "Enabled on create" : "Enabled"}</FieldTitle>
                <FieldDescription>
                  Disabled jobs stay visible but never dispatch on their schedule.
                </FieldDescription>
              </FieldContent>
            </Field>
          </div>
        </Section>
      </div>

      <DialogFooter variant="ruled">
        <Button onClick={onCancel} type="button" variant="outline">
          Cancel
        </Button>
        <Button
          className="min-w-32"
          data-testid="submit-job-form"
          disabled={!canSubmit || isPending}
          type="submit"
        >
          {isPending ? "Saving..." : mode === "create" ? "Create Job" : "Save Changes"}
        </Button>
      </DialogFooter>
    </form>
  );
}
