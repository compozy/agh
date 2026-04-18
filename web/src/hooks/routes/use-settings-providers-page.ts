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

export type ProviderDraft = {
  name: string;
  command: string;
  default_model: string;
  api_key_env: string;
};

function emptyDraft(): ProviderDraft {
  return { name: "", command: "", default_model: "", api_key_env: "" };
}

function toDraft(entry: SettingsProviderEntry): ProviderDraft {
  return {
    name: entry.name,
    command: entry.settings.command ?? "",
    default_model: entry.settings.default_model ?? "",
    api_key_env: entry.settings.api_key_env ?? "",
  };
}

function toRequest(draft: ProviderDraft): SettingsProviderRequest {
  const settings: SettingsProviderRequest["settings"] = {};
  if (draft.command.trim()) settings.command = draft.command.trim();
  if (draft.default_model.trim()) settings.default_model = draft.default_model.trim();
  if (draft.api_key_env.trim()) settings.api_key_env = draft.api_key_env.trim();
  return { settings };
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
      provider => provider.command_available && provider.api_key_env_present
    ).length;
    const binaryMissing = providers.filter(provider => !provider.command_available).length;
    const unconfigured = providers.filter(
      provider => provider.command_available && !provider.api_key_env_present
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
