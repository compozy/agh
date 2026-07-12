import { isReasoningEffort, type ReasoningEffort } from "@/lib/api-contract";
import {
  providerNeedsAuth,
  useRuntimeModelCatalog,
  type RuntimeCatalogProvider,
} from "@/systems/model-catalog";
import { useProviders } from "@/systems/providers";
import type {
  RuntimeModelOption,
  RuntimeProviderOption,
  RuntimeSelectorValue,
} from "@/systems/runtime";
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
  providersLoading: boolean;
  providersError: string | null;
  runtimeValue: RuntimeSelectorValue;
  runtimeProviders: RuntimeProviderOption[];
  runtimeModels: RuntimeModelOption[];
  authMode: OnboardingAuthMode;
  envVar: string;
  apiKey: string;
  catalogLoading: boolean;
  catalogLoaded: boolean;
  catalogRefreshing: boolean;
  catalogError: string | null;
  configurationError: string | null;
  isValid: boolean;
  isCommitting: boolean;
  onRuntimeChange: (next: RuntimeSelectorValue) => void;
  onRefreshCatalog: () => void;
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

function normalizeEffort(effort: string): ReasoningEffort | "" {
  return effort === "" ? "" : isReasoningEffort(effort) ? effort : "";
}

function updateRuntime(next: RuntimeSelectorValue): void {
  const store = useOnboardingDraftStore.getState();
  const providerChanged = next.provider !== store.provider;
  store.patch({
    provider: next.provider,
    model: next.model,
    reasoning: normalizeEffort(next.reasoning_effort),
    ...(providerChanged ? { envVar: "", apiKey: "" } : {}),
  });
}

function updateAuthMode(authMode: OnboardingAuthMode): void {
  useOnboardingDraftStore
    .getState()
    .patch(authMode === "native_cli" ? { authMode, envVar: "", apiKey: "" } : { authMode });
}

function updateEnvVar(envVar: string): void {
  useOnboardingDraftStore.getState().patch({ envVar });
}

function updateApiKey(apiKey: string): void {
  useOnboardingDraftStore.getState().patch({ apiKey });
}

export function useOnboardingDefaultModel(): OnboardingDefaultModelApi {
  const draft = useOnboardingDraftStore();
  const providersQuery = useProviders();
  const providers = providersQuery.data?.providers ?? [];

  const provider = draft.provider;

  const generalQuery = useSettingsGeneral();
  const providerDetailQuery = useSettingsProvider(provider, { enabled: provider.length > 0 });
  const updateGeneral = useUpdateSettingsGeneral();
  const putProvider = usePutSettingsProvider();

  const existingApiKeyTargetEnv =
    providerDetailQuery.data?.settings.credential_slots
      ?.find(slot => slot.name === "api_key")
      ?.target_env.trim() ?? "";

  const runtimeProviders: RuntimeProviderOption[] = providers.map(entry => ({
    id: entry.name,
    name: entry.display_name?.trim() || entry.name,
    runtime_provider: entry.name,
    needs_auth: providerNeedsAuth(entry.auth_status?.state),
  }));

  // Onboarding lets the operator browse every configured provider's catalog via
  // the single aggregate query, filtered to the configured providers.
  const catalogProviders: RuntimeCatalogProvider[] = runtimeProviders.map(entry => ({
    id: entry.id,
    needsAuth: entry.needs_auth,
  }));
  const catalog = useRuntimeModelCatalog(catalogProviders, { enabled: true });
  const runtimeModels = catalog.models;

  const runtimeValue: RuntimeSelectorValue = {
    provider: draft.provider,
    model: draft.model,
    reasoning_effort: draft.reasoning,
  };

  const onRefreshCatalog = catalog.refresh;

  const commit = async () => {
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
      reasoning: draft.reasoning,
      authMode: draft.authMode,
      envVar: draft.envVar.trim(),
      apiKey: draft.apiKey.trim(),
      provider: trimmedProvider,
    });
    await putProvider.mutateAsync({ name: trimmedProvider, body });
    await updateGeneral.mutateAsync({
      config: { ...config, defaults: { ...config.defaults, provider: trimmedProvider } },
    });
  };

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
    providersLoading: providersQuery.isLoading,
    providersError: providersQuery.error
      ? describeError("Failed to load providers.", providersQuery.error)
      : null,
    runtimeValue,
    runtimeProviders,
    runtimeModels,
    authMode: draft.authMode,
    envVar: draft.envVar,
    apiKey: draft.apiKey,
    catalogLoading: catalog.loading,
    catalogLoaded: catalog.loaded,
    catalogRefreshing: catalog.refreshing,
    catalogError: catalog.error,
    configurationError,
    isValid: canCommit,
    isCommitting: putProvider.isPending || updateGeneral.isPending,
    onRuntimeChange: updateRuntime,
    onRefreshCatalog,
    onAuthModeChange: updateAuthMode,
    onEnvVarChange: updateEnvVar,
    onApiKeyChange: updateApiKey,
    commit,
  };
}
