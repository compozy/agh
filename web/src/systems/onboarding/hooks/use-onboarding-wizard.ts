import { useState } from "react";
import { toast } from "sonner";

import { useOnboardingDraftStore } from "../stores/use-onboarding-draft-store";
import { useCompleteOnboarding } from "./use-complete-onboarding";
import {
  useOnboardingDefaultModel,
  type OnboardingDefaultModelApi,
} from "./use-onboarding-default-model";
import { useOnboardingWorkspaces, type OnboardingWorkspacesApi } from "./use-onboarding-workspaces";

export const ONBOARDING_STEP_COUNT = 2;

export interface OnboardingStepMeta {
  eyebrow: string;
  title: string;
  lead: string;
  hint: string;
}

const STEP_META: Record<number, OnboardingStepMeta> = {
  1: {
    eyebrow: "Step 1 of 2",
    title: "Choose your default model",
    lead: "This is the provider and model new agents use unless you override it per agent. AGH reaches it through your chosen credentials.",
    hint: "Default model",
  },
  2: {
    eyebrow: "Step 2 of 2",
    title: "Add your workspaces",
    lead: "Workspaces are the folders AGH operates inside. Add at least one — every agent session is scoped to a workspace.",
    hint: "Workspaces",
  },
};

export interface OnboardingWizardApi {
  step: number;
  maxStep: number;
  meta: OnboardingStepMeta;
  defaultModel: OnboardingDefaultModelApi;
  workspaces: OnboardingWorkspacesApi;
  canContinue: boolean;
  isLastStep: boolean;
  isBusy: boolean;
  commitError: string | null;
  goToStep: (step: number) => void;
  back: () => void;
  next: () => Promise<void>;
}

export function useOnboardingWizard(onComplete: () => void): OnboardingWizardApi {
  const step = useOnboardingDraftStore(state => state.step);
  const maxStep = useOnboardingDraftStore(state => state.maxStep);
  const setStep = useOnboardingDraftStore(state => state.setStep);
  const reset = useOnboardingDraftStore(state => state.reset);

  const defaultModel = useOnboardingDefaultModel();
  const workspaces = useOnboardingWorkspaces();
  const complete = useCompleteOnboarding();
  const [commitError, setCommitError] = useState<string | null>(null);

  const canContinue =
    step === 1 ? defaultModel.isValid : step === 2 ? workspaces.workspaces.length > 0 : true;

  const goToStep = (next: number) => {
    if (next < 1 || next > ONBOARDING_STEP_COUNT || next > maxStep) return;
    setStep(next);
  };

  const back = () => {
    if (step > 1) {
      setStep(step - 1);
    }
  };

  const finish = async () => {
    setCommitError(null);
    try {
      await complete.mutateAsync();
      reset();
      onComplete();
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to finish onboarding.";
      setCommitError(message);
      toast.error(message);
    }
  };

  const next = async () => {
    setCommitError(null);
    if (step === 1) {
      try {
        await defaultModel.commit();
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to save your default model.";
        setCommitError(message);
        toast.error(message);
        return;
      }
      setStep(2);
      return;
    }
    if (step === 2) {
      await finish();
      return;
    }
  };

  return {
    step,
    maxStep,
    meta: STEP_META[step] ?? STEP_META[1],
    defaultModel,
    workspaces,
    canContinue,
    isLastStep: step === ONBOARDING_STEP_COUNT,
    isBusy: defaultModel.isCommitting || complete.isPending,
    commitError,
    goToStep,
    back,
    next,
  };
}
