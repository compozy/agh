import { describe, expectTypeOf, it } from "vitest";

import type { OperationRequestBody, OperationResponse } from "./api-contract";

describe("agent authored context contract types", () => {
  it("keeps Soul contracts aligned with generated OpenAPI types", () => {
    type SoulPut = OperationRequestBody<"putAgentSoul">;
    type SoulPayload = OperationResponse<"getAgentSoul", 200>;
    type AgentContext = OperationResponse<"getAgentContext", 200>["context"];

    expectTypeOf<SoulPut>().toMatchTypeOf<{
      agent_name: string;
      body: string;
      expected_digest: string;
      workspace_id?: string;
      idempotency_key?: string;
    }>();
    expectTypeOf<SoulPayload["validation_status"]>().toEqualTypeOf<
      "missing" | "inactive" | "valid" | "invalid"
    >();
    expectTypeOf<SoulPayload>().toMatchTypeOf<{
      body?: string;
      diagnostics?: {
        severity: "info" | "warning" | "error";
      }[];
    }>();
    expectTypeOf<AgentContext["soul"]>().toMatchTypeOf<{
      validation_status?: SoulPayload["validation_status"];
      snapshot_id?: string;
      digest?: string;
      tone: string[];
      principles: string[];
    }>();
  });

  it("keeps Heartbeat health and wake contracts aligned with generated OpenAPI types", () => {
    type HeartbeatPut = OperationRequestBody<"putAgentHeartbeat">;
    type HeartbeatStatus = OperationResponse<"getAgentHeartbeatStatus", 200>;
    type SessionHealth = OperationResponse<"getSessionHealth", 200>["health"];
    type WakeDecision = OperationResponse<"wakeAgentHeartbeat", 200>["decision"];

    expectTypeOf<HeartbeatPut>().toMatchTypeOf<{
      agent_name: string;
      body: string;
      expected_digest: string;
      workspace_id?: string;
      idempotency_key?: string;
    }>();
    expectTypeOf<SessionHealth["state"]>().toEqualTypeOf<
      "idle" | "prompting" | "stopped" | "detached"
    >();
    expectTypeOf<SessionHealth["health"]>().toEqualTypeOf<
      "healthy" | "degraded" | "stale" | "dead" | "unknown"
    >();
    expectTypeOf<WakeDecision["result"]>().toEqualTypeOf<
      "sent" | "skipped" | "coalesced" | "rate_limited" | "failed"
    >();
    expectTypeOf<WakeDecision["reason"]>().toEqualTypeOf<
      | "wake_sent"
      | "heartbeat_disabled"
      | "heartbeat_invalid"
      | "heartbeat_no_policy"
      | "heartbeat_rate_limited"
      | "heartbeat_no_eligible_session"
      | "cooldown_active"
      | "quiet_window"
      | "session_not_found"
      | "session_unhealthy"
      | "session_not_attachable"
      | "session_prompt_active"
      | "session_prompt_active_race"
      | "synthetic_prompt_failed"
      | "wake_coalesced"
    >();
    expectTypeOf<HeartbeatStatus["wake_state"]>().toMatchTypeOf<
      | {
          last_result: WakeDecision["result"];
          last_reason?: WakeDecision["reason"];
          coalesced_count: number;
        }
      | null
      | undefined
    >();
  });
});
