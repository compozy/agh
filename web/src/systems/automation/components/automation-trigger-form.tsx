import type { FormEvent } from "react";
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import { Pill, PillButton } from "@/components/design-system";
import { Button } from "@agh/ui";

import {
  AutomationCheckbox,
  AutomationField,
  AutomationFormSection,
  AutomationInput,
  AutomationTextarea,
} from "./automation-form-primitives";
import { retryDraftForStrategy } from "../lib/automation-drafts";
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
  const [governanceExpanded, setGovernanceExpanded] = useState(mode === "edit");
  const retry = retryDraftForStrategy(draft.retry?.strategy ?? "none", draft.retry ?? undefined);

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
      className="flex max-h-[min(84vh,960px)] flex-col"
      data-testid="automation-trigger-form"
      onSubmit={handleSubmit}
    >
      <div className="border-b border-[color:var(--color-divider)] px-6 py-5">
        <div className="max-w-2xl space-y-1 pr-12">
          <h2 className="text-lg font-semibold text-[color:var(--color-text-primary)]">
            {mode === "create" ? "Create Trigger" : "Edit Trigger"}
          </h2>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            Event-driven triggers react to daemon events, webhooks, and extension signals.
          </p>
        </div>
      </div>

      <div className="flex-1 space-y-5 overflow-y-auto px-6 py-5">
        <AutomationFormSection
          description="Choose the event source, the destination agent, and the prompt template."
          title="Core"
        >
          <div className="grid gap-4 md:grid-cols-2">
            <AutomationField label="Name">
              <AutomationInput
                data-testid="trigger-name-input"
                onChange={event => onChange({ ...draft, name: event.target.value })}
                placeholder="review-on-stop"
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
          <AutomationField label="Event">
            <AutomationInput
              data-testid="trigger-event-input"
              onChange={event => onChange({ ...draft, event: event.target.value })}
              placeholder="session.stopped or webhook"
              value={draft.event}
            />
          </AutomationField>
          <div className="space-y-2">
            <span className="text-sm font-medium text-[color:var(--color-text-primary)]">
              Scope
            </span>
            <div className="flex flex-wrap items-center gap-2">
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
          </div>
          <AutomationField label="Prompt template">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <Pill emphasis="strong" kind="state" tone="violet">
                  GO TEMPLATE
                </Pill>
                <span className="text-xs text-[color:var(--color-text-tertiary)]">
                  Variables: .EventName, .Source, .Data, .Timestamp
                </span>
              </div>
              <AutomationTextarea
                data-testid="trigger-prompt-input"
                onChange={event => onChange({ ...draft, prompt: event.target.value })}
                placeholder="Review the session {{ .Data.session_id }} for agent {{ .Data.agent_name }}."
                value={draft.prompt}
              />
            </div>
          </AutomationField>
        </AutomationFormSection>

        <AutomationFormSection
          description="Optional exact-match filters narrow which activations should dispatch."
          title="Activation"
        >
          <AutomationField hint="One key=value pair per line." label="Filter rules">
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

        <section className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <button
            className="flex w-full items-center justify-between gap-3 text-left"
            data-testid="trigger-governance-toggle"
            onClick={() => setGovernanceExpanded(current => !current)}
            type="button"
          >
            <span className="flex items-center gap-2">
              {governanceExpanded ? (
                <ChevronDown className="size-4 text-[color:var(--color-text-label)]" />
              ) : (
                <ChevronRight className="size-4 text-[color:var(--color-text-label)]" />
              )}
              <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.1em] text-[color:var(--color-text-label)]">
                Governance
              </span>
            </span>
            <span className="text-sm text-[color:var(--color-text-secondary)]">
              Optional retry and rate limit settings
            </span>
          </button>

          {governanceExpanded ? (
            <div className="mt-4 space-y-4">
              <div className="grid gap-4 md:grid-cols-3">
                <AutomationField label="Retry policy">
                  <div className="flex flex-wrap items-center gap-2">
                    <PillButton
                      active={retry.strategy === "none"}
                      data-testid="trigger-retry-strategy-none"
                      onClick={() =>
                        onChange({ ...draft, retry: retryDraftForStrategy("none", retry) })
                      }
                      size="dense"
                    >
                      NONE
                    </PillButton>
                    <PillButton
                      active={retry.strategy === "backoff"}
                      data-testid="trigger-retry-strategy-backoff"
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
                    data-testid="trigger-retry-max"
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
                    data-testid="trigger-retry-delay"
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
                <AutomationField label="Window">
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
            </div>
          ) : null}
        </section>
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
          className="min-w-36"
          data-testid="submit-trigger-form"
          disabled={!canSubmit || isPending}
          size="lg"
          type="submit"
        >
          {isPending ? "Saving..." : mode === "create" ? "Create Trigger" : "Save Changes"}
        </Button>
      </div>
    </form>
  );
}
