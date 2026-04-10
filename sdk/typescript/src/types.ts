export type JSONPrimitive = string | number | boolean | null;
export type JSONValue = JSONPrimitive | JSONValue[] | { [key: string]: JSONValue };
export type ISODateTime = string;
export type ProtocolVersion = "1";
export type JSONRPCID = string | number;

export interface JSONRPCRequestEnvelope<Params = unknown> {
  jsonrpc: "2.0";
  id?: JSONRPCID;
  method: string;
  params?: Params;
}

export interface JSONRPCResponseEnvelope<Result = unknown, Data = unknown> {
  jsonrpc: "2.0";
  id: JSONRPCID;
  result?: Result;
  error?: JSONRPCErrorObject<Data>;
}

export interface JSONRPCErrorObject<Data = unknown> {
  code: number;
  message: string;
  data?: Data;
}

export type ToolSource = "builtin" | "mcp" | "extension" | "dynamic";

export interface Tool {
  name: string;
  description: string;
  input_schema: JSONValue;
  read_only: boolean;
  source: ToolSource;
}

export type ExtensionSourceTier = "bundled" | "user" | "workspace" | "marketplace";
export type HookEventFamily =
  | "session"
  | "input"
  | "prompt"
  | "event"
  | "agent"
  | "turn"
  | "message"
  | "tool"
  | "permission"
  | "context";

export type HookEvent =
  | "session.pre_create"
  | "session.post_create"
  | "session.pre_resume"
  | "session.post_resume"
  | "session.pre_stop"
  | "session.post_stop"
  | "input.pre_submit"
  | "prompt.post_assemble"
  | "event.pre_record"
  | "event.post_record"
  | "agent.pre_start"
  | "agent.spawned"
  | "agent.crashed"
  | "agent.stopped"
  | "turn.start"
  | "turn.end"
  | "message.start"
  | "message.delta"
  | "message.end"
  | "tool.pre_call"
  | "tool.post_call"
  | "tool.post_error"
  | "permission.request"
  | "permission.resolved"
  | "permission.denied"
  | "context.pre_compact"
  | "context.post_compact";

export type HookSource = "native" | "config" | "agent_definition" | "skill";
export type HookSkillSource = "bundled" | "marketplace" | "user" | "additional" | "workspace";
export type HookMode = "sync" | "async";
export type HookRunOutcome = "applied" | "denied" | "failed" | "skipped" | "dropped" | "rejected";
export type HookExecutorKind = "native" | "subprocess" | "wasm";

export interface ContextBlock {
  kind?: string;
  text?: string;
  metadata?: Record<string, string>;
}

export interface ToolCallRef {
  tool_call_id?: string;
  tool_name?: string;
  tool_namespace?: string;
  read_only?: boolean;
}

export interface ToolLocation {
  path?: string;
  start_line?: number;
  end_line?: number;
}

export interface PermissionOption {
  decision?: string;
  option_id?: string;
  kind?: string;
  label?: string;
}

export interface PermissionToolCall {
  id?: string;
  kind?: string;
  title?: string;
  status?: string;
  locations?: ToolLocation[];
}

export interface PayloadBase {
  event: HookEvent;
  timestamp?: ISODateTime;
}

export interface SessionContext {
  session_id?: string;
  session_name?: string;
  session_type?: string;
  agent_name?: string;
  workspace_id?: string;
  workspace?: string;
  acp_session_id?: string;
  state?: string;
  created_at?: ISODateTime;
  updated_at?: ISODateTime;
}

export interface TurnContext {
  turn_id?: string;
}

export interface ControlPatch {
  deny?: boolean;
  deny_reason?: string;
}

export interface SessionCreatePatch extends ControlPatch {
  session_name?: string;
  session_type?: string;
  agent_name?: string;
  workspace_id?: string;
  workspace?: string;
}

export interface InputPreSubmitPayload extends PayloadBase, SessionContext, TurnContext {
  input_class?: string;
  message?: string;
  context_blocks?: ContextBlock[];
}

export interface InputPreSubmitPatch extends ControlPatch {
  message?: string;
  context_blocks?: ContextBlock[];
}

export interface PromptPayload extends PayloadBase, SessionContext, TurnContext {
  input_class?: string;
  prompt?: string;
  context_blocks?: ContextBlock[];
}

export interface PromptPatch extends ControlPatch {
  prompt?: string;
  context_blocks?: ContextBlock[];
}

export interface EventRecordPayload extends PayloadBase, SessionContext, TurnContext {
  record_type?: string;
  sequence?: number;
  content?: JSONValue;
}

export interface EventRecordPatch {
  labels?: Record<string, string>;
}

export interface AgentPreStartPayload extends PayloadBase, SessionContext {
  command?: string;
  args?: string[];
  cwd?: string;
  provider?: string;
  model?: string;
}

export interface AgentLifecyclePayload extends PayloadBase, SessionContext {
  command?: string;
  args?: string[];
  cwd?: string;
  pid?: number;
  provider?: string;
  model?: string;
  error?: string;
}

export interface AgentStartPatch extends ControlPatch {
  command?: string;
  args?: string[];
  cwd?: string;
}

export interface AgentLifecyclePatch {
  labels?: Record<string, string>;
}

export interface TurnPayload extends PayloadBase, SessionContext, TurnContext {
  input_class?: string;
  user_message?: string;
}

export interface TurnPatch extends ControlPatch {
  labels?: Record<string, string>;
}

export interface MessagePayload extends PayloadBase, SessionContext, TurnContext {
  message_id?: string;
  role?: string;
  delta_type?: string;
  text?: string;
  raw?: JSONValue;
}

export interface MessagePatch extends ControlPatch {
  role?: string;
  delta_type?: string;
  text?: string;
}

export interface ToolPreCallPayload extends PayloadBase, SessionContext, TurnContext, ToolCallRef {
  tool_input?: JSONValue;
}

export interface ToolPostCallPayload extends PayloadBase, SessionContext, TurnContext, ToolCallRef {
  title?: string;
  tool_input?: JSONValue;
  tool_result?: JSONValue;
}

export interface ToolPostErrorPayload
  extends PayloadBase, SessionContext, TurnContext, ToolCallRef {
  title?: string;
  tool_input?: JSONValue;
  error?: string;
}

export interface ToolCallPatch extends ControlPatch {
  tool_name?: string;
  tool_namespace?: string;
  read_only?: boolean;
  tool_input?: JSONValue;
}

export interface ToolResultPatch extends ControlPatch {
  title?: string;
  tool_result?: JSONValue;
  error?: string;
}

export interface PermissionRequestPayload extends PayloadBase, SessionContext, TurnContext {
  request_id?: string;
  action?: string;
  resource?: string;
  decision?: string;
  decision_class?: string;
  tool_input?: JSONValue;
  tool_call?: PermissionToolCall;
  options?: PermissionOption[];
}

export interface PermissionResolutionPayload extends PayloadBase, SessionContext, TurnContext {
  request_id?: string;
  action?: string;
  resource?: string;
  decision?: string;
  decision_class?: string;
  tool_input?: JSONValue;
  tool_call?: PermissionToolCall;
}

export interface PermissionRequestPatch extends ControlPatch {
  decision?: string;
  decision_class?: string;
  reason?: string;
}

export interface ContextCompactPayload extends PayloadBase, SessionContext, TurnContext {
  reason?: string;
  strategy?: string;
  summary?: string;
  context_blocks?: ContextBlock[];
}

export interface ContextCompactionPatch extends ControlPatch {
  reason?: string;
  strategy?: string;
  context_blocks?: ContextBlock[];
}

export type SessionLifecyclePayload = PayloadBase & SessionContext;
export type SessionPreCreatePayload = PayloadBase & SessionContext;
export type SessionPostCreatePayload = SessionLifecyclePayload;
export type SessionPreResumePayload = SessionLifecyclePayload;
export type SessionPostResumePayload = SessionLifecyclePayload;
export type SessionPreStopPayload = SessionLifecyclePayload;
export type SessionPostStopPayload = SessionLifecyclePayload;
export type SessionPostCreatePatch = SessionCreatePatch;
export type SessionPreResumePatch = SessionCreatePatch;
export type SessionPostResumePatch = SessionCreatePatch;
export type SessionPreStopPatch = SessionCreatePatch;
export type SessionPostStopPatch = SessionCreatePatch;
export type EventPreRecordPayload = EventRecordPayload;
export type EventPostRecordPayload = EventRecordPayload;
export type EventPreRecordPatch = EventRecordPatch;
export type EventPostRecordPatch = EventRecordPatch;
export type AgentSpawnedPayload = AgentLifecyclePayload;
export type AgentCrashedPayload = AgentLifecyclePayload;
export type AgentStoppedPayload = AgentLifecyclePayload;
export type AgentSpawnedPatch = AgentLifecyclePatch;
export type AgentCrashedPatch = AgentLifecyclePatch;
export type AgentStoppedPatch = AgentLifecyclePatch;
export type TurnStartPayload = TurnPayload;
export type TurnEndPayload = TurnPayload;
export type TurnStartPatch = TurnPatch;
export type TurnEndPatch = TurnPatch;
export type MessageStartPayload = MessagePayload;
export type MessageDeltaPayload = MessagePayload;
export type MessageEndPayload = MessagePayload;
export type MessageStartPatch = MessagePatch;
export type MessageDeltaPatch = MessagePatch;
export type MessageEndPatch = MessagePatch;
export type ToolPostErrorPatch = ToolResultPatch;
export type PermissionResolvedPayload = PermissionResolutionPayload;
export type PermissionDeniedPayload = PermissionResolutionPayload;
export type PermissionResolvedPatch = Record<string, never>;
export type PermissionDeniedPatch = Record<string, never>;
export type ContextPreCompactPayload = ContextCompactPayload;
export type ContextPostCompactPayload = ContextCompactPayload;
export type ContextPreCompactPatch = ContextCompactionPatch;
export type ContextPostCompactPatch = ContextCompactionPatch;

export interface HookPayloadByEvent {
  "session.pre_create": SessionPreCreatePayload;
  "session.post_create": SessionPostCreatePayload;
  "session.pre_resume": SessionPreResumePayload;
  "session.post_resume": SessionPostResumePayload;
  "session.pre_stop": SessionPreStopPayload;
  "session.post_stop": SessionPostStopPayload;
  "input.pre_submit": InputPreSubmitPayload;
  "prompt.post_assemble": PromptPayload;
  "event.pre_record": EventPreRecordPayload;
  "event.post_record": EventPostRecordPayload;
  "agent.pre_start": AgentPreStartPayload;
  "agent.spawned": AgentSpawnedPayload;
  "agent.crashed": AgentCrashedPayload;
  "agent.stopped": AgentStoppedPayload;
  "turn.start": TurnStartPayload;
  "turn.end": TurnEndPayload;
  "message.start": MessageStartPayload;
  "message.delta": MessageDeltaPayload;
  "message.end": MessageEndPayload;
  "tool.pre_call": ToolPreCallPayload;
  "tool.post_call": ToolPostCallPayload;
  "tool.post_error": ToolPostErrorPayload;
  "permission.request": PermissionRequestPayload;
  "permission.resolved": PermissionResolvedPayload;
  "permission.denied": PermissionDeniedPayload;
  "context.pre_compact": ContextPreCompactPayload;
  "context.post_compact": ContextPostCompactPayload;
}

export interface HookPatchByEvent {
  "session.pre_create": SessionCreatePatch;
  "session.post_create": SessionPostCreatePatch;
  "session.pre_resume": SessionPreResumePatch;
  "session.post_resume": SessionPostResumePatch;
  "session.pre_stop": SessionPreStopPatch;
  "session.post_stop": SessionPostStopPatch;
  "input.pre_submit": InputPreSubmitPatch;
  "prompt.post_assemble": PromptPatch;
  "event.pre_record": EventPreRecordPatch;
  "event.post_record": EventPostRecordPatch;
  "agent.pre_start": AgentStartPatch;
  "agent.spawned": AgentSpawnedPatch;
  "agent.crashed": AgentCrashedPatch;
  "agent.stopped": AgentStoppedPatch;
  "turn.start": TurnStartPatch;
  "turn.end": TurnEndPatch;
  "message.start": MessageStartPatch;
  "message.delta": MessageDeltaPatch;
  "message.end": MessageEndPatch;
  "tool.pre_call": ToolCallPatch;
  "tool.post_call": ToolResultPatch;
  "tool.post_error": ToolPostErrorPatch;
  "permission.request": PermissionRequestPatch;
  "permission.resolved": PermissionResolvedPatch;
  "permission.denied": PermissionDeniedPatch;
  "context.pre_compact": ContextPreCompactPatch;
  "context.post_compact": ContextPostCompactPatch;
}

export interface HookMatcher {
  agent_name?: string;
  agent_type?: string;
  workspace_id?: string;
  workspace_root?: string;
  session_type?: string;
  input_class?: string;
  acp_event_type?: string;
  turn_id?: string;
  tool_name?: string;
  tool_namespace?: string;
  tool_read_only?: boolean;
  decision_class?: string;
  message_role?: string;
  message_delta_type?: string;
  compaction_reason?: string;
  compaction_strategy?: string;
}

export interface HookExecutorConfig {
  kind?: HookExecutorKind;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
}

export interface HookConfig {
  name: string;
  event: HookEvent;
  mode?: HookMode;
  required?: boolean;
  priority?: number;
  timeout?: string;
  matcher?: HookMatcher;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  executor?: HookExecutorConfig;
}

export interface HookDecl {
  name: string;
  event: HookEvent;
  source: HookSource;
  mode?: HookMode;
  required?: boolean;
  priority?: number;
  timeout?: number;
  matcher?: HookMatcher;
  executor_kind?: HookExecutorKind;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  metadata?: Record<string, string>;
  skill_source?: HookSkillSource;
}

export interface MCPServerConfig {
  command: string;
  args?: string[];
  env?: Record<string, string>;
}

export interface ResourcesConfig {
  skills?: string[];
  agents?: string[];
  hooks?: HookConfig[];
  mcp_servers?: Record<string, MCPServerConfig>;
}

export interface CapabilitiesConfig {
  provides?: string[];
}

export interface ActionsConfig {
  requires?: HostAPIMethod[];
}

export interface SubprocessConfig {
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  health_check_interval?: string;
  shutdown_timeout?: string;
}

export interface SecurityConfig {
  capabilities?: string[];
}

export interface ExtensionManifest {
  name: string;
  version: string;
  description?: string;
  min_agh_version?: string;
  resources?: ResourcesConfig;
  capabilities?: CapabilitiesConfig;
  actions?: ActionsConfig;
  subprocess?: SubprocessConfig;
  security?: SecurityConfig;
}

export interface ExtensionDefinition extends Pick<
  ExtensionManifest,
  "name" | "version" | "description" | "min_agh_version" | "capabilities" | "actions" | "security"
> {
  supported_hook_events?: HookEvent[];
}

export interface InitializeExtension {
  name: string;
  version: string;
  source_tier: ExtensionSourceTier;
}

export interface InitializeCapabilities {
  provides: string[];
  granted_actions: HostAPIMethod[];
  granted_security: string[];
}

export interface InitializeMethods {
  daemon_requests: string[];
  extension_services: string[];
}

export interface InitializeRuntime {
  health_check_interval_ms: number;
  health_check_timeout_ms: number;
  shutdown_timeout_ms: number;
  default_hook_timeout_ms: number;
}

export interface InitializeRequest {
  protocol_version: ProtocolVersion;
  supported_protocol_versions: ProtocolVersion[];
  agh_version: string;
  extension: InitializeExtension;
  capabilities: InitializeCapabilities;
  methods: InitializeMethods;
  runtime: InitializeRuntime;
}

export interface InitializeExtensionInfo {
  name: string;
  version: string;
  sdk_name?: string;
  sdk_version?: string;
}

export interface AcceptedCapabilities {
  provides: string[];
  actions: HostAPIMethod[];
  security: string[];
}

export interface InitializeSupports {
  health_check: boolean;
  provide_tools: boolean;
}

export interface InitializeResponse {
  protocol_version: ProtocolVersion;
  extension_info: InitializeExtensionInfo;
  accepted_capabilities: AcceptedCapabilities;
  implemented_methods: string[];
  supported_hook_events: HookEvent[];
  supports: InitializeSupports;
}

export interface ShutdownRequest {
  reason: string;
  deadline_ms: number;
}

export interface ShutdownResponse {
  acknowledged: boolean;
}

export interface HealthCheckResult {
  healthy: boolean;
  message?: string;
  details?: Record<string, JSONValue>;
}

export interface ProvideToolsResult {
  tools: Tool[];
}

export interface HookInvocation<TEvent extends HookEvent = HookEvent> {
  name: string;
  event: TEvent;
  mode: HookMode;
  required: boolean;
  timeout_ms: number;
  source: string;
  metadata?: Record<string, string>;
}

export interface ExecuteHookParams<TEvent extends HookEvent = HookEvent> {
  invocation_id: string;
  hook: HookInvocation<TEvent>;
  payload: HookPayloadByEvent[TEvent];
}

export type ExecuteHookResult<TEvent extends HookEvent = HookEvent> = HookPatchByEvent[TEvent];

export type SessionState = "starting" | "active" | "stopping" | "stopped";
export type StopReason =
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
export type MemoryScope = "global" | "workspace";
export type HostAPIMethod =
  | "sessions/list"
  | "sessions/create"
  | "sessions/prompt"
  | "sessions/stop"
  | "sessions/status"
  | "sessions/events"
  | "memory/recall"
  | "memory/store"
  | "memory/forget"
  | "observe/health"
  | "observe/events"
  | "skills/list";

export interface SessionsListParams {
  workspace?: string;
}

export interface SessionSummary {
  id: string;
  name?: string;
  agent: string;
  workspace?: string;
  state: SessionState;
  created_at: ISODateTime;
}

export interface SessionsCreateParams {
  agent: string;
  prompt?: string;
  workspace?: string;
}

export interface SessionCreateResult {
  session_id: string;
}

export interface SessionsPromptParams {
  session_id: string;
  message: string;
}

export interface SessionPromptResult {
  turn_id: string;
}

export interface SessionTargetParams {
  session_id: string;
}

export interface SessionStatus {
  session_id: string;
  name?: string;
  agent: string;
  workspace_id?: string;
  workspace?: string;
  state: SessionState;
  stop_reason?: StopReason;
  stop_detail?: string;
  acp_session_id?: string;
  created_at: ISODateTime;
  updated_at: ISODateTime;
}

export interface SessionEvent {
  type: string;
  timestamp: ISODateTime;
  data?: unknown;
}

export interface SessionEventsParams {
  session_id: string;
  type?: string;
  agent_name?: string;
  turn_id?: string;
  limit?: number;
  offset?: number;
  since?: ISODateTime;
}

export interface MemoryStoreParams {
  key: string;
  content: string;
  scope?: MemoryScope;
  workspace?: string;
  tags?: string[];
}

export interface MemoryRecallParams {
  query: string;
  limit?: number;
  scope?: MemoryScope;
  workspace?: string;
}

export interface MemoryForgetParams {
  key: string;
  scope?: MemoryScope;
  workspace?: string;
}

export interface MemoryRecallEntry {
  key: string;
  content: string;
  score: number;
}

export interface ObserveHealth {
  status: string;
  uptime_seconds: number;
  active_sessions: number;
  active_agents: number;
  global_db_size_bytes: number;
  session_db_size_bytes: number;
  version: string;
}

export interface ObserveEventsParams {
  session_id?: string;
  agent_name?: string;
  type?: string;
  since?: ISODateTime;
  limit?: number;
}

export interface SkillsListParams {
  workspace?: string;
}

export interface SkillSummary {
  name: string;
  description?: string;
  source: string;
}

export type EmptyResult = Record<string, never>;

export interface HostAPIMethodMap {
  "sessions/list": {
    params: SessionsListParams | undefined;
    result: SessionSummary[];
  };
  "sessions/create": {
    params: SessionsCreateParams;
    result: SessionCreateResult;
  };
  "sessions/prompt": {
    params: SessionsPromptParams;
    result: SessionPromptResult;
  };
  "sessions/stop": {
    params: SessionTargetParams;
    result: EmptyResult;
  };
  "sessions/status": {
    params: SessionTargetParams;
    result: SessionStatus;
  };
  "sessions/events": {
    params: SessionEventsParams;
    result: SessionEvent[];
  };
  "memory/recall": {
    params: MemoryRecallParams;
    result: MemoryRecallEntry[];
  };
  "memory/store": {
    params: MemoryStoreParams;
    result: EmptyResult;
  };
  "memory/forget": {
    params: MemoryForgetParams;
    result: EmptyResult;
  };
  "observe/health": {
    params: undefined;
    result: ObserveHealth;
  };
  "observe/events": {
    params: ObserveEventsParams | undefined;
    result: SessionEvent[];
  };
  "skills/list": {
    params: SkillsListParams | undefined;
    result: SkillSummary[];
  };
}
