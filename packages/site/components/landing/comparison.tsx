interface Approach {
  approach: string;
  focus: string;
  agentModel: string;
  coordination: string;
  deployment: string;
  highlight?: boolean;
}

const approaches: Approach[] = [
  {
    approach: "Assistant gateway",
    focus: "Personal AI assistant across chat channels",
    agentModel: "Single assistant with tool plugins",
    coordination: "None — one agent per user",
    deployment: "Cloud-hosted",
  },
  {
    approach: "All-in-one agent OS",
    focus: "Broad built-in capabilities, one integrated experience",
    agentModel: "Custom agents inside the platform",
    coordination: "Internal only — within the platform",
    deployment: "Cloud or self-hosted",
  },
  {
    approach: "Multi-tenant gateway",
    focus: "Enterprise AI platform with team management",
    agentModel: "Managed agents behind an API",
    coordination: "Centralized routing",
    deployment: "Cloud-hosted",
  },
  {
    approach: "AGH",
    focus: "Orchestrate real agent CLIs with an open coordination protocol",
    agentModel: "Your existing agents — any driver, any role",
    coordination: "AGH Network — open, cross-runtime protocol",
    deployment: "Local-first, single binary",
    highlight: true,
  },
];

const dimensions = [
  { key: "focus" as const, label: "Primary focus" },
  { key: "agentModel" as const, label: "Agent model" },
  { key: "coordination" as const, label: "Coordination" },
  { key: "deployment" as const, label: "Deployment" },
];

export function Comparison() {
  return (
    <section className="px-4 py-16 md:py-24">
      <div className="mx-auto max-w-[var(--site-layout-width)]">
        <div className="max-w-[600px]">
          <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
            Positioning
          </p>
          <h2 className="mt-4 text-[clamp(2.3rem,4.6vw,3.6rem)] leading-[1.0] font-normal tracking-[-0.03em] text-[var(--color-text-primary)]">
            Different approaches to agent infrastructure.
          </h2>
          <p className="mt-5 text-sm leading-7 text-[var(--color-text-secondary)]">
            Agent tools take different positions on where control lives and how agents coordinate.
            AGH is the only approach with an open cross-runtime coordination protocol built in.
          </p>
        </div>

        <div className="mt-10 overflow-hidden rounded-[12px] border border-[var(--color-divider)] bg-[var(--color-surface)]">
          {/* Header row */}
          <div className="hidden border-b border-[var(--color-divider)] px-6 py-4 md:grid md:grid-cols-[180px_repeat(4,minmax(0,1fr))] md:gap-4">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
              Approach
            </p>
            {dimensions.map(d => (
              <p
                key={d.key}
                className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]"
              >
                {d.label}
              </p>
            ))}
          </div>

          {approaches.map(row => (
            <div
              key={row.approach}
              className={`grid gap-3 border-t border-[var(--color-divider)] px-5 py-5 first:border-t-0 md:grid-cols-[180px_repeat(4,minmax(0,1fr))] md:gap-4 md:px-6 ${
                row.highlight
                  ? "border-t-[var(--color-accent-tint)] bg-[var(--color-accent-tint)]"
                  : ""
              }`}
            >
              <div>
                <h3
                  className={`text-base font-semibold ${row.highlight ? "text-[var(--color-accent)]" : "text-[var(--color-text-primary)]"}`}
                >
                  {row.approach}
                </h3>
              </div>
              {dimensions.map(d => (
                <div key={d.key}>
                  <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)] md:hidden">
                    {d.label}
                  </p>
                  <p
                    className={`text-sm leading-6 ${
                      row.highlight && d.key === "coordination"
                        ? "font-medium text-[var(--color-text-primary)]"
                        : "text-[var(--color-text-secondary)]"
                    }`}
                  >
                    {row[d.key]}
                  </p>
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
