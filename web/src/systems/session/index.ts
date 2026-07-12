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
  NormalizedSessionTranscriptResponse,
  SessionEventPayload,
  SessionApprovalResponse,
  SessionEventsResponse,
  SessionHistoryResponse,
  SessionLedgerEvent,
  SessionLedgerMeta,
  SessionLedgerResponse,
  SessionMessage,
  SessionByIDResponse,
  SessionPayload,
  SessionPromptPayload,
  SessionPromptRequest,
  SessionPromptResponse,
  SessionPromptResult,
  SessionGoalCommandResult,
  SessionGoalContext,
  SessionGoalResponse,
  SessionGoalSnapshot,
  SessionGoalStatus,
  GoalPromptMeta,
  SessionAttachResponse,
  SessionBadge,
  SessionRecapPayload,
  SessionRecapResponse,
  SessionUsagePayload,
  SessionUsageResponse,
  SessionRepairPayload,
  SessionRepairQuery,
  SessionRepairResponse,
  SessionResponse,
  SessionState,
  SessionListFilters,
  SessionTranscriptPage,
  SessionsResponse,
  SessionsQuery,
  SessionTranscriptResponse,
  TranscriptMarkerPayload,
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
  cancelQueuedSessionPrompt,
  cancelSessionPrompt,
  createSession,
  deleteSession,
  fetchSession,
  fetchSessionById,
  fetchSessionEvents,
  fetchSessionHistory,
  fetchSessionGoal,
  fetchSessionLedger,
  fetchSessionRecap,
  fetchSessionUsage,
  fetchSessionTranscript,
  fetchSessions,
  interruptSessionPrompt,
  repairSession,
  resumeSession,
  sendSessionPrompt,
  SessionApiError,
  SessionLedgerUnavailableError,
  SessionNotFoundError,
  steerSessionPrompt,
  stopSession,
} from "./adapters/session-api";

// Query infrastructure
export { formatMessageTimestamp, formatMessageTimestampFull } from "./lib/format-timestamp";
export { sessionKeys } from "./lib/query-keys";
export {
  sessionByIdOptions,
  sessionDetailOptions,
  sessionEventsOptions,
  sessionHistoryOptions,
  sessionGoalOptions,
  sessionLedgerOptions,
  sessionRecapOptions,
  sessionUsageOptions,
  sessionTranscriptOptions,
  sessionsListOptions,
} from "./lib/query-options";
export {
  hasRunningSession,
  idleAttachableAgentNames,
  isSessionRunning,
  isUserControllableSession,
  runningAgentNames,
} from "./lib/session-running";

// Stores
export { useSessionStore } from "./hooks/use-session-store";
export type {
  ComposerDraft,
  SessionState as SessionStoreState,
  SessionActions,
  SessionStore,
} from "./stores/session-store";

// Hooks
export {
  useSession,
  useSessionById,
  useSessionLedger,
  useSessionGoal,
  useSessionRecap,
  useSessionUsage,
  useSessions,
} from "./hooks/use-sessions";
export {
  useSessionTranscriptThreadMessages,
  useSessionTranscriptThreadState,
} from "./hooks/use-session-transcript-thread-messages";
export type {
  SessionTranscriptThreadState,
  SessionTranscriptThreadStatus,
} from "./lib/session-transcript-thread-context-value";
export {
  useClearSessionConversation,
  useCancelQueuedSessionPrompt,
  useCreateSession,
  useDeleteSession,
  useInterruptSessionPrompt,
  useQueueSessionPrompt,
  useRepairSession,
  useResumeSession,
  useSendSessionPrompt,
  useSteerSessionPrompt,
  useStopSession,
  type CancelQueuedSessionPromptParams,
  type RepairSessionParams,
  type SendSessionPromptParams,
  type SessionPromptActionParams,
} from "./hooks/use-session-actions";
export {
  useSessionCreateDialog,
  type SessionCreateDialogApi,
  type SessionCreateDialogDraft,
  type SessionCreateDialogState,
} from "./hooks/use-session-create-dialog";
export {
  SessionCreateProvider,
  type SessionCreateContextValue,
} from "./contexts/session-create-context";
export { useSessionCreate } from "./hooks/use-session-create";

// Components
export {
  SessionCreateDialog,
  type SessionCreateDialogProps,
} from "./components/session-create-dialog";
export {
  SessionResumeFailure,
  type SessionResumeFailureProps,
} from "./components/session-resume-failure";
export { SessionToolCallRow, type SessionToolCallRowProps } from "./components/tool-call-card";
export {
  SessionChatRuntimeProvider,
  type SessionChatRuntimeProviderProps,
} from "./components/session-chat-runtime-provider";
export { ThinkingBlock, type ThinkingBlockProps } from "./components/thinking-block";
export { PermissionPrompt, type PermissionPromptProps } from "./components/permission-prompt";
export {
  SessionInspector,
  type InspectorMemoryState,
  type InspectorSessionLedger,
  type InspectorUsage,
  type SessionInspectorProps,
} from "./components/session-inspector";
export {
  deriveFileReads,
  deriveTraceEvents,
  type InspectorFileEntry,
  type InspectorTraceEvent,
  type InspectorTraceKind,
  type InspectorTraceStatus,
} from "./components/session-inspector.logic";
