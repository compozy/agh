import { authorizeLabel, type SettingsMCPServerEntry } from "@/systems/settings";

export type MarketplaceMCPInstalledStatus = "running" | "authorize";

export function marketplaceMCPInstalledStatus(
  server: SettingsMCPServerEntry
): MarketplaceMCPInstalledStatus {
  if (authorizeLabel(server)) return "authorize";
  const runtime = server.runtime_status?.state;
  if (
    runtime === "auth_required" ||
    runtime === "auth_expired" ||
    runtime === "auth_invalid" ||
    runtime === "auth_refresh_failed"
  ) {
    return "authorize";
  }
  return "running";
}
