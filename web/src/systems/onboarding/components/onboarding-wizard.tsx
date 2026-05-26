import { Check, Lock } from "lucide-react";

import {
  Button,
  Eyebrow,
  Spinner,
  Stepper,
  StepperBody,
  StepperDescription,
  StepperIndicator,
  StepperItem,
  StepperNav,
  StepperRail,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from "@agh/ui";

import { useOnboardingWizard } from "../hooks/use-onboarding-wizard";
import { StepDefaultModel } from "./step-default-model";
import { StepOnboardingChat } from "./step-onboarding-chat";
import { StepWorkspaces } from "./step-workspaces";

const STEP_RAIL = [
  {
    step: 1,
    title: "Default model",
    description: "Provider, model, reasoning & how you authenticate.",
  },
  { step: 2, title: "Workspaces", description: "Add the folders AGH should operate inside." },
  { step: 3, title: "Onboarding agent", description: "Set up channels & agents in a quick chat." },
];

interface OnboardingWizardProps {
  onComplete: () => void;
}

export function OnboardingWizard({ onComplete }: OnboardingWizardProps) {
  const wizard = useOnboardingWizard(onComplete);

  return (
    <div
      data-testid="onboarding-wizard"
      className="grid h-dvh min-h-0 grid-cols-1 overflow-hidden bg-canvas md:grid-cols-[360px_minmax(0,1fr)]"
    >
      <aside className="relative hidden flex-col border-r border-line bg-rail p-7 md:flex">
        <div className="flex items-center gap-3">
          <span
            aria-hidden="true"
            className="grid size-8 place-items-center rounded-md bg-accent text-accent-ink"
          >
            <Check className="size-4" />
          </span>
          <div>
            <div className="text-base font-medium text-fg-strong">AGH</div>
            <div className="text-xs text-faint">Agent workplace</div>
          </div>
        </div>

        <div className="mt-7 mb-6">
          <Eyebrow className="text-accent">First-run setup</Eyebrow>
          <h1 className="mt-2 text-lg font-medium leading-tight text-fg-strong">
            Let&apos;s get your workspace running.
          </h1>
          <p className="mt-2 text-small-body leading-6 text-muted">
            A few essentials before AGH can host agents. You can refine everything later in
            Settings.
          </p>
        </div>

        <Stepper value={wizard.step} orientation="vertical" onValueChange={wizard.goToStep}>
          <StepperNav>
            {STEP_RAIL.map(item => (
              <StepperItem
                key={item.step}
                step={item.step}
                completed={wizard.step > item.step}
                disabled={item.step > wizard.maxStep}
              >
                <StepperTrigger>
                  <StepperRail>
                    <StepperIndicator>{item.step}</StepperIndicator>
                    {item.step < STEP_RAIL.length ? <StepperSeparator /> : null}
                  </StepperRail>
                  <StepperBody>
                    <StepperTitle>{item.title}</StepperTitle>
                    <StepperDescription>{item.description}</StepperDescription>
                  </StepperBody>
                </StepperTrigger>
              </StepperItem>
            ))}
          </StepperNav>
        </Stepper>

        <div className="mt-auto flex items-center gap-2 border-t border-line pt-4 text-xs text-faint">
          <Lock className="size-3 text-subtle" />
          <span>Runs locally · nothing leaves your machine</span>
        </div>
      </aside>

      <main className="flex min-h-0 flex-col bg-canvas">
        <header className="flex-none border-b border-line px-8 py-6">
          <div className="mx-auto max-w-2xl">
            <Eyebrow className="text-accent">{wizard.meta.eyebrow}</Eyebrow>
            <h2 className="mt-2 text-detail-h2 font-medium tracking-detail-h2 text-fg-strong">
              {wizard.meta.title}
            </h2>
            <p className="mt-2 max-w-xl text-small-body leading-6 text-muted">{wizard.meta.lead}</p>
          </div>
        </header>

        {wizard.step === 3 ? (
          <StepOnboardingChat chat={wizard.chat} />
        ) : (
          <div className="min-h-0 flex-1 overflow-y-auto px-8 py-7">
            <div className="mx-auto max-w-2xl">
              {wizard.step === 1 ? (
                <StepDefaultModel model={wizard.defaultModel} />
              ) : (
                <StepWorkspaces workspaces={wizard.workspaces} />
              )}
            </div>
          </div>
        )}

        <footer className="flex flex-none items-center justify-between gap-4 border-t border-line bg-canvas px-8 py-4">
          <p className="text-xs text-faint" data-testid="onboarding-footer-hint">
            Step <span className="font-medium text-muted tabular-nums">{wizard.step}</span> / 3 ·{" "}
            {wizard.meta.hint}
          </p>
          <div className="flex items-center gap-2.5">
            {wizard.commitError ? (
              <span
                className="text-xs text-danger"
                role="alert"
                data-testid="onboarding-commit-error"
              >
                {wizard.commitError}
              </span>
            ) : null}
            <Button variant="ghost" size="sm" onClick={wizard.back} disabled={wizard.step === 1}>
              Back
            </Button>
            <Button
              size="sm"
              onClick={() => void wizard.next()}
              disabled={!wizard.canContinue || wizard.isBusy}
              data-testid="onboarding-continue"
            >
              {wizard.isBusy ? <Spinner /> : null}
              {wizard.isLastStep ? "Finish setup" : "Continue"}
            </Button>
          </div>
        </footer>
      </main>
    </div>
  );
}
