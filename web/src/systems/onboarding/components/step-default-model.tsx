import { Check, KeyRound } from "lucide-react";

import {
  Eyebrow,
  Field,
  FieldLabel,
  Input,
  KindIcon,
  Spinner,
  cn,
  providerKindIconRegistry,
} from "@agh/ui";
import { ModelCommandSelect, ReasoningCommandSelect } from "@/systems/runtime";

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
        <div className="mb-3 flex items-center gap-2">
          <Eyebrow className="text-subtle">Provider</Eyebrow>
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
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
            {model.providers.map(provider => {
              const selected = provider.name === model.provider;
              return (
                <button
                  key={provider.name}
                  type="button"
                  aria-pressed={selected}
                  onClick={() => model.onProviderChange(provider.name)}
                  data-testid={`onboarding-provider-${provider.name}`}
                  className={cn(
                    "relative flex items-center gap-3 rounded-md bg-canvas-soft px-3.5 py-3 text-left ring-1 ring-inset ring-line transition-colors hover:bg-elevated",
                    selected && "bg-surface-glaze ring-[1.5px] ring-accent"
                  )}
                >
                  <span className="grid size-10 flex-none place-items-center rounded-icon-well bg-canvas-tint ring-1 ring-inset ring-line">
                    <KindIcon
                      kind={provider.name}
                      registry={providerKindIconRegistry}
                      size="md"
                      tone={selected ? "accent" : "default"}
                    />
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium text-fg-strong">
                      {provider.display_name || provider.name}
                    </span>
                    <span className="mt-0.5 block truncate font-mono text-xs text-subtle">
                      {provider.name}
                    </span>
                  </span>
                  {selected ? (
                    <span className="absolute right-2.5 top-2.5 grid size-4 place-items-center rounded-full bg-accent text-accent-ink">
                      <Check className="size-2.5" />
                    </span>
                  ) : null}
                </button>
              );
            })}
          </div>
        )}
      </section>

      <section className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <Field>
          <FieldLabel>Model</FieldLabel>
          <ModelCommandSelect
            options={model.modelOptions}
            value={model.model}
            loading={model.catalogLoading}
            disabled={model.provider.length === 0}
            onChange={model.onModelChange}
            placeholder="Provider default"
            triggerTestId="onboarding-model-select"
          />
          {model.catalogError ? (
            <p className="text-xs text-danger" role="alert">
              {model.catalogError}
            </p>
          ) : null}
        </Field>
        <Field>
          <FieldLabel>Reasoning effort</FieldLabel>
          <ReasoningCommandSelect
            options={model.reasoningOptions}
            value={model.reasoning}
            disabled={!model.reasoningSupported}
            disabledHint="This model does not expose reasoning effort."
            onChange={model.onReasoningChange}
            placeholder="Default"
            triggerTestId="onboarding-reasoning-select"
          />
        </Field>
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
                data-testid="onboarding-env-var"
              />
            </Field>
            <Field>
              <FieldLabel>
                API key <span className="font-normal text-faint">· optional now</span>
              </FieldLabel>
              <div className="relative">
                <KeyRound className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-subtle" />
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
      </section>
    </div>
  );
}
