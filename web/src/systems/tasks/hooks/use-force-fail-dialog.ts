import { useCallback, useId, useState } from "react";

export type ForceFailRunHandler = (reason: string) => Promise<void> | void;

export function useForceFailDialog(onForceFailRun?: ForceFailRunHandler) {
  const reasonId = useId();
  const reasonHintId = useId();
  const [isOpen, setIsOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleOpenChange = useCallback((open: boolean) => {
    setIsOpen(open);
    if (!open) {
      setReason("");
      setError(null);
    }
  }, []);

  const open = useCallback(() => {
    setIsOpen(true);
  }, []);

  const changeReason = useCallback((value: string) => {
    setReason(value);
    setError(null);
  }, []);

  const confirm = useCallback(async () => {
    const trimmed = reason.trim();
    if (!trimmed) {
      setError("Reason is required.");
      return;
    }
    try {
      await onForceFailRun?.(trimmed);
      handleOpenChange(false);
    } catch {
      setError(null);
    }
  }, [handleOpenChange, onForceFailRun, reason]);

  return {
    changeReason,
    confirm,
    error,
    handleOpenChange,
    isOpen,
    open,
    reason,
    reasonHintId,
    reasonId,
  };
}
