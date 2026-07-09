import { useCallback, useEffect, useMemo, useRef, useState } from "react";

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
import { useActiveWorkspace } from "@/systems/workspace";

export type MCPEnvPair = { key: string; value: string };

export type MCPDraft = {
  name: string;
  command: string;
  args: string[];
  env: MCPEnvPair[];
};

export type MCPActiveScope = "workspace" | "global";

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
    command: entry.command ?? "",
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
  const server: SettingsMCPServerRequest["server"] = { name, transport: "stdio", command };
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

interface UseMcpPageOptions {
  initialScope?: MCPActiveScope;
}

export function useMcpPage(options: UseMcpPageOptions = {}) {
  const page = useSettingsPage({ currentSlug: "mcp-servers" });
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [activeScope, setActiveScope] = useState<MCPActiveScope>(
    options.initialScope ?? "workspace"
  );
  const putMutation = usePutSettingsMCPServer();
  const deleteMutation = useDeleteSettingsMCPServer();

  const [editor, setEditor] = useState<MCPEditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<MCPDeleteState>({ mode: "closed" });
  const [lastAction, setLastAction] = useState<LastAction>(null);

  const filter = useMemo(() => {
    if (activeScope === "global") {
      return { scope: "global" as const };
    }
    if (!activeWorkspaceId) {
      return null;
    }
    return { scope: "workspace" as const, workspace_id: activeWorkspaceId };
  }, [activeScope, activeWorkspaceId]);

  const queryEnabled = filter !== null;
  const query = useSettingsMCPServers(filter ?? { scope: "global" }, { enabled: queryEnabled });

  const envelope = queryEnabled ? (query.data ?? null) : null;
  const servers = envelope?.mcp_servers ?? [];
  const availableScopes = envelope?.available_scopes ?? ["global", "workspace"];
  const workspaceScopeAvailable = availableScopes.includes("workspace");

  const counts = useMemo(() => {
    const total = servers.length;
    const shadowed = servers.reduce(
      (acc, entry) => acc + (entry.source_metadata.shadowed_sources?.length ?? 0),
      0
    );
    return { total, shadowed };
  }, [servers]);

  const resetTransientState = useCallback(() => {
    putMutation.reset();
    deleteMutation.reset();
    setEditor({ mode: "closed" });
    setDeleteTarget({ mode: "closed" });
  }, [deleteMutation, putMutation]);

  const selectScope = useCallback(
    (scope: MCPActiveScope) => {
      resetTransientState();
      setActiveScope(scope);
    },
    [resetTransientState]
  );

  const previousWorkspaceIdRef = useRef(activeWorkspaceId);
  useEffect(() => {
    if (previousWorkspaceIdRef.current === activeWorkspaceId) {
      return;
    }
    previousWorkspaceIdRef.current = activeWorkspaceId;
    resetTransientState();
  }, [activeWorkspaceId, resetTransientState]);

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
    return resolveAvailableTargets(editor.entry, activeScope);
  }, [activeScope, editor]);

  const saveEditor = useCallback(() => {
    if (editor.mode === "closed" || filter === null) return;
    const name = editor.draft.name.trim();
    const command = editor.draft.command.trim();
    if (!name || !command) return;
    const body = toRequest(editor.draft);
    const target = editor.target;
    const filterPayload =
      filter.scope === "workspace"
        ? { scope: "workspace" as const, workspace_id: filter.workspace_id, target }
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
  }, [editor, filter, putMutation]);

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
    return resolveAvailableTargets(deleteTarget.entry, activeScope);
  }, [activeScope, deleteTarget]);

  const confirmDelete = useCallback(() => {
    if (deleteTarget.mode === "closed" || filter === null) return;
    const target = deleteTarget.entry;
    const deleteFilter =
      filter.scope === "workspace"
        ? {
            scope: "workspace" as const,
            workspace_id: filter.workspace_id,
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
  }, [deleteMutation, deleteTarget, filter]);

  const dismissLastAction = useCallback(() => setLastAction(null), []);

  const needsActiveWorkspace = activeScope === "workspace" && !activeWorkspaceId;
  const isLoading = !needsActiveWorkspace && query.isLoading;
  const error = needsActiveWorkspace ? null : query.error;

  return {
    isLoading,
    error,
    envelope,
    servers,
    counts,
    restart: page.restart,
    activeScope,
    selectScope,
    activeWorkspace,
    activeWorkspaceId,
    availableScopes,
    workspaceScopeAvailable,
    needsActiveWorkspace,
    queryEnabled,
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
