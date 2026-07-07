import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  ApproveLoopRunRequest,
  CreateLoopRequest,
  LoopAnnotation,
  LoopAnnotationsUpdateRequest,
  LoopCatalogEntry,
  LoopConfig,
  LoopConfigUpdateRequest,
  LoopDetail,
  LoopRun,
  LoopRunActionResult,
  LoopRunAggregates,
  LoopRunDetail,
  LoopRunsFilter,
  LoopStreamFilter,
  PatchLoopRequest,
  RunLoopRequest,
  RunLoopResult,
  ValidateLoopRequest,
  ValidateLoopResult,
} from "../types";

export class LoopsApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "LoopsApiError";
  }
}

/**
 * A 422 lint rejection carrying the same per-node `{ node_id, code, message, severity }`
 * body the validate endpoint returns. Thrown by `patchLoop` so the editor can map a
 * publish rejection back onto nodes (task-22 MUST), not just show a generic banner.
 */
export class LoopValidationError extends LoopsApiError {
  constructor(
    message: string,
    public readonly result: ValidateLoopResult
  ) {
    super(message, 422);
    this.name = "LoopValidationError";
  }
}

function normalizeOptionalText(value?: string | null): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

/**
 * Builds the workspace-scoped Loop-run SSE URL, mirroring `buildTaskStreamUrl`.
 * `after_sequence` (incl. `0`, which opts into deterministic Last-Event-ID:0
 * precedence) is preserved; both the workspace id and run id are required.
 */
export function buildLoopStreamUrl(
  workspaceId: string,
  runId: string,
  filters: LoopStreamFilter = {}
): string {
  const trimmedWorkspace = workspaceId.trim();
  if (trimmedWorkspace === "") {
    throw new LoopsApiError("workspace id is required to build stream url", 400);
  }
  const trimmedRun = runId.trim();
  if (trimmedRun === "") {
    throw new LoopsApiError("loop run id is required to build stream url", 400);
  }
  const path = `/api/workspaces/${encodeURIComponent(trimmedWorkspace)}/loop-runs/${encodeURIComponent(
    trimmedRun
  )}/events`;
  if (filters.after_sequence === undefined) {
    return path;
  }
  return `${path}?after_sequence=${encodeURIComponent(String(filters.after_sequence))}`;
}

// Catalog + definition ------------------------------------------------------

export async function listLoops(
  workspaceId: string,
  signal?: AbortSignal
): Promise<LoopCatalogEntry[]> {
  const { data, error, response } = await apiClient.GET("/api/workspaces/{workspace_id}/loops", {
    params: { path: { workspace_id: workspaceId } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new LoopsApiError(
      defaultApiErrorMessage("Failed to fetch loops", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch loops").loops;
}

export async function getLoop(
  workspaceId: string,
  name: string,
  signal?: AbortSignal
): Promise<LoopDetail> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loops/{name}",
    {
      params: { path: { workspace_id: workspaceId, name } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to fetch loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch loop "${name}"`).loop;
}

export async function createLoop(
  workspaceId: string,
  body: CreateLoopRequest,
  signal?: AbortSignal
): Promise<LoopDetail> {
  const { data, error, response } = await apiClient.POST("/api/workspaces/{workspace_id}/loops", {
    params: { path: { workspace_id: workspaceId } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 409) {
      throw new LoopsApiError(defaultApiErrorMessage("Loop already exists", response, error), 409);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage("Failed to create loop", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to create loop").loop;
}

export async function patchLoop(
  workspaceId: string,
  name: string,
  body: PatchLoopRequest,
  signal?: AbortSignal
): Promise<LoopDetail> {
  const { data, error, response } = await apiClient.PATCH(
    "/api/workspaces/{workspace_id}/loops/{name}",
    {
      params: { path: { workspace_id: workspaceId, name } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    if (response.status === 409) {
      throw new LoopsApiError(
        defaultApiErrorMessage(`Loop "${name}" was modified by another editor`, response, error),
        409
      );
    }
    // A 422 is a per-node lint rejection, not a transport failure: carry the structured
    // `{ node_id, code, message, severity }` body so the editor maps it back onto nodes.
    if (response.status === 422 && isValidationResult(error)) {
      const count = error.errors?.length ?? 0;
      throw new LoopValidationError(
        `Publish rejected: ${name} has ${count} lint issue${count === 1 ? "" : "s"}`,
        error
      );
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to publish loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to publish loop "${name}"`).loop;
}

export async function deleteLoop(
  workspaceId: string,
  name: string,
  signal?: AbortSignal
): Promise<void> {
  const { error, response } = await apiClient.DELETE(
    "/api/workspaces/{workspace_id}/loops/{name}",
    {
      params: { path: { workspace_id: workspaceId, name } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to delete loop "${name}"`, response, error),
      response.status
    );
  }
}

function isValidationResult(value: unknown): value is ValidateLoopResult {
  return typeof value === "object" && value !== null && "valid" in value;
}

export async function validateLoop(
  workspaceId: string,
  name: string,
  body: ValidateLoopRequest,
  signal?: AbortSignal
): Promise<ValidateLoopResult> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/loops/{name}/validate",
    {
      params: { path: { workspace_id: workspaceId, name } },
      body,
      signal,
    }
  );

  // A 422 is a successful lint verdict, not a transport failure: openapi-fetch
  // parks the non-2xx body in `error`, which carries the same `{ valid, errors }`
  // shape as the 200 response — surface it instead of throwing.
  if (response.status === 422 && isValidationResult(error)) {
    return error;
  }

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to validate loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to validate loop "${name}"`);
}

// Config --------------------------------------------------------------------

export async function getLoopConfig(
  workspaceId: string,
  name: string,
  signal?: AbortSignal
): Promise<LoopConfig | null> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loops/{name}/config",
    {
      params: { path: { workspace_id: workspaceId, name } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      return null;
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to fetch config for loop "${name}"`, response, error),
      response.status
    );
  }

  return (
    requireResponseData(data, response, `Failed to fetch config for loop "${name}"`).config ?? null
  );
}

export async function putLoopConfig(
  workspaceId: string,
  name: string,
  body: LoopConfigUpdateRequest,
  signal?: AbortSignal
): Promise<LoopConfig | null> {
  const { data, error, response } = await apiClient.PUT(
    "/api/workspaces/{workspace_id}/loops/{name}/config",
    {
      params: { path: { workspace_id: workspaceId, name } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to update config for loop "${name}"`, response, error),
      response.status
    );
  }

  return (
    requireResponseData(data, response, `Failed to update config for loop "${name}"`).config ?? null
  );
}

// Editor annotations (node positions) ---------------------------------------

export async function getLoopAnnotations(
  workspaceId: string,
  name: string,
  signal?: AbortSignal
): Promise<LoopAnnotation[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loops/{name}/annotations",
    {
      params: { path: { workspace_id: workspaceId, name } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to fetch annotations for loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch annotations for loop "${name}"`)
    .annotations;
}

export async function putLoopAnnotations(
  workspaceId: string,
  name: string,
  body: LoopAnnotationsUpdateRequest,
  signal?: AbortSignal
): Promise<LoopAnnotation[]> {
  const { data, error, response } = await apiClient.PUT(
    "/api/workspaces/{workspace_id}/loops/{name}/annotations",
    {
      params: { path: { workspace_id: workspaceId, name } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to save annotations for loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to save annotations for loop "${name}"`)
    .annotations;
}

// Run start / dry-run -------------------------------------------------------

export async function runLoop(
  workspaceId: string,
  name: string,
  body: RunLoopRequest,
  options: { dry?: boolean } = {},
  signal?: AbortSignal
): Promise<RunLoopResult> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/loops/{name}/run",
    {
      params: {
        path: { workspace_id: workspaceId, name },
        query: options.dry ? { dry: true } : {},
      },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop not found: ${name}`, 404);
    }
    if (response.status === 409) {
      throw new LoopsApiError(
        defaultApiErrorMessage(`Loop "${name}" is already running`, response, error),
        409
      );
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to run loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to run loop "${name}"`);
}

// Runs ----------------------------------------------------------------------

export async function listLoopRuns(
  workspaceId: string,
  filters: LoopRunsFilter = {},
  signal?: AbortSignal
): Promise<{ runs: LoopRun[]; aggregates: LoopRunAggregates }> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loop-runs",
    {
      params: {
        path: { workspace_id: workspaceId },
        query: {
          loop: normalizeOptionalText(filters.loop),
          status: normalizeOptionalText(filters.status),
          limit: filters.limit,
        },
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    throw new LoopsApiError(
      defaultApiErrorMessage("Failed to fetch loop runs", response, error),
      response.status
    );
  }

  const payload = requireResponseData(data, response, "Failed to fetch loop runs");
  return { runs: payload.runs, aggregates: payload.aggregates };
}

export async function getLoopRun(
  workspaceId: string,
  runId: string,
  signal?: AbortSignal
): Promise<LoopRunDetail> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loop-runs/{run_id}",
    {
      params: { path: { workspace_id: workspaceId, run_id: runId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop run not found: ${runId}`, 404);
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to fetch loop run "${runId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch loop run "${runId}"`);
}

// Run controls --------------------------------------------------------------

function loopRunControlError(
  action: string,
  runId: string,
  response: Response,
  error: unknown
): LoopsApiError {
  if (response.status === 404) {
    return new LoopsApiError(`Loop run not found: ${runId}`, 404);
  }
  if (response.status === 409 || response.status === 422) {
    return new LoopsApiError(
      defaultApiErrorMessage(`Cannot ${action} loop run "${runId}"`, response, error),
      response.status
    );
  }
  return new LoopsApiError(
    defaultApiErrorMessage(`Failed to ${action} loop run "${runId}"`, response, error),
    response.status
  );
}

export async function pauseLoopRun(
  workspaceId: string,
  runId: string,
  signal?: AbortSignal
): Promise<LoopRunActionResult> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/loop-runs/{run_id}/pause",
    { params: { path: { workspace_id: workspaceId, run_id: runId } }, body: {}, signal }
  );

  if (apiRequestFailed(response, error)) {
    throw loopRunControlError("pause", runId, response, error);
  }
  return requireResponseData(data, response, `Failed to pause loop run "${runId}"`);
}

export async function resumeLoopRun(
  workspaceId: string,
  runId: string,
  signal?: AbortSignal
): Promise<LoopRunActionResult> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/loop-runs/{run_id}/resume",
    { params: { path: { workspace_id: workspaceId, run_id: runId } }, body: {}, signal }
  );

  if (apiRequestFailed(response, error)) {
    throw loopRunControlError("resume", runId, response, error);
  }
  return requireResponseData(data, response, `Failed to resume loop run "${runId}"`);
}

export async function stopLoopRun(
  workspaceId: string,
  runId: string,
  signal?: AbortSignal
): Promise<LoopRunActionResult> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/loop-runs/{run_id}/stop",
    { params: { path: { workspace_id: workspaceId, run_id: runId } }, body: {}, signal }
  );

  if (apiRequestFailed(response, error)) {
    throw loopRunControlError("stop", runId, response, error);
  }
  return requireResponseData(data, response, `Failed to stop loop run "${runId}"`);
}

export async function approveLoopRun(
  workspaceId: string,
  runId: string,
  body: ApproveLoopRunRequest,
  signal?: AbortSignal
): Promise<LoopRunActionResult> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/loop-runs/{run_id}/approve",
    {
      params: { path: { workspace_id: workspaceId, run_id: runId } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new LoopsApiError(`Loop run not found: ${runId}`, 404);
    }
    if (response.status === 409 || response.status === 422) {
      throw new LoopsApiError(
        defaultApiErrorMessage(`Cannot record gate decision for run "${runId}"`, response, error),
        response.status
      );
    }
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to approve loop run "${runId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to approve loop run "${runId}"`);
}
