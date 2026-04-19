import { describe, it, expect, beforeEach } from "vitest";
import { useSessionStore } from "../hooks/use-session-store";
import type { UIMessage, PermissionRequest } from "../types";

function makeMessage(overrides: Partial<UIMessage> = {}): UIMessage {
  return {
    id: `msg-${Math.random().toString(36).slice(2, 8)}`,
    role: "assistant",
    content: "test content",
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("session-store", () => {
  beforeEach(() => {
    // Reset store to initial state before each test
    useSessionStore.setState({
      activeSessionId: null,
      messages: [],
      isStreaming: false,
      pendingPermission: null,
      drafts: {},
    });
  });

  describe("setActiveSession", () => {
    it("replaces messages and sets activeSessionId", () => {
      const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];
      useSessionStore.getState().setActiveSession("session-1", messages);

      const state = useSessionStore.getState();
      expect(state.activeSessionId).toBe("session-1");
      expect(state.messages).toHaveLength(2);
      expect(state.messages[0].id).toBe("m1");
      expect(state.messages[1].id).toBe("m2");
    });

    it("clears streaming and permission state", () => {
      useSessionStore.setState({
        isStreaming: true,
        pendingPermission: {
          requestId: "r1",
          toolName: "Bash",
          toolInput: {},
          action: "exec",
          resource: "cmd",
        },
      });

      useSessionStore.getState().setActiveSession("session-2", []);

      const state = useSessionStore.getState();
      expect(state.isStreaming).toBe(false);
      expect(state.pendingPermission).toBeNull();
    });
  });

  describe("appendMessage", () => {
    it("adds message to end of array", () => {
      const existing = makeMessage({ id: "existing" });
      useSessionStore.setState({ messages: [existing] });

      const newMsg = makeMessage({ id: "new" });
      useSessionStore.getState().appendMessage(newMsg);

      const messages = useSessionStore.getState().messages;
      expect(messages).toHaveLength(2);
      expect(messages[0].id).toBe("existing");
      expect(messages[1].id).toBe("new");
    });

    it("works on empty messages array", () => {
      const msg = makeMessage({ id: "first" });
      useSessionStore.getState().appendMessage(msg);

      expect(useSessionStore.getState().messages).toHaveLength(1);
      expect(useSessionStore.getState().messages[0].id).toBe("first");
    });
  });

  describe("updateLastMessage", () => {
    it("merges partial into last message", () => {
      const msg1 = makeMessage({ id: "m1", content: "first" });
      const msg2 = makeMessage({ id: "m2", content: "original", isStreaming: true });
      useSessionStore.setState({ messages: [msg1, msg2] });

      useSessionStore.getState().updateLastMessage({
        content: "updated content",
        thinking: "some thinking",
        isStreaming: false,
      });

      const messages = useSessionStore.getState().messages;
      expect(messages).toHaveLength(2);
      expect(messages[0].content).toBe("first"); // unchanged
      expect(messages[1].content).toBe("updated content");
      expect(messages[1].thinking).toBe("some thinking");
      expect(messages[1].isStreaming).toBe(false);
      expect(messages[1].id).toBe("m2"); // preserves id
    });

    it("does nothing when messages array is empty", () => {
      useSessionStore.setState({ messages: [] });
      useSessionStore.getState().updateLastMessage({ content: "should not crash" });
      expect(useSessionStore.getState().messages).toHaveLength(0);
    });
  });

  describe("setPendingPermission", () => {
    it("sets permission state", () => {
      const permission: PermissionRequest = {
        requestId: "req-1",
        toolName: "Bash",
        toolInput: { command: "rm -rf /" },
        action: "execute",
        resource: "bash command",
      };

      useSessionStore.getState().setPendingPermission(permission);

      const state = useSessionStore.getState();
      expect(state.pendingPermission).toEqual(permission);
    });

    it("clears permission state with null", () => {
      useSessionStore.setState({
        pendingPermission: {
          requestId: "req-1",
          toolName: "Bash",
          toolInput: {},
          action: "exec",
          resource: "cmd",
        },
      });

      useSessionStore.getState().setPendingPermission(null);
      expect(useSessionStore.getState().pendingPermission).toBeNull();
    });
  });

  describe("clearSession", () => {
    it("resets all state to initial values", () => {
      useSessionStore.setState({
        activeSessionId: "session-1",
        messages: [makeMessage()],
        isStreaming: true,
        pendingPermission: {
          requestId: "r1",
          toolName: "Bash",
          toolInput: {},
          action: "exec",
          resource: "cmd",
        },
        drafts: { "session-1": { text: "draft" } },
      });

      useSessionStore.getState().clearSession();

      const state = useSessionStore.getState();
      expect(state.activeSessionId).toBeNull();
      expect(state.messages).toHaveLength(0);
      expect(state.isStreaming).toBe(false);
      expect(state.pendingPermission).toBeNull();
      expect(state.drafts).toEqual({});
    });
  });

  describe("setDraft + clearDraft", () => {
    it("stores a draft for a session and merges patches", () => {
      useSessionStore.getState().setDraft("session-a", { text: "Hello" });
      useSessionStore.getState().setDraft("session-a", { skillId: "no-workarounds" });

      const draft = useSessionStore.getState().drafts["session-a"];
      expect(draft.text).toBe("Hello");
      expect(draft.skillId).toBe("no-workarounds");
    });

    it("keeps drafts isolated per session", () => {
      useSessionStore.getState().setDraft("session-a", { text: "Alpha draft" });
      useSessionStore.getState().setDraft("session-b", { text: "Bravo draft" });

      const drafts = useSessionStore.getState().drafts;
      expect(drafts["session-a"].text).toBe("Alpha draft");
      expect(drafts["session-b"].text).toBe("Bravo draft");
    });

    it("removes the entry when the draft becomes empty", () => {
      useSessionStore.getState().setDraft("session-a", { text: "Hello" });
      useSessionStore.getState().setDraft("session-a", { text: "" });

      expect(useSessionStore.getState().drafts["session-a"]).toBeUndefined();
    });

    it("clearDraft drops the entry", () => {
      useSessionStore.getState().setDraft("session-a", { text: "Hello", skillId: "x" });
      useSessionStore.getState().clearDraft("session-a");

      expect(useSessionStore.getState().drafts["session-a"]).toBeUndefined();
    });
  });

  describe("setActiveSession draft preservation", () => {
    it("keeps drafts alive across session switches", () => {
      useSessionStore.getState().setDraft("session-a", { text: "Unsent thought" });
      useSessionStore.getState().setActiveSession("session-a", []);
      useSessionStore.getState().setActiveSession("session-b", []);
      useSessionStore.getState().setActiveSession("session-a", []);

      expect(useSessionStore.getState().drafts["session-a"]?.text).toBe("Unsent thought");
    });
  });
});
