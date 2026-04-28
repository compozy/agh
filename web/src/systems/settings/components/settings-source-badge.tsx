import { Pill, type PillTone } from "@agh/ui";

import type { SettingsSourceKind } from "../types";

interface SettingsSource {
  kind: SettingsSourceKind;
  scope: "global" | "workspace";
  workspace_id?: string;
}

interface SettingsSourceBadgeProps {
  source: SettingsSource;
  shadowed?: SettingsSource[];
  "data-testid"?: string;
}

const KIND_LABELS: Record<SettingsSourceKind, string> = {
  "builtin-provider": "BUILTIN",
  "global-config": "CONFIG",
  "workspace-config": "WORKSPACE",
  "global-mcp-sidecar": "MCP.JSON",
  "workspace-mcp-sidecar": "WS-MCP.JSON",
};

function badgeTone(kind: SettingsSourceKind): PillTone {
  switch (kind) {
    case "builtin-provider":
      return "neutral";
    case "global-config":
    case "global-mcp-sidecar":
      return "info";
    case "workspace-config":
    case "workspace-mcp-sidecar":
      return "warning";
    default:
      return "neutral";
  }
}

function sourceLabel(source: SettingsSource): string {
  const base = KIND_LABELS[source.kind];
  if (source.workspace_id) {
    return `${base} · ${source.workspace_id}`;
  }
  return base;
}

function SettingsSourceBadge({
  source,
  shadowed,
  "data-testid": testId,
}: SettingsSourceBadgeProps) {
  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid={testId}>
      <Pill
        mono
        tone={badgeTone(source.kind)}
        data-testid={testId ? `${testId}-effective` : undefined}
      >
        {sourceLabel(source)}
      </Pill>
      {shadowed && shadowed.length > 0 ? (
        <span
          className="flex flex-wrap items-center gap-1 font-mono text-[10px] font-semibold tracking-[0.1em] text-[color:var(--color-text-label)]"
          data-testid={testId ? `${testId}-shadowed` : undefined}
        >
          <span className="uppercase">shadows</span>
          {shadowed.map((entry, index) => (
            <Pill
              mono
              tone="neutral"
              // biome-ignore lint/suspicious/noArrayIndexKey: source list is stable per read
              key={`${entry.kind}-${entry.scope}-${entry.workspace_id ?? ""}-${index}`}
            >
              {sourceLabel(entry)}
            </Pill>
          ))}
        </span>
      ) : null}
    </div>
  );
}

export { SettingsSourceBadge };
export type { SettingsSource };
