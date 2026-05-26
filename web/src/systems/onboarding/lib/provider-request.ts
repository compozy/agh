import type { SettingsProviderRequest } from "@/systems/settings";

import type { OnboardingAuthMode } from "../stores/use-onboarding-draft-store";

export interface ProviderRequestInputs {
  model: string;
  reasoning: string;
  authMode: OnboardingAuthMode;
  envVar: string;
  apiKey: string;
  provider: string;
}

type ProviderSettings = NonNullable<SettingsProviderRequest["settings"]>;
type ProviderModelsPayload = NonNullable<ProviderSettings["models"]>;

function existingApiKeyTargetEnv(current: ProviderSettings): string {
  const slot = current.credential_slots?.find(entry => entry.name === "api_key");
  return slot?.target_env?.trim() ?? "";
}

function buildProviderModels(
  current: ProviderSettings,
  model: string,
  reasoning: string
): ProviderModelsPayload {
  const base = current.models ?? {};
  const models: ProviderModelsPayload = {
    ...base,
    ...(model.length > 0 ? { default: model } : {}),
  };
  if (model.length === 0 || reasoning.length === 0) {
    return models;
  }
  const curated = [...(base.curated ?? [])];
  const index = curated.findIndex(entry => entry.id === model);
  if (index >= 0) {
    curated[index] = { ...curated[index], default_reasoning_effort: reasoning };
  } else {
    curated.push({ id: model, default_reasoning_effort: reasoning, supports_reasoning: true });
  }
  models.curated = curated;
  return models;
}

// buildOnboardingProviderRequest produces a read-modify-write provider settings payload that
// persists the chosen default model + reasoning and the selected auth mode without dropping
// existing provider config. The API key (if any) is only ever sent as a vault-backed secret.
export function buildOnboardingProviderRequest(
  current: ProviderSettings,
  inputs: ProviderRequestInputs
): SettingsProviderRequest {
  const settings: ProviderSettings = {
    ...current,
    models: buildProviderModels(current, inputs.model, inputs.reasoning),
    auth_mode: inputs.authMode,
  };

  if (inputs.authMode !== "bound_secret") {
    delete settings.credential_slots;
    return { settings };
  }

  const targetEnv = inputs.envVar.trim() || existingApiKeyTargetEnv(current);
  if (targetEnv.length === 0) {
    throw new Error("Enter the environment variable the provider expects.");
  }
  const hasKey = inputs.apiKey.trim().length > 0;
  const secretRef = hasKey ? `vault:providers/${inputs.provider}/api_key` : `env:${targetEnv}`;
  settings.credential_slots = [
    {
      name: "api_key",
      target_env: targetEnv,
      secret_ref: secretRef,
      kind: "api_key",
      required: true,
    },
  ];
  if (!hasKey) {
    return { settings };
  }
  return {
    settings,
    secrets: [
      { name: "api_key", secret_ref: secretRef, kind: "api_key", value: inputs.apiKey.trim() },
    ],
  };
}
