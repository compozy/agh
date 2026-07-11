// The two automation page view-models live in dedicated files (each under the
// production-source line cap); this barrel preserves the original import path.
export { useAutomationJobsPage } from "./use-automation-jobs-page";
export { useAutomationTriggersPage } from "./use-automation-triggers-page";
export type { AutomationCreateSeed, AutomationRouteSearch } from "./use-automation-page-base";
