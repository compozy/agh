import { useCallback, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useDeleteSettingsProvider,
  usePutSettingsProvider,
  useSettingsProviders,
  type ProviderDraft,
  type SettingsMutationResult,
  type SettingsProviderEntry,
  type SettingsProviderRequest,
  type SettingsSourceKind,
} from "@/systems/settings";
import {
  applyProviderFilters,
  DEFAULT_PROVIDER_FILTERS,
  providerCredentialsConfigured,
  type ProviderAuthMode,
  type ProviderDefaultFilter,
  type ProviderFilterState,
  type ProviderHarness,
} from "@/systems/settings/lib/providers-list-filters";
import type { ProviderStateLabel } from "@/systems/settings/lib/provider-state";

type ProviderCredentialSlotDraft = ProviderDraft["credential_slots"][number];
type ProviderModelPayload = ProviderDraft["curated_snapshot"][number];

function emptyDraft(): ProviderDraft {
  return {
    name: "",
    command: "",
    display_name: "",
    model_default: "",
    curated_models: "",
    curated_snapshot: [],
    target_env: "",
    harness: "acp",
    runtime_provider: "",
    transport: "",
    base_url: "",
    auth_mode: "native_cli",
    env_policy: "filtered",
    home_policy: "operator",
    auth_status_command: "",
    auth_login_command: "",
    secret_ref: "",
    secret_value: "",
    credential_slots: [],
    credential_secret_values: [],
  };
}

function toDraft(entry: SettingsProviderEntry): ProviderDraft {
  const credentialSlots = credentialSlotsForDraft(entry.settings.credential_slots ?? []);
  const credentialSlot = credentialSlots[0];
  const curatedSnapshot = (entry.settings.models?.curated ?? []).map(model => ({ ...model }));
  return {
    name: entry.name,
    command: entry.settings.command ?? "",
    display_name: entry.settings.display_name ?? "",
    model_default: entry.settings.models?.default ?? "",
    curated_models: joinCuratedModels(curatedSnapshot),
    curated_snapshot: curatedSnapshot,
    target_env: credentialSlot?.target_env ?? "",
    harness: entry.settings.harness ?? "acp",
    runtime_provider: entry.settings.runtime_provider ?? "",
    transport: entry.settings.transport ?? "",
    base_url: entry.settings.base_url ?? "",
    auth_mode: entry.settings.auth_mode ?? "native_cli",
    env_policy: entry.settings.env_policy ?? "filtered",
    home_policy: entry.settings.home_policy ?? "operator",
    auth_status_command: entry.settings.auth_status_command ?? "",
    auth_login_command: entry.settings.auth_login_command ?? "",
    secret_ref: credentialSlot?.secret_ref ?? envSecretRef(credentialSlot?.target_env),
    secret_value: "",
    credential_slots: credentialSlots,
    credential_secret_values: credentialSlots.map(() => ""),
  };
}

function toRequest(draft: ProviderDraft): SettingsProviderRequest {
  const settings: SettingsProviderRequest["settings"] = {};
  if (draft.command.trim()) settings.command = draft.command.trim();
  if (draft.display_name.trim()) settings.display_name = draft.display_name.trim();
  settings.models = {
    ...(draft.model_default.trim() ? { default: draft.model_default.trim() } : {}),
    curated: parseCuratedModels(draft.curated_models, draft.curated_snapshot),
  };
  if (draft.harness.trim()) settings.harness = draft.harness.trim();
  if (draft.runtime_provider.trim()) settings.runtime_provider = draft.runtime_provider.trim();
  if (draft.transport.trim()) settings.transport = draft.transport.trim();
  if (draft.base_url.trim()) settings.base_url = draft.base_url.trim();
  if (draft.auth_mode.trim()) settings.auth_mode = draft.auth_mode.trim();
  if (draft.env_policy.trim()) settings.env_policy = draft.env_policy.trim();
  if (draft.home_policy.trim()) settings.home_policy = draft.home_policy.trim();
  if (draft.auth_status_command.trim()) {
    settings.auth_status_command = draft.auth_status_command.trim();
  }
  if (draft.auth_login_command.trim())
    settings.auth_login_command = draft.auth_login_command.trim();

  const credentialSlots = buildCredentialSlots(draft);
  if (credentialSlots.length > 0) {
    settings.credential_slots = credentialSlots;
  }

  const secrets: SettingsProviderRequest["secrets"] = [];
  for (const [index, credential] of credentialSlots.entries()) {
    const value = index === 0 ? draft.secret_value : (draft.credential_secret_values[index] ?? "");
    if (!value.trim() || !credential.secret_ref.startsWith("vault:")) {
      continue;
    }
    secrets.push({
      name: credential.name,
      secret_ref: credential.secret_ref,
      kind: credential.kind ?? "api_key",
      value,
    });
  }

  return secrets.length > 0 ? { settings, secrets } : { settings };
}

function envSecretRef(apiKeyEnv?: string): string {
  const envName = apiKeyEnv?.trim();
  return envName ? `env:${envName}` : "";
}

function joinCuratedModels(models: ProviderModelPayload[]): string {
  return models
    .map(model => model.id.trim())
    .filter(Boolean)
    .join("\n");
}

function parseCuratedModels(raw: string, snapshot: ProviderModelPayload[]): ProviderModelPayload[] {
  const seen = new Set<string>();
  const models: ProviderModelPayload[] = [];
  const snapshotById = new Map(
    snapshot.filter(entry => entry.id.trim().length > 0).map(entry => [entry.id.trim(), entry])
  );
  for (const part of raw.split(/[\n,]/u)) {
    const id = part.trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    const enrichment = snapshotById.get(id);
    if (enrichment) {
      models.push({ ...enrichment, id });
    } else {
      models.push({ id });
    }
  }
  return models;
}

function credentialSlotsForDraft(
  slots: ProviderCredentialSlotDraft[]
): ProviderCredentialSlotDraft[] {
  return slots.map(slot => ({ ...slot }));
}

function buildCredentialSlots(draft: ProviderDraft): ProviderCredentialSlotDraft[] {
  if (draft.auth_mode !== "bound_secret") {
    return [];
  }
  const primarySlot = normalizeCredentialSlot(draft.credential_slots[0]);
  const additionalSlots = draft.credential_slots
    .slice(1)
    .map(normalizeCredentialSlot)
    .filter(slot => slot !== null);
  const targetEnv = draft.target_env.trim();
  const secretRef = draft.secret_ref.trim() || envSecretRef(targetEnv);
  if (!targetEnv || !secretRef) {
    return additionalSlots;
  }

  return [
    {
      name: primarySlot?.name.trim() || "api_key",
      target_env: targetEnv,
      secret_ref: secretRef,
      kind: primarySlot?.kind?.trim() || "api_key",
      required: primarySlot?.required ?? true,
    },
    ...additionalSlots,
  ];
}

function normalizeCredentialSlot(
  slot: ProviderCredentialSlotDraft | undefined
): ProviderCredentialSlotDraft | null {
  if (!slot) {
    return null;
  }
  const name = slot.name.trim();
  const targetEnv = slot.target_env.trim();
  const secretRef = slot.secret_ref.trim();
  if (!name || !targetEnv || !secretRef) {
    return null;
  }
  const kind = slot.kind?.trim();
  return {
    name,
    target_env: targetEnv,
    secret_ref: secretRef,
    ...(kind ? { kind } : {}),
    required: slot.required,
  };
}

function errorMessage(error: unknown): string | null {
  if (error instanceof SettingsApiError) return error.message;
  if (error instanceof Error) return error.message;
  return null;
}

export type ProviderInspectorState =
  | { mode: "closed" }
  | { mode: "inspect"; entry: SettingsProviderEntry }
  | {
      mode: "edit";
      entry: SettingsProviderEntry;
      draft: ProviderDraft;
      cameFrom: "inspect" | "external";
    }
  | { mode: "create"; draft: ProviderDraft };

type DeleteState = { mode: "closed" } | { mode: "open"; entry: SettingsProviderEntry };

export type ProviderLastAction =
  | { kind: "saved"; name: string; result: SettingsMutationResult }
  | { kind: "deleted"; name: string; result: SettingsMutationResult; hadFallback: boolean };

type LastAction = ProviderLastAction | null;

export type { ProviderDraft };

export function useSettingsProvidersPage() {
  const query = useSettingsProviders();
  const putMutation = usePutSettingsProvider();
  const deleteMutation = useDeleteSettingsProvider();
  const page = useSettingsPage({ currentSlug: "providers" });

  const [inspector, setInspector] = useState<ProviderInspectorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<DeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<LastAction>(null);
  const [filters, setFilters] = useState<ProviderFilterState>(DEFAULT_PROVIDER_FILTERS);

  const envelope = query.data ?? null;
  const providers = envelope?.providers ?? [];

  const counts = useMemo(() => {
    const installed = providers.filter(
      provider => provider.command_available && providerCredentialsConfigured(provider)
    ).length;
    const binaryMissing = providers.filter(provider => !provider.command_available).length;
    const unconfigured = providers.filter(
      provider => provider.command_available && !providerCredentialsConfigured(provider)
    ).length;
    return { total: providers.length, installed, binaryMissing, unconfigured };
  }, [providers]);

  const filteredProviders = useMemo(
    () => applyProviderFilters(providers, filters),
    [providers, filters]
  );

  const setStatusFilter = useCallback((next: ProviderStateLabel | null) => {
    setFilters(current => ({ ...current, statusFilter: next }));
  }, []);
  const setSourceFilter = useCallback((next: SettingsSourceKind | null) => {
    setFilters(current => ({ ...current, sourceFilter: next }));
  }, []);
  const setHarnessFilter = useCallback((next: ProviderHarness | null) => {
    setFilters(current => ({ ...current, harnessFilter: next }));
  }, []);
  const setAuthModeFilter = useCallback((next: ProviderAuthMode | null) => {
    setFilters(current => ({ ...current, authModeFilter: next }));
  }, []);
  const setDefaultFilter = useCallback((next: ProviderDefaultFilter | null) => {
    setFilters(current => ({ ...current, defaultFilter: next }));
  }, []);
  const setNameQuery = useCallback((next: string) => {
    setFilters(current => ({ ...current, nameQuery: next }));
  }, []);

  const openInspect = useCallback(
    (entry: SettingsProviderEntry) => {
      putMutation.reset();
      setInspector({ mode: "inspect", entry });
    },
    [putMutation]
  );

  const openCreate = useCallback(() => {
    putMutation.reset();
    setInspector({ mode: "create", draft: emptyDraft() });
  }, [putMutation]);

  const switchToEdit = useCallback(() => {
    setInspector(current => {
      if (current.mode !== "inspect") return current;
      return {
        mode: "edit",
        entry: current.entry,
        draft: toDraft(current.entry),
        cameFrom: "inspect",
      };
    });
  }, []);

  const cancelEdit = useCallback(() => {
    putMutation.reset();
    setInspector(current => {
      if (current.mode === "edit" && current.cameFrom === "inspect") {
        return { mode: "inspect", entry: current.entry };
      }
      return { mode: "closed" };
    });
  }, [putMutation]);

  const closeInspector = useCallback(() => {
    setInspector({ mode: "closed" });
    putMutation.reset();
  }, [putMutation]);

  const updateDraft = useCallback((updater: (draft: ProviderDraft) => ProviderDraft) => {
    setInspector(current => {
      if (current.mode !== "edit" && current.mode !== "create") return current;
      return { ...current, draft: updater(current.draft) };
    });
  }, []);

  const inspectorIsValid = useMemo(() => {
    if (inspector.mode !== "edit" && inspector.mode !== "create") return false;
    const name = inspector.draft.name.trim();
    if (name.length === 0) return false;
    if (
      inspector.draft.auth_mode === "bound_secret" &&
      buildCredentialSlots(inspector.draft).length === 0
    ) {
      return false;
    }
    if (
      inspector.draft.secret_value.trim() &&
      !inspector.draft.secret_ref.trim().startsWith("vault:")
    ) {
      return false;
    }
    if (inspector.mode === "create") {
      return !providers.some(provider => provider.name.toLowerCase() === name.toLowerCase());
    }
    return true;
  }, [inspector, providers]);

  const saveInspector = useCallback(() => {
    if (inspector.mode !== "edit" && inspector.mode !== "create") return;
    const name = inspector.draft.name.trim();
    if (!name) return;
    const body = toRequest(inspector.draft);
    putMutation.mutate(
      { name, body },
      {
        onSuccess: result => {
          setLastAction({ kind: "saved", name, result });
          setInspector({ mode: "closed" });
        },
      }
    );
  }, [inspector, putMutation]);

  const openDelete = useCallback(
    (entry: SettingsProviderEntry) => {
      deleteMutation.reset();
      setDeleteTarget({ mode: "open", entry });
    },
    [deleteMutation]
  );

  const closeDelete = useCallback(() => {
    setDeleteTarget({ mode: "closed" });
    deleteMutation.reset();
  }, [deleteMutation]);

  const confirmDelete = useCallback(() => {
    if (deleteTarget.mode === "closed") return;
    const target = deleteTarget.entry;
    deleteMutation.mutate(target.name, {
      onSuccess: result => {
        setLastAction({
          kind: "deleted",
          name: target.name,
          result,
          hadFallback: Boolean(target.fallback),
        });
        setDeleteTarget({ mode: "closed" });
        setInspector(current =>
          current.mode === "inspect" && current.entry.name === target.name
            ? { mode: "closed" }
            : current
        );
      },
    });
  }, [deleteMutation, deleteTarget]);

  const dismissLastAction = useCallback(() => setLastAction(null), []);

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    providers,
    filteredProviders,
    filters,
    setStatusFilter,
    setSourceFilter,
    setHarnessFilter,
    setAuthModeFilter,
    setDefaultFilter,
    setNameQuery,
    counts,
    restart: page.restart,
    inspector,
    inspectorIsValid,
    inspectorError: errorMessage(putMutation.error),
    inspectorWarnings: putMutation.data?.warnings,
    inspectorIsSaving: putMutation.isPending,
    openInspect,
    openCreate,
    switchToEdit,
    cancelEdit,
    closeInspector,
    updateDraft,
    saveInspector,
    deleteTarget,
    deleteError: errorMessage(deleteMutation.error),
    deleteIsPending: deleteMutation.isPending,
    openDelete,
    closeDelete,
    confirmDelete,
    lastAction,
    dismissLastAction,
  };
}
