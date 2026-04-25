import { Activity, Database, FileCode2, Network, Plug } from "lucide-react";

const cardBase =
  "group relative isolate min-w-0 overflow-hidden rounded-(--radius-diagram) border border-(--color-divider) bg-[#11100f] p-7 transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_40%,var(--color-divider))] sm:p-8 xl:p-10";

const labelBase =
  "mb-5 flex items-center gap-3 font-mono text-[11px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)";

const imageBase = "h-full w-full select-none opacity-95";

export function BentoSection() {
  return (
    <section
      id="runtime-map"
      aria-label="AGH runtime map"
      className="scroll-mt-24 border-y border-(--color-divider) bg-(--color-canvas-deep) px-4 py-6 sm:px-5 md:py-10 lg:px-5 lg:py-24"
    >
      <div
        data-testid="bento-grid"
        className="mx-auto grid w-full max-w-[1200px] gap-4 md:grid-cols-2 lg:aspect-[1536/1200] lg:grid-cols-[minmax(0,0.67fr)_minmax(0,1fr)] lg:grid-rows-[2.5fr_2fr]"
      >
        <RuntimeCard />
        <NetworkCard />
        <BridgesCard />

        <div className="grid min-w-0 gap-4 md:grid-cols-2 lg:col-start-2 lg:row-start-2 lg:grid-cols-[minmax(0,0.42fr)_minmax(0,0.58fr)]">
          <MemoryCard />
          <TraceCard />
        </div>
      </div>
    </section>
  );
}

function RuntimeCard() {
  return (
    <article
      data-testid="bento-runtime"
      className={`${cardBase} min-h-[540px] md:min-h-[560px] lg:col-start-1 lg:row-start-1 lg:min-h-0`}
    >
      <div className="absolute inset-x-0 bottom-0 top-[0%] pointer-events-none">
        <img
          src="/images/bento-illustrations/runtime-v2.png"
          alt="AGH runtime device showing durable agent sessions and status indicators."
          loading="eager"
          decoding="async"
          className={`${imageBase} object-cover`}
        />
      </div>
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgba(14,14,15,0.68)_0%,rgba(14,14,15,0.18)_42%,rgba(14,14,15,0)_68%)]" />

      <div className="relative z-10 max-w-[21rem]">
        <div className={labelBase}>
          <Database className="h-4 w-4" />
          <span>Runtime</span>
        </div>
        <h3
          aria-label="Your agents. Under control."
          className="font-display text-[2rem] font-normal leading-[1.08] text-(--color-text-primary) sm:text-[2.35rem] xl:text-[2.5rem]"
        >
          Your agents.
          <br />
          <span className="text-(--color-accent)">Under control.</span>
        </h3>
        <span className="mt-5 block h-px w-8 bg-(--color-accent)" aria-hidden="true" />
      </div>
    </article>
  );
}

function NetworkCard() {
  return (
    <article
      data-testid="bento-network"
      className={`${cardBase} min-h-[420px] md:col-span-2 md:min-h-[500px] lg:col-span-1 lg:col-start-2 lg:row-start-1 lg:min-h-0`}
    >
      <div className="absolute inset-0 pointer-events-none">
        <img
          src="/images/bento-illustrations/network-v2.png"
          alt="AGH network diagram showing discovery, delegation, receipt, and peers."
          loading="eager"
          decoding="async"
          className={`${imageBase} object-contain object-[40%_100%]`}
        />
      </div>
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgba(14,14,15,0.94)_0%,rgba(14,14,15,0.78)_21%,rgba(14,14,15,0)_48%)]" />

      <div className="relative z-10 max-w-[30rem]">
        <div className={labelBase}>
          <Network className="h-4 w-4" />
          <span>Network</span>
        </div>
        <h3
          aria-label="Built-in network. Delegate. Deliver. Done."
          className="font-display text-[1.9rem] font-normal leading-[1.08] text-(--color-text-primary) sm:text-[2.2rem] xl:text-[2.50rem]"
        >
          Built-in network.
          <br />
          <span className="text-(--color-accent)">Delegate. Deliver.</span> Done.
        </h3>
      </div>
    </article>
  );
}

function BridgesCard() {
  return (
    <article
      data-testid="bento-bridges"
      className={`${cardBase} min-h-[360px] md:min-h-[390px] lg:col-start-1 lg:row-start-2 lg:min-h-0`}
    >
      <div className="absolute inset-0 pointer-events-none">
        <img
          src="/images/bento-illustrations/bridges-v2.png"
          alt="Bridge events from Slack, Discord, and Telegram entering an AGH device."
          loading="lazy"
          decoding="async"
          className={`${imageBase} object-cover object-[30%_100%]`}
        />
      </div>
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgba(14,14,15,0.84)_0%,rgba(14,14,15,0.4)_30%,rgba(14,14,15,0)_64%)]" />

      <div className="relative z-10 max-w-[18rem]">
        <div className={labelBase}>
          <Plug className="h-4 w-4" />
          <span>Bridges</span>
        </div>
        <h3
          aria-label="From anywhere. Into a session."
          className="font-display text-[1.65rem] font-normal leading-[1.08] text-(--color-text-primary) sm:text-[1.9rem] xl:text-[2rem]"
        >
          From anywhere.
          <br />
          <span className="text-(--color-accent)">Into a session.</span>
        </h3>
      </div>
    </article>
  );
}

function MemoryCard() {
  return (
    <article data-testid="bento-memory" className={`${cardBase} min-h-[390px] lg:min-h-0`}>
      <div className="absolute inset-x-0 bottom-0 top-[18%] pointer-events-none">
        <img
          src="/images/bento-illustrations/memory-v2.png"
          alt="Skill document carrying deployment intent into AGH memory."
          width={1076}
          height={1462}
          loading="lazy"
          decoding="async"
          className={`${imageBase} object-cover object-[50%_80%]`}
        />
      </div>
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgba(14,14,15,0.92)_0%,rgba(14,14,15,0.72)_27%,rgba(14,14,15,0)_58%)]" />

      <div className="relative z-10 max-w-[17rem]">
        <div className={labelBase}>
          <FileCode2 className="h-4 w-4" />
          <span>Memory</span>
        </div>
        <h3
          aria-label="Context that remembers."
          className="font-display text-[1.8rem] font-normal leading-[1.08] text-(--color-text-primary) sm:text-[2rem] xl:text-[2.2rem]"
        >
          Context that
          <br />
          <span className="text-(--color-accent)">remembers.</span>
        </h3>
      </div>
    </article>
  );
}

function TraceCard() {
  return (
    <article data-testid="bento-trace" className={`${cardBase} min-h-[390px] lg:min-h-0`}>
      <div className="absolute inset-0 pointer-events-none">
        <img
          src="/images/bento-illustrations/trace-v2.png"
          alt="Replay trace timeline with session events and health status."
          loading="lazy"
          decoding="async"
          className={`${imageBase} object-cover object-[50%_70%]`}
        />
      </div>
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgba(14,14,15,0.92)_0%,rgba(14,14,15,0.66)_26%,rgba(14,14,15,0)_58%)]" />

      <div className="relative z-10 max-w-[21rem]">
        <div className={labelBase}>
          <Activity className="h-4 w-4" />
          <span>Trace</span>
        </div>
        <h3
          aria-label="Every step Traceable."
          className="font-display text-[1.8rem] font-normal leading-[1.08] text-(--color-text-primary) sm:text-[2rem] xl:text-[2.2rem]"
        >
          Every step.
          <br />
          <span className="text-(--color-accent)">traceable</span>.
        </h3>
      </div>
    </article>
  );
}
