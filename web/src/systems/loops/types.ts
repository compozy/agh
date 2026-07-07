import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

// Catalog + definition ------------------------------------------------------

export type LoopCatalogEntry = OperationResponse<"listLoops", 200>["loops"][number];
export type LoopCatalogInfo = LoopCatalogEntry["catalog"];
export type LoopSource = LoopCatalogEntry["source"];
export type LoopContract = LoopCatalogEntry["contract"];
export type LoopContractBudget = LoopContract["budget"];
export type LoopContractVerification = NonNullable<LoopContract["verification"]>[number];
export type LoopInputSchema = NonNullable<LoopCatalogEntry["inputs"]>;
export type LoopInputSchemaField = LoopInputSchema[string];
export type LoopStartBinding = NonNullable<LoopCatalogEntry["start"]>[number];
export type LoopAggregate30d = LoopCatalogEntry["aggregate_30d"];

export type LoopDetail = OperationResponse<"getLoop", 200>["loop"];
export type LoopDefinition = LoopDetail["definition"];
export type LoopDefinitionMeta = LoopDefinition["meta"];
export type LoopDefinitionGraph = LoopDefinition["graph"];

// Runs ----------------------------------------------------------------------

export type LoopRunListResult = OperationResponse<"listLoopRuns", 200>;
export type LoopRun = LoopRunListResult["runs"][number];
export type LoopRunAggregates = LoopRunListResult["aggregates"];
export type LoopRunDetail = OperationResponse<"getLoopRun", 200>;
export type LoopRunRecord = LoopRunDetail["run"];
export type LoopRunGeneration = NonNullable<LoopRunDetail["generations"]>[number];
export type LoopRunGenerationOutput = LoopRunGeneration["outputs"][number];

// SSE -----------------------------------------------------------------------

export type LoopRunEventFrame = OperationResponse<"streamLoopRunEvents", 200>;
export type LoopStreamFilter = OperationQuery<"streamLoopRunEvents">;
export type LoopRunsFilter = OperationQuery<"listLoopRuns">;

export type LoopRunEventKind = LoopRunEventFrame["kind"];

// Config --------------------------------------------------------------------

export type LoopConfig = NonNullable<OperationResponse<"getLoopConfig", 200>["config"]>;
export type LoopConfigUpdateRequest = OperationRequestBody<"putLoopConfig">;

// Editor annotations (node positions) ---------------------------------------

export type LoopAnnotation = OperationResponse<"getLoopAnnotations", 200>["annotations"][number];
export type LoopAnnotationsUpdateRequest = OperationRequestBody<"putLoopAnnotations">;

// Write requests ------------------------------------------------------------

export type CreateLoopRequest = OperationRequestBody<"createLoop">;
export type PatchLoopRequest = OperationRequestBody<"patchLoop">;
export type RunLoopRequest = OperationRequestBody<"runLoop">;
export type RunLoopResult = OperationResponse<"runLoop", 201>;
export type LoopDryRunPreview = NonNullable<RunLoopResult["dry_run"]>;
export type LoopDryRunNode = LoopDryRunPreview["nodes"][number];
export type LoopEffectiveConfig = LoopDryRunPreview["effective_config"];
export type ValidateLoopRequest = OperationRequestBody<"validateLoop">;
export type ValidateLoopResult = OperationResponse<"validateLoop", 200>;
export type LoopValidationIssue = NonNullable<ValidateLoopResult["errors"]>[number];
export type ApproveLoopRunRequest = OperationRequestBody<"approveLoopRun">;

/**
 * Run control (pause/resume/stop/approve) responses. The OpenAPI contract defines
 * all four 200 bodies identically as a `Record<string, boolean>` verdict (e.g.
 * `{ ok: true }`); this aliases the `pauseLoopRun` shape as the single source. If a
 * future contract change diverges one of them, split this into per-operation types.
 */
export type LoopRunActionResult = OperationResponse<"pauseLoopRun", 200>;

// Status vocabulary ---------------------------------------------------------

export type LoopRunStatus = LoopRun["status"];
