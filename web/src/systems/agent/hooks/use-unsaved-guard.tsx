import { ConfirmDialog } from "@agh/ui";
import { useBlocker } from "@tanstack/react-router";
import { useCallback, useMemo, type ReactNode } from "react";

export interface UseUnsavedGuardOptions {
  dirty: boolean;
  /** Entity display name interpolated into the confirmation copy. */
  entityName: string;
}

export interface UseUnsavedGuardResult {
  status: "idle" | "blocked";
  proceed: (() => void) | undefined;
  reset: (() => void) | undefined;
  /** Ready-to-render ConfirmDialog; consumer mounts it beside the editor. */
  confirmDialog: ReactNode;
}

/**
 * Shared unsaved navigation guard for agent editors.
 * Pins TanStack Router 1.170.17 useBlocker resolver semantics:
 * blocked.proceed allows navigation; blocked.reset stays.
 */
export function useUnsavedGuard({
  dirty,
  entityName,
}: UseUnsavedGuardOptions): UseUnsavedGuardResult {
  const shouldBlock = useCallback(() => dirty, [dirty]);
  const blocker = useBlocker({
    shouldBlockFn: shouldBlock,
    withResolver: true,
    disabled: !dirty,
    enableBeforeUnload: dirty,
  });

  const proceed = useCallback(() => {
    blocker.proceed?.();
  }, [blocker]);

  const reset = useCallback(() => {
    blocker.reset?.();
  }, [blocker]);

  const confirmDialog = useMemo(
    () => (
      <ConfirmDialog
        open={blocker.status === "blocked"}
        onOpenChange={open => {
          if (!open) reset();
        }}
        tone="warning"
        title="Discard unsaved changes?"
        description={`Your edits to ${entityName} will be lost.`}
        confirmLabel="Discard changes"
        cancelLabel="Keep editing"
        onConfirm={proceed}
        confirmButtonProps={{ "data-testid": "unsaved-guard-discard" }}
        cancelButtonProps={{ "data-testid": "unsaved-guard-keep-editing" }}
        contentProps={{ "data-testid": "unsaved-guard-dialog" }}
      />
    ),
    [blocker.status, entityName, proceed, reset]
  );

  return {
    status: blocker.status,
    proceed: blocker.status === "blocked" ? proceed : undefined,
    reset: blocker.status === "blocked" ? reset : undefined,
    confirmDialog,
  };
}
