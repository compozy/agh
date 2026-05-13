import { useMemo } from "react";

import { useSessionDeleteDialog } from "@/hooks/routes/use-session-delete-dialog";
import { useSessionPageControls } from "@/hooks/routes/use-session-page-controls";
import { useSessionTopbarSlot } from "@/hooks/routes/use-session-topbar-slot";
import {
  useSessionLedger,
  type InspectorMemoryState,
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
  sessionVault: ReturnType<typeof useSessionVaultSecrets>;
  deleteDialog: ReturnType<typeof useSessionDeleteDialog>;
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
  const controls = useSessionPageControls(sessionId, session.state, {
    onDeleteSuccess,
    workspaceId: session.workspace_id,
  });
  const sessionVault = useSessionVaultSecrets(sessionId);
  const ledgerEnabled = session.state === "stopped";
  const sessionLedger = useSessionLedger(sessionId, session.workspace_id, {
    enabled: ledgerEnabled,
  });
  const inspectorMemory = useMemo<InspectorMemoryState>(
    () => ({
      ledger: sessionLedger.data ?? null,
      isLoading: sessionLedger.isLoading,
      error: sessionLedger.error,
    }),
    [sessionLedger.data, sessionLedger.isLoading, sessionLedger.error]
  );
  const deleteDialog = useSessionDeleteDialog(controls.handleDelete);

  useSessionTopbarSlot({
    session,
    isDeleting: controls.isDeleting,
    isStopping: controls.isStopping,
    isResuming: controls.isResuming,
    onDelete: deleteDialog.openDialog,
    onStop: controls.handleStop,
    onResume: controls.handleResume,
  });

  return { controls, inspectorMemory, sessionVault, deleteDialog };
}
