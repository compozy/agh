import Link from "next/link";
import { ArrowRight } from "lucide-react";

const runtimeFeatures = [
  "Sessions survive restarts — inspect, replay, and resume",
  "Workspace-scoped memory and skills",
  "CLI, HTTP/SSE, and web UI on one daemon",
  "Automation triggers and external bridges",
];

const networkFeatures = [
  "Agents discover peers by capability, not hardcoded config",
  "Delegate tasks across coders, reviewers, and deployers",
  "Structured updates and receipts between runtimes",
  "Adopt incrementally — keep your existing control plane",
  "Human-in-the-loop governance at the coordination edge",
];

export function TwoPillars() {
  return (
    <section className="bg-[var(--color-surface)] px-4 pt-10 pb-20 md:pt-14 md:pb-28">
      <div className="mx-auto max-w-[var(--site-layout-width)]">
        <div className="max-w-[640px] border-b border-[var(--color-divider)] pb-10">
          <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
            TWO SURFACES, ONE SYSTEM
          </p>
          <h2 className="mt-5 text-[clamp(2.4rem,5vw,3.6rem)] leading-[1.0] font-normal tracking-[-0.03em] text-[var(--color-text-primary)]">
            A runtime you control. A network agents share.
          </h2>
          <p className="mt-5 max-w-[56ch] text-base leading-relaxed text-[var(--color-text-secondary)]">
            Most agent tools stop at the runtime boundary. AGH adds an open coordination layer so
            specialized agents can find each other and move work between runtimes.
          </p>
        </div>

        <div className="mt-12 flex flex-col gap-6 lg:flex-row lg:items-stretch">
          {/* Network pillar — the differentiator, gets more space */}
          <div className="flex flex-col rounded-[12px] border border-[var(--color-accent)] border-opacity-30 bg-[var(--color-canvas)] p-8 md:p-10 lg:w-[55%]">
            <div className="flex items-center gap-3">
              <span className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
                AGH Network
              </span>
              <span className="rounded-[6px] bg-[var(--color-accent-tint)] px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-accent)]">
                differentiator
              </span>
            </div>
            <h3 className="mt-4 max-w-[18ch] text-2xl leading-[1.1] font-semibold tracking-[-0.02em] text-[var(--color-text-primary)]">
              Open coordination between agent runtimes
            </h3>
            <p className="mt-4 max-w-[48ch] text-sm leading-relaxed text-[var(--color-text-secondary)]">
              When a code agent needs an infra specialist or a data analyst, AGH Network gives them
              a shared protocol for discovery, delegation, and structured updates — without forcing
              everyone onto one tool.
            </p>

            <div className="mt-8 overflow-hidden rounded-[8px] border border-[var(--color-divider)]">
              {networkFeatures.map(feature => (
                <div
                  key={feature}
                  className="flex items-start gap-3 border-t border-[var(--color-divider)] px-5 py-3.5 first:border-t-0"
                >
                  <span className="mt-1.5 shrink-0 inline-block h-1.5 w-1.5 rounded-full bg-[var(--color-accent)]" />
                  <span className="text-sm leading-snug text-[var(--color-text-secondary)]">
                    {feature}
                  </span>
                </div>
              ))}
            </div>

            <Link
              href="/protocol"
              className="mt-auto flex items-center gap-2 pt-8 text-sm font-medium text-[var(--color-accent)] transition-colors hover:text-[var(--color-accent-hover)]"
            >
              Read the AGH Network docs
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>

          {/* Runtime pillar */}
          <div className="flex flex-col rounded-[12px] bg-[var(--color-canvas)] p-8 md:p-10 lg:w-[45%]">
            <span className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
              AGH Runtime
            </span>
            <h3 className="mt-4 max-w-[16ch] text-2xl leading-[1.1] font-semibold tracking-[-0.02em] text-[var(--color-text-primary)]">
              Local-first operator control plane
            </h3>
            <p className="mt-4 max-w-[48ch] text-sm leading-relaxed text-[var(--color-text-secondary)]">
              One binary that turns agent CLIs into managed work with sessions, memory, skills, and
              observability — all running on your machine.
            </p>

            <div className="mt-8 overflow-hidden rounded-[8px] border border-[var(--color-divider)]">
              {runtimeFeatures.map(feature => (
                <div
                  key={feature}
                  className="flex items-start gap-3 border-t border-[var(--color-divider)] px-5 py-3.5 first:border-t-0"
                >
                  <span className="mt-1.5 shrink-0 inline-block h-1.5 w-1.5 rounded-full bg-[var(--color-text-tertiary)]" />
                  <span className="text-sm leading-snug text-[var(--color-text-secondary)]">
                    {feature}
                  </span>
                </div>
              ))}
            </div>

            <Link
              href="/docs/getting-started"
              className="mt-auto flex items-center gap-2 pt-8 text-sm font-medium text-[var(--color-accent)] transition-colors hover:text-[var(--color-accent-hover)]"
            >
              Get started with the runtime
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}
