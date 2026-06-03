import { Bot, Box, Check, Clock, Filter, Info } from "lucide-react";

import { Button, DialogFooter, Field, FieldLabel, Input } from "@agh/ui";

import { useAutomationTriggerForm } from "../hooks/use-automation-trigger-form";
import type { WorkspaceOption } from "../lib/trigger-preview";
import type { CreateAutomationTriggerRequest } from "../types";
import { AgentPromptStep } from "./trigger-form/agent-prompt-step";
import { EventCatalog } from "./trigger-form/event-catalog";
import { FilterConditions } from "./trigger-form/filter-conditions";
import { FlowStep } from "./trigger-form/flow-step";
import { TriggerPreview } from "./trigger-form/preview/trigger-preview";
import { ReliabilitySection } from "./trigger-form/reliability-section";
import { ScopeStep } from "./trigger-form/scope-step";

interface AutomationTriggerFormProps {
  activeWorkspaceId?: string | null;
  draft: CreateAutomationTriggerRequest;
  isPending: boolean;
  mode: "create" | "edit";
  onCancel: () => void;
  onChange: (draft: CreateAutomationTriggerRequest) => void;
  onSubmit: () => void;
  /** Workspaces selectable for a workspace-scoped trigger. */
  workspaces?: ReadonlyArray<WorkspaceOption>;
  /** Known agent names; falls back to a free-text agent input when empty. */
  agents?: string[];
}

const EMPTY_AGENTS: string[] = [];

export function AutomationTriggerForm({
  activeWorkspaceId,
  draft,
  isPending,
  mode,
  onCancel,
  onChange,
  onSubmit,
  workspaces,
  agents = EMPTY_AGENTS,
}: AutomationTriggerFormProps) {
  const form = useAutomationTriggerForm({
    activeWorkspaceId,
    draft,
    isPending,
    mode,
    onChange,
    onSubmit,
    workspaces,
  });

  return (
    <form
      className="flex min-h-0 flex-col"
      data-testid="automation-trigger-form"
      onSubmit={form.handleSubmit}
    >
      <div className="grid min-h-0 flex-1 grid-cols-[1fr_var(--width-right-rail-default)] max-lg:grid-cols-1 max-lg:grid-rows-[minmax(0,1fr)_auto]">
        <section className="min-h-0 overflow-y-auto px-6 py-5">
          <Field className="mb-1">
            <FieldLabel htmlFor="trigger-name">Trigger name</FieldLabel>
            <Input
              className="font-mono"
              data-testid="trigger-name-input"
              id="trigger-name"
              onChange={event => form.onName(event.target.value)}
              placeholder="summarize-failures"
              value={draft.name}
            />
          </Field>

          <div className="mt-5">
            <FlowStep
              active
              icon={Box}
              kicker="For"
              subtitle="Choose where this trigger is active. A global trigger sees events from every workspace."
              title="A workspace, or the whole runtime"
            >
              <ScopeStep
                isWebhook={form.isWebhook}
                onScopeChange={form.onScopeChange}
                onWorkspaceChange={form.onWorkspaceChange}
                scope={draft.scope}
                workspaceId={draft.workspace_id}
                workspaces={form.resolvedWorkspaces}
              />
            </FlowStep>

            <FlowStep
              active={Boolean(form.selection.catalogId)}
              icon={Clock}
              kicker="When"
              subtitle="Pick the runtime event this trigger listens for. A few events need one more detail, shown right under your choice."
              title="An event happens"
            >
              <EventCatalog
                onSelectEvent={form.onSelectEvent}
                onSubConfigChange={form.onSubConfigChange}
                selection={form.selection}
                subConfigValues={form.subConfigValues}
              />
            </FlowStep>

            <FlowStep
              active={form.hasActiveFilters}
              icon={Filter}
              kicker="Only if"
              subtitle="Narrow it down. Each condition is an exact match on a field from the event above."
              title={
                <>
                  It matches these conditions
                  <span className="ml-1.5 text-small-body font-normal text-faint">(optional)</span>
                </>
              }
            >
              <FilterConditions
                eventKind={form.eventKind}
                filter={draft.filter ?? {}}
                keyOptions={form.keyOptions}
                onChange={form.onFilterChange}
                openPayload={form.openPayload}
              />
            </FlowStep>

            <FlowStep
              active={draft.agent_name.trim() !== ""}
              icon={Bot}
              kicker="Then"
              last
              subtitle="The agent receives a prompt rendered from the event's data."
              title="Run this agent"
            >
              <AgentPromptStep
                agent={draft.agent_name}
                agents={agents}
                onAgentChange={form.onAgentChange}
                onPromptChange={form.onPromptChange}
                prompt={draft.prompt}
                variables={form.variables}
              />
            </FlowStep>
          </div>

          <ReliabilitySection
            badge={form.preview.reliabilityBadge}
            defaultOpen={form.reliabilityDefaultOpen}
            enabled={draft.enabled ?? true}
            fireLimit={draft.fire_limit ?? undefined}
            onEnabledChange={form.onEnabledChange}
            onFireLimitChange={form.onFireLimitChange}
            onRetryChange={form.onRetryChange}
            retry={form.retry}
          />
        </section>

        <TriggerPreview preview={form.preview} />
      </div>

      <DialogFooter variant="ruled">
        <div className="flex flex-1 items-center gap-2 text-form-hint text-subtle">
          <Info aria-hidden="true" className="size-3 shrink-0 text-faint" />
          <span>
            Created as a <b className="font-medium text-muted">dynamic</b> trigger: editable and
            deletable anytime.
          </span>
        </div>
        <Button onClick={onCancel} type="button" variant="outline">
          Cancel
        </Button>
        <Button
          className="min-w-36"
          data-testid="submit-trigger-form"
          disabled={!form.canSubmit || isPending}
          type="submit"
        >
          {isPending ? (
            "Saving..."
          ) : mode === "create" ? (
            <>
              <Check aria-hidden="true" className="size-4" />
              Create trigger
            </>
          ) : (
            "Save changes"
          )}
        </Button>
      </DialogFooter>
    </form>
  );
}
