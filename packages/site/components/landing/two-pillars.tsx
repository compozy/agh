import Link from "next/link";
import { ArrowRight } from "lucide-react";

const pillars = [
  {
    label: "RUNTIME",
    title: "Agent Operating System",
    description:
      "Single binary. No sidecars. No external services. AGH spawns and manages agent CLIs as first-class sessions with memory, skills, and workspace isolation.",
    features: [
      "Session lifecycle management",
      "Persistent dual-scope memory",
      "Skills and extension marketplace",
      "Workspace-aware configuration",
    ],
    href: "/runtime",
    cta: "Explore the Runtime",
  },
  {
    label: "PROTOCOL",
    title: "Agent Network Protocol",
    description:
      "MCP connects agents to tools. AGH connects agents to agents. An open wire format that any harness can implement — discover peers, exchange messages, coordinate work.",
    features: ["7 message kinds", "Interaction lifecycle", "Peer discovery", "Transport-agnostic"],
    href: "/protocol",
    cta: "Read the Protocol Spec",
  },
];

export function TwoPillars() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-16 md:py-24">
      <div className="mx-auto max-w-5xl">
        <p className="text-center font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
          TWO PRODUCTS, ONE BINARY
        </p>
        <h2 className="mt-3 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          Runtime + Protocol
        </h2>
        <div className="mt-12 grid gap-6 md:grid-cols-2">
          {pillars.map(pillar => (
            <div
              key={pillar.label}
              className="flex flex-col rounded-xl border border-[var(--color-divider)] bg-[var(--color-canvas)] p-6 md:p-8"
            >
              <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-accent)]">
                {pillar.label}
              </span>
              <h3 className="mt-2 text-xl font-semibold text-[var(--color-text-primary)]">
                {pillar.title}
              </h3>
              <p className="mt-3 text-sm leading-relaxed text-[var(--color-text-secondary)]">
                {pillar.description}
              </p>
              <ul className="mt-6 flex flex-col gap-2">
                {pillar.features.map(feature => (
                  <li
                    key={feature}
                    className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]"
                  >
                    <span className="inline-block h-1.5 w-1.5 rounded-full bg-[var(--color-accent)]" />
                    {feature}
                  </li>
                ))}
              </ul>
              <Link
                href={pillar.href}
                className="mt-auto flex items-center gap-1.5 pt-6 text-sm font-medium text-[var(--color-accent)] transition-colors hover:text-[var(--color-accent-hover)]"
              >
                {pillar.cta}
                <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
