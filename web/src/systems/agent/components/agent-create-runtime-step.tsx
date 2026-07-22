import { Settings2 } from "lucide-react";

import {
  Button,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  FieldTitle,
  FormSection,
  Input,
} from "@agh/ui";

import {
  RuntimeSelector,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
} from "@/systems/runtime";

import type { AgentCreateDialogDraft } from "../lib/agent-create-draft";

export interface AgentCreateRuntimeStepProps {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  modelCatalogError: string | null;
  modelCatalogLoading: boolean;
  modelCatalogLoaded: boolean;
  modelCatalogRefreshing: boolean;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
  onRefreshCatalog: () => void;
  onOpenProviderSettings: () => void;
  providerOptions: RuntimeProviderOption[];
  providersLoading: boolean;
  runtimeModels: RuntimeModelOption[];
}

export function AgentCreateRuntimeStep({
  draft,
  errors,
  modelCatalogError,
  modelCatalogLoading,
  modelCatalogLoaded,
  modelCatalogRefreshing,
  onDraftChange,
  onRefreshCatalog,
  onOpenProviderSettings,
  providerOptions,
  providersLoading,
  runtimeModels,
}: AgentCreateRuntimeStepProps) {
  const runtimeValue: RuntimeSelectorValue = {
    provider: draft.provider,
    model: draft.model,
    reasoning_effort: draft.reasoningEffort,
  };
  const hasRuntimeOverride = Boolean(
    draft.provider.trim() || draft.model.trim() || draft.reasoningEffort
  );
  return (
    <FormSection
      data-testid="agent-create-runtime"
      icon={Settings2}
      size="compact"
      title="Runtime"
      description="Choose the provider, model, and reasoning defaults for new sessions."
    >
      <Field data-invalid={Boolean(errors.provider || errors.reasoningEffort)}>
        <FieldTitle id="agent-create-runtime-label">Runtime</FieldTitle>
        <FieldDescription>
          Leave this unchanged to inherit the project provider, model, and reasoning defaults.
          Selecting a runtime creates agent-level overrides.
        </FieldDescription>
        {draft.provider.trim().length === 0 ? (
          <p className="text-form-hint text-info" data-testid="agent-create-runtime-inherited">
            Project runtime defaults will be used.
          </p>
        ) : null}
        <RuntimeSelector
          value={runtimeValue}
          onChange={next =>
            onDraftChange({
              ...draft,
              provider: next.provider,
              model: next.model,
              reasoningEffort: next.reasoning_effort,
            })
          }
          providers={providerOptions}
          models={runtimeModels}
          loading={modelCatalogLoading}
          catalogLoaded={modelCatalogLoaded}
          refreshing={modelCatalogRefreshing}
          onRefreshCatalog={onRefreshCatalog}
          onOpenProviderSettings={onOpenProviderSettings}
          disabled={providersLoading || providerOptions.length === 0}
          ariaLabelledby="agent-create-runtime-label"
          triggerId="agent-create-runtime-trigger"
          triggerTestId="agent-create-runtime-select"
        />
        {hasRuntimeOverride ? (
          <Button
            className="mt-2"
            data-testid="agent-create-runtime-use-project-defaults"
            onClick={() =>
              onDraftChange({
                ...draft,
                provider: "",
                model: "",
                reasoningEffort: "",
              })
            }
            size="sm"
            type="button"
            variant="ghost"
          >
            Use project defaults
          </Button>
        ) : null}
        <FieldError data-testid="agent-create-provider-error">
          {errors.provider ?? errors.reasoningEffort}
        </FieldError>
        {modelCatalogError ? (
          <p className="text-small-body text-warning" data-testid="agent-create-model-error">
            {modelCatalogError}
          </p>
        ) : null}
      </Field>

      <Field>
        <FieldLabel htmlFor="agent-create-command">Command</FieldLabel>
        <FieldDescription>Optional provider command override for this agent.</FieldDescription>
        <Input
          data-testid="agent-create-command"
          id="agent-create-command"
          onChange={event => onDraftChange({ ...draft, command: event.target.value })}
          placeholder="Leave blank to use the provider default"
          value={draft.command}
        />
      </Field>
    </FormSection>
  );
}
