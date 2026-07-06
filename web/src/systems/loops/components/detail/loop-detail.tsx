import { ArrowRight, GitFork, Play, SlidersHorizontal } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Button, DetailHeader, Pill, Section } from "@agh/ui";

import type { LoopBindingRow } from "../../lib/loop-bindings";
import type { LoopGraph } from "../../lib/loop-graph";
import type { LoopAggregate30d, LoopDetail as LoopDetailData, LoopRun } from "../../types";
import { LoopBodyDag } from "./loop-body-dag";
import { LoopContractPanel } from "./loop-contract-panel";
import { LoopDeclaredInputs } from "./loop-declared-inputs";
import { LoopLimitsPanel } from "./loop-limits-panel";
import { LoopRecentRuns } from "./loop-recent-runs";
import { LoopStartBindingsPanel } from "./loop-start-bindings-panel";
import { LoopStatsPanel } from "./loop-stats-panel";
import { LoopVersionsPanel } from "./loop-versions-panel";

interface LoopDetailProps {
  loop: LoopDetailData;
  graph: LoopGraph;
  recentRuns: readonly LoopRun[];
  bindings: readonly LoopBindingRow[];
  bindingsLoading: boolean;
  successRate: number | null;
  aggregate: LoopAggregate30d | null;
  onBack: () => void;
  onRun: () => void;
  onConfigure: () => void;
  onFork: () => void;
  onAddTrigger: () => void;
  onAddSchedule: () => void;
}

/**
 * The Loop definition page (design §4.2): detail header + Run/Configure/Fork
 * actions over a two-column layout — contract, read-only body graph, and recent
 * runs on the left; declared inputs, start bindings, limits, versions, and 30d
 * stats on the right rail.
 */
export function LoopDetailView({
  loop,
  graph,
  recentRuns,
  bindings,
  bindingsLoading,
  successRate,
  aggregate,
  onBack,
  onRun,
  onConfigure,
  onFork,
  onAddTrigger,
  onAddSchedule,
}: LoopDetailProps) {
  const definition = loop.definition;
  const category = loop.catalog?.category;
  const sourceLabel = loop.source === "workspace" ? "Workspace" : "Read-only";
  const declaredKinds = (definition.start ?? []).map(binding => binding.kind);
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto" data-testid="loop-detail">
      <DetailHeader
        back={onBack}
        backLabel="Back to Loops"
        crumbs={[{ label: "Loops", to: "/loops" }, { label: loop.name }]}
        title={loop.name}
        pills={
          <>
            <Pill size="xs" tone="neutral">
              {sourceLabel}
            </Pill>
            <Pill size="xs" tone="neutral">
              v{loop.version} · published
            </Pill>
          </>
        }
        meta={
          <>
            <span className="font-mono">{definition.apiVersion}</span>
            {category ? <span>{category}</span> : null}
            <span>{graph.nodes.length} nodes</span>
            {loop.description ? <span className="truncate">{loop.description}</span> : null}
          </>
        }
        actions={
          <>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onConfigure}
              data-testid="loop-configure-action"
            >
              <SlidersHorizontal aria-hidden="true" className="size-3.5" />
              Configure
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onFork}
              data-testid="loop-fork-action"
            >
              <GitFork aria-hidden="true" className="size-3.5" />
              Fork & edit
            </Button>
            <Button type="button" size="sm" onClick={onRun} data-testid="loop-run-action">
              <Play aria-hidden="true" className="size-3.5" />
              Run loop
            </Button>
          </>
        }
      />
      <div className="mx-auto w-full max-w-[1240px] px-9 py-6">
        <div className="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_320px]">
          <div className="flex flex-col gap-7">
            <LoopContractPanel
              contract={definition.contract}
              concurrency={definition.concurrency}
            />
            <Section
              label="Body · DAG"
              right={
                <button
                  type="button"
                  onClick={onFork}
                  className="inline-flex items-center gap-1.5 text-[11px] font-medium text-muted transition-colors hover:text-fg-strong"
                  data-testid="loop-open-builder"
                >
                  Open in builder
                  <ArrowRight aria-hidden="true" className="size-3" />
                </button>
              }
            >
              <LoopBodyDag graph={graph} />
            </Section>
            <Section
              label="Recent runs"
              right={
                <Link
                  to="/loop-runs"
                  className="inline-flex items-center gap-1.5 text-[11px] font-medium text-muted transition-colors hover:text-fg-strong"
                  data-testid="loop-all-runs"
                >
                  All runs
                  <ArrowRight aria-hidden="true" className="size-3" />
                </Link>
              }
            >
              <LoopRecentRuns runs={recentRuns} />
            </Section>
          </div>
          <aside className="flex flex-col gap-6">
            <LoopDeclaredInputs inputs={definition.inputs} />
            <LoopStartBindingsPanel
              declaredKinds={declaredKinds}
              bindings={bindings}
              isLoading={bindingsLoading}
              onAddTrigger={onAddTrigger}
              onAddSchedule={onAddSchedule}
            />
            <LoopLimitsPanel contract={definition.contract} />
            <LoopVersionsPanel version={loop.version} />
            {aggregate && successRate !== null ? (
              <LoopStatsPanel successRate={successRate} aggregate={aggregate} />
            ) : null}
          </aside>
        </div>
      </div>
    </div>
  );
}
