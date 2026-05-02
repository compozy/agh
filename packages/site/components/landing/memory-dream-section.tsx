import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { CodeBlock } from "./primitives/code-block";
import { SectionFrame } from "./primitives/section-frame";

const MEMORY_CODE = `agh memory write personal-notes \\
  --type user \\
  --description "Pedro prefers BR-PT in conversation"
agh memory search "BR-PT"
agh memory consolidate`;

const STEPS = [
  {
    eyebrow: "Plain files",
    title: "Memory at ~/.agh/memory/*.md",
    description:
      "Four typed files — user, feedback, project, reference — with YAML frontmatter, scoped to global or workspace. Version it. Diff it. Port it across providers.",
  },
  {
    eyebrow: "Dream consolidation",
    title: "Time → Sessions → Lock cascade",
    description:
      "Default gates: 24h, 5 touched sessions, file-lock. When all three pass, AGH spawns an ephemeral session that synthesizes recent activity into durable facts. No surprise compute.",
  },
  {
    eyebrow: "Agent-managed",
    title: "Same surface for you and the agent",
    description:
      "agh memory write | search | consolidate works from CLI, HTTP, and UDS. Operators inspect the same files agents write — no privileged path.",
  },
];

export function MemoryDreamSection() {
  return (
    <SectionFrame
      className="relative border-b border-(--color-divider)"
      background="canvas"
      padY="lg"
      ariaLabel="Memory and dream consolidation"
    >
      <div className="grid min-w-0 gap-12 lg:grid-cols-[minmax(0,400px)_1fr] lg:items-start lg:gap-16">
        <div className="flex h-full min-w-0 flex-col justify-between lg:sticky lg:top-24">
          <div>
            <p className="font-mono text-[11px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
              Memory
            </p>
            <h2 className="mt-3 text-[clamp(1.9rem,3.4vw,2.6rem)] leading-[1.05] font-normal tracking-[-0.025em] text-(--color-text-primary)">
              Memory that compounds
              <br />
              <span className="italic text-(--color-text-tertiary)">while you sleep.</span>
            </h2>
            <p className="mt-4 max-w-[50ch] text-sm leading-relaxed text-(--color-text-secondary)">
              Memory is not a vector database. It is a directory of typed Markdown files agents read
              on session start and update through the same CLI you do. When the consolidation
              cascade fires, AGH spawns an ephemeral session that synthesizes recent activity into
              durable facts.
            </p>
            <Link
              href="/runtime"
              className="mt-6 inline-flex items-center gap-1.5 text-sm font-medium text-(--color-accent) transition-colors hover:text-(--color-accent-hover)"
            >
              Read the memory and dream guide
              <ArrowUpRight aria-hidden className="h-4 w-4" />
            </Link>
          </div>
          <div className="mt-12 hidden lg:block">
            <img
              src="/images/runtime/memory-dream-landing-v1.png"
              alt="AGH memory interface diagram showing scoped Markdown files, memory indexing, and dream consolidation into durable memory."
              loading="lazy"
              decoding="async"
              className="block w-full max-w-[400px] select-none object-contain opacity-95"
            />
          </div>
        </div>

        <div className="flex min-w-0 flex-col gap-0">
          <ol className="flex flex-col divide-y divide-(--color-divider)">
            {STEPS.map((step, index) => (
              <li
                key={step.eyebrow}
                className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-x-6 gap-y-2 py-7 first:pt-0"
              >
                <span
                  aria-hidden="true"
                  className="font-mono text-[12px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent) tabular-nums"
                >
                  {String(index + 1).padStart(2, "0")}
                </span>
                <div className="min-w-0">
                  <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
                    {step.eyebrow}
                  </p>
                  <h3 className="mt-2 text-base font-medium leading-snug text-(--color-text-primary)">
                    {step.title}
                  </h3>
                  <p className="mt-2 max-w-[60ch] text-sm leading-relaxed text-(--color-text-secondary)">
                    {step.description}
                  </p>
                </div>
              </li>
            ))}
          </ol>

          <div className="mt-10">
            <CodeBlock code={MEMORY_CODE} caption="agh memory" shell />
          </div>
        </div>
      </div>
    </SectionFrame>
  );
}
