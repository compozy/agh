import { NotInitializedError } from "./errors.js";
import type {
  BridgeInstance,
  BridgesInstancesReportStateParams,
  BridgesMessagesIngestResult,
  HostAPIMethod,
  HostAPIMethodMap,
  InboundMessageEnvelope,
  ObserveEventsParams,
  ObserveHealth,
  SessionCreateResult,
  SessionEvent,
  SessionPromptResult,
  SessionStatus,
  SessionSummary,
  SkillSummary,
  SkillsListParams,
  MemoryRecallEntry,
  MemoryStoreParams,
  MemoryRecallParams,
  MemoryForgetParams,
  SessionsCreateParams,
  SessionsListParams,
  SessionsPromptParams,
  SessionTargetParams,
  SessionEventsParams,
} from "./types.js";

interface HostAPITransport {
  call<TResult = unknown>(method: string, params?: unknown): Promise<TResult>;
}

interface HostAPIOptions {
  isReady?: () => boolean;
}

type HostMethodParams<TMethod extends HostAPIMethod> = HostAPIMethodMap[TMethod]["params"];
type HostMethodResult<TMethod extends HostAPIMethod> = HostAPIMethodMap[TMethod]["result"];

async function callHostMethod<TMethod extends HostAPIMethod>(
  transport: HostAPITransport,
  method: TMethod,
  params: HostMethodParams<TMethod>,
  isReady: () => boolean
): Promise<HostMethodResult<TMethod>> {
  if (!isReady()) {
    throw new NotInitializedError();
  }
  return (await transport.call(method, params)) as HostMethodResult<TMethod>;
}

export class HostAPI {
  private readonly isReady: () => boolean;

  public readonly sessions: {
    list: (params?: SessionsListParams) => Promise<SessionSummary[]>;
    create: (params: SessionsCreateParams) => Promise<SessionCreateResult>;
    prompt: (params: SessionsPromptParams) => Promise<SessionPromptResult>;
    stop: (params: SessionTargetParams) => Promise<Record<string, never>>;
    status: (params: SessionTargetParams) => Promise<SessionStatus>;
    events: (params: SessionEventsParams) => Promise<SessionEvent[]>;
  };

  public readonly memory: {
    recall: (params: MemoryRecallParams) => Promise<MemoryRecallEntry[]>;
    store: (params: MemoryStoreParams) => Promise<Record<string, never>>;
    forget: (params: MemoryForgetParams) => Promise<Record<string, never>>;
  };

  public readonly observe: {
    health: () => Promise<ObserveHealth>;
    events: (params?: ObserveEventsParams) => Promise<SessionEvent[]>;
  };

  public readonly skills: {
    list: (params?: SkillsListParams) => Promise<SkillSummary[]>;
  };

  public readonly bridges: {
    ingest: (params: InboundMessageEnvelope) => Promise<BridgesMessagesIngestResult>;
    get: () => Promise<BridgeInstance>;
    reportState: (params: BridgesInstancesReportStateParams) => Promise<BridgeInstance>;
  };

  public constructor(
    private readonly transport: HostAPITransport,
    options: HostAPIOptions = {}
  ) {
    this.isReady = options.isReady ?? (() => true);

    this.sessions = {
      list: async params => await this.request("sessions/list", params),
      create: async params => await this.request("sessions/create", params),
      prompt: async params => await this.request("sessions/prompt", params),
      stop: async params => await this.request("sessions/stop", params),
      status: async params => await this.request("sessions/status", params),
      events: async params => await this.request("sessions/events", params),
    };

    this.memory = {
      recall: async params => await this.request("memory/recall", params),
      store: async params => await this.request("memory/store", params),
      forget: async params => await this.request("memory/forget", params),
    };

    this.observe = {
      health: async () => await this.request("observe/health", undefined),
      events: async params => await this.request("observe/events", params),
    };

    this.skills = {
      list: async params => await this.request("skills/list", params),
    };

    this.bridges = {
      ingest: async params => await this.request("bridges/messages/ingest", params),
      get: async () => await this.request("bridges/instances/get", undefined),
      reportState: async params => await this.request("bridges/instances/report_state", params),
    };
  }

  public async request<TMethod extends HostAPIMethod>(
    method: TMethod,
    params: HostMethodParams<TMethod>
  ): Promise<HostMethodResult<TMethod>> {
    return await callHostMethod(this.transport, method, params, this.isReady);
  }
}
