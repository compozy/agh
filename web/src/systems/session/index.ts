// Types
export type {
  ACPCaps,
  AgentEventPayload,
  AghPermissionData,
  ApproveSessionParams,
  CreateSessionParams,
  FetchSessionEventsParams,
  PermissionDecision,
  PermissionRequest,
  SessionEventPayload,
  SessionApprovalResponse,
  SessionEventsResponse,
  SessionHistoryResponse,
  SessionMessage,
  SessionPayload,
  SessionResponse,
  SessionState,
  SessionsResponse,
  SessionTranscriptResponse,
  SessionDataParts,
  TokenUsagePayload,
  ToolUseResult,
  TranscriptMessage,
  TranscriptMessageRole,
  TurnHistoryPayload,
} from "./types";

// Adapters
export {
  approveSession,
  createSession,
  fetchSession,
  fetchSessionEvents,
  fetchSessionHistory,
  fetchSessionTranscript,
  fetchSessions,
  resumeSession,
  stopSession,
} from "./adapters/session-api";

// Query infrastructure
export { sessionKeys } from "./lib/query-keys";
export {
  sessionDetailOptions,
  sessionEventsOptions,
  sessionHistoryOptions,
  sessionTranscriptOptions,
  sessionsListOptions,
} from "./lib/query-options";

// Stores
export { useSessionStore } from "./hooks/use-session-store";
export type {
  ComposerDraft,
  SessionState as SessionStoreState,
  SessionActions,
  SessionStore,
} from "./stores/session-store";

// Hooks
export { useSession, useSessions } from "./hooks/use-sessions";
export {
  useClearSessionConversation,
  useCreateSession,
  useResumeSession,
  useStopSession,
} from "./hooks/use-session-actions";

// Components
export { ToolCallCard, type ToolCallCardProps } from "./components/tool-call-card";
export { ChatHeader, type ChatHeaderProps } from "./components/chat-header";
export {
  SessionChatRuntimeProvider,
  type SessionChatRuntimeProviderProps,
} from "./components/session-chat-runtime-provider";
export { ThinkingBlock, type ThinkingBlockProps } from "./components/thinking-block";
export { PermissionPrompt, type PermissionPromptProps } from "./components/permission-prompt";
export {
  SessionInspector,
  SessionInspectorDrawer,
  deriveFileReads,
  deriveTraceEvents,
  type InspectorFileEntry,
  type InspectorMemoryDoc,
  type InspectorTraceEvent,
  type InspectorTraceKind,
  type InspectorTraceStatus,
  type InspectorUsage,
  type SessionInspectorProps,
} from "./components/session-inspector";
