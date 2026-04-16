import { Activity, Boxes, Database, FileCode2, Network, Plug, Sparkles, Timer } from "lucide-react";
import { FeatureCard } from "./primitives/feature-card";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

const FEATURES = [
  {
    icon: <Database className="h-4 w-4" />,
    eyebrow: "Sessions",
    title: "Resume any agent run",
    description:
      "Every agent run is a durable session. Stop, resume, inspect every step, fork from any point.",
  },
  {
    icon: <Sparkles className="h-4 w-4" />,
    eyebrow: "Memory",
    title: "Context that survives restarts",
    description:
      "Global and per-workspace memory in plain Markdown. Four types, one index per scope.",
  },
  {
    icon: <FileCode2 className="h-4 w-4" />,
    eyebrow: "Skills",
    title: "Reusable playbooks",
    description:
      "Drop-in SKILL.md bundles with YAML frontmatter. Bundled library, workspace overrides, community catalog.",
  },
  {
    icon: <Boxes className="h-4 w-4" />,
    eyebrow: "Workspaces",
    title: "Per-project everything",
    description:
      "Agents, skills, memory, and config overlay per workspace. Switch projects, switch context.",
  },
  {
    icon: <Timer className="h-4 w-4" />,
    eyebrow: "Automation",
    title: "Cron + webhooks, durable",
    description:
      "Schedule recurring work. Trigger sessions from external events. Every run tracked in SQLite.",
  },
  {
    icon: <Activity className="h-4 w-4" />,
    eyebrow: "Observability",
    title: "Everything logged, everything replayable",
    description:
      "Token usage, permission audit, tool calls, errors — streamed over SSE, persisted to disk.",
  },
  {
    icon: <Plug className="h-4 w-4" />,
    eyebrow: "Hooks",
    title: "Inject logic anywhere",
    description:
      "Run scripts or sub-agents on ~40 lifecycle events — permission checks, tool calls, network receipts.",
  },
  {
    icon: <Network className="h-4 w-4" />,
    eyebrow: "Bridges",
    title: "Slack, Discord, Telegram in — replies out",
    description:
      "Platform webhooks become sessions. Response events stream back to the original thread.",
  },
];

export function FeaturesSection() {
  return (
    <SectionFrame background="canvas" padY="lg">
      <SectionHeader
        align="start"
        eyebrow="What you get"
        title="Everything a modern agent runtime should have."
        description="You already know you need sessions, memory, and skills. AGH ships all of it, local-first, with an operator surface you can script."
      />

      <ul className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {FEATURES.map(feature => (
          <li key={feature.eyebrow}>
            <FeatureCard
              icon={feature.icon}
              eyebrow={feature.eyebrow}
              title={feature.title}
              description={feature.description}
              className="h-full"
            />
          </li>
        ))}
      </ul>
    </SectionFrame>
  );
}
