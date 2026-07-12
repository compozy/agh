import { useRef } from "react";

export function useLatestAgentSettingsActions(onSave: () => void, onDiscard: () => void) {
  const saveRef = useRef(onSave);
  const discardRef = useRef(onDiscard);
  saveRef.current = onSave;
  discardRef.current = onDiscard;

  return {
    onSave: () => saveRef.current(),
    onDiscard: () => discardRef.current(),
  };
}
