import { Play, Brain, Sparkles, FolderOpen, Timer, Network, Webhook, Puzzle } from "lucide-react";
import type { LucideIcon } from "lucide-react";

interface Feature {
  icon: LucideIcon;
  label: string;
  title: string;
  description: string;
}

const features: Feature[] = [
  {
    icon: Play,
    label: "SESSIONS",
    title: "Session Lifecycle",
    description:
      "Spawn, pause, resume, and replay agent sessions. Full event persistence with SQLite — nothing is lost.",
  },
  {
    icon: Brain,
    label: "MEMORY",
    title: "Persistent Memory",
    description:
      "Dual-scope memory — global knowledge and workspace-specific context. Dream consolidation distills insights automatically.",
  },
  {
    icon: Sparkles,
    label: "SKILLS",
    title: "Skills System",
    description:
      "Declarative SKILL.md format. Bundled skills, marketplace discovery, workspace-scoped activation.",
  },
  {
    icon: FolderOpen,
    label: "WORKSPACES",
    title: "Workspace Isolation",
    description:
      "Copy the directory, and the agent works. Config overlays, multi-root support, portable by default.",
  },
  {
    icon: Timer,
    label: "AUTOMATION",
    title: "Jobs & Triggers",
    description:
      "Cron schedules, event triggers, webhook endpoints. Automate agent work without manual intervention.",
  },
  {
    icon: Network,
    label: "BRIDGES",
    title: "Platform Bridges",
    description:
      "Route agent interactions to Slack, Discord, Telegram. Platform adapters with bidirectional message flow.",
  },
  {
    icon: Webhook,
    label: "HOOKS",
    title: "Event Hooks",
    description:
      "Declare matchers and executors for any event in the system. Extend behavior without modifying core code.",
  },
  {
    icon: Puzzle,
    label: "EXTENSIONS",
    title: "Extension System",
    description:
      "Install, develop, and share extensions via the marketplace. First-class plugin architecture.",
  },
];

export function RuntimeFeatures() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-16 md:py-24">
      <div className="mx-auto max-w-5xl">
        <p className="text-center font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
          RUNTIME
        </p>
        <h2 className="mt-3 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          Orchestrates real agent CLIs, not API wrappers
        </h2>
        <p className="mx-auto mt-4 max-w-2xl text-center text-sm leading-relaxed text-[var(--color-text-secondary)]">
          Single binary. No sidecars. No external services. Everything your agents need to operate,
          built into one daemon.
        </p>
        <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {features.map(feature => {
            const Icon = feature.icon;
            return (
              <div
                key={feature.label}
                className="flex flex-col rounded-xl bg-[var(--color-canvas)] p-5"
              >
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-[var(--color-surface-elevated)]">
                  <Icon className="h-4 w-4 text-[var(--color-accent)]" />
                </div>
                <span className="mt-4 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
                  {feature.label}
                </span>
                <h3 className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                  {feature.title}
                </h3>
                <p className="mt-2 text-xs leading-relaxed text-[var(--color-text-secondary)]">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
