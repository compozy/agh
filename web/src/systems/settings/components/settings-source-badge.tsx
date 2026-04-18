import type { SettingsSourceKind } from "../types";
import { Pill } from "@/components/design-system";

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

function badgeTone(kind: SettingsSourceKind): "neutral" | "amber" | "green" | "violet" {
  if (kind === "builtin-provider") return "violet";
  if (kind === "workspace-config" || kind === "workspace-mcp-sidecar") return "amber";
  return "green";
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
        emphasis="strong"
        kind="state"
        tone={badgeTone(source.kind)}
        data-testid={testId ? `${testId}-effective` : undefined}
      >
        {sourceLabel(source)}
      </Pill>
      {shadowed && shadowed.length > 0 ? (
        <span
          className="flex flex-wrap items-center gap-1 text-[0.625rem] tracking-[0.12em] text-[color:var(--color-text-label)]"
          data-testid={testId ? `${testId}-shadowed` : undefined}
        >
          <span className="font-mono uppercase">shadows</span>
          {shadowed.map((entry, index) => (
            <Pill
              // biome-ignore lint/suspicious/noArrayIndexKey: source list is stable per read
              key={`${entry.kind}-${entry.scope}-${entry.workspace_id ?? ""}-${index}`}
              emphasis="muted"
              kind="state"
              tone="neutral"
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
