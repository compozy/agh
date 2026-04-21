import { beforeEach, describe, expect, it } from "vitest";

import { useSessionStore } from "../hooks/use-session-store";
import type { PermissionRequest, UIMessage } from "../types";

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

  describe("setActiveSession", () => {
    it("replaces history messages and sets activeSessionId", () => {
      const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];
      useSessionStore.getState().setActiveSession("session-1", messages);

      const state = useSessionStore.getState();
      expect(state.activeSessionId).toBe("session-1");
      expect(state.historyMessages).toHaveLength(2);
      expect(state.historyMessages[0].id).toBe("m1");
      expect(state.historyMessages[1].id).toBe("m2");
      expect(state.liveMessages).toEqual([]);
    });

    it("clears transient streaming state when switching sessions", () => {
      useSessionStore.setState({
        liveMessages: [makeMessage({ id: "live-1", isStreaming: true })],
        isStreaming: true,
        awaitingTranscriptSync: true,
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
      expect(state.liveMessages).toEqual([]);
      expect(state.isStreaming).toBe(false);
      expect(state.awaitingTranscriptSync).toBe(false);
      expect(state.pendingPermission).toBeNull();
    });
  });

  describe("history + live message actions", () => {
    it("replaceHistoryMessages swaps only the durable transcript", () => {
      const live = [makeMessage({ id: "live-1" })];
      useSessionStore.setState({ liveMessages: live });

      useSessionStore.getState().replaceHistoryMessages([makeMessage({ id: "history-1" })]);

      const state = useSessionStore.getState();
      expect(state.historyMessages.map(message => message.id)).toEqual(["history-1"]);
      expect(state.liveMessages).toBe(live);
    });

    it("setLiveMessages replaces only the in-flight tail", () => {
      useSessionStore.setState({ historyMessages: [makeMessage({ id: "history-1" })] });

      useSessionStore.getState().setLiveMessages([makeMessage({ id: "live-1" })]);

      const state = useSessionStore.getState();
      expect(state.historyMessages.map(message => message.id)).toEqual(["history-1"]);
      expect(state.liveMessages.map(message => message.id)).toEqual(["live-1"]);
    });

    it("clearLiveMessages removes the in-flight tail without touching history", () => {
      useSessionStore.setState({
        historyMessages: [makeMessage({ id: "history-1" })],
        liveMessages: [makeMessage({ id: "live-1" })],
      });

      useSessionStore.getState().clearLiveMessages();

      const state = useSessionStore.getState();
      expect(state.historyMessages.map(message => message.id)).toEqual(["history-1"]);
      expect(state.liveMessages).toEqual([]);
    });

    it("resetConversation clears history, live tail, and transient flags", () => {
      useSessionStore.setState({
        activeSessionId: "session-1",
        historyMessages: [makeMessage({ id: "history-1" })],
        liveMessages: [makeMessage({ id: "live-1" })],
        isStreaming: true,
        awaitingTranscriptSync: true,
        pendingPermission: {
          requestId: "r1",
          toolName: "Bash",
          toolInput: {},
          action: "exec",
          resource: "cmd",
        },
      });

      useSessionStore.getState().resetConversation();

      const state = useSessionStore.getState();
      expect(state.activeSessionId).toBe("session-1");
      expect(state.historyMessages).toEqual([]);
      expect(state.liveMessages).toEqual([]);
      expect(state.isStreaming).toBe(false);
      expect(state.awaitingTranscriptSync).toBe(false);
      expect(state.pendingPermission).toBeNull();
    });
  });

  describe("pending state", () => {
    it("sets permission state", () => {
      const permission: PermissionRequest = {
        requestId: "req-1",
        toolName: "Bash",
        toolInput: { command: "rm -rf /" },
        action: "execute",
        resource: "bash command",
      };

      useSessionStore.getState().setPendingPermission(permission);
      expect(useSessionStore.getState().pendingPermission).toEqual(permission);
    });

    it("tracks transcript sync state", () => {
      useSessionStore.getState().setAwaitingTranscriptSync(true);
      expect(useSessionStore.getState().awaitingTranscriptSync).toBe(true);

      useSessionStore.getState().setAwaitingTranscriptSync(false);
      expect(useSessionStore.getState().awaitingTranscriptSync).toBe(false);
    });
  });

  describe("clearSession", () => {
    it("resets all session state to initial values", () => {
      useSessionStore.setState({
        activeSessionId: "session-1",
        historyMessages: [makeMessage()],
        liveMessages: [makeMessage({ id: "live-1" })],
        isStreaming: true,
        awaitingTranscriptSync: true,
        pendingPermission: {
          requestId: "r1",
          toolName: "Bash",
          toolInput: {},
          action: "exec",
          resource: "cmd",
        },
        drafts: { "session-1": { text: "draft", channel: "release" } },
      });

      useSessionStore.getState().clearSession();

      const state = useSessionStore.getState();
      expect(state.activeSessionId).toBeNull();
      expect(state.historyMessages).toEqual([]);
      expect(state.liveMessages).toEqual([]);
      expect(state.isStreaming).toBe(false);
      expect(state.awaitingTranscriptSync).toBe(false);
      expect(state.pendingPermission).toBeNull();
      expect(state.drafts).toEqual({});
    });
  });

  describe("setDraft + clearDraft", () => {
    it("stores a draft for a session and merges patches", () => {
      useSessionStore.getState().setDraft("session-a", { text: "Hello" });
      useSessionStore.getState().setDraft("session-a", { channel: "release" });

      const draft = useSessionStore.getState().drafts["session-a"];
      expect(draft).toEqual({ text: "Hello", channel: "release" });
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
      useSessionStore.getState().setDraft("session-a", { text: "Hello", channel: "release" });
      useSessionStore.getState().clearDraft("session-a");

      expect(useSessionStore.getState().drafts["session-a"]).toBeUndefined();
    });
  });
});
