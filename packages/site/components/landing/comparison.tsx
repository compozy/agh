import { Check, Minus } from "lucide-react";

interface ComparisonRow {
  feature: string;
  typical: string;
  agh: string;
  aghHighlight: boolean;
}

const rows: ComparisonRow[] = [
  {
    feature: "Agent execution",
    typical: "API wrapper calls",
    agh: "Real CLI subprocesses via ACP",
    aghHighlight: true,
  },
  {
    feature: "Agent-to-agent communication",
    typical: "None / custom glue",
    agh: "Built-in network protocol",
    aghHighlight: true,
  },
  {
    feature: "Session persistence",
    typical: "In-memory or external DB",
    agh: "SQLite per-session event store",
    aghHighlight: true,
  },
  {
    feature: "Memory system",
    typical: "None or basic RAG",
    agh: "Dual-scope with dream consolidation",
    aghHighlight: true,
  },
  {
    feature: "Deployment",
    typical: "Docker + sidecars + services",
    agh: "Single binary, zero dependencies",
    aghHighlight: true,
  },
  {
    feature: "Configuration",
    typical: "Environment variables",
    agh: "TOML + AGENT.md + workspace overlays",
    aghHighlight: true,
  },
  {
    feature: "Extensibility",
    typical: "Fork or monkey-patch",
    agh: "Skills, hooks, extensions, bridges",
    aghHighlight: true,
  },
  {
    feature: "Observability",
    typical: "Logs to stdout",
    agh: "Structured events, SSE streaming, health metrics",
    aghHighlight: true,
  },
];

export function Comparison() {
  return (
    <section className="px-4 py-16 md:py-24">
      <div className="mx-auto max-w-5xl">
        <p className="text-center font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
          COMPARISON
        </p>
        <h2 className="mt-3 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          AGH vs typical agent harness
        </h2>

        {/* Desktop table */}
        <div className="mt-12 hidden overflow-hidden rounded-xl border border-[var(--color-divider)] md:block">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-[var(--color-divider)] bg-[var(--color-surface)]">
                <th className="px-5 py-3 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
                  Feature
                </th>
                <th className="px-5 py-3 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
                  Typical Harness
                </th>
                <th className="px-5 py-3 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-accent)]">
                  AGH
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr
                  key={row.feature}
                  className={i % 2 === 0 ? "bg-transparent" : "bg-[var(--color-surface)]"}
                >
                  <td className="px-5 py-3 font-medium text-[var(--color-text-primary)]">
                    {row.feature}
                  </td>
                  <td className="px-5 py-3 text-[var(--color-text-tertiary)]">
                    <span className="flex items-center gap-2">
                      <Minus className="h-3.5 w-3.5 text-[var(--color-text-tertiary)]" />
                      {row.typical}
                    </span>
                  </td>
                  <td className="px-5 py-3 text-[var(--color-text-primary)]">
                    <span className="flex items-center gap-2">
                      <Check className="h-3.5 w-3.5 text-[var(--color-success)]" />
                      {row.agh}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Mobile cards */}
        <div className="mt-12 flex flex-col gap-4 md:hidden">
          {rows.map(row => (
            <div
              key={row.feature}
              className="rounded-xl border border-[var(--color-divider)] bg-[var(--color-surface)] p-4"
            >
              <p className="text-sm font-medium text-[var(--color-text-primary)]">{row.feature}</p>
              <div className="mt-3 flex flex-col gap-2">
                <div className="flex items-start gap-2">
                  <Minus className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--color-text-tertiary)]" />
                  <span className="text-xs text-[var(--color-text-tertiary)]">{row.typical}</span>
                </div>
                <div className="flex items-start gap-2">
                  <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--color-success)]" />
                  <span className="text-xs text-[var(--color-text-primary)]">{row.agh}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
