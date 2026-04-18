import { useCallback, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useDeleteSettingsMCPServer,
  usePutSettingsMCPServer,
  useSettingsMCPServers,
  type SettingsMCPServerEntry,
  type SettingsMCPServerRequest,
  type SettingsMCPServerTarget,
  type SettingsMutationResult,
  type SettingsScope,
} from "@/systems/settings";
import { useWorkspaces } from "@/systems/workspace";
import type { WorkspacePayload } from "@/systems/workspace";

export type MCPEnvPair = { key: string; value: string };

export type MCPDraft = {
  name: string;
  command: string;
  args: string[];
  env: MCPEnvPair[];
};

export type MCPScopeSelection = { scope: "global" } | { scope: "workspace"; workspaceId: string };

export type MCPEditorState =
  | { mode: "closed" }
  | {
      mode: "create";
      draft: MCPDraft;
      target: SettingsMCPServerTarget;
    }
  | {
      mode: "edit";
      name: string;
      draft: MCPDraft;
      entry: SettingsMCPServerEntry;
      target: SettingsMCPServerTarget;
    };

export type MCPDeleteState =
  | { mode: "closed" }
  | { mode: "open"; entry: SettingsMCPServerEntry; target: SettingsMCPServerTarget };

export type MCPLastAction =
  | { kind: "saved"; name: string; result: SettingsMutationResult }
  | {
      kind: "deleted";
      name: string;
      result: SettingsMutationResult;
      remainingShadowed: number;
    };

type LastAction = MCPLastAction | null;

function emptyDraft(): MCPDraft {
  return { name: "", command: "", args: [], env: [] };
}

function toDraft(entry: SettingsMCPServerEntry): MCPDraft {
  const env = entry.env ? Object.entries(entry.env).map(([key, value]) => ({ key, value })) : [];
  return {
    name: entry.name,
    command: entry.command,
    args: [...(entry.args ?? [])],
    env,
  };
}

function toRequest(draft: MCPDraft): SettingsMCPServerRequest {
  const name = draft.name.trim();
  const command = draft.command.trim();
  const args = draft.args.map(arg => arg.trim()).filter(arg => arg.length > 0);
  const envEntries = draft.env
    .map(entry => ({ key: entry.key.trim(), value: entry.value }))
    .filter(entry => entry.key.length > 0);
  const env: Record<string, string> = {};
  for (const entry of envEntries) {
    env[entry.key] = entry.value;
  }
  const server: SettingsMCPServerRequest["server"] = { name, command };
  if (args.length > 0) server.args = args;
  if (envEntries.length > 0) server.env = env;
  return { server };
}

function errorMessage(error: unknown): string | null {
  if (error instanceof SettingsApiError) return error.message;
  if (error instanceof Error) return error.message;
  return null;
}

function resolveAvailableTargets(
  entry: SettingsMCPServerEntry | null,
  scope: SettingsScope
): SettingsMCPServerTarget[] {
  const base: SettingsMCPServerTarget[] = ["auto", "config", "sidecar"];
  if (!entry) return base;
  const available = entry.source_metadata.available_targets;
  const hasConfig = available.some(target =>
    scope === "workspace"
      ? target === "workspace-config" || target === "global-config"
      : target === "global-config"
  );
  const hasSidecar = available.some(target =>
    scope === "workspace"
      ? target === "workspace-mcp-sidecar" || target === "global-mcp-sidecar"
      : target === "global-mcp-sidecar"
  );
  const result: SettingsMCPServerTarget[] = ["auto"];
  if (hasConfig) result.push("config");
  if (hasSidecar) result.push("sidecar");
  return result;
}

interface UseSettingsMCPServersPageOptions {
  initialScope?: MCPScopeSelection;
}

export function useSettingsMCPServersPage(options: UseSettingsMCPServersPageOptions = {}) {
  const page = useSettingsPage({ currentSlug: "mcp-servers" });
  const workspaceQuery = useWorkspaces();
  const workspaces: WorkspacePayload[] = workspaceQuery.data ?? [];

  const defaultSelection: MCPScopeSelection = options.initialScope ?? { scope: "global" };
  const [selection, setSelection] = useState<MCPScopeSelection>(defaultSelection);
  const filter = useMemo(
    () =>
      selection.scope === "workspace"
        ? { scope: "workspace" as const, workspace_id: selection.workspaceId }
        : { scope: "global" as const },
    [selection]
  );
  const query = useSettingsMCPServers(filter);
  const putMutation = usePutSettingsMCPServer();
  const deleteMutation = useDeleteSettingsMCPServer();

  const [editor, setEditor] = useState<MCPEditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<MCPDeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<LastAction>(null);

  const envelope = query.data ?? null;
  const servers = envelope?.mcp_servers ?? [];
  const availableScopes = envelope?.available_scopes ?? ["global"];

  const counts = useMemo(() => {
    const total = servers.length;
    const shadowed = servers.reduce(
      (acc, entry) => acc + (entry.source_metadata.shadowed_sources?.length ?? 0),
      0
    );
    return { total, shadowed };
  }, [servers]);

  const selectedWorkspace = useMemo(
    () =>
      selection.scope === "workspace"
        ? (workspaces.find(workspace => workspace.id === selection.workspaceId) ?? null)
        : null,
    [selection, workspaces]
  );

  const selectGlobal = useCallback(() => {
    putMutation.reset();
    deleteMutation.reset();
    setSelection({ scope: "global" });
    setEditor({ mode: "closed" });
    setDeleteTarget({ mode: "closed" });
  }, [deleteMutation, putMutation]);

  const selectWorkspace = useCallback(
    (workspaceId: string) => {
      putMutation.reset();
      deleteMutation.reset();
      setSelection({ scope: "workspace", workspaceId });
      setEditor({ mode: "closed" });
      setDeleteTarget({ mode: "closed" });
    },
    [deleteMutation, putMutation]
  );

  const openCreate = useCallback(() => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft(), target: "auto" });
  }, [putMutation]);

  const openEdit = useCallback(
    (entry: SettingsMCPServerEntry) => {
      putMutation.reset();
      setEditor({
        mode: "edit",
        name: entry.name,
        draft: toDraft(entry),
        entry,
        target: "auto",
      });
    },
    [putMutation]
  );

  const closeEditor = useCallback(() => {
    setEditor({ mode: "closed" });
    putMutation.reset();
  }, [putMutation]);

  const updateDraft = useCallback((updater: (draft: MCPDraft) => MCPDraft) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, draft: updater(current.draft) };
    });
  }, []);

  const setEditorTarget = useCallback((target: SettingsMCPServerTarget) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, target };
    });
  }, []);

  const editorIsValid = useMemo(() => {
    if (editor.mode === "closed") return false;
    const name = editor.draft.name.trim();
    const command = editor.draft.command.trim();
    if (name.length === 0 || command.length === 0) return false;
    if (editor.mode === "create") {
      return !servers.some(entry => entry.name.toLowerCase() === name.toLowerCase());
    }
    return true;
  }, [editor, servers]);

  const editorAvailableTargets = useMemo<SettingsMCPServerTarget[]>(() => {
    if (editor.mode === "closed") return ["auto", "config", "sidecar"];
    if (editor.mode === "create") return ["auto", "config", "sidecar"];
    return resolveAvailableTargets(editor.entry, selection.scope);
  }, [editor, selection]);

  const saveEditor = useCallback(() => {
    if (editor.mode === "closed") return;
    const name = editor.draft.name.trim();
    const command = editor.draft.command.trim();
    if (!name || !command) return;
    const body = toRequest(editor.draft);
    const target = editor.target;
    const filterPayload =
      selection.scope === "workspace"
        ? { scope: "workspace" as const, workspace_id: selection.workspaceId, target }
        : { scope: "global" as const, target };
    putMutation.mutate(
      { name, body, filter: filterPayload },
      {
        onSuccess: result => {
          setLastAction({ kind: "saved", name, result });
          setEditor({ mode: "closed" });
        },
      }
    );
  }, [editor, putMutation, selection]);

  const openDelete = useCallback(
    (entry: SettingsMCPServerEntry) => {
      deleteMutation.reset();
      setDeleteTarget({ mode: "open", entry, target: "auto" });
    },
    [deleteMutation]
  );

  const closeDelete = useCallback(() => {
    setDeleteTarget({ mode: "closed" });
    deleteMutation.reset();
  }, [deleteMutation]);

  const setDeleteTargetKind = useCallback((target: SettingsMCPServerTarget) => {
    setDeleteTarget(current => {
      if (current.mode === "closed") return current;
      return { ...current, target };
    });
  }, []);

  const deleteAvailableTargets = useMemo<SettingsMCPServerTarget[]>(() => {
    if (deleteTarget.mode === "closed") return ["auto", "config", "sidecar"];
    return resolveAvailableTargets(deleteTarget.entry, selection.scope);
  }, [deleteTarget, selection]);

  const confirmDelete = useCallback(() => {
    if (deleteTarget.mode === "closed") return;
    const target = deleteTarget.entry;
    const deleteFilter =
      selection.scope === "workspace"
        ? {
            scope: "workspace" as const,
            workspace_id: selection.workspaceId,
            target: deleteTarget.target,
          }
        : { scope: "global" as const, target: deleteTarget.target };
    const remainingShadowed = target.source_metadata.shadowed_sources?.length ?? 0;
    deleteMutation.mutate(
      { name: target.name, filter: deleteFilter },
      {
        onSuccess: result => {
          setLastAction({
            kind: "deleted",
            name: target.name,
            result,
            remainingShadowed,
          });
          setDeleteTarget({ mode: "closed" });
        },
      }
    );
  }, [deleteMutation, deleteTarget, selection]);

  const dismissLastAction = useCallback(() => setLastAction(null), []);

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    servers,
    counts,
    restart: page.restart,
    selection,
    selectedWorkspace,
    workspaces,
    workspacesLoading: workspaceQuery.isLoading,
    availableScopes,
    selectGlobal,
    selectWorkspace,
    editor,
    editorIsValid,
    editorAvailableTargets,
    editorError: errorMessage(putMutation.error),
    editorWarnings: putMutation.data?.warnings,
    editorIsSaving: putMutation.isPending,
    openCreate,
    openEdit,
    closeEditor,
    updateDraft,
    setEditorTarget,
    saveEditor,
    deleteTarget,
    deleteAvailableTargets,
    deleteError: errorMessage(deleteMutation.error),
    deleteIsPending: deleteMutation.isPending,
    openDelete,
    closeDelete,
    setDeleteTargetKind,
    confirmDelete,
    lastAction,
    dismissLastAction,
  };
}
