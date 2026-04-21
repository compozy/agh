import { beforeEach, describe, expect, it } from "vitest";

import { useSessionStore } from "../hooks/use-session-store";
import type { UIMessage } from "../types";

function makeMessage(overrides: Partial<UIMessage> = {}): UIMessage {
  return {
    id: `msg-${Math.random().toString(36).slice(2, 8)}`,
    role: "assistant",
    content: "test content",
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("session-store session switch", () => {
  beforeEach(() => {
    useSessionStore.setState({
      activeSessionId: null,
      historyMessages: [],
      liveMessages: [],
      isStreaming: false,
      awaitingTranscriptSync: false,
      pendingPermission: null,
      drafts: {},
    });
  });

  it("loads history messages into store for the target session", () => {
    const historyMessages: UIMessage[] = [
      makeMessage({ id: "h1", role: "user", content: "Previous question" }),
      makeMessage({ id: "h2", role: "assistant", content: "Previous answer", isStreaming: false }),
    ];

    useSessionStore.getState().setActiveSession("session-with-history", historyMessages);

    const state = useSessionStore.getState();
    expect(state.activeSessionId).toBe("session-with-history");
    expect(state.historyMessages).toHaveLength(2);
    expect(state.historyMessages[0].content).toBe("Previous question");
    expect(state.historyMessages[1].content).toBe("Previous answer");
    expect(state.liveMessages).toEqual([]);
    expect(state.isStreaming).toBe(false);
    expect(state.pendingPermission).toBeNull();
  });

  it("clears the live tail on session switch", () => {
    useSessionStore.setState({
      activeSessionId: "session-streaming",
      historyMessages: [makeMessage({ id: "history-1" })],
      liveMessages: [makeMessage({ id: "live-1", isStreaming: true })],
      isStreaming: true,
      awaitingTranscriptSync: true,
    });

    useSessionStore.getState().setActiveSession("session-new", []);

    const state = useSessionStore.getState();
    expect(state.historyMessages).toEqual([]);
    expect(state.liveMessages).toEqual([]);
    expect(state.isStreaming).toBe(false);
    expect(state.awaitingTranscriptSync).toBe(false);
  });

  it("keeps drafts alive across session switches", () => {
    useSessionStore.getState().setDraft("session-a", { text: "Unsent thought" });
    useSessionStore.getState().setActiveSession("session-a", []);
    useSessionStore.getState().setActiveSession("session-b", []);
    useSessionStore.getState().setActiveSession("session-a", []);

    expect(useSessionStore.getState().drafts["session-a"]?.text).toBe("Unsent thought");
  });
});
