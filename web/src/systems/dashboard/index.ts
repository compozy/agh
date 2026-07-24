export { DashboardApiError } from "./adapters/dashboard-api-errors";
export { HomeDashboard } from "./components/home-dashboard";
export { buildHomeLogsStreamUrl, getHomeActivity, getHomeOverview } from "./adapters/overview-api";
export {
  homeActivityTone,
  isQuietActivityEvent,
  isTaskLifecycleEvent,
  type HomeActivityTone,
} from "./lib/activity-classes";
export {
  useHomePrefsActions,
  useHomePrefsStore,
  useHomeSystemOpen,
  useHomeUsageWindow,
} from "./hooks/use-home-prefs-store";
export { homeScopeForActiveWorkspace, type HomeScope } from "./lib/home-scope";
export { dashboardKeys } from "./lib/query-keys";
export {
  ACTIVITY_STALE_TIME,
  HOME_ACTIVITY_LIMIT,
  homeActivityOptions,
  homeOverviewOptions,
  OVERVIEW_REFETCH_INTERVAL,
  OVERVIEW_STALE_TIME,
} from "./lib/query-options";
export { elapsedSecondsSince, useElapsedNowSeconds } from "./hooks/use-elapsed-ticker";
export { useHomeDashboard, type HomeDashboardModel } from "./hooks/use-home-dashboard";
export {
  useHomeLive,
  type HomeLiveEventSource,
  type HomeLiveEventSourceFactory,
  type UseHomeLiveOptions,
} from "./hooks/use-home-live";
export {
  HOME_USAGE_WINDOWS,
  normalizeHomeUsageWindow,
  type HomeActivityEvent,
  type HomeActivityFilter,
  type HomeAgentShare,
  type HomeAttention,
  type HomeAttentionItem,
  type HomeSurfaceStatus,
  type HomeOutcomeDay,
  type HomeOverview,
  type HomeOverviewFilter,
  type HomePulseBucket,
  type HomeUsageDay,
  type HomeUsageWindow,
} from "./types";
