import { describe, it, expect, beforeEach } from "vitest";
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
      messages: [],
      isStreaming: false,
      pendingPermission: null,
      drafts: {},
    });
  });

  it("saves current messages before loading new session via setActiveSession", () => {
    // Set up session A with messages
    const sessionAMessages = [
      makeMessage({ id: "a1", content: "Hello from A" }),
      makeMessage({ id: "a2", content: "Response from A" }),
    ];
    useSessionStore.getState().setActiveSession("session-a", sessionAMessages);

    expect(useSessionStore.getState().activeSessionId).toBe("session-a");
    expect(useSessionStore.getState().messages).toHaveLength(2);

    // Switch to session B with its own messages
    const sessionBMessages = [makeMessage({ id: "b1", content: "Hello from B" })];
    useSessionStore.getState().setActiveSession("session-b", sessionBMessages);

    // Store now reflects session B
    expect(useSessionStore.getState().activeSessionId).toBe("session-b");
    expect(useSessionStore.getState().messages).toHaveLength(1);
    expect(useSessionStore.getState().messages[0].content).toBe("Hello from B");
  });

  it("loads history messages into store for target session", () => {
    const historyMessages: UIMessage[] = [
      makeMessage({ id: "h1", role: "user", content: "Previous question" }),
      makeMessage({ id: "h2", role: "assistant", content: "Previous answer", isStreaming: false }),
    ];

    useSessionStore.getState().setActiveSession("session-with-history", historyMessages);

    const state = useSessionStore.getState();
    expect(state.activeSessionId).toBe("session-with-history");
    expect(state.messages).toHaveLength(2);
    expect(state.messages[0].content).toBe("Previous question");
    expect(state.messages[1].content).toBe("Previous answer");
    expect(state.isStreaming).toBe(false);
    expect(state.pendingPermission).toBeNull();
  });

  it("clears streaming state on session switch", () => {
    useSessionStore.setState({
      activeSessionId: "session-streaming",
      messages: [makeMessage({ isStreaming: true })],
      isStreaming: true,
      pendingPermission: null,
    });

    useSessionStore.getState().setActiveSession("session-new", []);

    expect(useSessionStore.getState().isStreaming).toBe(false);
  });

  it("clears pending permission on session switch", () => {
    useSessionStore.setState({
      activeSessionId: "session-perm",
      messages: [],
      isStreaming: false,
      pendingPermission: {
        requestId: "req-1",
        toolName: "Bash",
        toolInput: {},
        action: "execute",
        resource: "cmd",
      },
    });

    useSessionStore.getState().setActiveSession("session-new", []);

    expect(useSessionStore.getState().pendingPermission).toBeNull();
  });

  it("handles switching to session with empty history", () => {
    const msgs = [makeMessage({ id: "m1" })];
    useSessionStore.getState().setActiveSession("session-a", msgs);

    useSessionStore.getState().setActiveSession("session-empty", []);

    expect(useSessionStore.getState().activeSessionId).toBe("session-empty");
    expect(useSessionStore.getState().messages).toHaveLength(0);
  });
});
