// Types
export type {
  AutomationFireLimit,
  AutomationJob,
  AutomationJobListFilter,
  AutomationJobsListResponse,
  AutomationJobStableFilter,
  AutomationKind,
  AutomationRetry,
  AutomationRun,
  AutomationRunHistoryFilter,
  AutomationRunListFilter,
  AutomationRunStatus,
  AutomationSchedule,
  AutomationScheduleMode,
  AutomationSchedulerState,
  AutomationScope,
  AutomationScopeFilter,
  AutomationSource,
  AutomationTrigger,
  AutomationTriggerFilter,
  AutomationTriggerListFilter,
  AutomationTriggersListResponse,
  AutomationTriggerStableFilter,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
  UpdateAutomationJobRequest,
  UpdateAutomationTriggerRequest,
} from "./types";

// Adapters
export {
  AutomationApiError,
  createAutomationJob,
  createAutomationTrigger,
  deleteAutomationJob,
  deleteAutomationTrigger,
  getAutomationJob,
  getAutomationTrigger,
  listAutomationJobRuns,
  listAutomationJobs,
  listAutomationRuns,
  listAutomationTriggerRuns,
  listAutomationTriggers,
  triggerAutomationJob,
  updateAutomationJob,
  updateAutomationTrigger,
} from "./adapters/automation-api";

// Query infrastructure
export { automationKeys } from "./lib/query-keys";
export {
  buildAutomationJobRequest,
  buildAutomationTriggerRequest,
} from "./lib/automation-requests";
export {
  automationJobDetailOptions,
  automationJobRunsOptions,
  automationJobsListOptions,
  automationRunsListOptions,
  automationTriggerDetailOptions,
  automationTriggerRunsOptions,
  automationTriggersListOptions,
} from "./lib/query-options";
export type { AutomationLoopTarget, AutomationTargetMode } from "./lib/automation-drafts";
export {
  LOOP_TARGET_KIND,
  automationJobToDraft,
  automationTargetMode,
  automationTriggerToDraft,
  createAutomationJobDraft,
  createAutomationTriggerDraft,
  createLoopTargetJobDraft,
  createLoopTargetTriggerDraft,
  emptyLoopTarget,
  bindLoopTargetWorkspace,
  loopTargetWorkspaceId,
  normalizeAutomationRetry,
  retryDraftForStrategy,
  setJobTargetMode,
  setTriggerTargetMode,
} from "./lib/automation-drafts";
export type { AutomationDialogHandle } from "./lib/dialog-handle";
export { createAutomationDialogHandle } from "./lib/dialog-handle";
export {
  automationSourceLabel,
  automationScopeLabel,
  automationScopeTone,
  automationSourceTone,
  automationStatusTone,
  describeFireLimit,
  describeRetry,
  describeSchedule,
  describeTrigger,
  formatAutomationListSummary,
  formatDate,
  formatDateTime,
  formatPromptPreview,
  formatRelativeTime,
  formatRunDuration,
  formatRunTitle,
} from "./lib/automation-formatters";
// Hooks
export {
  useAutomationJob,
  useAutomationJobs,
  useAutomationJobRuns,
  useAutomationRuns,
  useAutomationTrigger,
  useAutomationTriggers,
  useAutomationTriggerRuns,
} from "./hooks/use-automation";
export {
  useCreateAutomationJob,
  useCreateAutomationTrigger,
  useDeleteAutomationJob,
  useDeleteAutomationTrigger,
  useTriggerAutomationJob,
  useUpdateAutomationJob,
  useUpdateAutomationTrigger,
} from "./hooks/use-automation-actions";

// Components
export { AutomationDetailPanel } from "./components/automation-detail-panel";
export { AutomationEditorDialog } from "./components/automation-editor-dialog";
export { AutomationJobForm } from "./components/automation-job-form";
export { AutomationListPanel } from "./components/automation-list-panel";
export { AutomationOperationsPage } from "./components/automation-operations-page";
export { AutomationRunHistory } from "./components/automation-run-history";
export { AutomationTriggerForm } from "./components/automation-trigger-form";
