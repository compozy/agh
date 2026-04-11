import { describe, expect, expectTypeOf, it } from "vitest";

import type {
  ACPCaps,
  AgentEventPayload,
  ApproveSessionParams,
  CreateSessionParams,
  SessionEventPayload,
  SessionPayload,
  SessionState,
  TranscriptMessage,
  TranscriptToolResult,
  UIMessageRole,
} from "./types";
import { uiMessageRoles } from "./types";

describe("session contract types", () => {
  it("keeps REST session payloads aligned with the generated contract", () => {
    expectTypeOf<ACPCaps>().toMatchTypeOf<{
      supports_load_session: boolean;
      supported_modes?: string[];
      supported_models?: string[];
    }>();

    expectTypeOf<SessionState>().toEqualTypeOf<"starting" | "active" | "stopping" | "stopped">();

    expectTypeOf<SessionPayload>().toMatchTypeOf<{
      id: string;
      agent_name: string;
      state: SessionState;
      created_at: string;
      updated_at: string;
      name?: string;
      workspace_id?: string;
      workspace_path?: string;
      stop_reason?:
        | "completed"
        | "user_canceled"
        | "max_iterations"
        | "loop_detected"
        | "timeout"
        | "budget_exceeded"
        | "error"
        | "agent_crashed"
        | "hook_stopped"
        | "shutdown";
      stop_detail?: string;
      acp_session_id?: string;
      acp_caps?: ACPCaps | null;
    }>();

    expectTypeOf<SessionEventPayload>().toMatchTypeOf<{
      id: string;
      session_id: string;
      sequence: number;
      turn_id: string;
      type: string;
      agent_name: string;
      content: unknown;
      timestamp: string;
      workspace_id?: string;
      workspace_path?: string;
    }>();

    expectTypeOf<TranscriptMessage>().toMatchTypeOf<{
      id: string;
      role: string;
      content: string;
      thinking_complete: boolean;
      tool_error: boolean;
      timestamp: string;
      tool_result?: TranscriptToolResult | null;
    }>();

    expectTypeOf<CreateSessionParams>().toMatchTypeOf<{
      agent_name?: string;
      name?: string;
      workspace?: string;
      workspace_path?: string;
    }>();

    expectTypeOf<ApproveSessionParams>().toMatchTypeOf<{
      request_id: string;
      turn_id: string;
      decision: string;
    }>();
  });

  it("keeps streaming-only local types separate from REST contract types", () => {
    expectTypeOf<AgentEventPayload>().toMatchTypeOf<{
      type: string;
      usage?: {
        input_tokens?: number;
        output_tokens?: number;
      };
      raw?: unknown;
    }>();

    expectTypeOf<UIMessageRole>().toEqualTypeOf<
      "user" | "assistant" | "tool_call" | "tool_result" | "system"
    >();
    expect(uiMessageRoles).toEqual(["user", "assistant", "tool_call", "tool_result", "system"]);
  });
});
