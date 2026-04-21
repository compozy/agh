import { useEffect, useEffectEvent, useRef } from "react";
import { useAui, useAuiEvent, useAuiState } from "@assistant-ui/react";

import { useSessionStore } from "@/systems/session/hooks/use-session-store";

export function useSessionComposerState(sessionId: string) {
  const aui = useAui();
  const draftText = useSessionStore(state => state.drafts[sessionId]?.text ?? "");
  const setDraft = useSessionStore(state => state.setDraft);
  const clearDraft = useSessionStore(state => state.clearDraft);
  const composerText = useAuiState(state => state.composer.text);
  const isRunning = useAuiState(state => state.thread.isRunning);
  const hasHydratedDraftRef = useRef(false);

  const clearDraftForSession = useEffectEvent(() => {
    clearDraft(sessionId);
  });

  useEffect(() => {
    if (hasHydratedDraftRef.current) {
      return;
    }

    hasHydratedDraftRef.current = true;

    if (!draftText) {
      return;
    }

    aui.composer().setText(draftText);
  }, [aui, draftText]);

  useEffect(() => {
    setDraft(sessionId, { text: composerText });
  }, [composerText, sessionId, setDraft]);

  useAuiEvent("composer.send", clearDraftForSession);

  return { isRunning };
}
