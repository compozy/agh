import { HeroPlayer } from "./hero-player";
import { CtaButton } from "./primitives/cta-button";

const signalItems = [
  {
    label: "Complete agent runtime",
    detail: "Sessions, memory, skills, workspaces, automation, bridges — one binary.",
  },
  {
    label: "Built-in agent network",
    detail: "Agents discover peers, delegate work, and collect receipts across machines.",
  },
  {
    label: "Local-first, self-hosted",
    detail: "No Docker. No Postgres. Start with agh daemon start.",
  },
  {
    label: "Open protocol, open source",
    detail: "agh-network/v0 is an open wire spec. Bring any agent you like.",
  },
];

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-(--color-divider) px-4 pt-8 pb-16 md:pt-12 md:pb-20">
      {/* Background mesh — restored and faded so it textures the whole hero. */}
      <div
        className="pointer-events-none absolute inset-0 bg-size-[100%_auto] bg-position-[0%_0%] bg-no-repeat opacity-20 mix-blend-screen"
        style={{ backgroundImage: "url('/hero-bg.png')" }}
        aria-hidden="true"
      />

      <div className="relative mx-auto max-w-(--site-layout-width)">
        <div className="grid gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,540px)] lg:items-center lg:gap-14">
          <div className="order-2 lg:order-0 lg:pr-2">
            <div className="flex items-center gap-3 font-mono text-[11px] font-medium uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
              <span className="text-(--color-accent)">AGH</span>
              <span className="h-px w-10 bg-(--color-divider)" />
              <span>Agent Operating System</span>
            </div>

            <h1 className="mt-6 max-w-[18ch] text-[clamp(2.8rem,6.5vw,5.4rem)] leading-[0.96] font-normal tracking-[-0.035em] text-(--color-text-primary)">
              An agent runtime with a network built in.
            </h1>

            <p className="mt-6 max-w-[58ch] text-base leading-relaxed text-(--color-text-secondary) md:text-lg">
              Sessions, memory, skills, workspaces, automation, bridges — the whole runtime in a
              single local binary. Then the part nobody else ships: an open protocol so your agents
              discover peers, delegate work, and collect receipts across machines.
            </p>

            <div className="mt-8 flex flex-col items-start gap-3 sm:flex-row sm:flex-wrap">
              <CtaButton href="/runtime/core/getting-started/installation" variant="primary">
                Install the runtime
              </CtaButton>
              <CtaButton href="/protocol" variant="ghost">
                See the network
              </CtaButton>
            </div>
          </div>

          {/* Visual comes after copy on desktop (and on mobile flows under). */}
          <div className="order-1 lg:order-0">
            <HeroPlayer />
          </div>
        </div>
        <dl className="mt-10 grid grid-cols-2 gap-3 md:grid-cols-4">
          {signalItems.map(item => (
            <div
              key={item.label}
              className="rounded-(--radius-diagram) border border-white/10 p-4 backdrop-blur-sm"
            >
              <dt className="font-mono text-[12px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
                {item.label}
              </dt>
              <dd className="mt-1.5 text-[12px] leading-relaxed text-(--color-text-secondary)">
                {item.detail}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
