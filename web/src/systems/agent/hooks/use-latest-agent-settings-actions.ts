import { useCallback, useRef } from "react";

export function useLatestAgentSettingsActions(onSave: () => void, onDiscard: () => void) {
  const saveRef = useRef(onSave);
  const discardRef = useRef(onDiscard);
  saveRef.current = onSave;
  discardRef.current = onDiscard;

  return {
    onSave: useCallback(() => saveRef.current(), []),
    onDiscard: useCallback(() => discardRef.current(), []),
  };
}
