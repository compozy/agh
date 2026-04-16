import { Play, Brain, Sparkles, FolderOpen, Timer, Network, Webhook } from "lucide-react";
import type { LucideIcon } from "lucide-react";

interface Feature {
  icon: LucideIcon;
  label: string;
  title: string;
  description: string;
}

interface FeatureGroup {
  label: string;
  title: string;
  description: string;
  items: Feature[];
}

const featureGroups: FeatureGroup[] = [
  {
    label: "SESSION CONTROL",
    title: "Durable execution with operator visibility",
    description:
      "The runtime is designed to keep the core unit of work alive, inspectable, and easy to resume.",
    items: [
      {
        icon: Play,
        label: "SESSIONS",
        title: "Durable Sessions",
        description:
          "Resume long-running work, recover from interruptions, and stop treating agent runs like throwaway chats.",
      },
      {
        icon: Brain,
        label: "REPLAY",
        title: "Replayable History",
        description:
          "Inspect prompts, tool use, and outcomes when something breaks or needs review.",
      },
      {
        icon: Webhook,
        label: "OBSERVE",
        title: "One Operator Surface",
        description:
          "Watch live activity, inspect health, and manage runtime behavior from the CLI, APIs, or the web UI.",
      },
    ],
  },
  {
    label: "CONTEXT SYSTEM",
    title: "State that stays attached to the workspace",
    description:
      "Context should not depend on the operator remembering which prompt fragments to paste back in.",
    items: [
      {
        icon: Sparkles,
        label: "MEMORY",
        title: "Memory That Sticks",
        description:
          "Keep global preferences and workspace context without rebuilding it every time you start a session.",
      },
      {
        icon: FolderOpen,
        label: "SKILLS",
        title: "Skills Without Glue Code",
        description: "Package repeatable behavior once and apply it across agents and projects.",
      },
      {
        icon: Brain,
        label: "WORKSPACES",
        title: "Workspace-Aware Defaults",
        description:
          "Change agents, policies, and context per project instead of managing one flat global setup.",
      },
    ],
  },
  {
    label: "EVENTING EDGE",
    title: "External hooks when work leaves the core loop",
    description:
      "Automation and bridges let agents participate in the systems around them without the runtime turning into a black box.",
    items: [
      {
        icon: Timer,
        label: "AUTOMATION",
        title: "Automation and Triggers",
        description:
          "Kick off recurring work from schedules, events, or webhooks when humans should not be the queue.",
      },
      {
        icon: Network,
        label: "BRIDGES",
        title: "Bridges to Real Work",
        description:
          "Route work into Slack, Discord, or Telegram when agents need humans or external surfaces.",
      },
    ],
  },
];

export function RuntimeFeatures() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-20 md:py-28">
      <div className="mx-auto max-w-[var(--site-layout-width)]">
        <div className="flex flex-col gap-12 lg:flex-row lg:items-start lg:justify-between lg:gap-16">
          <div className="max-w-[500px] lg:sticky lg:top-24">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
              AGH RUNTIME
            </p>
            <h2 className="mt-5 text-[clamp(2.4rem,5vw,3.8rem)] leading-[0.98] font-semibold tracking-[-0.03em] text-[var(--color-text-primary)]">
              Control, continuity, and operator context.
            </h2>
            <p className="mt-6 text-[1.05rem] leading-relaxed text-[var(--color-text-secondary)]">
              AGH Runtime is for teams that want the power of real agent CLIs without giving up
              inspection, replay, or day-two control. The system groups around three jobs:
              execution, context, and external coordination.
            </p>
          </div>

          <div className="flex w-full flex-col gap-8 lg:max-w-[640px]">
            {featureGroups.map(group => (
              <div key={group.label} className="rounded-[24px] bg-[var(--color-canvas)] p-6 md:p-8">
                <div className="flex flex-col gap-4 border-b border-[var(--color-divider)] pb-6 md:flex-row md:items-start md:justify-between">
                  <div className="max-w-[340px]">
                    <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
                      {group.label}
                    </p>
                    <h3 className="mt-3 text-[1.5rem] leading-[1.1] font-medium tracking-[-0.02em] text-[var(--color-text-primary)]">
                      {group.title}
                    </h3>
                  </div>
                  <p className="mt-1 max-w-[260px] text-[0.95rem] leading-relaxed text-[var(--color-text-secondary)] md:mt-0">
                    {group.description}
                  </p>
                </div>

                <div className="mt-6 grid gap-6 md:grid-cols-2">
                  {group.items.map(feature => {
                    const Icon = feature.icon;
                    return (
                      <div key={feature.label} className="flex flex-col items-start gap-4">
                        <div className="flex h-12 w-12 items-center justify-center rounded-[12px] bg-[var(--color-surface-elevated)]">
                          <Icon className="h-5 w-5 text-[var(--color-accent)]" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="text-[1.1rem] font-medium text-[var(--color-text-primary)]">
                              {feature.title}
                            </h4>
                          </div>
                          <p className="mt-2 text-[0.95rem] leading-relaxed text-[var(--color-text-secondary)]">
                            {feature.description}
                          </p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
