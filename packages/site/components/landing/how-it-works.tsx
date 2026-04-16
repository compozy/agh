const steps = [
  {
    step: "01",
    title: "Install the runtime",
    description: "One binary. No containers, no sidecars, no cloud dependencies.",
    code: `curl -fsSL https://get.agh.compozy.com | sh`,
  },
  {
    step: "02",
    title: "Start the daemon",
    description:
      "One local process that exposes the same surface to CLI, HTTP/SSE, and the web UI.",
    code: `agh daemon start`,
  },
  {
    step: "03",
    title: "Launch a session",
    description:
      "Create managed work you can inspect and resume — not a disposable terminal window.",
    code: `agh session new --agent coder`,
  },
];

export function HowItWorks() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-20 md:py-28">
      <div className="mx-auto max-w-[var(--site-layout-width)]">
        <div className="mx-auto max-w-[640px] text-center">
          <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
            Getting started
          </p>
          <h2 className="mt-5 text-[clamp(2.4rem,4.5vw,3.6rem)] leading-[1.0] font-normal tracking-[-0.03em] text-[var(--color-text-primary)]">
            Three commands to your first managed session.
          </h2>
        </div>

        <div className="mx-auto mt-14 max-w-[720px]">
          <div className="flex flex-col gap-6">
            {steps.map(item => (
              <div
                key={item.step}
                className="flex flex-col gap-4 rounded-[12px] border border-[var(--color-divider)] p-6 md:p-8"
              >
                <div className="flex items-start gap-4">
                  <span className="font-mono text-lg font-medium text-[var(--color-accent)] mt-0.5">
                    {item.step}
                  </span>
                  <div>
                    <h3 className="text-lg font-medium text-[var(--color-text-primary)]">
                      {item.title}
                    </h3>
                    <p className="mt-2 max-w-[42ch] text-sm leading-relaxed text-[var(--color-text-secondary)]">
                      {item.description}
                    </p>
                  </div>
                </div>
                <div className="ml-[2.75rem] overflow-x-auto rounded-[8px] bg-[var(--color-surface-elevated)] p-4">
                  <pre className="font-mono text-[13px] text-[var(--color-text-primary)]">
                    <span className="text-[var(--color-accent)]">$ </span>
                    <code>{item.code}</code>
                  </pre>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
