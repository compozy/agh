import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

const FEATURES = [
  {
    eyebrow: "Sessions",
    title: "Resume any agent run",
    description:
      "Every agent run is a durable session. Stop, resume, inspect every step, fork from any point.",
    image: "/images/everything/illustration_01.png",
    imageAlt: "Durable session timeline with code edits and replay checkpoints.",
  },
  {
    eyebrow: "Memory",
    title: "Context that survives restarts",
    description:
      "Global and per-workspace memory in plain Markdown. Four types, one index per scope.",
    image: "/images/everything/illustration_02.png",
    imageAlt: "Memory cards stored in a global Markdown index.",
  },
  {
    eyebrow: "Skills",
    title: "Reusable playbooks",
    description:
      "Drop-in SKILL.md bundles with YAML frontmatter. Bundled library, workspace overrides, community catalog.",
    image: "/images/everything/illustration_04.png",
    imageAlt: "Playbook YAML connected to read, run, analyze, and propose steps.",
  },
  {
    eyebrow: "Workspaces",
    title: "Per-project everything",
    description:
      "Agents, skills, memory, and config overlay per workspace. Switch projects, switch context.",
    image: "/images/everything/illustration_05.png",
    imageAlt: "Workspace folders with isolated context, memory, and config cards.",
  },
  {
    eyebrow: "Automation",
    title: "Cron + webhooks, durable",
    description:
      "Schedule recurring work. Trigger sessions from external events. Every run tracked in SQLite.",
    image: "/images/everything/illustration_06.png",
    imageAlt: "Automation job fan-out to archive, notify, webhook, and summary actions.",
  },
  {
    eyebrow: "Observability",
    title: "Everything logged, everything replayable",
    description:
      "Token usage, permission audit, tool calls, errors — streamed over SSE, persisted to disk.",
    image: "/images/everything/illustration_03.png",
    imageAlt: "Replay trace, event chart, and top tool usage panels.",
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

      <ul className="mt-12 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {FEATURES.map(feature => (
          <li key={feature.eyebrow}>
            <article
              data-testid="feature-card"
              className="group flex h-full min-h-[420px] flex-col overflow-hidden rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) p-4 transition-colors duration-300 hover:border-[color-mix(in_srgb,var(--color-accent)_40%,var(--color-divider))] sm:p-5"
            >
              <div className="overflow-hidden rounded-[8px]">
                <img
                  src={feature.image}
                  alt={feature.imageAlt}
                  loading="lazy"
                  decoding="async"
                  className="block aspect-[16/10] w-full object-contain opacity-95 transition-transform duration-500 ease-out group-hover:scale-[1.02]"
                />
              </div>
              <div className="flex flex-1 flex-col pt-5">
                <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
                  {feature.eyebrow}
                </p>
                <h3 className="mt-3 text-[1.0625rem] font-medium leading-snug tracking-[-0.01em] text-(--color-text-primary)">
                  {feature.title}
                </h3>
                <p className="mt-3 text-sm leading-relaxed text-(--color-text-secondary)">
                  {feature.description}
                </p>
              </div>
            </article>
          </li>
        ))}
      </ul>
    </SectionFrame>
  );
}
