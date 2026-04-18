import { useCallback, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useDeleteSettingsEnvironment,
  usePutSettingsEnvironment,
  useSettingsEnvironments,
  type SettingsEnvironmentEntry,
  type SettingsEnvironmentRequest,
  type SettingsMutationResult,
} from "@/systems/settings";

type Profile = SettingsEnvironmentRequest["profile"];

export type EnvironmentDraft = {
  name: string;
  backend: string;
  sync_mode: string;
  persistence: string;
  runtime_root: string;
  preserved: Omit<Profile, "backend" | "sync_mode" | "persistence" | "runtime_root">;
};

function emptyDraft(): EnvironmentDraft {
  return {
    name: "",
    backend: "local",
    sync_mode: "",
    persistence: "",
    runtime_root: "",
    preserved: {},
  };
}

function toDraft(entry: SettingsEnvironmentEntry): EnvironmentDraft {
  const { backend, sync_mode, persistence, runtime_root, ...preserved } = entry.profile;
  return {
    name: entry.name,
    backend,
    sync_mode: sync_mode ?? "",
    persistence: persistence ?? "",
    runtime_root: runtime_root ?? "",
    preserved,
  };
}

function toRequest(draft: EnvironmentDraft): SettingsEnvironmentRequest {
  const profile: Profile = { backend: draft.backend.trim(), ...draft.preserved };
  if (draft.sync_mode.trim()) profile.sync_mode = draft.sync_mode.trim();
  if (draft.persistence.trim()) profile.persistence = draft.persistence.trim();
  if (draft.runtime_root.trim()) profile.runtime_root = draft.runtime_root.trim();
  return { profile };
}

function errorMessage(error: unknown): string | null {
  if (error instanceof SettingsApiError) return error.message;
  if (error instanceof Error) return error.message;
  return null;
}

export type EnvironmentEditorState =
  | { mode: "closed" }
  | { mode: "create"; draft: EnvironmentDraft }
  | {
      mode: "edit";
      name: string;
      draft: EnvironmentDraft;
      entry: SettingsEnvironmentEntry;
    };

type DeleteState = { mode: "closed" } | { mode: "open"; entry: SettingsEnvironmentEntry };

export type EnvironmentLastAction =
  | { kind: "saved"; name: string; result: SettingsMutationResult }
  | {
      kind: "deleted";
      name: string;
      result: SettingsMutationResult;
      usageCount: number;
    };

type LastAction = EnvironmentLastAction | null;

export function useSettingsEnvironmentsPage() {
  const query = useSettingsEnvironments();
  const putMutation = usePutSettingsEnvironment();
  const deleteMutation = useDeleteSettingsEnvironment();
  const page = useSettingsPage({ currentSlug: "environments" });

  const [editor, setEditor] = useState<EnvironmentEditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<DeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<LastAction>(null);

  const envelope = query.data ?? null;
  const environments = envelope?.environments ?? [];

  const counts = useMemo(() => {
    const total = environments.length;
    const totalWorkspaces = environments.reduce(
      (acc, entry) => acc + entry.workspace_usage_count,
      0
    );
    return { total, totalWorkspaces };
  }, [environments]);

  const openCreate = useCallback(() => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft() });
  }, [putMutation]);

  const openEdit = useCallback(
    (entry: SettingsEnvironmentEntry) => {
      putMutation.reset();
      setEditor({ mode: "edit", name: entry.name, draft: toDraft(entry), entry });
    },
    [putMutation]
  );

  const closeEditor = useCallback(() => {
    setEditor({ mode: "closed" });
    putMutation.reset();
  }, [putMutation]);

  const updateDraft = useCallback((updater: (draft: EnvironmentDraft) => EnvironmentDraft) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, draft: updater(current.draft) };
    });
  }, []);

  const editorIsValid = useMemo(() => {
    if (editor.mode === "closed") return false;
    const name = editor.draft.name.trim();
    if (name.length === 0) return false;
    if (editor.draft.backend.trim().length === 0) return false;
    if (editor.mode === "create") {
      return !environments.some(entry => entry.name.toLowerCase() === name.toLowerCase());
    }
    return true;
  }, [editor, environments]);

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
    (entry: SettingsEnvironmentEntry) => {
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
          usageCount: target.workspace_usage_count,
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
    environments,
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
