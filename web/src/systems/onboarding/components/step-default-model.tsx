import { KeyRound } from "lucide-react";

import { cn, Eyebrow, Field, FieldLabel, Input, Spinner } from "@agh/ui";
import { RuntimeSelector } from "@/systems/runtime";

import type { OnboardingDefaultModelApi } from "../hooks/use-onboarding-default-model";
import type { OnboardingAuthMode } from "../stores/use-onboarding-draft-store";

const AUTH_OPTIONS: { mode: OnboardingAuthMode; title: string; description: string }[] = [
  {
    mode: "native_cli",
    title: "Native CLI auth",
    description: "Reuse the provider CLI already signed in on this machine.",
  },
  {
    mode: "bound_secret",
    title: "Provide an API key",
    description: "Bind a key from an environment variable or paste it directly.",
  },
];

interface StepDefaultModelProps {
  model: OnboardingDefaultModelApi;
}

export function StepDefaultModel({ model }: StepDefaultModelProps) {
  return (
    <div className="flex flex-col gap-8" data-testid="onboarding-step-default-model">
      <section>
        <div id="onboarding-runtime-label" className="mb-3 flex items-center gap-2">
          <Eyebrow className="text-subtle">Runtime</Eyebrow>
        </div>
        {model.providersLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted">
            <Spinner /> Loading providers…
          </div>
        ) : model.providersError ? (
          <p className="text-sm text-danger" role="alert">
            {model.providersError}
          </p>
        ) : (
          <RuntimeSelector
            value={model.runtimeValue}
            onChange={model.onRuntimeChange}
            providers={model.runtimeProviders}
            models={model.runtimeModels}
            loading={model.catalogLoading}
            catalogLoaded={model.catalogLoaded}
            refreshing={model.catalogRefreshing}
            onRefreshCatalog={model.onRefreshCatalog}
            disabled={model.runtimeProviders.length === 0}
            ariaLabelledby="onboarding-runtime-label"
            triggerTestId="onboarding-runtime-select"
          />
        )}
        {model.catalogError ? (
          <p className="mt-2 text-xs text-danger" role="alert">
            {model.catalogError}
          </p>
        ) : null}
      </section>

      <section>
        <div className="mb-3 flex items-center gap-2">
          <Eyebrow className="text-subtle">Authentication</Eyebrow>
        </div>
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          {AUTH_OPTIONS.map(option => {
            const selected = option.mode === model.authMode;
            return (
              <button
                key={option.mode}
                type="button"
                aria-pressed={selected}
                onClick={() => model.onAuthModeChange(option.mode)}
                data-testid={`onboarding-auth-${option.mode}`}
                className={cn(
                  "flex gap-3 rounded-md bg-canvas-soft p-3.5 text-left ring-1 ring-inset ring-line transition-colors hover:bg-elevated",
                  selected && "bg-surface-glaze ring-[1.5px] ring-accent"
                )}
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "mt-0.5 size-4 flex-none rounded-full ring-[1.5px] ring-inset ring-line-strong",
                    selected && "ring-[5px] ring-accent"
                  )}
                />
                <span>
                  <span className="block text-sm font-medium text-fg-strong">{option.title}</span>
                  <span className="mt-0.5 block text-xs leading-5 text-subtle">
                    {option.description}
                  </span>
                </span>
              </button>
            );
          })}
        </div>

        {model.authMode === "bound_secret" ? (
          <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel>Environment variable</FieldLabel>
              <Input
                value={model.envVar}
                spellCheck={false}
                onChange={event => model.onEnvVarChange(event.currentTarget.value)}
                placeholder="PROVIDER_API_KEY"
                aria-invalid={
                  model.configurationError ===
                  "Enter the environment variable the provider expects."
                }
                data-testid="onboarding-env-var"
              />
            </Field>
            <Field>
              <FieldLabel>
                API key <span className="font-normal text-faint">(optional)</span>
              </FieldLabel>
              <div className="relative">
                <KeyRound className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-subtle" />
                <Input
                  type="password"
                  value={model.apiKey}
                  spellCheck={false}
                  onChange={event => model.onApiKeyChange(event.currentTarget.value)}
                  placeholder="sk-…"
                  className="pl-8"
                  data-testid="onboarding-api-key"
                />
              </div>
            </Field>
          </div>
        ) : null}
        {model.configurationError ? (
          <p className="mt-3 text-xs text-danger" role="alert">
            {model.configurationError}
          </p>
        ) : null}
      </section>
    </div>
  );
}
