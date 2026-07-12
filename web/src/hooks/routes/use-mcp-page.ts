import { useEffect, useEffectEvent, useRef, useState } from "react";

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
  const args = draft.args.flatMap(arg => {
    const value = arg.trim();
    return value ? [value] : [];
  });
  const envEntries = draft.env.flatMap(entry => {
    const key = entry.key.trim();
    return key ? [{ key, value: entry.value }] : [];
  });
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

  const filter =
    activeScope === "global"
      ? { scope: "global" as const }
      : activeWorkspaceId
        ? { scope: "workspace" as const, workspace_id: activeWorkspaceId }
        : null;

  const queryEnabled = filter !== null;
  const query = useSettingsMCPServers(filter ?? { scope: "global" }, { enabled: queryEnabled });

  const envelope = queryEnabled ? (query.data ?? null) : null;
  const servers = envelope?.mcp_servers ?? [];
  const availableScopes = envelope?.available_scopes ?? ["global", "workspace"];
  const workspaceScopeAvailable = availableScopes.includes("workspace");

  const counts = {
    total: servers.length,
    shadowed: servers.reduce(
      (acc, entry) => acc + (entry.source_metadata.shadowed_sources?.length ?? 0),
      0
    ),
  };

  const resetTransientState = () => {
    putMutation.reset();
    deleteMutation.reset();
    setEditor({ mode: "closed" });
    setDeleteTarget({ mode: "closed" });
  };
  const resetTransientStateAfterWorkspaceChange = useEffectEvent(resetTransientState);

  const selectScope = (scope: MCPActiveScope) => {
    resetTransientState();
    setActiveScope(scope);
  };

  const previousWorkspaceIdRef = useRef(activeWorkspaceId);
  useEffect(() => {
    if (previousWorkspaceIdRef.current === activeWorkspaceId) {
      return;
    }
    previousWorkspaceIdRef.current = activeWorkspaceId;
    resetTransientStateAfterWorkspaceChange();
  }, [activeWorkspaceId]);

  const openCreate = () => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft(), target: "auto" });
  };

  const openEdit = (entry: SettingsMCPServerEntry) => {
    putMutation.reset();
    setEditor({
      mode: "edit",
      name: entry.name,
      draft: toDraft(entry),
      entry,
      target: "auto",
    });
  };

  const closeEditor = () => {
    setEditor({ mode: "closed" });
    putMutation.reset();
  };

  const updateDraft = (updater: (draft: MCPDraft) => MCPDraft) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, draft: updater(current.draft) };
    });
  };

  const setEditorTarget = (target: SettingsMCPServerTarget) => {
    setEditor(current => {
      if (current.mode === "closed") return current;
      return { ...current, target };
    });
  };

  const editorName = editor.mode === "closed" ? "" : editor.draft.name.trim();
  const editorCommand = editor.mode === "closed" ? "" : editor.draft.command.trim();
  const editorIsValid =
    editor.mode !== "closed" &&
    editorName.length > 0 &&
    editorCommand.length > 0 &&
    (editor.mode !== "create" ||
      !servers.some(entry => entry.name.toLowerCase() === editorName.toLowerCase()));

  const editorAvailableTargets: SettingsMCPServerTarget[] =
    editor.mode === "edit"
      ? resolveAvailableTargets(editor.entry, activeScope)
      : ["auto", "config", "sidecar"];

  const saveEditor = () => {
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
  };

  const openDelete = (entry: SettingsMCPServerEntry) => {
    deleteMutation.reset();
    setDeleteTarget({ mode: "open", entry, target: "auto" });
  };

  const closeDelete = () => {
    setDeleteTarget({ mode: "closed" });
    deleteMutation.reset();
  };

  const setDeleteTargetKind = (target: SettingsMCPServerTarget) => {
    setDeleteTarget(current => {
      if (current.mode === "closed") return current;
      return { ...current, target };
    });
  };

  const deleteAvailableTargets: SettingsMCPServerTarget[] =
    deleteTarget.mode === "open"
      ? resolveAvailableTargets(deleteTarget.entry, activeScope)
      : ["auto", "config", "sidecar"];

  const confirmDelete = () => {
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
  };

  const dismissLastAction = () => setLastAction(null);

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
