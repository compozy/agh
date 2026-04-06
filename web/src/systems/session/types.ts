import { z } from "zod";

// --- ACPCaps ---

export const acpCapsSchema = z.object({
  supports_load_session: z.boolean(),
  supported_modes: z.array(z.string()).optional(),
  supported_models: z.array(z.string()).optional(),
});

export type ACPCaps = z.infer<typeof acpCapsSchema>;

// --- SessionPayload ---

export const sessionStateSchema = z.enum(["starting", "active", "stopping", "stopped"]);

export type SessionState = z.infer<typeof sessionStateSchema>;

export const sessionPayloadSchema = z.object({
  id: z.string(),
  name: z.string().optional(),
  agent_name: z.string(),
  workspace: z.string(),
  state: sessionStateSchema,
  acp_session_id: z.string().optional(),
  acp_caps: acpCapsSchema.optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type SessionPayload = z.infer<typeof sessionPayloadSchema>;

// --- SessionEventPayload ---

export const sessionEventPayloadSchema = z.object({
  id: z.string(),
  session_id: z.string(),
  sequence: z.number(),
  turn_id: z.string(),
  type: z.string(),
  agent_name: z.string(),
  content: z.unknown(),
  timestamp: z.string(),
});

export type SessionEventPayload = z.infer<typeof sessionEventPayloadSchema>;

// --- TokenUsagePayload ---

export const tokenUsagePayloadSchema = z.object({
  turn_id: z.string().optional(),
  input_tokens: z.number().optional(),
  output_tokens: z.number().optional(),
  total_tokens: z.number().optional(),
  thought_tokens: z.number().optional(),
  cache_read_tokens: z.number().optional(),
  cache_write_tokens: z.number().optional(),
  context_used: z.number().optional(),
  context_size: z.number().optional(),
  cost_amount: z.number().optional(),
  cost_currency: z.string().optional(),
  timestamp: z.string().optional(),
});

export type TokenUsagePayload = z.infer<typeof tokenUsagePayloadSchema>;

// --- AgentEventPayload ---

export const agentEventPayloadSchema = z.object({
  type: z.string(),
  session_id: z.string().optional(),
  turn_id: z.string().optional(),
  request_id: z.string().optional(),
  timestamp: z.string().optional(),
  text: z.string().optional(),
  title: z.string().optional(),
  tool_call_id: z.string().optional(),
  stop_reason: z.string().optional(),
  action: z.string().optional(),
  resource: z.string().optional(),
  decision: z.string().optional(),
  error: z.string().optional(),
  usage: tokenUsagePayloadSchema.optional(),
  raw: z.unknown().optional(),
});

export type AgentEventPayload = z.infer<typeof agentEventPayloadSchema>;

// --- TurnHistoryPayload ---

export const turnHistoryPayloadSchema = z.object({
  turn_id: z.string(),
  events: z.array(sessionEventPayloadSchema),
});

export type TurnHistoryPayload = z.infer<typeof turnHistoryPayloadSchema>;

// --- TranscriptPayload ---

export const transcriptToolResultSchema = z
  .object({
    stdout: z.string().optional(),
    stderr: z.string().optional(),
    file_path: z.string().optional(),
    content: z.string().optional(),
    structured_patch: z.unknown().optional(),
    error: z.string().optional(),
    raw_output: z.unknown().optional(),
  })
  .passthrough();

export type TranscriptToolResult = z.infer<typeof transcriptToolResultSchema>;

export const transcriptMessageRoleSchema = z.enum([
  "user",
  "assistant",
  "tool_call",
  "tool_result",
]);

export const transcriptMessageSchema = z
  .object({
    id: z.string(),
    role: transcriptMessageRoleSchema,
    content: z.string(),
    thinking: z.string().optional(),
    thinking_complete: z.boolean(),
    tool_name: z.string().optional(),
    tool_input: z.record(z.string(), z.unknown()).optional(),
    tool_result: transcriptToolResultSchema.optional(),
    tool_error: z.boolean(),
    timestamp: z.string(),
  })
  .passthrough();

export type TranscriptMessage = z.infer<typeof transcriptMessageSchema>;

// --- ToolUseResult ---

export interface ToolUseResult {
  stdout?: string;
  stderr?: string;
  filePath?: string;
  content?: string;
  structuredPatch?: unknown[];
  error?: string;
  rawOutput?: unknown;
}

// --- UIMessage ---

export const uiMessageRoleSchema = z.enum([
  "user",
  "assistant",
  "tool_call",
  "tool_result",
  "system",
]);

export type UIMessageRole = z.infer<typeof uiMessageRoleSchema>;

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

// --- PermissionRequest ---

export interface PermissionRequest {
  requestId: string;
  toolName: string;
  toolInput: Record<string, unknown>;
  action: string;
  resource: string;
}

// --- API Response Envelopes ---

export const sessionsResponseSchema = z.object({
  sessions: z.array(sessionPayloadSchema),
});

export type SessionsResponse = z.infer<typeof sessionsResponseSchema>;

export const sessionResponseSchema = z.object({
  session: sessionPayloadSchema,
});

export type SessionResponse = z.infer<typeof sessionResponseSchema>;

export const sessionEventsResponseSchema = z.object({
  events: z.array(sessionEventPayloadSchema),
});

export type SessionEventsResponse = z.infer<typeof sessionEventsResponseSchema>;

export const sessionHistoryResponseSchema = z.object({
  history: z.array(turnHistoryPayloadSchema),
});

export type SessionHistoryResponse = z.infer<typeof sessionHistoryResponseSchema>;

export const sessionTranscriptResponseSchema = z.object({
  messages: z.array(transcriptMessageSchema),
});

export type SessionTranscriptResponse = z.infer<typeof sessionTranscriptResponseSchema>;
