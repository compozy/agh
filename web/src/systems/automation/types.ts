import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type AutomationJob = OperationResponse<"getAutomationJob", 200>["job"];
export type AutomationTrigger = OperationResponse<"getAutomationTrigger", 200>["trigger"];
export type AutomationRun = OperationResponse<"getAutomationRun", 200>["run"];

export type AutomationJobListFilter = OperationQuery<"listAutomationJobs">;
export type AutomationTriggerListFilter = OperationQuery<"listAutomationTriggers">;
export type AutomationRunListFilter = OperationQuery<"listAutomationRuns">;
export type AutomationRunHistoryFilter = OperationQuery<"listAutomationJobRuns">;

export type CreateAutomationJobRequest = OperationRequestBody<"createAutomationJob">;
export type UpdateAutomationJobRequest = OperationRequestBody<"updateAutomationJob">;
export type CreateAutomationTriggerRequest = OperationRequestBody<"createAutomationTrigger">;
export type UpdateAutomationTriggerRequest = OperationRequestBody<"updateAutomationTrigger">;

export type AutomationScope = AutomationJob["scope"];
export type AutomationSource = AutomationJob["source"];
export type AutomationRunStatus = AutomationRun["status"];
export type AutomationSchedule = NonNullable<AutomationJob["schedule"]>;
export type AutomationScheduleMode = AutomationSchedule["mode"];
export type AutomationRetry = AutomationJob["retry"];
export type AutomationFireLimit = AutomationJob["fire_limit"];
export type AutomationTriggerFilter = NonNullable<AutomationTrigger["filter"]>;

export type AutomationKind = "jobs" | "triggers";
export type AutomationScopeFilter = "all" | AutomationScope;
