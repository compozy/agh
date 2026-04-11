import { format } from "node:util";

import {
  CapabilityDeniedError,
  InternalError,
  InvalidParamsError,
  MethodNotFoundError,
  NotInitializedError,
  ShutdownInProgressError,
} from "./errors.js";
import { HostAPI } from "./host-api.js";
import { StdioTransport } from "./transport.js";
import type {
  AcceptedCapabilities,
  ExtensionDefinition,
  HealthCheckResult,
  HookEvent,
  HostAPIMethod,
  InitializeRequest,
  InitializeResponse,
  InitializeRuntime,
  JSONRPCRequestEnvelope,
  ProvideToolsResult,
  ShutdownRequest,
  ShutdownResponse,
} from "./types.js";
import type { TransportLike } from "./transport.js";

const SDK_NAME = "@agh/extension-sdk";
const SDK_VERSION = "0.1.0";
const SUPPORTED_PROTOCOL_VERSIONS = ["1"];
const REQUIRED_PROVIDES_METHODS: Record<string, string[]> = {
  "channel.adapter": ["channels/deliver"],
  "memory.backend": ["memory/store", "memory/recall", "memory/forget"],
};

export interface ExtensionOptions {
  transport?: TransportLike;
  stderr?: NodeJS.WritableStream;
  sdkVersion?: string;
}

export interface ExtensionSession {
  initializeRequest: InitializeRequest;
  initializeResponse: InitializeResponse;
  runtime: InitializeRuntime;
  acceptedCapabilities: AcceptedCapabilities;
}

export interface ExtensionContext {
  readonly request: JSONRPCRequestEnvelope;
  readonly requestId: string | number;
  readonly host: HostAPI;
  readonly session: ExtensionSession;
  log: (...values: unknown[]) => void;
}

export type ExtensionHandler<TParams = unknown, TResult = unknown> = (
  context: ExtensionContext,
  params: TParams
) => Promise<TResult> | TResult;

type ReadyCallback = (host: HostAPI, session: ExtensionSession) => Promise<void> | void;

export class Extension {
  private transport: TransportLike;
  private readonly stderr: NodeJS.WritableStream;
  private readonly sdkVersion: string;
  private readonly handlers = new Map<string, ExtensionHandler>();
  private readonly readyCallbacks = new Set<ReadyCallback>();
  private readonly transportBindings = new Set<string>();
  private readonly host: HostAPI;
  private initialized = false;
  private shutdownStarted = false;
  private shutdownDeadlineMS: number | undefined;
  private session: ExtensionSession | undefined;
  private startPromise: Promise<HostAPI> | undefined;
  private resolveStart: ((host: HostAPI) => void) | undefined;
  private rejectStart: ((reason: unknown) => void) | undefined;

  public constructor(
    public readonly definition: ExtensionDefinition,
    options: ExtensionOptions = {}
  ) {
    this.transport = options.transport ?? new StdioTransport();
    this.stderr = options.stderr ?? process.stderr;
    this.sdkVersion = options.sdkVersion ?? SDK_VERSION;
    this.host = new HostAPI(
      {
        call: async <TResult>(method: string, params?: unknown): Promise<TResult> =>
          await this.transport.call<TResult>(method, params),
      },
      { isReady: () => this.initialized && !this.shutdownStarted }
    );

    this.bindTransportHandlers();
  }

  public bindTransport(transport: TransportLike): this {
    if (this.startPromise) {
      throw new Error("transport may only be swapped before start()");
    }
    this.transport = transport;
    this.transportBindings.clear();
    this.bindTransportHandlers();
    return this;
  }

  public handle<TParams = unknown, TResult = unknown>(
    method: string,
    handler: ExtensionHandler<TParams, TResult>
  ): this {
    const cleanMethod = method.trim();
    if (cleanMethod === "initialize") {
      throw new Error("initialize is reserved by the SDK");
    }
    this.handlers.set(cleanMethod, handler as ExtensionHandler);
    this.bindMethod(cleanMethod);
    return this;
  }

  public onReady(callback: ReadyCallback): this {
    this.readyCallbacks.add(callback);
    if (this.initialized && this.session) {
      queueMicrotask(() => {
        void this.runReadyCallback(callback, this.session!);
      });
    }
    return this;
  }

  public async start(): Promise<HostAPI> {
    if (this.startPromise) {
      return await this.startPromise;
    }

    this.startPromise = new Promise<HostAPI>((resolve, reject) => {
      this.resolveStart = resolve;
      this.rejectStart = reject;
    });

    this.transport.onTransportError(error => {
      if (!this.initialized && this.rejectStart) {
        this.rejectStart(error);
      }
      this.logError("transport error", error);
    });
    this.transport.start();

    return await this.startPromise;
  }

  public getImplementedMethods(): string[] {
    const methods = new Set<string>(["health_check", "shutdown"]);
    for (const method of this.handlers.keys()) {
      methods.add(method);
    }
    if (this.handlers.has("provide_tools")) {
      methods.add("provide_tools");
    }
    return Array.from(methods).sort();
  }

  public getSupportedHookEvents(): HookEvent[] {
    return [...(this.definition.supported_hook_events ?? [])];
  }

  private bindTransportHandlers(): void {
    this.bindMethod("initialize");
    this.bindMethod("health_check");
    this.bindMethod("shutdown");
    this.bindMethod("provide_tools");
    for (const method of this.handlers.keys()) {
      this.bindMethod(method);
    }
  }

  private bindMethod(method: string): void {
    if (this.transportBindings.has(method)) {
      this.transport.handle(
        method,
        async (params, request) => await this.dispatch(method, params, request)
      );
      return;
    }
    this.transportBindings.add(method);
    this.transport.handle(
      method,
      async (params, request) => await this.dispatch(method, params, request)
    );
  }

  private async dispatch(
    method: string,
    params: unknown,
    request: JSONRPCRequestEnvelope
  ): Promise<unknown> {
    if (method === "initialize") {
      return await this.handleInitialize(params);
    }
    if (!this.initialized || !this.session) {
      throw new NotInitializedError();
    }
    if (this.shutdownStarted && method !== "shutdown") {
      throw new ShutdownInProgressError(
        this.shutdownDeadlineMS === undefined ? {} : { deadline_ms: this.shutdownDeadlineMS }
      );
    }

    switch (method) {
      case "health_check":
        return await this.handleHealthCheck(request, params);
      case "shutdown":
        return await this.handleShutdown(request, params);
      case "provide_tools":
        return await this.handleProvideTools(request, params);
      default:
        return await this.handleUserMethod(method, request, params);
    }
  }

  private async handleInitialize(params: unknown): Promise<InitializeResponse> {
    if (this.initialized) {
      throw new InvalidParamsError("initialize may only be called once");
    }

    const request = this.parseInitializeRequest(params);
    this.validateProtocolVersion(request);

    const requestedProvides = normalizeStringList(this.definition.capabilities?.provides);
    const requestedActions = normalizeHostMethodList(this.definition.actions?.requires);
    const requestedSecurity = normalizeStringList(this.definition.security?.capabilities);

    ensureSubset("provides", requestedProvides, request.capabilities.provides);
    ensureSubset("actions", requestedActions, request.capabilities.granted_actions);
    ensureSubset("security", requestedSecurity, request.capabilities.granted_security);

    const implementedMethods = this.getImplementedMethods();
    validateProvidedMethodCoverage(requestedProvides, implementedMethods);

    const response: InitializeResponse = {
      protocol_version: "1",
      extension_info: {
        name: this.definition.name,
        version: this.definition.version,
        sdk_name: SDK_NAME,
        sdk_version: this.sdkVersion,
      },
      accepted_capabilities: {
        provides: requestedProvides,
        actions: requestedActions,
        security: requestedSecurity,
      },
      implemented_methods: implementedMethods,
      supported_hook_events: this.getSupportedHookEvents(),
      supports: {
        health_check: true,
        provide_tools: this.handlers.has("provide_tools"),
      },
    };

    this.initialized = true;
    this.session = {
      initializeRequest: request,
      initializeResponse: response,
      runtime: request.runtime,
      acceptedCapabilities: response.accepted_capabilities,
    };

    setImmediate(() => {
      void this.finishInitialization();
    });

    return response;
  }

  private async finishInitialization(): Promise<void> {
    if (!this.session) {
      return;
    }
    for (const callback of this.readyCallbacks) {
      await this.runReadyCallback(callback, this.session);
    }
    this.resolveStart?.(this.host);
    this.resolveStart = undefined;
    this.rejectStart = undefined;
  }

  private async runReadyCallback(
    callback: ReadyCallback,
    session: ExtensionSession
  ): Promise<void> {
    try {
      await callback(this.host, session);
    } catch (error) {
      this.logError("onReady callback failed", error);
    }
  }

  private async handleHealthCheck(
    request: JSONRPCRequestEnvelope,
    params: unknown
  ): Promise<HealthCheckResult> {
    const customHandler = this.handlers.get("health_check");
    if (!customHandler) {
      return {
        healthy: true,
        message: "",
        details: {},
      };
    }
    return (await customHandler(this.makeContext(request), params as never)) as HealthCheckResult;
  }

  private async handleShutdown(
    request: JSONRPCRequestEnvelope,
    params: unknown
  ): Promise<ShutdownResponse> {
    const shutdownRequest = parseShutdownRequest(params);
    this.shutdownStarted = true;
    this.shutdownDeadlineMS = shutdownRequest.deadline_ms;

    const customHandler = this.handlers.get("shutdown");
    if (customHandler) {
      const result = (await customHandler(
        this.makeContext(request),
        shutdownRequest as never
      )) as ShutdownResponse;
      return result ?? { acknowledged: true };
    }
    return { acknowledged: true };
  }

  private async handleProvideTools(
    request: JSONRPCRequestEnvelope,
    params: unknown
  ): Promise<ProvideToolsResult> {
    const customHandler = this.handlers.get("provide_tools");
    if (!customHandler) {
      throw new MethodNotFoundError("provide_tools");
    }
    const result = (await customHandler(
      this.makeContext(request),
      params as never
    )) as ProvideToolsResult;
    if (!Array.isArray(result?.tools)) {
      throw new InvalidParamsError("provide_tools must return a tools array");
    }
    return result;
  }

  private async handleUserMethod(
    method: string,
    request: JSONRPCRequestEnvelope,
    params: unknown
  ): Promise<unknown> {
    const handler = this.handlers.get(method);
    if (!handler) {
      throw new MethodNotFoundError(method);
    }
    return await handler(this.makeContext(request), params as never);
  }

  private makeContext(request: JSONRPCRequestEnvelope): ExtensionContext {
    if (!this.session) {
      throw new InternalError("session is not initialized");
    }
    if (request.id === undefined) {
      throw new InternalError("request id is required");
    }
    return {
      request,
      requestId: request.id,
      host: this.host,
      session: this.session,
      log: (...values: unknown[]) => {
        this.stderr.write(`${format(...values)}\n`);
      },
    };
  }

  private parseInitializeRequest(params: unknown): InitializeRequest {
    if (typeof params !== "object" || params === null) {
      throw new InvalidParamsError("initialize params must be an object");
    }

    const request = params as InitializeRequest;
    if (!request.protocol_version) {
      throw new InvalidParamsError("protocol_version is required");
    }
    if (
      !Array.isArray(request.supported_protocol_versions) ||
      request.supported_protocol_versions.length === 0
    ) {
      throw new InvalidParamsError("supported_protocol_versions is required");
    }
    if (!request.extension?.name || !request.extension?.version) {
      throw new InvalidParamsError("extension identity is required");
    }
    if (!request.runtime) {
      throw new InvalidParamsError("runtime is required");
    }
    for (const field of [
      "health_check_interval_ms",
      "health_check_timeout_ms",
      "shutdown_timeout_ms",
      "default_hook_timeout_ms",
    ] as const) {
      if (!Number.isFinite(request.runtime[field]) || request.runtime[field] <= 0) {
        throw new InvalidParamsError(`${field} must be greater than zero`);
      }
    }
    return request;
  }

  private validateProtocolVersion(request: InitializeRequest): void {
    if (!SUPPORTED_PROTOCOL_VERSIONS.includes(request.protocol_version)) {
      throw new InvalidParamsError("unsupported protocol version", {
        reason: "unsupported_protocol_version",
        protocol_version: request.protocol_version,
        supported_protocol_versions: SUPPORTED_PROTOCOL_VERSIONS,
      });
    }
    if (!request.supported_protocol_versions.includes("1")) {
      throw new InvalidParamsError("supported protocol versions must include 1");
    }
  }

  private logError(message: string, error: unknown): void {
    const detail = error instanceof Error ? (error.stack ?? error.message) : String(error);
    this.stderr.write(`${message}: ${detail}\n`);
  }
}

function normalizeStringList(values: readonly string[] | undefined): string[] {
  return Array.from(new Set((values ?? []).map(value => value.trim()).filter(Boolean))).sort();
}

function normalizeHostMethodList(values: readonly HostAPIMethod[] | undefined): HostAPIMethod[] {
  return normalizeStringList(values) as HostAPIMethod[];
}

function ensureSubset(label: string, requested: string[], granted: readonly string[]): void {
  const grantedSet = new Set(granted.map(value => value.trim()));
  const missing = requested.filter(value => !grantedSet.has(value));
  if (missing.length > 0) {
    throw new CapabilityDeniedError({
      field: label,
      required: missing,
      granted: [...grantedSet],
    });
  }
}

function validateProvidedMethodCoverage(
  provides: readonly string[],
  implementedMethods: readonly string[]
): void {
  const implemented = new Set(implementedMethods);
  for (const capability of provides) {
    const requiredMethods = REQUIRED_PROVIDES_METHODS[capability];
    if (!requiredMethods) {
      continue;
    }
    const missing = requiredMethods.filter(method => !implemented.has(method));
    if (missing.length > 0) {
      throw new InternalError(`capability ${capability} requires methods ${missing.join(", ")}`);
    }
  }
}

function parseShutdownRequest(params: unknown): ShutdownRequest {
  if (typeof params !== "object" || params === null) {
    throw new InvalidParamsError("shutdown params must be an object");
  }
  const request = params as ShutdownRequest;
  if (!Number.isFinite(request.deadline_ms) || request.deadline_ms <= 0) {
    throw new InvalidParamsError("deadline_ms must be greater than zero");
  }
  return {
    reason: request.reason ?? "shutdown",
    deadline_ms: request.deadline_ms,
  };
}
