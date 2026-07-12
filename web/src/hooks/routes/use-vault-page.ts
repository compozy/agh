import { useState } from "react";

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

function countVaultSecrets(secrets: VaultSecret[]) {
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
}

export function useVaultPage() {
  const [namespace, setNamespace] = useState<VaultNamespaceFilter>("all");
  const [prefix, setPrefix] = useState("");
  const [editor, setEditor] = useState<VaultEditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<VaultDeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<VaultLastAction | null>(null);

  const filter = filterFor(namespace, prefix);
  const query = useVaultSecrets(filter);
  const putMutation = usePutVaultSecret();
  const deleteMutation = useDeleteVaultSecret();

  const secrets = query.data ?? [];
  const counts = countVaultSecrets(secrets);

  const openCreate = () => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft() });
  };

  const closeEditor = () => {
    setEditor({ mode: "closed" });
    putMutation.reset();
  };

  const updateDraft = (updater: (draft: VaultDraft) => VaultDraft) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, draft: updater(current.draft) };
    });
  };

  const editorIsValid =
    editor.mode !== "closed" &&
    editor.draft.ref.trim().startsWith("vault:") &&
    editor.draft.secretValue.trim() !== "";

  const saveEditor = () => {
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
  };

  const openDelete = (secret: VaultSecret) => {
    deleteMutation.reset();
    setDeleteTarget({ mode: "open", secret });
  };

  const closeDelete = () => {
    setDeleteTarget({ mode: "closed" });
    deleteMutation.reset();
  };

  const confirmDelete = () => {
    if (deleteTarget.mode !== "open") return;
    const ref = deleteTarget.secret.ref;
    deleteMutation.mutate(ref, {
      onSuccess: () => {
        setLastAction({ kind: "deleted", ref });
        setDeleteTarget({ mode: "closed" });
      },
    });
  };

  const dismissLastAction = () => {
    setLastAction(null);
  };

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
