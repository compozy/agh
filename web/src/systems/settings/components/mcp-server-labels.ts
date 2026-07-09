import type { SettingsMCPServerTarget, SettingsWriteTarget } from "../types";

export function mcpTargetLabel(target: SettingsMCPServerTarget): string {
  if (target === "auto") return "auto (highest precedence)";
  if (target === "config") return "config (.agh/config.toml)";
  return "sidecar (mcp.json)";
}

export function mcpWriteTargetLabel(target: SettingsWriteTarget): string {
  if (target === "global-config") return "GLOBAL CFG";
  if (target === "workspace-config") return "WS CFG";
  if (target === "global-mcp-sidecar") return "GLOBAL MCP";
  if (target === "workspace-mcp-sidecar") return "WS MCP";
  if (target === "global-agent-file") return "GLOBAL AGENT";
  return "WS AGENT";
}
