import { format } from "node:util";

import {
  CapabilityDeniedError,
  InternalError,
  InvalidParamsError,
  MethodNotFoundError,
  NotInitializedError,
  ShutdownInProgressError,
  ToolExecutionError,
  isRPCError,
} from "./errors.js";
import { HostAPI } from "./host-api.js";
import { schemaDigest } from "./schema-digest.js";
import { StdioTransport } from "./transport.js";
import type {
  AcceptedCapabilities,
  ExtensionDefinition,
  ExtensionProvideToolsResponse,
  HealthCheckResult,
  HookEvent,
  HostAPIMethod,
  ExtensionToolCallRequest,
  ExtensionToolCallResponse,
  ExtensionToolRuntimeDescriptor,
  InitializeRequest,
  InitializeResponse,
  InitializeRuntime,
  JSONRPCRequestEnvelope,
  JSONValue,
  RiskClass,
  ShutdownRequest,
  ShutdownResponse,
  ToolID,
  ToolResult,
} from "./types.js";
import type { TransportLike } from "./transport.js";

const SDK_NAME = "@agh/extension-sdk";
const SDK_VERSION = "0.1.0";
const SUPPORTED_PROTOCOL_VERSIONS = ["1"];
const TOOL_PROVIDER_CAPABILITY = "tool.provider";
const PROVIDE_TOOLS_METHOD = "provide_tools";
const TOOLS_CALL_METHOD = "tools/call";
const REQUIRED_PROVIDES_METHODS: Record<string, string[]> = {
  "bridge.adapter": ["bridges/deliver"],
  "memory.backend": ["memory/store", "memory/recall", "memory/forget"],
  [TOOL_PROVIDER_CAPABILITY]: [PROVIDE_TOOLS_METHOD, TOOLS_CALL_METHOD],
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

export interface ExtensionToolOptions {
  id?: ToolID;
  description?: string;
  inputSchema: JSONValue;
  outputSchema?: JSONValue;
  readOnly?: boolean;
  risk?: RiskClass;
  capabilities?: string[];
  sensitiveInputFields?: string[];
}

export interface ExtensionToolContext<TInput = unknown> {
  readonly input: TInput;
  readonly context: ExtensionContext;
  readonly host: HostAPI;
  readonly session: ExtensionSession;
  readonly toolID: ToolID;
  readonly handler: string;
}

export type ExtensionToolHandler<TInput = unknown> = (
  request: ExtensionToolContext<TInput>
) => Promise<ToolResult | JSONValue | string | void> | ToolResult | JSONValue | string | void;

type ReadyCallback = (host: HostAPI, session: ExtensionSession) => Promise<void> | void;

interface RegisteredTool {
  descriptor: ExtensionToolRuntimeDescriptor;
  handler: ExtensionToolHandler;
  sensitiveInputFields: string[];
}

export class Extension {
  private transport: TransportLike;
  private readonly stderr: NodeJS.WritableStream;
  private readonly sdkVersion: string;
  private readonly handlers = new Map<string, ExtensionHandler>();
  private readonly toolHandlers = new Map<string, RegisteredTool>();
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
    if (this.toolHandlers.size > 0 && isToolProviderMethod(cleanMethod)) {
      throw new Error(`${cleanMethod} is reserved by extension.tool()`);
    }
    this.handlers.set(cleanMethod, handler as ExtensionHandler);
    this.bindMethod(cleanMethod);
    return this;
  }

  public tool<TInput = unknown>(
    handler: string,
    options: ExtensionToolOptions,
    toolHandler: ExtensionToolHandler<TInput>
  ): this {
    const cleanHandler = handler.trim();
    if (!cleanHandler) {
      throw new Error("tool handler is required");
    }
    if (this.toolHandlers.has(cleanHandler)) {
      throw new Error(`tool handler already registered: ${cleanHandler}`);
    }
    if (this.handlers.has(PROVIDE_TOOLS_METHOD) || this.handlers.has(TOOLS_CALL_METHOD)) {
      throw new Error("provide_tools and tools/call are reserved by extension.tool()");
    }
    const inputSchema = normalizeSchema(options.inputSchema, "inputSchema");
    const outputSchema =
      options.outputSchema === undefined
        ? undefined
        : normalizeSchema(options.outputSchema, "outputSchema");
    const descriptor: ExtensionToolRuntimeDescriptor = {
      id: options.id ?? canonicalExtensionToolID(this.definition.name, cleanHandler),
      handler: cleanHandler,
      input_schema_digest: schemaDigest(inputSchema),
      ...(outputSchema ? { output_schema_digest: schemaDigest(outputSchema) } : {}),
      read_only: Boolean(options.readOnly),
      risk: options.risk ?? (options.readOnly ? "read" : "mutating"),
      capabilities: normalizeStringList(options.capabilities),
    };
    this.toolHandlers.set(cleanHandler, {
      descriptor,
      handler: toolHandler as ExtensionToolHandler,
      sensitiveInputFields: normalizeStringList(options.sensitiveInputFields),
    });
    this.ensureToolProviderCapability();
    this.ensureToolProviderHandlers();
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
    if (this.toolHandlers.size > 0) {
      methods.add(PROVIDE_TOOLS_METHOD);
      methods.add(TOOLS_CALL_METHOD);
    }
    return Array.from(methods).sort();
  }

  public getSupportedHookEvents(): HookEvent[] {
    return [...(this.definition.supported_hook_events ?? [])];
  }

  public getToolDescriptors(): ExtensionToolRuntimeDescriptor[] {
    return [...this.toolHandlers.values()].map(tool => ({
      ...tool.descriptor,
      capabilities: [...(tool.descriptor.capabilities ?? [])],
    }));
  }

  private bindTransportHandlers(): void {
    this.bindMethod("initialize");
    this.bindMethod("health_check");
    this.bindMethod("shutdown");
    for (const method of this.handlers.keys()) {
      this.bindMethod(method);
    }
    if (this.toolHandlers.size > 0) {
      this.bindMethod(PROVIDE_TOOLS_METHOD);
      this.bindMethod(TOOLS_CALL_METHOD);
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
      case PROVIDE_TOOLS_METHOD:
        return this.handleProvideTools();
      case TOOLS_CALL_METHOD:
        return await this.handleToolCall(request, params);
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

  private handleProvideTools(): ExtensionProvideToolsResponse {
    return {
      tools: this.getToolDescriptors(),
    };
  }

  private async handleToolCall(
    request: JSONRPCRequestEnvelope,
    params: unknown
  ): Promise<ExtensionToolCallResponse> {
    const call = parseToolCallRequest(params);
    const registered = this.toolHandlers.get(call.handler);
    if (!registered) {
      throw new MethodNotFoundError(call.handler);
    }
    if (registered.descriptor.id !== call.tool_id) {
      throw new InvalidParamsError("tool_id does not match handler", {
        expected_tool_id: registered.descriptor.id,
        actual_tool_id: call.tool_id,
        handler: call.handler,
      });
    }

    const context = this.makeContext(request);
    try {
      const result = await registered.handler({
        input: call.input,
        context,
        host: context.host,
        session: context.session,
        toolID: call.tool_id,
        handler: call.handler,
      });
      return { result: normalizeToolResult(result) };
    } catch (error) {
      if (isRPCError(error)) {
        throw error;
      }
      throw toolExecutionError(error, call, registered.sensitiveInputFields);
    }
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
    if (typeof request.session_nonce !== "string" || request.session_nonce.trim() === "") {
      throw new InvalidParamsError("session_nonce is required");
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
    if (!request.capabilities || typeof request.capabilities !== "object") {
      throw new InvalidParamsError("capabilities are required");
    }
    if (!Array.isArray(request.capabilities.provides)) {
      throw new InvalidParamsError("capabilities.provides must be an array");
    }
    if (!Array.isArray(request.capabilities.granted_actions)) {
      throw new InvalidParamsError("capabilities.granted_actions must be an array");
    }
    if (!Array.isArray(request.capabilities.granted_security)) {
      throw new InvalidParamsError("capabilities.granted_security must be an array");
    }
    if (!Array.isArray(request.capabilities.granted_resource_kinds)) {
      throw new InvalidParamsError("capabilities.granted_resource_kinds must be an array");
    }
    if (!Array.isArray(request.capabilities.granted_resource_scopes)) {
      throw new InvalidParamsError("capabilities.granted_resource_scopes must be an array");
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

  private ensureToolProviderCapability(): void {
    const capabilities = this.definition.capabilities ?? {};
    capabilities.provides = normalizeStringList([
      ...(capabilities.provides ?? []),
      TOOL_PROVIDER_CAPABILITY,
    ]);
    this.definition.capabilities = capabilities;
  }

  private ensureToolProviderHandlers(): void {
    this.bindMethod(PROVIDE_TOOLS_METHOD);
    this.bindMethod(TOOLS_CALL_METHOD);
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

function parseToolCallRequest(params: unknown): ExtensionToolCallRequest {
  if (typeof params !== "object" || params === null) {
    throw new InvalidParamsError("tools/call params must be an object");
  }
  const request = params as ExtensionToolCallRequest;
  if (typeof request.tool_id !== "string" || request.tool_id.trim() === "") {
    throw new InvalidParamsError("tool_id is required");
  }
  if (typeof request.handler !== "string" || request.handler.trim() === "") {
    throw new InvalidParamsError("handler is required");
  }
  return {
    tool_id: request.tool_id.trim(),
    handler: request.handler.trim(),
    ...(request.session_id ? { session_id: request.session_id } : {}),
    input: (request.input ?? {}) as JSONValue,
  };
}

function normalizeToolResult(value: ToolResult | JSONValue | string | void): ToolResult {
  if (value === undefined) {
    return emptyToolResult();
  }
  if (isToolResult(value)) {
    return { ...emptyToolResult(), ...value };
  }
  if (typeof value === "string") {
    return {
      ...emptyToolResult(),
      content: [{ type: "text", text: value }],
      preview: value,
    };
  }
  return {
    ...emptyToolResult(),
    structured: value,
  };
}

function emptyToolResult(): ToolResult {
  return {
    truncated: false,
    bytes: 0,
    duration_ms: 0,
  };
}

function isToolResult(value: ToolResult | JSONValue | string | void): value is ToolResult {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return false;
  }
  return [
    "content",
    "structured",
    "preview",
    "artifacts",
    "metadata",
    "redactions",
    "truncated",
    "bytes",
    "duration_ms",
  ].some(key => Object.prototype.hasOwnProperty.call(value, key));
}

function toolExecutionError(
  error: unknown,
  call: ExtensionToolCallRequest,
  sensitiveInputFields: string[]
): ToolExecutionError {
  const errorMessage = redactSensitiveText(
    error instanceof Error ? error.message : String(error),
    call.input,
    sensitiveInputFields
  );
  return new ToolExecutionError({
    tool_id: call.tool_id,
    handler: call.handler,
    error: errorMessage,
    ...(sensitiveInputFields.length > 0
      ? {
          input: redactSensitiveInput(call.input, sensitiveInputFields),
          sensitive_input_fields: sensitiveInputFields,
        }
      : {}),
  });
}

function redactSensitiveInput(value: JSONValue, sensitiveInputFields: string[]): JSONValue {
  const cloned = cloneJSON(value);
  for (const field of sensitiveInputFields) {
    redactPath(
      cloned,
      field
        .split(".")
        .map(part => part.trim())
        .filter(Boolean)
    );
  }
  return cloned;
}

function redactPath(value: JSONValue, path: string[]): void {
  if (path.length === 0 || typeof value !== "object" || value === null || Array.isArray(value)) {
    return;
  }
  const [head, ...rest] = path;
  if (head === undefined) {
    return;
  }
  if (!(head in value)) {
    return;
  }
  if (rest.length === 0) {
    value[head] = "[REDACTED]";
    return;
  }
  redactPath(value[head] as JSONValue, rest);
}

function redactSensitiveText(
  text: string,
  input: JSONValue,
  sensitiveInputFields: string[]
): string {
  let redacted = text;
  for (const field of sensitiveInputFields) {
    const value = readPath(
      input,
      field
        .split(".")
        .map(part => part.trim())
        .filter(Boolean)
    );
    if (typeof value === "string" && value !== "") {
      redacted = redacted.replaceAll(value, "[REDACTED]");
    }
  }
  return redacted;
}

function readPath(value: JSONValue, path: string[]): JSONValue | undefined {
  if (path.length === 0) {
    return value;
  }
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return undefined;
  }
  const [head, ...rest] = path;
  if (head === undefined) {
    return undefined;
  }
  if (!(head in value)) {
    return undefined;
  }
  return readPath(value[head] as JSONValue, rest);
}

function cloneJSON(value: JSONValue): JSONValue {
  if (value === null || typeof value !== "object") {
    return value;
  }
  if (Array.isArray(value)) {
    return value.map(item => cloneJSON(item));
  }
  return Object.fromEntries(Object.entries(value).map(([key, entry]) => [key, cloneJSON(entry)]));
}

function normalizeSchema(value: JSONValue, field: string): JSONValue {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error(`${field} must be a JSON object`);
  }
  return cloneJSON(value);
}

function isToolProviderMethod(method: string): boolean {
  return method === PROVIDE_TOOLS_METHOD || method === TOOLS_CALL_METHOD;
}

function canonicalExtensionToolID(extensionName: string, handler: string): ToolID {
  return `ext__${canonicalIDSegment(extensionName)}__${canonicalIDSegment(handler)}`;
}

function canonicalIDSegment(raw: string): string {
  const normalized = raw.trim().toLowerCase();
  let output = "";
  let lastUnderscore = false;
  for (const char of normalized) {
    if ((char >= "a" && char <= "z") || (char >= "0" && char <= "9")) {
      output += char;
      lastUnderscore = false;
      continue;
    }
    if (output.length > 0 && !lastUnderscore) {
      output += "_";
      lastUnderscore = true;
    }
  }
  const segment = output.replaceAll(/^_+|_+$/g, "");
  if (!/^[a-z][a-z0-9_]*$/.test(segment) || segment.includes("__")) {
    throw new Error(`invalid tool id segment: ${raw}`);
  }
  return segment;
}
