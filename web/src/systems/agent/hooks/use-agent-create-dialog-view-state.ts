import { useEffect, useMemo, useState } from "react";

import {
  validateAgentCreateDraft,
  type AgentCreateDialogDraft,
  type AgentCreateProviderOption,
  type AgentCreateStep,
} from "../lib/agent-create-draft";

const AGENT_CREATE_STEPS: readonly AgentCreateStep[] = [
  "basics",
  "runtime",
  "instructions",
  "access",
];

interface AgentCreateDialogViewStateArgs {
  draft: AgentCreateDialogDraft;
  hasActiveWorkspace: boolean;
  initialStep: AgentCreateStep;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  providerOptions: AgentCreateProviderOption[];
  providersError: string | null;
  providersLoading: boolean;
}

function useAgentCreateDialogViewState({
  draft,
  hasActiveWorkspace,
  initialStep,
  onOpenChange,
  open,
  providerOptions,
  providersError,
  providersLoading,
}: AgentCreateDialogViewStateArgs) {
  const [step, setStep] = useState<AgentCreateStep>(initialStep);
  const validation = useMemo(
    () =>
      validateAgentCreateDraft(draft, {
        hasActiveWorkspace,
        providerOptions,
        providersError,
        providersLoading,
      }),
    [draft, hasActiveWorkspace, providerOptions, providersError, providersLoading]
  );
  const visibleErrors = useMemo(
    () =>
      visibleAgentCreateErrors(draft, validation.fields, {
        providerOptions,
        providersError,
        providersLoading,
      }),
    [draft, providerOptions, providersError, providersLoading, validation.fields]
  );

  useEffect(() => {
    if (!open) {
      setStep(initialStep);
    }
  }, [initialStep, open]);

  const currentIndex = AGENT_CREATE_STEPS.indexOf(step);
  const previousStep = currentIndex > 0 ? AGENT_CREATE_STEPS[currentIndex - 1] : undefined;
  const nextStep =
    currentIndex < AGENT_CREATE_STEPS.length - 1 ? AGENT_CREATE_STEPS[currentIndex + 1] : undefined;
  const canAdvance = validation.stepValidity[step];
  const activeProvider = providerOptions.find(option => option.name === draft.provider);

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setStep(initialStep);
    }
    onOpenChange(next);
  };

  return {
    activeProvider,
    canAdvance,
    currentIndex,
    handleOpenChange,
    nextStep,
    previousStep,
    setStep,
    step,
    validation,
    visibleErrors,
  };
}

function visibleAgentCreateErrors(
  draft: AgentCreateDialogDraft,
  errors: Record<string, string | undefined>,
  context: {
    providerOptions: readonly AgentCreateProviderOption[];
    providersError: string | null;
    providersLoading: boolean;
  }
): Record<string, string | undefined> {
  return {
    scope: errors.scope,
    name: draft.name.trim().length > 0 ? errors.name : undefined,
    categoryPath: draft.categoryPath.trim().length > 0 ? errors.categoryPath : undefined,
    provider:
      context.providersError ||
      context.providersLoading ||
      context.providerOptions.length === 0 ||
      draft.provider.trim().length > 0
        ? errors.provider
        : undefined,
    prompt: draft.prompt.trim().length > 0 ? errors.prompt : undefined,
    tools: draft.tools.length > 0 ? errors.tools : undefined,
    toolsets: draft.toolsets.length > 0 ? errors.toolsets : undefined,
    denyTools: draft.denyTools.length > 0 ? errors.denyTools : undefined,
  };
}

export { useAgentCreateDialogViewState };
