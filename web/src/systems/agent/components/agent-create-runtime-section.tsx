import { Settings2 } from "lucide-react";

import { Button, Field, FieldDescription, FieldError, FieldTitle, FormSection } from "@agh/ui";

import {
  RuntimeSelector,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
} from "@/systems/runtime";

import type { AgentCreateDialogDraft } from "../lib/agent-create-draft";

export interface AgentCreateRuntimeSectionProps {
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

/**
 * Simple tier: provider, model, and reasoning as one segmented control.
 *
 * Catalog state stays visible here rather than behind Advanced — a Simple view
 * that hides catalog truth would let an operator submit against a stale or
 * failed catalog without knowing it.
 */
export function AgentCreateRuntimeSection({
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
}: AgentCreateRuntimeSectionProps) {
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
      description="Backed by the live provider and model catalogs."
      icon={Settings2}
      size="compact"
      title="Runtime"
    >
      <Field data-invalid={Boolean(errors.provider || errors.reasoningEffort)}>
        <FieldTitle id="agent-create-runtime-label">Runtime</FieldTitle>
        <FieldDescription>
          Leave this unchanged to inherit the project defaults; selecting a runtime creates
          agent-level overrides.
        </FieldDescription>
        {draft.provider.trim().length === 0 ? (
          <p className="text-form-hint text-info" data-testid="agent-create-runtime-inherited">
            Project runtime defaults will be used.
          </p>
        ) : null}
        <RuntimeSelector
          ariaLabelledby="agent-create-runtime-label"
          catalogLoaded={modelCatalogLoaded}
          disabled={providersLoading || providerOptions.length === 0}
          loading={modelCatalogLoading}
          models={runtimeModels}
          onChange={next =>
            onDraftChange({
              ...draft,
              provider: next.provider,
              model: next.model,
              reasoningEffort: next.reasoning_effort,
            })
          }
          onOpenProviderSettings={onOpenProviderSettings}
          onRefreshCatalog={onRefreshCatalog}
          providers={providerOptions}
          refreshing={modelCatalogRefreshing}
          triggerId="agent-create-runtime-trigger"
          triggerTestId="agent-create-runtime-select"
          value={runtimeValue}
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
    </FormSection>
  );
}
