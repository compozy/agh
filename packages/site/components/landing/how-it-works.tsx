const steps = [
  {
    step: "01",
    title: "Install AGH",
    description: "Single binary, no dependencies. Download and run.",
    code: `curl -fsSL https://get.agh.compozy.com | sh`,
  },
  {
    step: "02",
    title: "Start the daemon",
    description: "AGH runs as a background daemon managing all your agent sessions.",
    code: `agh daemon start`,
  },
  {
    step: "03",
    title: "Create your first session",
    description:
      "Spawn an agent CLI as a managed session with memory, skills, and workspace context.",
    code: `agh session create --agent claude-code \\
  --workspace ~/my-project`,
  },
];

export function HowItWorks() {
  return (
    <section className="px-4 py-16 md:py-24">
      <div className="mx-auto max-w-5xl">
        <p className="text-center font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
          GETTING STARTED
        </p>
        <h2 className="mt-3 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          Up and running in three steps
        </h2>
        <div className="mt-12 flex flex-col gap-8">
          {steps.map(item => (
            <div
              key={item.step}
              className="flex flex-col gap-4 md:flex-row md:items-start md:gap-8"
            >
              <div className="flex shrink-0 items-start gap-4 md:w-64">
                <span className="font-mono text-sm font-semibold text-[var(--color-accent)]">
                  {item.step}
                </span>
                <div>
                  <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">
                    {item.title}
                  </h3>
                  <p className="mt-1 text-sm leading-relaxed text-[var(--color-text-secondary)]">
                    {item.description}
                  </p>
                </div>
              </div>
              <div className="flex-1 overflow-x-auto rounded-lg border border-[var(--color-divider)] bg-[var(--color-surface-elevated)] p-4">
                <pre className="font-mono text-sm leading-relaxed text-[var(--color-text-primary)]">
                  <code>{item.code}</code>
                </pre>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
