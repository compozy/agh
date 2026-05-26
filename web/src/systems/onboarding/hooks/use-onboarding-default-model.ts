import { useCallback, useMemo } from "react";

import {
  deriveActiveSessionOptions,
  modelAvailabilityLabel,
  modelAvailabilityTone,
  useProviderModels,
  type ProviderModelPayload,
  type ReasoningOption,
} from "@/systems/model-catalog";
import { useProviders, type ProviderSummary } from "@/systems/providers";
import type { ModelSelectOption } from "@/systems/runtime";
import {
  useSettingsGeneral,
  useSettingsProvider,
  useUpdateSettingsGeneral,
  usePutSettingsProvider,
} from "@/systems/settings";

import { buildOnboardingProviderRequest } from "../lib/provider-request";
import {
  useOnboardingDraftStore,
  type OnboardingAuthMode,
} from "../stores/use-onboarding-draft-store";

export interface OnboardingDefaultModelApi {
  providers: ProviderSummary[];
  providersLoading: boolean;
  providersError: string | null;
  provider: string;
  model: string;
  reasoning: string;
  authMode: OnboardingAuthMode;
  envVar: string;
  apiKey: string;
  modelOptions: ModelSelectOption[];
  reasoningOptions: ReasoningOption[];
  reasoningSupported: boolean;
  defaultReasoning: string | null;
  catalogLoading: boolean;
  catalogError: string | null;
  configurationError: string | null;
  isValid: boolean;
  isCommitting: boolean;
  onProviderChange: (provider: string) => void;
  onModelChange: (model: string) => void;
  onReasoningChange: (reasoning: string) => void;
  onAuthModeChange: (mode: OnboardingAuthMode) => void;
  onEnvVarChange: (envVar: string) => void;
  onApiKeyChange: (apiKey: string) => void;
  commit: () => Promise<void>;
}

function describeError(fallback: string, error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}

export function useOnboardingDefaultModel(): OnboardingDefaultModelApi {
  const draft = useOnboardingDraftStore();
  const providersQuery = useProviders();
  const providers = useMemo(() => providersQuery.data?.providers ?? [], [providersQuery.data]);

  const provider = draft.provider;

  const generalQuery = useSettingsGeneral();
  const providerDetailQuery = useSettingsProvider(provider, { enabled: provider.length > 0 });
  const updateGeneral = useUpdateSettingsGeneral();
  const putProvider = usePutSettingsProvider();

  const catalogQuery = useProviderModels({
    providerId: provider,
    includeStale: true,
    enabled: provider.length > 0,
  });
  const catalogModels = useMemo<ProviderModelPayload[]>(
    () => catalogQuery.data?.models ?? [],
    [catalogQuery.data]
  );
  const existingApiKeyTargetEnv = useMemo(
    () =>
      providerDetailQuery.data?.settings.credential_slots
        ?.find(slot => slot.name === "api_key")
        ?.target_env.trim() ?? "",
    [providerDetailQuery.data]
  );

  const derived = useMemo(
    () =>
      deriveActiveSessionOptions({
        catalog: catalogModels,
        selectedModel: draft.model.length > 0 ? draft.model : null,
      }),
    [catalogModels, draft.model]
  );

  const modelOptions = useMemo<ModelSelectOption[]>(
    () =>
      derived.modelOptions.map(option => ({
        id: option.id,
        label: option.displayName,
        availability: {
          label: modelAvailabilityLabel(option.availabilityState),
          tone: modelAvailabilityTone(option.availabilityState),
          state: option.availabilityState,
        },
      })),
    [derived.modelOptions]
  );

  const onProviderChange = useCallback((next: string) => {
    useOnboardingDraftStore.getState().patch({
      provider: next,
      model: "",
      reasoning: "",
      envVar: "",
      apiKey: "",
    });
  }, []);

  const onModelChange = useCallback((model: string) => {
    useOnboardingDraftStore.getState().patch({ model, reasoning: "" });
  }, []);

  const onReasoningChange = useCallback((reasoning: string) => {
    useOnboardingDraftStore.getState().patch({ reasoning });
  }, []);

  const onAuthModeChange = useCallback((authMode: OnboardingAuthMode) => {
    useOnboardingDraftStore
      .getState()
      .patch(authMode === "native_cli" ? { authMode, envVar: "", apiKey: "" } : { authMode });
  }, []);

  const onEnvVarChange = useCallback((envVar: string) => {
    useOnboardingDraftStore.getState().patch({ envVar });
  }, []);

  const onApiKeyChange = useCallback((apiKey: string) => {
    useOnboardingDraftStore.getState().patch({ apiKey });
  }, []);

  const commit = useCallback(async () => {
    const trimmedProvider = draft.provider.trim();
    if (trimmedProvider.length === 0) {
      throw new Error("Select a provider before continuing.");
    }
    const detail = providerDetailQuery.data;
    if (!detail) {
      throw new Error(
        providerDetailQuery.error
          ? describeError("Failed to load provider settings.", providerDetailQuery.error)
          : "Provider settings are still loading."
      );
    }
    const config = generalQuery.data?.config;
    if (!config) {
      throw new Error(
        generalQuery.error
          ? describeError("Failed to load general settings.", generalQuery.error)
          : "General settings are still loading."
      );
    }
    const body = buildOnboardingProviderRequest(detail.settings, {
      model: draft.model.trim(),
      reasoning: draft.reasoning.trim(),
      authMode: draft.authMode,
      envVar: draft.envVar.trim(),
      apiKey: draft.apiKey.trim(),
      provider: trimmedProvider,
    });
    await putProvider.mutateAsync({ name: trimmedProvider, body });
    await updateGeneral.mutateAsync({
      config: { ...config, defaults: { ...config.defaults, provider: trimmedProvider } },
    });
  }, [
    draft,
    generalQuery.data,
    generalQuery.error,
    providerDetailQuery.data,
    providerDetailQuery.error,
    putProvider,
    updateGeneral,
  ]);

  const providerSettingsError =
    provider.length > 0 && providerDetailQuery.error
      ? describeError("Failed to load provider settings.", providerDetailQuery.error)
      : null;
  const generalSettingsError = generalQuery.error
    ? describeError("Failed to load general settings.", generalQuery.error)
    : null;
  const missingBoundSecretTarget =
    draft.authMode === "bound_secret" &&
    draft.envVar.trim().length === 0 &&
    existingApiKeyTargetEnv.length === 0;
  const credentialTargetError =
    provider.length > 0 && providerDetailQuery.isSuccess && missingBoundSecretTarget
      ? "Enter the environment variable the provider expects."
      : null;
  const configurationError =
    providerSettingsError ?? generalSettingsError ?? credentialTargetError ?? null;
  const canCommit =
    provider.trim().length > 0 &&
    providerDetailQuery.isSuccess &&
    generalQuery.isSuccess &&
    !missingBoundSecretTarget;

  return {
    providers,
    providersLoading: providersQuery.isLoading,
    providersError: providersQuery.error
      ? describeError("Failed to load providers.", providersQuery.error)
      : null,
    provider,
    model: draft.model,
    reasoning: draft.reasoning,
    authMode: draft.authMode,
    envVar: draft.envVar,
    apiKey: draft.apiKey,
    modelOptions,
    reasoningOptions: derived.reasoningOptions,
    reasoningSupported: derived.reasoningSupported,
    defaultReasoning: derived.defaultReasoning,
    catalogLoading: catalogQuery.isLoading || catalogQuery.isFetching,
    catalogError: catalogQuery.error
      ? describeError("Failed to load provider models.", catalogQuery.error)
      : null,
    configurationError,
    isValid: canCommit,
    isCommitting: putProvider.isPending || updateGeneral.isPending,
    onProviderChange,
    onModelChange,
    onReasoningChange,
    onAuthModeChange,
    onEnvVarChange,
    onApiKeyChange,
    commit,
  };
}
