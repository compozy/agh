import { useCallback, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useDeleteSettingsProvider,
  usePutSettingsProvider,
  useSettingsProviders,
  type SettingsMutationResult,
  type SettingsProviderEntry,
  type SettingsProviderRequest,
} from "@/systems/settings";

type ProviderCredentialSlotDraft = NonNullable<
  NonNullable<SettingsProviderRequest["settings"]>["credential_slots"]
>[number];
type ProviderModelsPayload = NonNullable<
  NonNullable<SettingsProviderRequest["settings"]>["models"]
>;
type ProviderModelPayload = NonNullable<ProviderModelsPayload["curated"]>[number];

export type ProviderDraft = {
  name: string;
  command: string;
  display_name: string;
  model_default: string;
  curated_models: string;
  curated_snapshot: ProviderModelPayload[];
  target_env: string;
  harness: string;
  runtime_provider: string;
  transport: string;
  base_url: string;
  auth_mode: string;
  env_policy: string;
  home_policy: string;
  auth_status_command: string;
  auth_login_command: string;
  secret_ref: string;
  secret_value: string;
  credential_slots: ProviderCredentialSlotDraft[];
  credential_secret_values: string[];
};

function providerCredentialsConfigured(provider: SettingsProviderEntry): boolean {
  const credentials = provider.credentials ?? [];
  if (credentials.length === 0) {
    return true;
  }
  return credentials.every(credential => !credential.required || credential.present);
}

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

export type ProviderEditorState =
  | { mode: "closed" }
  | { mode: "create"; draft: ProviderDraft }
  | { mode: "edit"; name: string; draft: ProviderDraft; entry: SettingsProviderEntry };

type DeleteState = { mode: "closed" } | { mode: "open"; entry: SettingsProviderEntry };

export type ProviderLastAction =
  | { kind: "saved"; name: string; result: SettingsMutationResult }
  | { kind: "deleted"; name: string; result: SettingsMutationResult; hadFallback: boolean };

type LastAction = ProviderLastAction | null;

export function useSettingsProvidersPage() {
  const query = useSettingsProviders();
  const putMutation = usePutSettingsProvider();
  const deleteMutation = useDeleteSettingsProvider();
  const page = useSettingsPage({ currentSlug: "providers" });

  const [editor, setEditor] = useState<ProviderEditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<DeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<LastAction>(null);

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

  const openCreate = useCallback(() => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft() });
  }, [putMutation]);

  const openEdit = useCallback(
    (entry: SettingsProviderEntry) => {
      putMutation.reset();
      setEditor({ mode: "edit", name: entry.name, draft: toDraft(entry), entry });
    },
    [putMutation]
  );

  const closeEditor = useCallback(() => {
    setEditor({ mode: "closed" });
    putMutation.reset();
  }, [putMutation]);

  const updateDraft = useCallback((updater: (draft: ProviderDraft) => ProviderDraft) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, draft: updater(current.draft) };
    });
  }, []);

  const editorIsValid = useMemo(() => {
    if (editor.mode === "closed") return false;
    const name = editor.draft.name.trim();
    if (name.length === 0) return false;
    if (
      editor.draft.auth_mode === "bound_secret" &&
      buildCredentialSlots(editor.draft).length === 0
    ) {
      return false;
    }
    if (editor.draft.secret_value.trim() && !editor.draft.secret_ref.trim().startsWith("vault:")) {
      return false;
    }
    if (editor.mode === "create") {
      return !providers.some(provider => provider.name.toLowerCase() === name.toLowerCase());
    }
    return true;
  }, [editor, providers]);

  const saveEditor = useCallback(() => {
    if (editor.mode === "closed") return;
    const name = editor.draft.name.trim();
    if (!name) return;
    const body = toRequest(editor.draft);
    putMutation.mutate(
      { name, body },
      {
        onSuccess: result => {
          setLastAction({ kind: "saved", name, result });
          setEditor({ mode: "closed" });
        },
      }
    );
  }, [editor, putMutation]);

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
      },
    });
  }, [deleteMutation, deleteTarget]);

  const dismissLastAction = useCallback(() => setLastAction(null), []);

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    providers,
    counts,
    restart: page.restart,
    editor,
    editorIsValid,
    editorError: errorMessage(putMutation.error),
    editorWarnings: putMutation.data?.warnings,
    editorIsSaving: putMutation.isPending,
    openCreate,
    openEdit,
    closeEditor,
    updateDraft,
    saveEditor,
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
