import { ArrowRight, Layers3, Plus, Sparkles } from "lucide-react";

import { cn } from "@/lib/utils";

import { PageContent } from "./page-content";
import { MetricStrip } from "./metric-strip";
import { Panel, PanelBody, PanelDescription, PanelFooter, PanelHeader, PanelTitle } from "./panel";
import { Pill, pillVariants } from "./pill";
import { SectionHeading } from "./section-heading";
import { StatusDot } from "./status-dot";
import { TextureCanvas } from "./texture-canvas";
import { Toolbar, ToolbarAction, ToolbarGroup, ToolbarSearch } from "./toolbar";

const filters = [
  { active: true, label: "Foundations" },
  { active: false, label: "Panels" },
  { active: false, label: "Density" },
  { active: false, label: "Status" },
] as const;

const missionThreads = [
  {
    id: "#4238",
    meta: "Runtime choreography",
    progress: 3,
    status: "Live",
    tags: ["ACP", "Permissions", "Streams"],
    title: "Stabilize interactive approval flows across live agent sessions",
    tone: "amber",
  },
  {
    id: "#4298",
    meta: "Design system extraction",
    progress: 2,
    status: "Shaping",
    tags: ["Tokens", "Panels", "Typography"],
    title: "Turn raw shadcn surfaces into a compact AGH command language",
    tone: "violet",
  },
  {
    id: "#4316",
    meta: "Observability frame",
    progress: 4,
    status: "Stable",
    tags: ["SSE", "Metrics", "Health"],
    title: "Expose session health as dense, scannable system surfaces",
    tone: "green",
  },
] as const;

const integrations = [
  {
    description: "Daemon, CLI, and web share one visual grammar for state and action.",
    name: "Control plane",
    tone: "green",
  },
  {
    description: "Session timelines, permission prompts, and status edges stay compact.",
    name: "Runtime telemetry",
    tone: "amber",
  },
  {
    description: "Shared surface primitives now exist without forcing a full migration yet.",
    name: "Foundation layer",
    tone: "violet",
  },
] as const;

const swatches = [
  { description: "Canvas", name: "Graphite", token: "--color-canvas" },
  { description: "Surface", name: "Panel Base", token: "--color-surface" },
  { description: "Accent", name: "Amber", token: "--color-accent" },
  { description: "Signal", name: "Green", token: "--color-success" },
  { description: "Utility", name: "Violet", token: "--color-info" },
] as const;

function DesignSystemShowcase() {
  return (
    <TextureCanvas>
      <PageContent>
        <section className="animate-in fade-in slide-in-from-bottom-3 duration-700">
          <SectionHeading
            action={
              <Panel tone="accented" className="max-w-sm gap-4 p-4 sm:p-5">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="flex size-11 items-center justify-center rounded-full border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)]">
                      <Sparkles className="size-5 text-[color:var(--color-accent)]" />
                    </div>
                    <div>
                      <p className="font-mono text-[0.625rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                        First pass
                      </p>
                      <p className="text-sm text-[color:var(--color-text-secondary)]">
                        Foundations only, built to migrate forward.
                      </p>
                    </div>
                  </div>
                  <ArrowRight className="size-4 text-[color:var(--color-text-tertiary)]" />
                </div>
              </Panel>
            }
            description="A project-native surface language for AGH: graphite canvas, compact metadata, warm action cues, and dense operator panels that feel built for live system control."
            eyebrow="AGH / web design foundations v0.1"
            title="Command surfaces that feel authored, not scaffolded."
          />
        </section>

        <section className="animate-in fade-in slide-in-from-bottom-3 duration-700 delay-100">
          <Toolbar>
            <ToolbarGroup>
              {filters.map(filter => (
                <button
                  className={cn(
                    pillVariants({
                      emphasis: filter.active ? "strong" : "muted",
                      kind: "filter",
                      tone: filter.active ? "amber" : "neutral",
                    }),
                    "cursor-pointer hover:border-[color:var(--color-divider)] hover:text-[color:var(--color-text-primary)]"
                  )}
                  key={filter.label}
                  type="button"
                >
                  {filter.label}
                </button>
              ))}
            </ToolbarGroup>

            <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
              <ToolbarSearch placeholder="Search surfaces, tokens, primitives..." />
              <ToolbarAction>
                <Plus className="mr-2 size-4" />
                Build surface
              </ToolbarAction>
            </div>
          </Toolbar>
        </section>

        <section className="grid gap-5 lg:grid-cols-[1.5fr_1fr]">
          <Panel className="animate-in fade-in slide-in-from-bottom-3 duration-700 delay-150">
            <PanelHeader>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="space-y-2">
                  <p className="font-mono text-[0.64rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                    Mission threads
                  </p>
                  <PanelTitle>
                    Dense operational rows with clear hierarchy and signal color.
                  </PanelTitle>
                </div>
                <Pill emphasis="strong" kind="state" tone="green">
                  3 surfaces live
                </Pill>
              </div>
              <PanelDescription>
                The rows below turn the reference language into reusable composition patterns:
                compact metadata, rounded shells, low-noise tags, and obvious action/status cues.
              </PanelDescription>
            </PanelHeader>

            <PanelBody className="gap-3">
              {missionThreads.map(thread => (
                <article
                  className="rounded-[0.6rem] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4 transition-colors duration-200 hover:border-[color:var(--color-divider)]"
                  key={thread.id}
                >
                  <div className="flex flex-wrap items-start justify-between gap-4">
                    <div className="space-y-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <StatusDot tone={thread.tone} />
                        <p className="font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                          {thread.id} / {thread.meta}
                        </p>
                      </div>
                      <h3 className="max-w-2xl text-lg leading-7 font-medium text-[color:var(--color-text-primary)]">
                        {thread.title}
                      </h3>
                    </div>
                    <Pill emphasis="strong" kind="state" tone={thread.tone}>
                      {thread.status}
                    </Pill>
                  </div>

                  <div className="mt-4 flex flex-wrap gap-2">
                    {thread.tags.map(tag => (
                      <Pill key={tag}>{tag}</Pill>
                    ))}
                  </div>

                  <div className="mt-4 grid grid-cols-4 gap-2">
                    {Array.from({ length: 4 }, (_, index) => {
                      const active = index < thread.progress;

                      return (
                        <span
                          className={cn(
                            "h-2 rounded-full border border-transparent transition-colors",
                            active
                              ? "bg-[color:var(--color-success)]"
                              : "bg-[color:var(--color-surface-elevated)] border-[color:var(--color-divider)]"
                          )}
                          key={`${thread.id}-${index + 1}`}
                        />
                      );
                    })}
                  </div>
                </article>
              ))}
            </PanelBody>
          </Panel>

          <div className="grid gap-4">
            <Panel
              className="animate-in fade-in slide-in-from-bottom-3 duration-700 delay-200"
              tone="accented"
            >
              <PanelHeader>
                <p className="font-mono text-[0.64rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                  Core metrics
                </p>
                <PanelTitle>Foundations that can scale across future AGH views.</PanelTitle>
              </PanelHeader>

              <PanelBody>
                <MetricStrip
                  detail="Tokenized surface, line, text, accent, and depth roles are all defined."
                  label="Foundation kit"
                  tone="amber"
                  value="08"
                />
                <MetricStrip
                  detail="The first route now proves the system without forcing a broad migration."
                  label="Routes migrated"
                  tone="green"
                  value="01"
                />
              </PanelBody>
            </Panel>

            <Panel
              className="animate-in fade-in slide-in-from-bottom-3 duration-700 delay-300"
              tone="elevated"
            >
              <PanelHeader>
                <p className="font-mono text-[0.64rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                  Integration health
                </p>
                <PanelTitle>Shared primitives for the surfaces AGH uses most.</PanelTitle>
              </PanelHeader>

              <PanelBody className="gap-3">
                {integrations.map(item => (
                  <div
                    className="rounded-[0.45rem] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
                    key={item.name}
                  >
                    <div className="flex items-center gap-3">
                      <StatusDot tone={item.tone} />
                      <div>
                        <p className="text-sm font-medium text-[color:var(--color-text-primary)]">
                          {item.name}
                        </p>
                        <p className="mt-1 text-sm leading-6 text-[color:var(--color-text-secondary)]">
                          {item.description}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </PanelBody>
            </Panel>
          </div>
        </section>

        <section className="animate-in fade-in slide-in-from-bottom-3 duration-700 delay-[250ms]">
          <Panel tone="elevated">
            <PanelHeader>
              <div className="flex flex-wrap items-center justify-between gap-4">
                <div className="space-y-2">
                  <p className="font-mono text-[0.64rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                    Token preview
                  </p>
                  <PanelTitle>The route doubles as a living reference surface.</PanelTitle>
                </div>
                <Pill emphasis="strong" kind="tag" tone="violet">
                  Extracted from image references
                </Pill>
              </div>
            </PanelHeader>

            <PanelBody className="gap-5">
              <div className="grid gap-3 md:grid-cols-5">
                {swatches.map(swatch => (
                  <div
                    className="rounded-[0.55rem] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-3"
                    key={swatch.token}
                  >
                    <div
                      className="mb-3 h-20 rounded-[0.35rem] border border-[color:var(--color-divider)]"
                      style={{ background: `var(${swatch.token})` }}
                    />
                    <p className="text-sm font-medium text-[color:var(--color-text-primary)]">
                      {swatch.name}
                    </p>
                    <p className="mt-1 font-mono text-[0.58rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
                      {swatch.description}
                    </p>
                  </div>
                ))}
              </div>

              <div className="grid gap-4 lg:grid-cols-[1.15fr_0.85fr]">
                <div className="rounded-[0.6rem] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <Pill emphasis="strong" kind="filter" tone="amber">
                      Filters
                    </Pill>
                    <Pill kind="filter">Tags</Pill>
                    <Pill emphasis="strong" kind="tag" tone="green">
                      Stable
                    </Pill>
                    <Pill emphasis="strong" kind="tag" tone="violet">
                      Utility
                    </Pill>
                  </div>
                </div>

                <div className="flex items-center gap-3 rounded-[0.6rem] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
                  <div className="flex size-11 items-center justify-center rounded-full border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)]">
                    <Layers3 className="size-5 text-[color:var(--color-info)]" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-[color:var(--color-text-primary)]">
                      Foundation layer is isolated from the raw primitive inventory.
                    </p>
                    <p className="mt-1 text-sm leading-6 text-[color:var(--color-text-secondary)]">
                      New routes can migrate toward these surfaces without forcing a full library
                      rewrite.
                    </p>
                  </div>
                </div>
              </div>
            </PanelBody>

            <PanelFooter>
              <p className="font-mono text-[0.625rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                Graphite canvas / warm action cues / compact operator density
              </p>
              <div className="flex items-center gap-2 text-sm text-[color:var(--color-text-secondary)]">
                <StatusDot tone="green" />
                Ready for incremental migration
              </div>
            </PanelFooter>
          </Panel>
        </section>
      </PageContent>
    </TextureCanvas>
  );
}

export { DesignSystemShowcase };
