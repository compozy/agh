import { useCallback, useMemo, useState } from "react";

import {
  useDeleteVaultSecret,
  usePutVaultSecret,
  useVaultSecrets,
  VAULT_NAMESPACES,
  VaultApiError,
  type VaultListFilter,
  type VaultNamespace,
  type VaultSecret,
} from "@/systems/vault";

export type VaultNamespaceFilter = VaultNamespace | "all";

export interface VaultDraft {
  ref: string;
  kind: string;
  secretValue: string;
}

export type VaultEditorState = { mode: "closed" } | { mode: "create"; draft: VaultDraft };
export type VaultDeleteState = { mode: "closed" } | { mode: "open"; secret: VaultSecret };
export type VaultLastAction =
  | { kind: "saved"; ref: string; secret: VaultSecret }
  | { kind: "deleted"; ref: string };

function emptyDraft(): VaultDraft {
  return {
    ref: "vault:sessions/",
    kind: "",
    secretValue: "",
  };
}

function errorMessage(error: unknown): string | null {
  if (error instanceof VaultApiError) return error.message;
  if (error instanceof Error) return error.message;
  return null;
}

function normalizePrefix(value: string): string {
  return value.trim();
}

function filterFor(namespace: VaultNamespaceFilter, prefix: string): VaultListFilter {
  const filter: VaultListFilter = {};
  if (namespace !== "all") {
    filter.namespace = namespace;
  }
  const normalizedPrefix = normalizePrefix(prefix);
  if (normalizedPrefix) {
    filter.prefix = normalizedPrefix;
  }
  return filter;
}

export function useVaultPage() {
  const [namespace, setNamespace] = useState<VaultNamespaceFilter>("all");
  const [prefix, setPrefix] = useState("");
  const [editor, setEditor] = useState<VaultEditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<VaultDeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<VaultLastAction | null>(null);

  const filter = useMemo(() => filterFor(namespace, prefix), [namespace, prefix]);
  const query = useVaultSecrets(filter);
  const putMutation = usePutVaultSecret();
  const deleteMutation = useDeleteVaultSecret();

  const secrets = query.data ?? [];
  const counts = useMemo(() => {
    const byNamespace = Object.fromEntries(VAULT_NAMESPACES.map(item => [item, 0])) as Record<
      VaultNamespace,
      number
    >;
    for (const secret of secrets) {
      if (secret.namespace in byNamespace) {
        byNamespace[secret.namespace as VaultNamespace] += 1;
      }
    }
    return {
      total: secrets.length,
      sessions: byNamespace.sessions,
      providers: byNamespace.providers,
      byNamespace,
    };
  }, [secrets]);

  const openCreate = useCallback(() => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft() });
  }, [putMutation]);

  const closeEditor = useCallback(() => {
    setEditor({ mode: "closed" });
    putMutation.reset();
  }, [putMutation]);

  const updateDraft = useCallback((updater: (draft: VaultDraft) => VaultDraft) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, draft: updater(current.draft) };
    });
  }, []);

  const editorIsValid = useMemo(() => {
    if (editor.mode === "closed") return false;
    return editor.draft.ref.trim().startsWith("vault:") && editor.draft.secretValue.trim() !== "";
  }, [editor]);

  const saveEditor = useCallback(() => {
    if (editor.mode === "closed" || !editorIsValid) return;
    const ref = editor.draft.ref.trim();
    const kind = editor.draft.kind.trim();
    putMutation.mutate(
      {
        ref,
        secret_value: editor.draft.secretValue,
        ...(kind ? { kind } : {}),
      },
      {
        onSuccess: secret => {
          setLastAction({ kind: "saved", ref, secret });
          setEditor({ mode: "closed" });
        },
      }
    );
  }, [editor, editorIsValid, putMutation]);

  const openDelete = useCallback(
    (secret: VaultSecret) => {
      deleteMutation.reset();
      setDeleteTarget({ mode: "open", secret });
    },
    [deleteMutation]
  );

  const closeDelete = useCallback(() => {
    setDeleteTarget({ mode: "closed" });
    deleteMutation.reset();
  }, [deleteMutation]);

  const confirmDelete = useCallback(() => {
    if (deleteTarget.mode !== "open") return;
    const ref = deleteTarget.secret.ref;
    deleteMutation.mutate(ref, {
      onSuccess: () => {
        setLastAction({ kind: "deleted", ref });
        setDeleteTarget({ mode: "closed" });
      },
    });
  }, [deleteMutation, deleteTarget]);

  const dismissLastAction = useCallback(() => {
    setLastAction(null);
  }, []);

  return {
    counts,
    deleteError: errorMessage(deleteMutation.error),
    deleteIsPending: deleteMutation.isPending,
    deleteTarget,
    dismissLastAction,
    editor,
    editorError: errorMessage(putMutation.error),
    editorIsSaving: putMutation.isPending,
    editorIsValid,
    filter,
    isLoading: query.isLoading,
    isRefetching: query.isFetching && !query.isLoading,
    lastAction,
    namespace,
    prefix,
    queryError: errorMessage(query.error),
    refetch: query.refetch,
    secrets,
    setNamespace,
    setPrefix,
    closeDelete,
    closeEditor,
    confirmDelete,
    openCreate,
    openDelete,
    saveEditor,
    updateDraft,
  };
}
