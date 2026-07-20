import { useSessionClearDialog } from "@/hooks/routes/use-session-clear-dialog";
import { useSessionDeleteDialog } from "@/hooks/routes/use-session-delete-dialog";
import { useSessionPageControls } from "@/hooks/routes/use-session-page-controls";
import { useSessionTopbarSlot } from "@/hooks/routes/use-session-topbar-slot";
import {
  useSessionLedger,
  useSessionUsage,
  type InspectorMemoryState,
  type InspectorUsage,
  type SessionPayload,
} from "@/systems/session";
import { useSessionVaultSecrets } from "@/systems/vault";

export interface UseSessionDetailPageInput {
  sessionId: string;
  session: SessionPayload;
  onDeleteSuccess: () => void;
}

export interface UseSessionDetailPageResult {
  controls: ReturnType<typeof useSessionPageControls>;
  inspectorMemory: InspectorMemoryState;
  inspectorUsage: InspectorUsage | null;
  sessionVault: ReturnType<typeof useSessionVaultSecrets>;
  deleteDialog: ReturnType<typeof useSessionDeleteDialog>;
  clearDialog: ReturnType<typeof useSessionClearDialog>;
}

/**
 * Bundles every hook used by the `/agents/$name/sessions/$id` route under a
 * single composable surface so the route component stays under the
 * `compozy-react/max-component-complexity` ceiling.
 */
export function useSessionDetailPage({
  sessionId,
  session,
  onDeleteSuccess,
}: UseSessionDetailPageInput): UseSessionDetailPageResult {
  const controls = useSessionPageControls(sessionId, session, {
    onDeleteSuccess,
    workspaceId: session.workspace_id,
  });
  const sessionVault = useSessionVaultSecrets(sessionId);
  const ledgerEnabled = session.state === "stopped";
  const sessionLedger = useSessionLedger(sessionId, session.workspace_id, {
    enabled: ledgerEnabled,
  });
  const inspectorMemory: InspectorMemoryState = {
    ledger: sessionLedger.data ?? null,
    isLoading: sessionLedger.isLoading,
    error: sessionLedger.error,
  };
  const sessionUsage = useSessionUsage(sessionId, session.workspace_id, session.state);
  const usage = sessionUsage.data;
  const inspectorUsage: InspectorUsage | null = usage
    ? {
        tokensIn: usage.input_tokens ?? undefined,
        tokensOut: usage.output_tokens ?? undefined,
        totalTokens: usage.total_tokens ?? undefined,
        costUsd: usage.total_cost ?? undefined,
        costCurrency: usage.cost_currency || undefined,
        costStatus: usage.cost_status ?? undefined,
        costSource: usage.cost_source ?? undefined,
        turnCount: usage.turn_count,
      }
    : null;
  const deleteDialog = useSessionDeleteDialog(controls.handleDelete);
  const clearDialog = useSessionClearDialog(controls.handleClear);

  useSessionTopbarSlot({
    session,
    isDeleting: controls.isDeleting,
    isStopping: controls.isStopping,
    isResuming: controls.isResuming,
    isClearing: controls.isClearing,
    canClear: controls.canClear,
    onDelete: deleteDialog.openDialog,
    onStop: controls.handleStop,
    onResume: controls.handleResume,
    onClear: clearDialog.openDialog,
  });

  return { controls, inspectorMemory, inspectorUsage, sessionVault, deleteDialog, clearDialog };
}
