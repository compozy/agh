import { describe, expect, it } from "vitest";

import {
  agentEventPayloadSchema,
  sessionEventPayloadSchema,
  sessionEventsResponseSchema,
  sessionHistoryResponseSchema,
  sessionPayloadSchema,
  sessionResponseSchema,
  sessionTranscriptResponseSchema,
  sessionsResponseSchema,
  tokenUsagePayloadSchema,
  transcriptMessageSchema,
  transcriptToolResultSchema,
  turnHistoryPayloadSchema,
  uiMessageRoleSchema,
} from "./types";

describe("sessionPayloadSchema", () => {
  const validSession = {
    id: "sess-123",
    agent_name: "claude",
    workspace_id: "ws_alpha",
    workspace_path: "/home/user/project",
    state: "active",
    created_at: "2026-04-03T10:00:00Z",
    updated_at: "2026-04-03T10:00:00Z",
  };

  it("validates a minimal valid session", () => {
    const result = sessionPayloadSchema.safeParse(validSession);
    expect(result.success).toBe(true);
  });

  it("validates a session with all optional fields", () => {
    const full = {
      ...validSession,
      name: "my-session",
      acp_session_id: "acp-456",
      acp_caps: {
        supports_load_session: true,
        supported_modes: ["interactive"],
        supported_models: ["claude-opus-4-6"],
      },
    };
    const result = sessionPayloadSchema.safeParse(full);
    expect(result.success).toBe(true);
  });

  it("rejects missing required field: id", () => {
    const { id: _, ...noId } = validSession;
    const result = sessionPayloadSchema.safeParse(noId);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: agent_name", () => {
    const { agent_name: _, ...noAgent } = validSession;
    const result = sessionPayloadSchema.safeParse(noAgent);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: workspace_id", () => {
    const { workspace_id: _, ...noWorkspaceID } = validSession;
    const result = sessionPayloadSchema.safeParse(noWorkspaceID);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: workspace_path", () => {
    const { workspace_path: _, ...noWorkspacePath } = validSession;
    const result = sessionPayloadSchema.safeParse(noWorkspacePath);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: state", () => {
    const { state: _, ...noState } = validSession;
    const result = sessionPayloadSchema.safeParse(noState);
    expect(result.success).toBe(false);
  });

  it("validates all valid state values", () => {
    for (const state of ["starting", "active", "stopping", "stopped"]) {
      const result = sessionPayloadSchema.safeParse({ ...validSession, state });
      expect(result.success).toBe(true);
    }
  });

  it("rejects invalid state value", () => {
    const result = sessionPayloadSchema.safeParse({ ...validSession, state: "unknown" });
    expect(result.success).toBe(false);
  });
});

describe("sessionEventPayloadSchema", () => {
  const validEvent = {
    id: "evt-1",
    session_id: "sess-123",
    sequence: 1,
    turn_id: "turn-1",
    type: "agent_message",
    agent_name: "claude",
    content: { text: "hello" },
    timestamp: "2026-04-03T10:00:00Z",
  };

  it("validates a valid session event", () => {
    const result = sessionEventPayloadSchema.safeParse(validEvent);
    expect(result.success).toBe(true);
  });

  it("accepts unknown content shapes", () => {
    const result = sessionEventPayloadSchema.safeParse({
      ...validEvent,
      content: [1, 2, 3],
    });
    expect(result.success).toBe(true);
  });

  it("rejects missing session_id", () => {
    const { session_id: _, ...noSessionId } = validEvent;
    const result = sessionEventPayloadSchema.safeParse(noSessionId);
    expect(result.success).toBe(false);
  });
});

describe("tokenUsagePayloadSchema", () => {
  it("validates an empty object (all fields optional)", () => {
    const result = tokenUsagePayloadSchema.safeParse({});
    expect(result.success).toBe(true);
  });

  it("validates a full token usage payload", () => {
    const result = tokenUsagePayloadSchema.safeParse({
      turn_id: "turn-1",
      input_tokens: 100,
      output_tokens: 200,
      total_tokens: 300,
      thought_tokens: 50,
      cache_read_tokens: 10,
      cache_write_tokens: 5,
      context_used: 1000,
      context_size: 8000,
      cost_amount: 0.05,
      cost_currency: "USD",
      timestamp: "2026-04-03T10:00:00Z",
    });
    expect(result.success).toBe(true);
  });

  it("rejects non-number token values", () => {
    const result = tokenUsagePayloadSchema.safeParse({ input_tokens: "not-a-number" });
    expect(result.success).toBe(false);
  });
});

describe("agentEventPayloadSchema", () => {
  it("validates a minimal agent event", () => {
    const result = agentEventPayloadSchema.safeParse({ type: "agent_message" });
    expect(result.success).toBe(true);
  });

  it("validates a tool_call event with all fields", () => {
    const result = agentEventPayloadSchema.safeParse({
      type: "tool_call",
      session_id: "sess-1",
      turn_id: "turn-1",
      request_id: "req-1",
      timestamp: "2026-04-03T10:00:00Z",
      title: "Read",
      tool_call_id: "tc-1",
      action: "read",
      resource: "/path/to/file",
      raw: { file_path: "/path" },
    });
    expect(result.success).toBe(true);
  });

  it("validates an event with nested usage", () => {
    const result = agentEventPayloadSchema.safeParse({
      type: "usage",
      usage: {
        input_tokens: 100,
        output_tokens: 200,
      },
    });
    expect(result.success).toBe(true);
  });

  it("rejects missing type", () => {
    const result = agentEventPayloadSchema.safeParse({ text: "hello" });
    expect(result.success).toBe(false);
  });
});

describe("transcriptToolResultSchema", () => {
  it("validates a tool result payload", () => {
    const result = transcriptToolResultSchema.safeParse({
      stdout: "ok",
      file_path: "/tmp/demo.ts",
      raw_output: "ok",
    });
    expect(result.success).toBe(true);
  });
});

describe("transcriptMessageSchema", () => {
  it("validates an assistant transcript message", () => {
    const result = transcriptMessageSchema.safeParse({
      id: "msg-1",
      role: "assistant",
      content: "Hello",
      thinking_complete: true,
      tool_error: false,
      timestamp: "2026-04-03T10:00:00Z",
    });
    expect(result.success).toBe(true);
  });

  it("validates a tool transcript message", () => {
    const result = transcriptMessageSchema.safeParse({
      id: "tool-1",
      role: "tool_call",
      content: "",
      tool_name: "Read",
      tool_input: { file_path: "/tmp/demo.ts" },
      thinking_complete: false,
      tool_error: false,
      timestamp: "2026-04-03T10:00:01Z",
    });
    expect(result.success).toBe(true);
  });
});

describe("turnHistoryPayloadSchema", () => {
  it("validates a turn with events", () => {
    const result = turnHistoryPayloadSchema.safeParse({
      turn_id: "turn-1",
      events: [
        {
          id: "evt-1",
          session_id: "sess-1",
          sequence: 1,
          turn_id: "turn-1",
          type: "agent_message",
          agent_name: "claude",
          content: { text: "hello" },
          timestamp: "2026-04-03T10:00:00Z",
        },
      ],
    });
    expect(result.success).toBe(true);
  });

  it("validates a turn with empty events", () => {
    const result = turnHistoryPayloadSchema.safeParse({
      turn_id: "turn-1",
      events: [],
    });
    expect(result.success).toBe(true);
  });

  it("rejects missing turn_id", () => {
    const result = turnHistoryPayloadSchema.safeParse({ events: [] });
    expect(result.success).toBe(false);
  });
});

describe("uiMessageRoleSchema", () => {
  it("validates all role variants", () => {
    for (const role of ["user", "assistant", "tool_call", "tool_result", "system"]) {
      const result = uiMessageRoleSchema.safeParse(role);
      expect(result.success).toBe(true);
    }
  });

  it("rejects invalid role", () => {
    const result = uiMessageRoleSchema.safeParse("admin");
    expect(result.success).toBe(false);
  });
});

describe("API response envelopes", () => {
  const validSession = {
    id: "sess-1",
    agent_name: "claude",
    workspace_id: "ws_alpha",
    workspace_path: "/tmp",
    state: "active",
    created_at: "2026-04-03T10:00:00Z",
    updated_at: "2026-04-03T10:00:00Z",
  };

  it("sessionsResponseSchema validates sessions list", () => {
    const result = sessionsResponseSchema.safeParse({
      sessions: [validSession],
    });
    expect(result.success).toBe(true);
  });

  it("sessionResponseSchema validates single session", () => {
    const result = sessionResponseSchema.safeParse({
      session: validSession,
    });
    expect(result.success).toBe(true);
  });

  it("sessionEventsResponseSchema validates events list", () => {
    const result = sessionEventsResponseSchema.safeParse({
      events: [
        {
          id: "evt-1",
          session_id: "sess-1",
          sequence: 1,
          turn_id: "turn-1",
          type: "agent_message",
          agent_name: "claude",
          content: null,
          timestamp: "2026-04-03T10:00:00Z",
        },
      ],
    });
    expect(result.success).toBe(true);
  });

  it("sessionHistoryResponseSchema validates history", () => {
    const result = sessionHistoryResponseSchema.safeParse({
      history: [{ turn_id: "turn-1", events: [] }],
    });
    expect(result.success).toBe(true);
  });

  it("sessionTranscriptResponseSchema validates transcript messages", () => {
    const result = sessionTranscriptResponseSchema.safeParse({
      messages: [
        {
          id: "msg-1",
          role: "assistant",
          content: "Hello",
          thinking_complete: false,
          tool_error: false,
          timestamp: "2026-04-03T10:00:00Z",
        },
      ],
    });
    expect(result.success).toBe(true);
  });
});
