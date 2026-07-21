import { useState } from "react";

import { useSettingsPage } from "./use-settings-page";
import {
  SettingsApiError,
  useDeleteSettingsProvider,
  usePutSettingsProvider,
  useSettingsProviders,
  type ProviderDraft,
  type SettingsMutationResult,
  type SettingsProviderEntry,
  type SettingsProviderModelRequest,
  type SettingsProviderRequest,
} from "@/systems/settings";
import {
  applyProviderFilters,
  DEFAULT_PROVIDER_FILTERS,
  type ProviderFilterState,
} from "@/systems/settings/lib/providers-list-filters";
import {
  deriveProviderStateLabel,
  type ProviderStateLabel,
} from "@/systems/settings/lib/provider-state";

type ProviderCredentialSlotDraft = ProviderDraft["credential_slots"][number];

function emptyDraft(): ProviderDraft {
  return {
    name: "",
    command: "",
    display_name: "",
    model_default: "",
    curated_models: "",
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
  return {
    name: entry.name,
    command: entry.settings.command ?? "",
    display_name: entry.settings.display_name ?? "",
    model_default: entry.settings.models?.default ?? "",
    curated_models: joinCuratedModels(entry.settings.models?.curated ?? []),
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
    curated: parseCuratedModels(draft.curated_models),
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
    const value = credentialSecretValue(draft, index);
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

function credentialSecretValue(draft: ProviderDraft, index: number): string {
  return index === 0 ? draft.secret_value : (draft.credential_secret_values[index] ?? "");
}

function envSecretRef(apiKeyEnv?: string): string {
  const envName = apiKeyEnv?.trim();
  return envName ? `env:${envName}` : "";
}

function joinCuratedModels(models: SettingsProviderModelRequest[]): string {
  return models
    .flatMap(model => {
      const id = model.id.trim();
      return id ? [id] : [];
    })
    .join("\n");
}

function parseCuratedModels(raw: string): SettingsProviderModelRequest[] {
  const seen = new Set<string>();
  const models: SettingsProviderModelRequest[] = [];
  for (const part of raw.split(/[\n,]/u)) {
    const id = part.trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    models.push({ id });
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

function isProviderInspectorValid(
  inspector: ProviderInspectorState,
  providers: readonly SettingsProviderEntry[]
): boolean {
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
    buildCredentialSlots(inspector.draft).some(
      (slot, index) =>
        credentialSecretValue(inspector.draft, index).trim().length > 0 &&
        !slot.secret_ref.startsWith("vault:")
    )
  ) {
    return false;
  }
  return (
    inspector.mode !== "create" ||
    !providers.some(provider => provider.name.toLowerCase() === name.toLowerCase())
  );
}

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

  const providerStates = providers.map(deriveProviderStateLabel);
  const counts = {
    total: providers.length,
    installed: providerStates.filter(state => state === "installed").length,
    binaryMissing: providerStates.filter(state => state === "binary-missing").length,
    needsSetup: providerStates.filter(state => state !== "installed" && state !== "binary-missing")
      .length,
  };

  const filteredProviders = applyProviderFilters(providers, filters);

  const setStatusFilter = (next: ProviderStateLabel | null) => {
    setFilters(current => ({ ...current, statusFilter: next }));
  };
  const setNameQuery = (next: string) => {
    setFilters(current => ({ ...current, nameQuery: next }));
  };

  const openInspect = (entry: SettingsProviderEntry) => {
    putMutation.reset();
    setInspector({ mode: "inspect", entry });
  };

  const openCreate = () => {
    putMutation.reset();
    setInspector({ mode: "create", draft: emptyDraft() });
  };

  const switchToEdit = () => {
    setInspector(current => {
      if (current.mode !== "inspect") return current;
      return {
        mode: "edit",
        entry: current.entry,
        draft: toDraft(current.entry),
        cameFrom: "inspect",
      };
    });
  };

  const cancelEdit = () => {
    putMutation.reset();
    setInspector(current => {
      if (current.mode === "edit" && current.cameFrom === "inspect") {
        return { mode: "inspect", entry: current.entry };
      }
      return { mode: "closed" };
    });
  };

  const closeInspector = () => {
    setInspector({ mode: "closed" });
    putMutation.reset();
  };

  const updateDraft = (updater: (draft: ProviderDraft) => ProviderDraft) => {
    setInspector(current => {
      if (current.mode !== "edit" && current.mode !== "create") return current;
      return { ...current, draft: updater(current.draft) };
    });
  };

  const inspectorIsValid = isProviderInspectorValid(inspector, providers);

  const saveInspector = () => {
    if (inspector.mode !== "edit" && inspector.mode !== "create") return;
    if (!inspectorIsValid) return;
    const name = inspector.draft.name.trim();
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
  };

  const openDelete = (entry: SettingsProviderEntry) => {
    deleteMutation.reset();
    setDeleteTarget({ mode: "open", entry });
  };

  const closeDelete = () => {
    setDeleteTarget({ mode: "closed" });
    deleteMutation.reset();
  };

  const confirmDelete = () => {
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
  };

  const dismissLastAction = () => setLastAction(null);

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    providers,
    filteredProviders,
    filters,
    setStatusFilter,
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
