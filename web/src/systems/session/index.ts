// Types
export type {
  ACPCaps,
  AgentEventPayload,
  ApproveSessionParams,
  CreateSessionParams,
  FetchSessionEventsParams,
  PermissionDecision,
  PermissionRequest,
  SessionEventPayload,
  SessionApprovalResponse,
  SessionEventsResponse,
  SessionHistoryResponse,
  SessionPayload,
  SessionResponse,
  SessionState,
  SessionsResponse,
  SessionTranscriptResponse,
  TokenUsagePayload,
  ToolUseResult,
  TranscriptMessage,
  TranscriptMessageRole,
  TranscriptToolResult,
  TurnHistoryPayload,
  UIMessage,
  UIMessageRole,
} from "./types";
export { uiMessageRoles } from "./types";

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

// Lib
export { SimpleStreamingBuffer, mergeStreamingChunk } from "./lib/streaming-buffer";
export {
  mapAgentEventToUIMessage,
  extractPermissionRequest,
  mapHistoryToMessages,
} from "./lib/event-mapper";
export { mapTranscriptToMessages } from "./lib/transcript-mapper";
export {
  getToolIcon,
  getToolLabel,
  getToolCompactSummary,
  type ToolLabelTense,
} from "./lib/tool-labels";

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
export { useCreateSession, useResumeSession, useStopSession } from "./hooks/use-session-actions";
export {
  useSessionChat,
  type UseSessionChatOptions,
  type UseSessionChatReturn,
} from "./hooks/use-session-chat";
export { useSessionHistory, type UseSessionHistoryReturn } from "./hooks/use-session-history";
export {
  useSessionTranscript,
  type UseSessionTranscriptReturn,
} from "./hooks/use-session-transcript";

// Components
export { SessionSidebarItem } from "./components/session-sidebar-item";
export {
  ChatView,
  buildRows,
  mergeToolPairs,
  type RowDescriptor,
  type ChatViewProps,
} from "./components/chat-view";
export { ToolCallCard, type ToolCallCardProps } from "./components/tool-call-card";
export { ChatHeader, type ChatHeaderProps } from "./components/chat-header";
export { MessageBubble, type MessageBubbleProps } from "./components/message-bubble";
export {
  MessageComposer,
  type MessageComposerAttachment,
  type MessageComposerChannel,
  type MessageComposerPayload,
  type MessageComposerProps,
  type MessageComposerSkill,
} from "./components/message-composer";
export { ThinkingBlock, type ThinkingBlockProps } from "./components/thinking-block";
export {
  ProcessingIndicator,
  type ProcessingIndicatorProps,
} from "./components/processing-indicator";
export { PermissionPrompt, type PermissionPromptProps } from "./components/permission-prompt";
