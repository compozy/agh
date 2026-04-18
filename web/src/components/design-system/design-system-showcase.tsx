import { useState } from "react";
import { Layers3, Sparkles } from "lucide-react";

import {
  Button,
  Empty,
  Metric,
  PageHeader,
  Pill,
  Pills,
  SearchInput,
  Section,
  StatusDot,
  Toolbar,
} from "@agh/ui";

/**
 * Showcase surface at `/design-system`. Task 06 trimmed this to consume only `@agh/ui`
 * primitives that exist today + the legacy metric/status components that still live in
 * this folder (both scheduled for migration in tasks 07/15).
 */
const SURFACE_FILTERS = [
  { label: "Foundations", value: "foundations" },
  { label: "Panels", value: "panels" },
  { label: "Density", value: "density" },
  { label: "Status", value: "status" },
] as const;

const INTEGRATIONS = [
  {
    description: "Daemon, CLI, and web share one visual grammar for state and action.",
    name: "Control plane",
    tone: "success",
  },
  {
    description: "Session timelines, permission prompts, and status edges stay compact.",
    name: "Runtime telemetry",
    tone: "warning",
  },
  {
    description: "Shared surface primitives now live in @agh/ui without forcing a full migration.",
    name: "Foundation layer",
    tone: "info",
  },
] as const;

function DesignSystemShowcase() {
  const [filter, setFilter] = useState<(typeof SURFACE_FILTERS)[number]["value"]>("foundations");
  const [search, setSearch] = useState("");

  return (
    <main className="flex min-h-dvh flex-col gap-6 bg-[color:var(--color-canvas)] px-6 py-8">
      <PageHeader
        title="AGH design surfaces"
        icon={Sparkles}
        count="v0.2"
        meta={
          <Button size="sm" type="button" variant="outline">
            View DESIGN.md
          </Button>
        }
      />

      <Toolbar aria-label="Surface filters">
        <Pills
          value={filter}
          onChange={setFilter}
          items={SURFACE_FILTERS.map(item => ({ label: item.label, value: item.value }))}
        />
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search surfaces, tokens, primitives..."
          containerClassName="ml-auto w-64"
          aria-label="Search primitives"
        />
      </Toolbar>

      <Section label="Integration health" right={<Pill variant="success">3 surfaces live</Pill>}>
        <div className="grid gap-3 md:grid-cols-3">
          {INTEGRATIONS.map(item => (
            <article
              key={item.name}
              className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
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
            </article>
          ))}
        </div>
      </Section>

      <Section label="Foundation signal">
        <div className="grid gap-3 md:grid-cols-2">
          <Metric
            subtext="Surface, line, text, accent, and depth roles are tokenized."
            label="Foundation kit"
            tone="accent"
            value="08"
          />
          <Metric
            subtext="Routes are consuming the new primitives without coexistence hacks."
            label="Routes migrated"
            tone="success"
            value="06"
          />
        </div>
      </Section>

      <Section label="Sample empty state">
        <Empty
          icon={Layers3}
          title="Ready for the next surface"
          description="Task 07 will add Metric, MonoBadge, KindChip, StatusDot, and ConnectionIndicator to @agh/ui."
          action={
            <Button size="sm" variant="outline">
              Open task queue
            </Button>
          }
        />
      </Section>
    </main>
  );
}

export { DesignSystemShowcase };
