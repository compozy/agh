import { InvalidParamsError, NotInitializedError } from "./errors.js";
import type {
  BridgeInstance,
  BridgeInstanceTargetParams,
  BridgesInstancesReportStateParams,
  BridgesMessagesIngestResult,
  HostAPIMethod,
  HostAPIMethodMap,
  InboundMessageEnvelope,
  ObserveEventsParams,
  ObserveHealth,
  ResourceGetParams,
  ResourceRecord,
  ResourcesListParams,
  ResourcesSnapshotParams,
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
    list: () => Promise<BridgeInstance[]>;
    ingest: (params: InboundMessageEnvelope) => Promise<BridgesMessagesIngestResult>;
    get: (params: BridgeInstanceTargetParams) => Promise<BridgeInstance>;
    reportState: (params: BridgesInstancesReportStateParams) => Promise<BridgeInstance>;
  };

  public readonly resources: {
    list: (params?: ResourcesListParams) => Promise<ResourceRecord[]>;
    get: (params: ResourceGetParams) => Promise<ResourceRecord>;
    snapshot: (params: ResourcesSnapshotParams) => Promise<Record<string, never>>;
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
      list: async () => await this.request("bridges/instances/list", undefined),
      ingest: async params => await this.request("bridges/messages/ingest", params),
      get: async params => await this.request("bridges/instances/get", params),
      reportState: async params => await this.request("bridges/instances/report_state", params),
    };

    this.resources = {
      list: async params => {
        validateResourcesListParams(params);
        return await this.request("resources/list", params);
      },
      get: async params => {
        validateResourceGetParams(params);
        return await this.request("resources/get", params);
      },
      snapshot: async params => {
        validateResourcesSnapshotParams(params);
        return await this.request("resources/snapshot", params);
      },
    };
  }

  public async request<TMethod extends HostAPIMethod>(
    method: TMethod,
    params: HostMethodParams<TMethod>
  ): Promise<HostMethodResult<TMethod>> {
    return await callHostMethod(this.transport, method, params, this.isReady);
  }
}

function validateResourcesListParams(params: ResourcesListParams | undefined): void {
  if (params === undefined) {
    return;
  }
  if (!isRecord(params)) {
    throw new InvalidParamsError("resources.list params must be an object");
  }
  if (params.kind !== undefined) {
    assertNonEmptyString(params.kind, "resources.list kind");
  }
  if (params.scope !== undefined) {
    validateResourceScope(params.scope, "resources.list scope");
  }
  const limit = params.limit;
  if (
    limit !== undefined &&
    limit !== null &&
    (typeof limit !== "number" || !Number.isInteger(limit) || limit < 0)
  ) {
    throw new InvalidParamsError("resources.list limit must be a non-negative integer");
  }
}

function validateResourceGetParams(params: ResourceGetParams): void {
  if (!isRecord(params)) {
    throw new InvalidParamsError("resources.get params must be an object");
  }
  assertNonEmptyString(params.kind, "resources.get kind");
  assertNonEmptyString(params.id, "resources.get id");
}

function validateResourcesSnapshotParams(params: ResourcesSnapshotParams): void {
  if (!isRecord(params)) {
    throw new InvalidParamsError("resources.snapshot params must be an object");
  }
  if (!Number.isInteger(params.source_version) || params.source_version <= 0) {
    throw new InvalidParamsError("resources.snapshot source_version must be a positive integer");
  }
  if (!Array.isArray(params.records)) {
    throw new InvalidParamsError("resources.snapshot records must be an array");
  }

  for (const [index, record] of params.records.entries()) {
    if (!isRecord(record)) {
      throw new InvalidParamsError(`resources.snapshot records[${index}] must be an object`);
    }
    assertNonEmptyString(record.kind, `resources.snapshot records[${index}].kind`);
    assertNonEmptyString(record.id, `resources.snapshot records[${index}].id`);
    validateResourceScope(record.scope, `resources.snapshot records[${index}].scope`);
    if (!Object.prototype.hasOwnProperty.call(record, "spec") || record.spec === undefined) {
      throw new InvalidParamsError(`resources.snapshot records[${index}].spec is required`);
    }
  }
}

function validateResourceScope(scope: unknown, field: string): void {
  if (!isRecord(scope)) {
    throw new InvalidParamsError(`${field} must be an object`);
  }
  if (scope.kind !== "global" && scope.kind !== "workspace") {
    throw new InvalidParamsError(`${field}.kind must be "global" or "workspace"`);
  }

  const id = typeof scope.id === "string" ? scope.id.trim() : "";
  if (scope.kind === "global" && id !== "") {
    throw new InvalidParamsError(`${field}.id must be empty for global scope`);
  }
  if (scope.kind === "workspace" && id === "") {
    throw new InvalidParamsError(`${field}.id is required for workspace scope`);
  }
}

function assertNonEmptyString(value: unknown, field: string): void {
  if (typeof value !== "string" || value.trim() === "") {
    throw new InvalidParamsError(`${field} is required`);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
