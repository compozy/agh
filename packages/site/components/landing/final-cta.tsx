import Link from "next/link";

export function FinalCta() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-16 md:py-24">
      <div className="mx-auto max-w-[var(--site-layout-width)] border-t border-[var(--color-divider)] pt-10">
        <div className="grid gap-8 rounded-[12px] border border-[var(--color-divider)] bg-[var(--color-canvas)] px-6 py-8 lg:grid-cols-[minmax(0,1fr)_340px] lg:items-center lg:px-8">
          <div>
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
              Next step
            </p>
            <h2 className="mt-4 max-w-[16ch] text-[clamp(2.2rem,4.5vw,3.6rem)] leading-[1.0] font-normal tracking-[-0.03em] text-[var(--color-text-primary)]">
              Install the runtime. Connect the network.
            </h2>
            <p className="mt-5 max-w-[52ch] text-sm leading-7 text-[var(--color-text-secondary)]">
              Start with one binary and managed sessions. Add AGH Network when your agents need to
              coordinate across runtimes.
            </p>
          </div>

          <div>
            <div className="flex flex-col items-start gap-3">
              <Link
                href="/docs/getting-started"
                className="inline-flex h-9 w-full items-center justify-center rounded-[8px] bg-[var(--color-accent)] px-5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] sm:w-auto"
              >
                Get Started
              </Link>
              <Link
                href="/protocol"
                className="inline-flex h-9 w-full items-center justify-center rounded-[8px] border border-[var(--color-divider)] bg-[var(--color-surface)] px-5 text-sm font-medium text-[var(--color-text-primary)] transition-colors hover:border-[var(--color-accent)] hover:text-[var(--color-accent)] sm:w-auto"
              >
                Explore AGH Network
              </Link>
            </div>

            <p className="mt-4 text-sm leading-6 text-[var(--color-text-secondary)]">
              The runtime gives you managed agent sessions today. The network becomes useful when
              work crosses specialist boundaries.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
