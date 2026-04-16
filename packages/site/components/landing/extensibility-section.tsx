import { Brain, FileCode2, Plug, Sparkles, Timer } from "lucide-react";
import { CodeBlock } from "./primitives/code-block";
import { FeatureCard } from "./primitives/feature-card";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

const FEATURES = [
  {
    icon: <FileCode2 className="h-4 w-4" />,
    eyebrow: "Hooks",
    title: "Observe and mutate lifecycle events",
    description:
      "Run shell, builtin, or sub-agent actions on ~40 lifecycle events — session start, tool call, permission request, network receipt.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "hooks catalog" },
  },
  {
    icon: <Sparkles className="h-4 w-4" />,
    eyebrow: "Skills",
    title: "Drop-in SKILL.md bundles",
    description:
      "Share reusable instruction sets with YAML frontmatter and Markdown body. Bundled defaults + global + workspace scopes.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "skills guide" },
  },
  {
    icon: <Brain className="h-4 w-4" />,
    eyebrow: "Memory",
    title: "Global + workspace, with dream consolidation",
    description:
      "Four memory types (user, feedback, project, reference). A consolidation pass synthesizes indexes from recent sessions.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "memory scopes" },
  },
  {
    icon: <Timer className="h-4 w-4" />,
    eyebrow: "Automation",
    title: "Cron + webhook + event triggers",
    description:
      "Durable jobs and triggers stored in SQLite. Schedule work. Delegate to peers. Track runs.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "automation" },
  },
  {
    icon: <Plug className="h-4 w-4" />,
    eyebrow: "Extensions",
    title: "Install from local or marketplace",
    description:
      "Extensions bundle skills, hooks, bridge adapters, and MCP servers. Ship them as zip files or via a GitHub registry.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "extensions" },
  },
];

const SKILL_MD = `---
name: deploy-staging
description: Trigger a staged deploy from any session
kind: skill
capabilities: [deploy, rollback]
---

Runs a validated deploy to staging and streams logs
back as a trace. Delegates to the deployer peer if
one is online; otherwise runs locally.

# [[hooks]]
# event = "session.prompt.before"
# run = "sh .agh/hooks/redact-secrets.sh"`;

export function ExtensibilitySection() {
  return (
    <SectionFrame background="canvas" padY="lg">
      <SectionHeader
        align="start"
        eyebrow="Extensibility"
        title="Hooks, skills, memory, automation, extensions."
        description="The daemon is extensible at every seam you actually need. No plugins to write — contracts are plain files."
      />

      <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-[repeat(5,minmax(0,1fr))]">
        {FEATURES.map(feature => (
          <FeatureCard
            key={feature.eyebrow}
            icon={feature.icon}
            eyebrow={feature.eyebrow}
            title={feature.title}
            description={feature.description}
            cite={feature.cite}
          />
        ))}
      </div>

      <div className="mt-10 grid gap-6 lg:grid-cols-[1fr_minmax(0,520px)] lg:items-center">
        <div className="max-w-[56ch] text-sm leading-relaxed text-(--color-text-secondary)">
          <p>
            A skill is a Markdown file with frontmatter. A hook is a TOML block in your config.
            Everything the daemon loads is inspectable with{" "}
            <code className="font-mono text-(--color-text-primary)">agh skill view</code>,{" "}
            <code className="font-mono text-(--color-text-primary)">agh hooks list</code>, and{" "}
            <code className="font-mono text-(--color-text-primary)">agh extension list</code>.
          </p>
          <p className="mt-4 font-mono text-[11px] uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
            Contract on disk — not a plugin API.
          </p>
        </div>
        <CodeBlock code={SKILL_MD} caption="deploy-staging.skill.md" language="markdown" />
      </div>
    </SectionFrame>
  );
}
