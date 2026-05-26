import { useEffect, useMemo, useRef } from "react";
import {
  ExportedMessageRepository,
  useAui,
  useAuiState,
  type ThreadMessageLike,
} from "@assistant-ui/react";

function transcriptMessageKey(messages: readonly ThreadMessageLike[]) {
  return messages.map(message => message.id).join("\n");
}

export function useRuntimeTranscriptHydration(messages: readonly ThreadMessageLike[]) {
  const aui = useAui();
  const importedKeyRef = useRef("");
  const isRunning = useAuiState(state => state.thread.isRunning);
  const runtimeMessageCount = useAuiState(state => state.thread.messages.length);
  const runtimeMessageKey = useAuiState(state =>
    state.thread.messages.map(message => message.id).join("\n")
  );
  const transcriptKey = useMemo(() => transcriptMessageKey(messages), [messages]);

  useEffect(() => {
    if (messages.length === 0 || isRunning) {
      return;
    }
    if (runtimeMessageKey === transcriptKey) {
      importedKeyRef.current = transcriptKey;
      return;
    }
    if (runtimeMessageCount > messages.length || importedKeyRef.current === transcriptKey) {
      return;
    }

    aui.thread().import(ExportedMessageRepository.fromArray(messages));
    importedKeyRef.current = transcriptKey;
  }, [aui, isRunning, messages, runtimeMessageCount, runtimeMessageKey, transcriptKey]);
}
