import { useQuery } from "@tanstack/react-query";
import { useEffect, useEffectEvent, useRef, useState } from "react";

import { useMCPAuthorize } from "@/hooks/routes/use-mcp-authorize";
import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  emptyDraft,
  SETTINGS_QUERY_INTERVALS,
  type MCPDraft,
  type SettingsMCPAuthFilter,
  SettingsApiError,
  type SettingsMCPServerEntry,
  type SettingsMCPServerTarget,
  type SettingsMutationResult,
  type SettingsScope,
  toDraft,
  toRequest,
  useDeleteSettingsMCPServer,
  usePutSettingsMCPServer,
  useSettingsMCPServers,
  validateDraft,
} from "@/systems/settings";
import { vaultSecretsListOptions } from "@/systems/vault";
import { useActiveWorkspace } from "@/systems/workspace";

export type { MCPDraft, MCPEnvPair } from "@/systems/settings";

export type MCPActiveScope = "workspace" | "global";

export type MCPEditorState =
  | { mode: "closed" }
  | { mode: "create"; draft: MCPDraft; target: SettingsMCPServerTarget }
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

// Route owns the URL search; the page hook consumes scope/server and reports
// changes back so scope and selection stay deep-linkable and refresh-safe.
export interface UseMcpPageOptions {
  scope: MCPActiveScope;
  selectedServer: string;
  onScopeChange: (scope: MCPActiveScope) => void;
  onSelectServer: (server: string) => void;
}

export function useMcpPage(options: UseMcpPageOptions) {
  const page = useSettingsPage({ currentSlug: "mcp-servers" });
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const activeScope = options.scope;
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

  const authFilter: SettingsMCPAuthFilter | null =
    filter === null
      ? null
      : filter.scope === "workspace"
        ? { scope: "workspace", workspace_id: filter.workspace_id }
        : { scope: "global" };
  const authorize = useMCPAuthorize(authFilter);
  const queryEnabled = filter !== null;
  const query = useSettingsMCPServers(filter ?? { scope: "global" }, {
    enabled: queryEnabled,
    refetchInterval: authorize.isAwaiting
      ? SETTINGS_QUERY_INTERVALS.mcpAuthStatusPollInterval
      : SETTINGS_QUERY_INTERVALS.collectionRefetchInterval,
  });

  const envelope = queryEnabled ? (query.data ?? null) : null;
  const servers = envelope?.mcp_servers ?? [];
  const availableScopes = envelope?.available_scopes ?? ["global", "workspace"];
  const workspaceScopeAvailable = availableScopes.includes("workspace");

  const editorOpen = editor.mode !== "closed";
  const vaultQuery = useQuery({
    ...vaultSecretsListOptions({ namespace: "mcp" }),
    enabled: editorOpen,
  });
  const vaultRefs = (vaultQuery.data ?? []).map(secret => secret.ref);

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
    authorize.cancel();
  };
  const resetTransientStateAfterWorkspaceChange = useEffectEvent(resetTransientState);

  const selectScope = (scope: MCPActiveScope) => {
    resetTransientState();
    options.onScopeChange(scope);
  };

  // Active workspace is Zustand-backed (sidebar), not URL-scoped, so a switch never
  // re-runs route scope validation. Clear any open editor/delete/authorize flow so a
  // pending save/delete/authorize can't target the newly selected workspace.
  const previousWorkspaceIdRef = useRef(activeWorkspaceId);
  useEffect(() => {
    if (previousWorkspaceIdRef.current === activeWorkspaceId) {
      return;
    }
    previousWorkspaceIdRef.current = activeWorkspaceId;
    resetTransientStateAfterWorkspaceChange();
  }, [activeWorkspaceId]);

  const selectServer = (name: string) => options.onSelectServer(name);
  const clearSelection = () => options.onSelectServer("");

  const selectedServer = options.selectedServer;
  const selectedEntry = selectedServer
    ? (servers.find(entry => entry.name === selectedServer) ?? null)
    : null;

  // Auto-completion path: the browser step finishes server-side; when the polled
  // list shows the authorizing server confirmed, advance the state machine.
  const authServerEntry = authorize.server
    ? (servers.find(entry => entry.name === authorize.server) ?? null)
    : null;
  useEffect(() => {
    const authStatus = authServerEntry?.auth_status;
    if (authorize.isAwaiting && authStatus?.status) {
      authorize.acknowledgeStatus(authStatus.status, Boolean(authStatus.token_present));
    }
  }, [authorize.isAwaiting, authorize.acknowledgeStatus, authServerEntry]);

  const openCreate = () => {
    putMutation.reset();
    setEditor({ mode: "create", draft: emptyDraft("stdio"), target: "auto" });
  };

  const openEdit = (entry: SettingsMCPServerEntry) => {
    putMutation.reset();
    setEditor({ mode: "edit", name: entry.name, draft: toDraft(entry), entry, target: "auto" });
  };

  const openAuthorize = (entry: SettingsMCPServerEntry) => {
    if (!authFilter) return;
    void authorize.beginAuthorize(entry.name, {
      status: entry.auth_status?.status ?? "needs_login",
      tokenPresent: Boolean(entry.auth_status?.token_present),
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

  const draftValidation = editor.mode === "closed" ? null : validateDraft(editor.draft);
  const editorName = editor.mode === "closed" ? "" : editor.draft.name.trim();
  const nameConflict =
    editor.mode === "create" &&
    editorName.length > 0 &&
    servers.some(entry => entry.name.toLowerCase() === editorName.toLowerCase());
  const editorErrors =
    draftValidation === null
      ? {}
      : nameConflict
        ? { ...draftValidation.errors, name: `An MCP server named "${editorName}" already exists.` }
        : draftValidation.errors;
  const editorIsValid = draftValidation !== null && draftValidation.valid && !nameConflict;

  const editorAvailableTargets: SettingsMCPServerTarget[] =
    editor.mode === "edit"
      ? resolveAvailableTargets(editor.entry, activeScope)
      : ["auto", "config", "sidecar"];

  const saveEditor = () => {
    if (editor.mode === "closed" || filter === null || !editorIsValid) return;
    const name = editor.draft.name.trim();
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

  const requestRemoveFromEditor = () => {
    if (editor.mode !== "edit") return;
    const entry = editor.entry;
    setEditor({ mode: "closed" });
    openDelete(entry);
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
          setLastAction({ kind: "deleted", name: target.name, result, remainingShadowed });
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
    selectServer,
    clearSelection,
    selectedServer,
    selectedEntry,
    activeWorkspace,
    activeWorkspaceId,
    availableScopes,
    workspaceScopeAvailable,
    needsActiveWorkspace,
    queryEnabled,
    refetch: query.refetch,
    editor,
    editorErrors,
    editorIsValid,
    editorAvailableTargets,
    editorVaultRefs: vaultRefs,
    editorSaveError: errorMessage(putMutation.error),
    editorWarnings: putMutation.data?.warnings,
    editorIsSaving: putMutation.isPending,
    openCreate,
    openEdit,
    closeEditor,
    updateDraft,
    setEditorTarget,
    saveEditor,
    requestRemoveFromEditor,
    authorize,
    authorizeEntry: authServerEntry,
    openAuthorize,
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
