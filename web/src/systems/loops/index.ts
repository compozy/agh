// Types
export type {
  ApproveLoopRunRequest,
  CreateLoopRequest,
  LoopAggregate30d,
  LoopAnnotation,
  LoopAnnotationsUpdateRequest,
  LoopCatalogEntry,
  LoopCatalogStableFilter,
  LoopsListResponse,
  LoopCatalogInfo,
  LoopConfig,
  LoopConfigUpdateRequest,
  LoopContract,
  LoopContractBudget,
  LoopContractVerification,
  LoopDefinition,
  LoopDefinitionGraph,
  LoopDefinitionMeta,
  LoopDetail,
  LoopDryRunNode,
  LoopDryRunPreview,
  LoopEffectiveConfig,
  LoopInputSchema,
  LoopInputSchemaField,
  LoopRun,
  LoopRunActionResult,
  LoopRunAggregates,
  LoopRunDetail,
  LoopRunEventFrame,
  LoopRunEventKind,
  LoopRunGeneration,
  LoopRunGenerationOutput,
  LoopRunListResult,
  LoopRunRecord,
  LoopRunStatus,
  LoopRunsFilter,
  LoopStartBinding,
  LoopStreamFilter,
  LoopValidationIssue,
  LoopWatchEventSubscription,
  LoopWatchEventsState,
  PatchLoopRequest,
  RunLoopRequest,
  RunLoopResult,
  ValidateLoopRequest,
  ValidateLoopResult,
} from "./types";

// Adapters
export {
  LoopsApiError,
  approveLoopRun,
  buildLoopStreamUrl,
  createLoop,
  deleteLoop,
  getLoop,
  getLoopAnnotations,
  getLoopConfig,
  getLoopRun,
  listLoopRuns,
  listLoops,
  patchLoop,
  pauseLoopRun,
  putLoopAnnotations,
  putLoopConfig,
  resumeLoopRun,
  runLoop,
  stopLoopRun,
  validateLoop,
} from "./adapters/loops-api";

// Query infrastructure
export { loopsKeys } from "./lib/query-keys";
export {
  loopAnnotationsOptions,
  loopConfigOptions,
  loopDetailOptions,
  loopRunDetailOptions,
  loopRunsOptions,
  loopsCatalogOptions,
} from "./lib/query-options";

// Catalog helpers
export type {
  LoopCatalogFilter,
  LoopCatalogGroup,
  LoopKind,
  LoopKindFilter,
  LoopStatusFilter,
} from "./lib/loop-catalog";
export {
  UNBOUNDED_CAP,
  countByKind,
  groupLoopCatalog,
  hasActiveLoopFilters,
  hasHumanGate,
  isUnboundedCap,
  iterationCapLabel,
  loopCategories,
  loopCategory,
  loopInputCount,
  loopKind,
  loopStatuses,
  matchesLoopFilter,
  successRateLabel,
} from "./lib/loop-catalog";

// Listing filter bridge (reui Filters + URL parsers)
export type { LoopFilterFieldKey, LoopFilterHandlers } from "./lib/loop-list-filters";
export {
  applyLoopFilterChips,
  buildLoopFilterFields,
  loopFiltersToChips,
  parseLoopCategoryFilter,
  parseLoopKindFilter,
  parseLoopStatusFilter,
} from "./lib/loop-list-filters";

// Start-binding helpers
export type { LoopBindingKind, LoopBindingRow } from "./lib/loop-bindings";
export { bindingKindLabel, summarizeBindingKinds } from "./lib/loop-bindings";

// Read-only graph projection
export type { LoopGraph, LoopGraphEdge, LoopGraphNode, LoopNodeClass } from "./lib/loop-graph";
export { fanOutSummary, nodeClassLabel, readLoopGraph } from "./lib/loop-graph";

// Visual editor — bijective codec, layout, linter, node schema, references (task 22)
export type { EditorEdge, EditorGraph, EditorNode, RawLoopEdge, RawLoopNode } from "./lib/codec";
export { definitionToGraph, editorEdgeId, graphToDefinition } from "./lib/codec";
export { layoutEditorGraph, annotationsToPositions } from "./lib/loop-editor-layout";
export type { LoopInvariantKey, LoopInvariantStatus, LoopLintState } from "./lib/loop-editor-lint";
export {
  LOOP_INVARIANTS,
  applyLintToNodes,
  buildLintState,
  classifyInvariant,
  emptyLintState,
} from "./lib/loop-editor-lint";
export type { FieldPath, FieldSpec } from "./lib/loop-node-schema";
export { buildNodeFields } from "./lib/loop-node-schema";
export type { LoopReferenceKind, LoopReferenceSuggestion } from "./lib/loop-references";
export {
  activeReferenceQuery,
  buildReferenceNamespace,
  filterReferences,
} from "./lib/loop-references";
export type { DslLine } from "./lib/loop-dsl";
export { buildDslView } from "./lib/loop-dsl";
export type { PaletteGroup, PaletteItem } from "./lib/loop-palette";
export { LOOP_PALETTE, uniqueNodeId } from "./lib/loop-palette";
export {
  getAtPath,
  isNodeIdPath,
  renameNodeId,
  setAtPath,
  setNodeField,
} from "./lib/loop-editor-draft";
export { useLoopEditor } from "./hooks/use-loop-editor";
export type {
  LoopEditorStatus,
  LoopEditorView,
  UseLoopEditorResult,
} from "./hooks/use-loop-editor";
export { LoopEditor } from "./components/editor/loop-editor";

// Limits & budget
export type { LoopLimitRow } from "./lib/loop-limits";
export {
  LOOP_CEILINGS,
  buildLoopLimits,
  formatTokenBudget,
  formatWallClock,
} from "./lib/loop-limits";

// Runs view-model
export type {
  LoopBudgetBar,
  LoopBudgetTone,
  LoopKpi,
  LoopOutcomeSegment,
  LoopRunKpis,
  LoopRunPartition,
} from "./lib/loop-runs-view";
export {
  buildOutcomeSegments,
  buildRunKpis,
  formatTokenCount,
  loopBudgetBar,
  partitionRuns,
} from "./lib/loop-runs-view";

// Run-form model
export type { LoopRunInputs } from "./lib/loop-run-form";
export {
  hasInputValue,
  initialRunInputs,
  isRunFormValid,
  missingRequiredInputs,
  serializeRunInputs,
} from "./lib/loop-run-form";
export type {
  LoopBudgetPolicy,
  LoopOverrideDraft,
  LoopOverrideField,
  LoopOverrideKey,
} from "./lib/loop-overrides";
export {
  buildConfigOverrides,
  buildOverrideFields,
  clampOverrideValue,
  hasActiveOverrides,
  initialOverrideDraft,
} from "./lib/loop-overrides";

// Configure-sheet model
export type {
  EnabledChecksMap,
  LoopConfigCheckDescriptor,
  LoopConfigCheckState,
} from "./lib/loop-config-checks";
export {
  buildCheckDescriptors,
  defaultCheckStates,
  initialCheckStates,
  parseEnabledChecks,
  serializeEnabledChecks,
} from "./lib/loop-config-checks";
export type { LoopConfigDraft, LoopReattemptStrategy } from "./lib/loop-config-draft";
export {
  buildConfigureModel,
  buildLoopConfigRequest,
  initialConfigDraft,
  resetConfigDraft,
} from "./lib/loop-config-draft";

// Run-page model
export type { LoopMeterKey, LoopMeterTone, LoopRunMeter } from "./lib/loop-run-meters";
export { LOOP_COST_PER_1M_TOKENS, buildRunMeters, deriveCostEstimate } from "./lib/loop-run-meters";
export type { LoopTimelineGeneration, LoopTimelineNode } from "./lib/loop-timeline";
export { buildRunTimeline, latestGenerationBreadth } from "./lib/loop-timeline";
export type {
  LoopApprovalFact,
  LoopApprovalRequest,
  LoopChannelMessage,
  LoopGateVerdict,
  LoopLiveEvent,
  LoopRunLiveState,
} from "./lib/loop-events";
export { applyLoopEventFrame, emptyLoopRunLiveState } from "./lib/loop-events";

// Formatters and helpers
export type { LoopStatusSignal } from "./lib/loop-formatters";
export {
  LOOP_STATUS_LABELS,
  LOOP_STATUS_TONE,
  isLoopRunStatus,
  isTerminalLoopStatus,
  loopStatusLabel,
  loopStatusPulse,
  loopStatusSignal,
  loopStatusTone,
} from "./lib/loop-formatters";

// Read hooks
export {
  useLoop,
  useLoopAnnotations,
  useLoopConfig,
  useLoopRun,
  useLoopRuns,
  useLoops,
} from "./hooks/use-loops";

// Mutation hooks
export {
  useApproveLoopRun,
  useCreateLoop,
  useDeleteLoop,
  usePatchLoop,
  usePauseLoopRun,
  usePutLoopAnnotations,
  usePutLoopConfig,
  useResumeLoopRun,
  useRunLoop,
  useStopLoopRun,
  useValidateLoop,
} from "./hooks/use-loop-actions";

// Run-form view-model hook
export { useLoopRunForm } from "./hooks/use-loop-run-form";

// Configure-sheet view-model hook
export { useLoopConfigure } from "./hooks/use-loop-configure";
export type { UseLoopConfigureResult } from "./hooks/use-loop-configure";

// SSE stream hook
export { useLoopStream } from "./hooks/use-loop-stream";
export type {
  LoopStreamEventSource,
  LoopStreamEventSourceFactory,
  UseLoopStreamOptions,
} from "./hooks/use-loop-stream";

// Components
export { LoopStatusPill } from "./components/loop-status-pill";
export type { LoopStatusPillProps } from "./components/loop-status-pill";
export { LoopCatalog } from "./components/catalog/loop-catalog";
export { LoopCatalogCard } from "./components/catalog/loop-catalog-card";
export { LoopCatalogFilters } from "./components/catalog/loop-catalog-filters";
export { MonoTag } from "./components/mono-tag";
export { LoopDetailView } from "./components/detail/loop-detail";
export { LoopStartBindingsPanel } from "./components/detail/loop-start-bindings-panel";
export { LoopRunsView } from "./components/runs/loop-runs-view";
export type { LoopOutcomeValue } from "./components/runs/loop-runs-outcome-filter";
export { LoopTargetFields } from "./components/target/loop-target-fields";

// Run form
export { LoopRunForm } from "./components/run-form/loop-run-form";
export { LoopRunInputField } from "./components/run-form/loop-run-input-field";
export { LoopRunOverrides } from "./components/run-form/loop-run-overrides";
export { LoopRunPreview } from "./components/run-form/loop-run-preview";

// Configure sheet
export { LoopConfigureSheet } from "./components/configure/loop-configure-sheet";

// Run page
export { LoopRunContractHeader } from "./components/run-page/loop-run-contract-header";
export { LoopRunControls } from "./components/run-page/loop-run-controls";
export { LoopRunMeters } from "./components/run-page/loop-run-meters";
export { LoopGenerationTimeline } from "./components/run-page/loop-generation-timeline";
export { LoopGenerationCard } from "./components/run-page/loop-generation-card";
export { LoopNodeRow } from "./components/run-page/loop-node-row";
export { LoopGateCard } from "./components/run-page/loop-gate-card";
export { LoopRunChannel } from "./components/run-page/loop-run-channel";
export { LoopApprovalGate } from "./components/run-page/loop-approval-gate";
export type { LoopGateDecision } from "./components/run-page/loop-approval-gate";
export { LoopRunEventsRail } from "./components/run-page/loop-run-events-rail";
export { LoopRunFacts } from "./components/run-page/loop-run-facts";
export { LoopStatusLegend } from "./components/run-page/loop-status-legend";
export { LoopWatchEventsPanel } from "./components/run-page/loop-watch-events-panel";

// Loop-target editing (automation Target step)
export type { LoopTargetDraft } from "./lib/loop-target";
export { setLoopTargetInput, setLoopTargetLoop, setLoopTargetMapping } from "./lib/loop-target";
