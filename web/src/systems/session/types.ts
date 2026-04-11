import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type SessionsResponse = OperationResponse<"listSessions", 200>;
export type SessionPayload = SessionsResponse["sessions"][number];
export type SessionResponse = OperationResponse<"getSession", 200>;
export type ACPCaps = NonNullable<SessionPayload["acp_caps"]>;
export type SessionState = SessionPayload["state"];

export type SessionEventsResponse = OperationResponse<"listSessionEvents", 200>;
export type SessionEventPayload = SessionEventsResponse["events"][number];
export type FetchSessionEventsParams = OperationQuery<"listSessionEvents">;

export type SessionHistoryResponse = OperationResponse<"getSessionHistory", 200>;
export type TurnHistoryPayload = SessionHistoryResponse["history"][number];

export type SessionTranscriptResponse = OperationResponse<"getSessionTranscript", 200>;
export type TranscriptMessage = SessionTranscriptResponse["messages"][number];
export type TranscriptMessageRole = TranscriptMessage["role"];
export type TranscriptToolResult = NonNullable<TranscriptMessage["tool_result"]>;

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
  usage?: TokenUsagePayload;
  raw?: unknown;
}

export const uiMessageRoles = ["user", "assistant", "tool_call", "tool_result", "system"] as const;

export type UIMessageRole = (typeof uiMessageRoles)[number];

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
  timestamp: number;
}

export interface PermissionRequest {
  requestId: string;
  toolName: string;
  toolInput: Record<string, unknown>;
  action: string;
  resource: string;
}
