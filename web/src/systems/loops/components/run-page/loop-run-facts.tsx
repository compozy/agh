import { Eyebrow } from "@agh/ui";

import type { LoopRunRecord } from "../../types";

interface LoopRunFactsProps {
  run: LoopRunRecord;
  loopVersion?: number;
  concurrency?: string;
}

interface Fact {
  label: string;
  value: string;
  mono?: boolean;
}

function reattemptLabel(strategy: string): string {
  return strategy === "full_body" ? "full-body" : "failed-only";
}

/** The run's start descriptor (`schedule` / `cli · operator`), truthful to the projection. */
function startLabel(run: LoopRunRecord): string {
  const origin = run.started_origin_kind;
  const by = run.started_by_ref;
  if (origin && by) return `${origin} · ${by}`;
  return origin ?? by ?? "manual";
}

/**
 * The "Run facts" rail (§4.4): the durable read-only descriptors of this run — loop,
 * executed loop version, re-attempt mode, start source, workspace, run id, and
 * concurrency policy. All read from daemon truth; version and concurrency come from the
 * executed definition pinned by the run detail when available.
 */
export function LoopRunFacts({ run, loopVersion, concurrency }: LoopRunFactsProps) {
  const facts: Fact[] = [
    { label: "Loop", value: run.loop_name },
    ...(loopVersion !== undefined
      ? [{ label: "Loop version", value: `v${loopVersion}`, mono: true }]
      : []),
    { label: "Re-attempt", value: reattemptLabel(run.reattempt_strategy) },
    { label: "Start", value: startLabel(run), mono: true },
    ...(concurrency ? [{ label: "Concurrency", value: concurrency, mono: true }] : []),
    { label: "Run ID", value: run.id, mono: true },
    { label: "Workspace", value: run.workspace_id, mono: true },
  ];
  return (
    <div className="border-b border-line-soft px-4 py-4" data-testid="loop-run-facts">
      <Eyebrow className="mb-3 text-faint">Run facts</Eyebrow>
      <div className="flex flex-col">
        {facts.map(fact => (
          <div
            key={fact.label}
            className="flex items-baseline justify-between gap-3 border-t border-line-soft py-1.5 text-[12px] first:border-t-0 first:pt-0"
          >
            <span className="text-subtle">{fact.label}</span>
            <span
              className={`min-w-0 truncate text-right font-medium text-fg ${fact.mono ? "font-mono text-[11px]" : ""}`}
              title={fact.value}
            >
              {fact.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
