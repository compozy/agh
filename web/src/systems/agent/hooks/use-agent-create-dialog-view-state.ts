import { useState } from "react";

import type { RuntimeProviderOption } from "@/systems/runtime";

import {
  validateAgentCreateDraft,
  type AgentCreateDialogDraft,
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
  providerOptions: RuntimeProviderOption[];
  providersError: string | null;
  providersLoading: boolean;
}

function useAgentCreateDialogViewState({
  draft,
  hasActiveWorkspace,
  initialStep,
  onOpenChange,
  providerOptions,
  providersError,
  providersLoading,
}: AgentCreateDialogViewStateArgs) {
  const [step, setStep] = useState<AgentCreateStep>(initialStep);
  const validation = validateAgentCreateDraft(draft, {
    hasActiveWorkspace,
    providerOptions,
    providersError,
    providersLoading,
  });
  const visibleErrors = visibleAgentCreateErrors(draft, validation.fields, {
    providerOptions,
    providersError,
    providersLoading,
  });

  const currentIndex = AGENT_CREATE_STEPS.indexOf(step);
  const previousStep = currentIndex > 0 ? AGENT_CREATE_STEPS[currentIndex - 1] : undefined;
  const nextStep =
    currentIndex < AGENT_CREATE_STEPS.length - 1 ? AGENT_CREATE_STEPS[currentIndex + 1] : undefined;
  const canAdvance = validation.stepValidity[step];
  const activeProvider = providerOptions.find(option => option.id === draft.provider);

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
    providerOptions: readonly RuntimeProviderOption[];
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
    reasoningEffort: draft.reasoningEffort !== "" ? errors.reasoningEffort : undefined,
    prompt: draft.prompt.trim().length > 0 ? errors.prompt : undefined,
    tools: draft.tools.length > 0 ? errors.tools : undefined,
    toolsets: draft.toolsets.length > 0 ? errors.toolsets : undefined,
    denyTools: draft.denyTools.length > 0 ? errors.denyTools : undefined,
  };
}

export { useAgentCreateDialogViewState };
