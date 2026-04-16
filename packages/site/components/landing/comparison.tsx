import { Check, Minus } from "lucide-react";
import { cn } from "@agh/ui/utils";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

type Approach = {
  approach: string;
  focus: string;
  agentModel: string;
  coordination: string;
  deployment: string;
  agents: string;
  /** Whether this approach ships a cross-runtime protocol today. */
  crossRuntime: boolean;
  highlight?: boolean;
};

const APPROACHES: Approach[] = [
  {
    approach: "Assistant gateway",
    focus: "Personal AI across chat channels",
    agentModel: "Single assistant with plugins",
    coordination: "None — one agent per user",
    deployment: "Cloud-hosted",
    agents: "1 (built-in)",
    crossRuntime: false,
  },
  {
    approach: "All-in-one agent OS",
    focus: "Broad built-in capabilities",
    agentModel: "Custom agents in platform",
    coordination: "Internal only",
    deployment: "Cloud or self-hosted",
    agents: "custom",
    crossRuntime: false,
  },
  {
    approach: "Multi-tenant gateway",
    focus: "Enterprise AI platform",
    agentModel: "Managed agents behind an API",
    coordination: "Centralized routing",
    deployment: "Cloud-hosted",
    agents: "managed",
    crossRuntime: false,
  },
  {
    approach: "AGH",
    focus: "Orchestrate real agent CLIs",
    agentModel: "Your existing ACP agents",
    coordination: "agh-network/v0 — shipped",
    deployment: "Local-first, single binary",
    agents: "8 ACP CLIs",
    crossRuntime: true,
    highlight: true,
  },
];

const DIMENSIONS = [
  { key: "focus" as const, label: "Primary focus" },
  { key: "agentModel" as const, label: "Agent model" },
  { key: "agents" as const, label: "Agents today" },
  { key: "coordination" as const, label: "Coordination" },
  { key: "deployment" as const, label: "Deployment" },
];

export function Comparison() {
  return (
    <SectionFrame background="canvas" padY="lg">
      <SectionHeader
        align="start"
        eyebrow="Positioning"
        title="Other tools stop at the runtime boundary."
        description="AGH is the only approach with a shipped cross-runtime protocol. The rest centralize coordination or skip it entirely."
      />

      <div className="mt-10 overflow-hidden rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface)">
        {/* Header row */}
        <div className="hidden border-b border-(--color-divider) px-5 py-4 md:grid md:grid-cols-[160px_repeat(5,minmax(0,1fr))_60px] md:gap-4">
          <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
            Approach
          </p>
          {DIMENSIONS.map(d => (
            <p
              key={d.key}
              className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)"
            >
              {d.label}
            </p>
          ))}
          <p className="text-right font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
            Cross-runtime
          </p>
        </div>

        {APPROACHES.map(row => (
          <div
            key={row.approach}
            className={cn(
              "grid gap-3 border-t border-(--color-divider) px-5 py-5 first:border-t-0 md:grid-cols-[160px_repeat(5,minmax(0,1fr))_60px] md:items-center md:gap-4",
              row.highlight &&
                "border-l-4 border-l-(--color-accent) bg-[color-mix(in_srgb,var(--color-accent-tint)_40%,transparent)]"
            )}
          >
            <div>
              <h3
                className={cn(
                  "text-[14px] font-semibold",
                  row.highlight ? "text-(--color-accent)" : "text-(--color-text-primary)"
                )}
              >
                {row.approach}
              </h3>
            </div>
            {DIMENSIONS.map(d => (
              <div key={d.key}>
                <p className="font-mono text-[10px] font-medium uppercase tracking-(--tracking-mono) text-(--color-text-tertiary) md:hidden">
                  {d.label}
                </p>
                <p
                  className={cn(
                    "text-[13px] leading-6",
                    row.highlight && d.key === "coordination"
                      ? "font-medium text-(--color-text-primary)"
                      : "text-(--color-text-secondary)"
                  )}
                >
                  {row[d.key]}
                </p>
              </div>
            ))}
            <div className="flex md:justify-end">
              <span
                aria-label={row.crossRuntime ? "Cross-runtime: yes" : "Cross-runtime: no"}
                className={cn(
                  "inline-flex h-6 w-6 items-center justify-center rounded-[6px]",
                  row.crossRuntime
                    ? "bg-(--color-success-tint) text-(--color-success)"
                    : "bg-(--color-surface-elevated) text-(--color-text-tertiary)"
                )}
              >
                {row.crossRuntime ? (
                  <Check className="h-3.5 w-3.5" strokeWidth={3} />
                ) : (
                  <Minus className="h-3.5 w-3.5" strokeWidth={3} />
                )}
              </span>
            </div>
          </div>
        ))}
      </div>
    </SectionFrame>
  );
}
