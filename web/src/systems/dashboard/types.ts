import type { OperationQuery, OperationResponse } from "@/lib/api-contract";

export type HomeOverview = OperationResponse<"getObserveOverview", 200>["overview"];
export type HomeOverviewWireFilter = OperationQuery<"getObserveOverview">;

export interface HomeOverviewFilter {
  workspace?: string;
  usageWindow?: HomeUsageWindow;
}
export type HomeAttention = HomeOverview["attention"];
export type HomeAttentionItem = HomeAttention["items"][number];
export type HomeOutcomeDay = HomeOverview["outcomes"]["days"][number];
export type HomeUsageDay = HomeOverview["usage"]["days"][number];
export type HomeAgentShare = HomeOverview["usage"]["agent_share"][number];
export type HomePulseBucket = HomeOverview["pulse"]["buckets"][number];

export type HomeActivityEvent = OperationResponse<"listLogs", 200>["events"][number];
export type HomeActivityFilter = OperationQuery<"listLogs">;

export type HomeUsageWindow = 7 | 30 | 90;

export type HomeSurfaceStatus = "loading" | "error" | "ready";

export const HOME_USAGE_WINDOWS: readonly HomeUsageWindow[] = [7, 30, 90];

export function normalizeHomeUsageWindow(value: number): HomeUsageWindow {
  return HOME_USAGE_WINDOWS.find(window => window === value) ?? 30;
}
