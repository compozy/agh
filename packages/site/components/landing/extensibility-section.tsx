import { FileCode2, Plug, Sparkles, Timer } from "lucide-react";
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
    cite: { href: "/runtime", label: "hooks catalog" },
  },
  {
    icon: <Sparkles className="h-4 w-4" />,
    eyebrow: "Skills",
    title: "Drop-in SKILL.md bundles",
    description:
      "Share reusable instruction sets with YAML frontmatter and Markdown body. Bundled defaults + global + workspace scopes.",
    cite: { href: "/runtime", label: "skills guide" },
  },
  {
    icon: <Timer className="h-4 w-4" />,
    eyebrow: "Automation",
    title: "Cron + webhook + event triggers",
    description:
      "Durable jobs and triggers stored in SQLite. Schedule work. Delegate to peers. Track runs.",
    cite: { href: "/runtime", label: "automation" },
  },
  {
    icon: <Plug className="h-4 w-4" />,
    eyebrow: "Extensions",
    title: "Install from local or marketplace",
    description:
      "Extensions bundle skills, hooks, bridge adapters, and MCP servers. Ship them as zip files or via a GitHub registry.",
    cite: { href: "/runtime", label: "extensions" },
  },
];

export function ExtensibilitySection() {
  return (
    <SectionFrame background="canvas" padY="lg">
      <SectionHeader
        align="start"
        eyebrow="Extensibility"
        title="Hooks, skills, automation, extensions."
        description="The daemon is extensible at every seam you actually need. No plugins to write — contracts are plain files."
      />

      <div className="mt-20 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
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

      <div className="mt-10 grid gap-8 lg:grid-cols-[minmax(0,360px)_minmax(0,1fr)] lg:items-center lg:gap-10">
        <div className="max-w-[56ch] text-sm leading-relaxed text-(--color-text-secondary)">
          <h3 className="font-display text-2xl mb-2 mt-8 text-white">
            A skill is a Markdown file with frontmatter.
          </h3>
          <p>
            A hook is a TOML block in your config. Everything the daemon loads is inspectable with{" "}
            <code className="font-mono text-(--color-text-primary)">agh skill view</code>,{" "}
            <code className="font-mono text-(--color-text-primary)">agh hooks list</code>, and{" "}
            <code className="font-mono text-(--color-text-primary)">agh extension list</code>.
          </p>
          <p className="mt-4 font-mono text-[11px] uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
            Contract on disk — not a plugin API.
          </p>
        </div>
        <img
          src="/images/extensibility-skill-contract-v1.png"
          alt="deploy-staging.skill.md shown as a Markdown skill contract with frontmatter, deployment capabilities, and a staged execution trace."
          loading="lazy"
          decoding="async"
          className="block w-full object-cover object-center opacity-95"
        />
      </div>
    </SectionFrame>
  );
}
