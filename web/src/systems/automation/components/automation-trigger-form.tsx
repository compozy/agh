import type { FormEvent } from "react";
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import {
  Button,
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldTitle,
  Input,
  Pill,
  PillGroup,
  Section,
  Switch,
  Textarea,
} from "@agh/ui";

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
  if (!filter || Object.keys(filter).length === 0) return "";
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
      if (key === "") return accumulator;

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
    if (!canSubmit || isPending) return;
    onSubmit();
  };

  return (
    <form
      className="flex max-h-[min(84vh,960px)] flex-col"
      data-testid="automation-trigger-form"
      onSubmit={handleSubmit}
    >
      <div className="flex-1 space-y-6 overflow-y-auto px-5 py-5">
        <Section label="Core">
          <div className="space-y-4 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="trigger-name">Name</FieldLabel>
                <Input
                  data-testid="trigger-name-input"
                  id="trigger-name"
                  onChange={event => onChange({ ...draft, name: event.target.value })}
                  placeholder="review-on-stop"
                  value={draft.name}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="trigger-agent">Agent</FieldLabel>
                <Input
                  data-testid="trigger-agent-input"
                  id="trigger-agent"
                  onChange={event => onChange({ ...draft, agent_name: event.target.value })}
                  placeholder="reviewer"
                  value={draft.agent_name}
                />
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="trigger-event">Event</FieldLabel>
              <Input
                data-testid="trigger-event-input"
                id="trigger-event"
                onChange={event => onChange({ ...draft, event: event.target.value })}
                placeholder="session.stopped or webhook"
                value={draft.event}
              />
            </Field>
            <Field>
              <FieldTitle>Scope</FieldTitle>
              <PillGroup
                aria-label="Scope"
                items={[
                  { value: "global", label: "GLOBAL", testId: "trigger-scope-global" },
                  { value: "workspace", label: "WORKSPACE", testId: "trigger-scope-workspace" },
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
            </Field>
            <Field>
              <FieldLabel htmlFor="trigger-prompt">Prompt template</FieldLabel>
              <div className="flex flex-wrap items-center gap-2">
                <Pill mono tone="info">
                  GO TEMPLATE
                </Pill>
                <span className="text-[12px] text-[color:var(--color-text-tertiary)]">
                  Variables: .EventName, .Source, .Data, .Timestamp
                </span>
              </div>
              <Textarea
                data-testid="trigger-prompt-input"
                id="trigger-prompt"
                onChange={event => onChange({ ...draft, prompt: event.target.value })}
                placeholder="Review the session {{ .Data.session_id }} for agent {{ .Data.agent_name }}."
                rows={4}
                value={draft.prompt}
              />
            </Field>
          </div>
        </Section>

        <Section label="Activation">
          <div className="space-y-4 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
            <Field>
              <FieldLabel htmlFor="trigger-filter">Filter rules</FieldLabel>
              <Textarea
                data-testid="trigger-filter-input"
                id="trigger-filter"
                onChange={event =>
                  onChange({
                    ...draft,
                    filter: parseFilterText(event.target.value),
                  })
                }
                placeholder={"data.branch=main\ndata.author=pedro"}
                rows={3}
                value={formatFilterText(draft.filter)}
              />
              <FieldDescription>One key=value pair per line.</FieldDescription>
            </Field>
            {draft.event === "webhook" ? (
              <div className="grid gap-4 md:grid-cols-3">
                <Field>
                  <FieldLabel htmlFor="trigger-endpoint-slug">Endpoint slug</FieldLabel>
                  <Input
                    data-testid="trigger-endpoint-slug-input"
                    id="trigger-endpoint-slug"
                    onChange={event => onChange({ ...draft, endpoint_slug: event.target.value })}
                    placeholder="repo-push"
                    value={draft.endpoint_slug ?? ""}
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="trigger-webhook-id">Webhook id</FieldLabel>
                  <Input
                    data-testid="trigger-webhook-id-input"
                    id="trigger-webhook-id"
                    onChange={event => onChange({ ...draft, webhook_id: event.target.value })}
                    placeholder="wbh_repo_push"
                    value={draft.webhook_id ?? ""}
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="trigger-webhook-secret">Webhook secret</FieldLabel>
                  <Input
                    data-testid="trigger-webhook-secret-input"
                    id="trigger-webhook-secret"
                    onChange={event => onChange({ ...draft, webhook_secret: event.target.value })}
                    placeholder="shared-secret"
                    value={draft.webhook_secret ?? ""}
                  />
                </Field>
              </div>
            ) : null}
          </div>
        </Section>

        <section className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <button
            className="flex w-full items-center justify-between gap-3 text-left"
            data-testid="trigger-governance-toggle"
            onClick={() => setGovernanceExpanded(current => !current)}
            type="button"
          >
            <span className="flex items-center gap-2">
              {governanceExpanded ? (
                <ChevronDown
                  aria-hidden="true"
                  className="size-4 text-[color:var(--color-text-tertiary)]"
                />
              ) : (
                <ChevronRight
                  aria-hidden="true"
                  className="size-4 text-[color:var(--color-text-tertiary)]"
                />
              )}
              <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                Governance
              </span>
            </span>
            <span className="text-[13px] text-[color:var(--color-text-secondary)]">
              Optional retry and rate limit settings
            </span>
          </button>

          {governanceExpanded ? (
            <div className="mt-4 space-y-4">
              <div className="grid gap-4 md:grid-cols-3">
                <Field>
                  <FieldTitle>Retry policy</FieldTitle>
                  <PillGroup
                    aria-label="Retry policy"
                    items={[
                      { value: "none", label: "NONE", testId: "trigger-retry-strategy-none" },
                      {
                        value: "backoff",
                        label: "BACKOFF",
                        testId: "trigger-retry-strategy-backoff",
                      },
                    ]}
                    onChange={next =>
                      onChange({ ...draft, retry: retryDraftForStrategy(next, retry) })
                    }
                    size="sm"
                    value={retry.strategy}
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="trigger-retry-max">Max retries</FieldLabel>
                  <Input
                    data-testid="trigger-retry-max"
                    disabled={retry.strategy !== "backoff"}
                    id="trigger-retry-max"
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
                  <FieldLabel htmlFor="trigger-retry-delay">Base delay</FieldLabel>
                  <Input
                    data-testid="trigger-retry-delay"
                    disabled={retry.strategy !== "backoff"}
                    id="trigger-retry-delay"
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
                  <FieldLabel htmlFor="trigger-fire-limit-max">Max fires</FieldLabel>
                  <Input
                    data-testid="trigger-fire-limit-max"
                    id="trigger-fire-limit-max"
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
                  <FieldLabel htmlFor="trigger-fire-limit-window">Window</FieldLabel>
                  <Input
                    data-testid="trigger-fire-limit-window"
                    id="trigger-fire-limit-window"
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
                  data-testid="trigger-enabled-toggle"
                  onCheckedChange={checked => onChange({ ...draft, enabled: checked })}
                />
                <FieldContent>
                  <FieldTitle>Trigger enabled</FieldTitle>
                  <FieldDescription>
                    Disabled triggers stay visible but ignore matching activation envelopes.
                  </FieldDescription>
                </FieldContent>
              </Field>
            </div>
          ) : null}
        </section>
      </div>

      <div className="flex items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-3">
        <Button onClick={onCancel} type="button" variant="outline">
          Cancel
        </Button>
        <Button
          className="min-w-36"
          data-testid="submit-trigger-form"
          disabled={!canSubmit || isPending}
          type="submit"
        >
          {isPending ? "Saving..." : mode === "create" ? "Create Trigger" : "Save Changes"}
        </Button>
      </div>
    </form>
  );
}
