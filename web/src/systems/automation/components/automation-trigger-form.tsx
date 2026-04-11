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
import type { AutomationTriggerFilter, CreateAutomationTriggerRequest } from "../types";

interface AutomationTriggerFormProps {
  activeWorkspaceId?: string | null;
  draft: CreateAutomationTriggerRequest;
  isPending: boolean;
  mode: "create" | "edit";
  onCancel: () => void;
  onChange: (draft: CreateAutomationTriggerRequest) => void;
  onSubmit: () => void;
}

function formatFilterText(filter?: AutomationTriggerFilter): string {
  if (!filter || Object.keys(filter).length === 0) {
    return "";
  }

  return Object.entries(filter)
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}

function parseFilterText(text: string): AutomationTriggerFilter {
  return text
    .split("\n")
    .map(line => line.trim())
    .filter(Boolean)
    .reduce<AutomationTriggerFilter>((accumulator, line) => {
      const separatorIndex = line.indexOf("=");
      if (separatorIndex === -1) {
        accumulator[line] = "";
        return accumulator;
      }

      const key = line.slice(0, separatorIndex).trim();
      if (key === "") {
        return accumulator;
      }

      accumulator[key] = line.slice(separatorIndex + 1).trim();
      return accumulator;
    }, {});
}

export function AutomationTriggerForm({
  activeWorkspaceId,
  draft,
  isPending,
  mode,
  onCancel,
  onChange,
  onSubmit,
}: AutomationTriggerFormProps) {
  const canSubmit =
    draft.name.trim() !== "" &&
    draft.agent_name.trim() !== "" &&
    draft.prompt.trim() !== "" &&
    draft.event.trim() !== "" &&
    (draft.scope === "global" || Boolean(draft.workspace_id));

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
      data-testid="automation-trigger-form"
      onSubmit={handleSubmit}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h2 className="text-base font-semibold text-[color:var(--color-text-primary)]">
            {mode === "create" ? "Create trigger" : "Edit trigger"}
          </h2>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            Triggers react to daemon events, webhooks, and extension signals.
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
            data-testid="submit-trigger-form"
            disabled={!canSubmit || isPending}
            type="submit"
          >
            {isPending ? "Saving..." : mode === "create" ? "Create trigger" : "Save changes"}
          </button>
        </div>
      </div>

      <AutomationFormSection
        description="Choose the event source, the target agent, and the prompt template to render."
        title="Core"
      >
        <div className="grid gap-4 md:grid-cols-2">
          <AutomationField label="Name">
            <AutomationInput
              data-testid="trigger-name-input"
              onChange={event => onChange({ ...draft, name: event.target.value })}
              placeholder="review-on-session-stop"
              value={draft.name}
            />
          </AutomationField>
          <AutomationField label="Agent">
            <AutomationInput
              data-testid="trigger-agent-input"
              onChange={event => onChange({ ...draft, agent_name: event.target.value })}
              placeholder="reviewer"
              value={draft.agent_name}
            />
          </AutomationField>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <AutomationField label="Event">
            <AutomationInput
              data-testid="trigger-event-input"
              onChange={event => onChange({ ...draft, event: event.target.value })}
              placeholder="session.stopped or webhook"
              value={draft.event}
            />
          </AutomationField>
          <AutomationField label="Scope">
            <div className="flex items-center gap-2">
              <PillButton
                active={draft.scope === "global"}
                data-testid="trigger-scope-global"
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
                data-testid="trigger-scope-workspace"
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
          </AutomationField>
        </div>
        <AutomationField label="Prompt template">
          <AutomationTextarea
            data-testid="trigger-prompt-input"
            onChange={event => onChange({ ...draft, prompt: event.target.value })}
            placeholder="React to {{ .Source }} with payload {{ .Data }}"
            value={draft.prompt}
          />
        </AutomationField>
      </AutomationFormSection>

      <AutomationFormSection
        description="Optional exact-match filters narrow which activations should fire this trigger."
        title="Activation"
      >
        <AutomationField hint="One `path=value` pair per line." label="Filter rules">
          <AutomationTextarea
            data-testid="trigger-filter-input"
            onChange={event =>
              onChange({
                ...draft,
                filter: parseFilterText(event.target.value),
              })
            }
            placeholder={"data.branch=main\ndata.author=pedro"}
            value={formatFilterText(draft.filter)}
          />
        </AutomationField>
        {draft.event === "webhook" ? (
          <div className="grid gap-4 md:grid-cols-3">
            <AutomationField label="Webhook endpoint slug">
              <AutomationInput
                data-testid="trigger-endpoint-slug-input"
                onChange={event => onChange({ ...draft, endpoint_slug: event.target.value })}
                placeholder="repo-push"
                value={draft.endpoint_slug ?? ""}
              />
            </AutomationField>
            <AutomationField label="Webhook id">
              <AutomationInput
                data-testid="trigger-webhook-id-input"
                onChange={event => onChange({ ...draft, webhook_id: event.target.value })}
                placeholder="wbh_repo_push"
                value={draft.webhook_id ?? ""}
              />
            </AutomationField>
            <AutomationField label="Webhook secret">
              <AutomationInput
                data-testid="trigger-webhook-secret-input"
                onChange={event => onChange({ ...draft, webhook_secret: event.target.value })}
                placeholder="shared-secret"
                value={draft.webhook_secret ?? ""}
              />
            </AutomationField>
          </div>
        ) : null}
      </AutomationFormSection>

      <AutomationFormSection
        description="Retry and fire-limit settings protect downstream sessions from noisy events."
        title="Governance"
      >
        <div className="grid gap-4 md:grid-cols-3">
          <AutomationField label="Retry strategy">
            <AutomationSelect
              data-testid="trigger-retry-strategy"
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
              data-testid="trigger-retry-max"
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
              data-testid="trigger-retry-delay"
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
              data-testid="trigger-fire-limit-max"
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
              data-testid="trigger-fire-limit-window"
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
          description="Disabled triggers stay visible but ignore matching activation envelopes."
          label="Trigger enabled"
          onCheckedChange={checked => onChange({ ...draft, enabled: checked })}
          testId="trigger-enabled-toggle"
        />
      </AutomationFormSection>
    </form>
  );
}
