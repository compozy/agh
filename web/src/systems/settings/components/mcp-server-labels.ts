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

const SOURCE_KIND_LABEL: Record<string, string> = {
  "workspace-config": "workspace config",
  "global-config": "global config",
  "workspace-mcp-sidecar": "workspace mcp.json",
  "global-mcp-sidecar": "global mcp.json",
  "workspace-agent-file": "workspace agent file",
  "global-agent-file": "global agent file",
};

/** Human-readable config source for a server row (non-catalog provenance). */
export function mcpSourceKindLabel(kind: string): string {
  return SOURCE_KIND_LABEL[kind] ?? kind.replace(/-/g, " ");
}

/** Catalog provenance line, or null for a hand-configured server (nothing invented). */
export function mcpProvenanceLine(
  catalogEntry: string | undefined,
  catalogVersion: string | undefined
): string | null {
  const entry = catalogEntry?.trim();
  if (!entry) return null;
  const version = catalogVersion?.trim();
  return `installed from catalog · ${entry}${version ? `@${version}` : ""}`;
}
