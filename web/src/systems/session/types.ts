import type { UIMessage as AIUIMessage } from "ai";

import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type SessionsResponse = OperationResponse<"listSessions", 200>;
export type SessionPayload = SessionsResponse["sessions"][number];
export type SessionResponse = OperationResponse<"getSession", 200>;
export type ACPCaps = NonNullable<SessionPayload["acp_caps"]>;
export type SessionState = SessionPayload["state"];
export type SessionFailurePayload = NonNullable<SessionPayload["failure"]>;
export type SessionLineagePayload = NonNullable<SessionPayload["lineage"]>;
export type AgentMePayload = OperationResponse<"getAgentMe", 200>["me"];
export type AgentContextPayload = OperationResponse<"getAgentContext", 200>["context"];
export type AgentSpawnPayload = OperationResponse<"spawnAgentSession", 201>["spawn"];
export type CoordinatorConfigPayload = OperationResponse<
  "getAgentCoordinatorConfig",
  200
>["coordinator"];

export type SessionEventsResponse = OperationResponse<"listSessionEvents", 200>;
export type SessionEventPayload = SessionEventsResponse["events"][number];
export type FetchSessionEventsParams = OperationQuery<"listSessionEvents">;

export type SessionHistoryResponse = OperationResponse<"getSessionHistory", 200>;
export type TurnHistoryPayload = SessionHistoryResponse["history"][number];

export type SessionTranscriptResponse = OperationResponse<"getSessionTranscript", 200>;
export type SessionBadge = SessionPayload["badge"];
export type SessionAttachResponse = OperationResponse<"attachSession", 200>;
export type SessionRecapResponse = OperationResponse<"getSessionRecap", 200>;
export type SessionRecapPayload = SessionRecapResponse["recap"];
export type TranscriptMarkerPayload = SessionRecapPayload["recent_markers"][number];
export type SessionRepairResponse = OperationResponse<"repairSession", 200>;
export type SessionRepairPayload = SessionRepairResponse["repair"];
export type SessionRepairQuery = OperationQuery<"repairSession">;

export type SessionLedgerResponse = OperationResponse<"getMemorySessionLedger", 200>;
export type SessionLedgerMeta = SessionLedgerResponse["meta"];
export type SessionLedgerEvent = SessionLedgerResponse["events"][number];

export type CreateSessionParams = OperationRequestBody<"createSession">;
export type SessionApprovalResponse = OperationResponse<"approveSession", 200>;
export type ApproveSessionParams = OperationRequestBody<"approveSession">;
export type PermissionDecision = ApproveSessionParams["decision"];

export interface ToolUseResult {
  stdout?: string;
  stderr?: string;
  filePath?: string;
  content?: string;
  structuredPatch?: unknown[];
  error?: string;
  rawOutput?: unknown;
}

export interface TokenUsagePayload {
  turn_id?: string;
  input_tokens?: number;
  output_tokens?: number;
  total_tokens?: number;
  thought_tokens?: number;
  cache_read_tokens?: number;
  cache_write_tokens?: number;
  context_used?: number;
  context_size?: number;
  cost_amount?: number;
  cost_currency?: string;
  timestamp?: string;
}

export interface RuntimeActivityPayload {
  turn_id?: string;
  turn_source?: string;
  turn_started_at?: string | null;
  deadline_at?: string | null;
  last_activity_at?: string | null;
  last_activity_kind?: string;
  last_activity_detail?: string;
  current_tool?: string;
  tool_call_id?: string;
  last_progress_at?: string | null;
  iteration_current?: number;
  iteration_max?: number;
  idle_seconds?: number;
  elapsed_ms: number;
  elapsed_seconds?: number;
}

export interface AgentEventPayload {
  type: string;
  session_id?: string;
  turn_id?: string;
  request_id?: string;
  timestamp?: string;
  text?: string;
  title?: string;
  tool_call_id?: string;
  stop_reason?: string;
  action?: string;
  resource?: string;
  decision?: string;
  error?: string;
  failure?: SessionFailurePayload;
  usage?: TokenUsagePayload;
  runtime?: RuntimeActivityPayload;
  marker?: TranscriptMarkerPayload;
  raw?: unknown;
}

export interface AghPermissionData extends AgentEventPayload {
  request_id: string;
  raw?: Record<string, unknown>;
}

export interface SessionDataParts extends Record<string, unknown> {
  "agh-event": AgentEventPayload;
  "agh-permission": AghPermissionData;
}

export type SessionMessage = AIUIMessage<unknown, SessionDataParts>;
export type TranscriptMessage = SessionMessage;
export type TranscriptMessageRole = TranscriptMessage["role"];

export const uiMessageRoles = [
  "user",
  "assistant",
  "tool_call",
  "tool_result",
  "system",
  "diff",
] as const;

export type UIMessageRole = (typeof uiMessageRoles)[number];

export interface UIMessageDiff {
  language?: string;
  content: string;
  path?: string;
  additions?: number;
  removals?: number;
}

export interface UIMessage {
  id: string;
  role: UIMessageRole;
  content: string;
  toolName?: string;
  toolInput?: Record<string, unknown>;
  toolResult?: ToolUseResult;
  toolError?: boolean;
  thinking?: string;
  thinkingComplete?: boolean;
  isStreaming?: boolean;
  diff?: UIMessageDiff;
  timestamp: number;
}

export interface PermissionRequest {
  requestId: string;
  toolName: string;
  toolInput: Record<string, unknown>;
  action: string;
  resource: string;
  supportedDecisions?: PermissionDecision[];
  turnId?: string;
  toolCallId?: string;
}
