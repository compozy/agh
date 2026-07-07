import { Link } from "@tanstack/react-router";
import { Play, Repeat2 } from "lucide-react";

import { Pill } from "@agh/ui";

import type { LoopBindingKind } from "../../lib/loop-bindings";
import {
  hasHumanGate,
  isUnboundedCap,
  iterationCapLabel,
  loopCategory,
  loopInputCount,
  successRateLabel,
} from "../../lib/loop-catalog";
import type { LoopCatalogEntry } from "../../types";
import { LoopStatusPill } from "../loop-status-pill";
import { LoopBindingBadge } from "./loop-binding-badge";

interface LoopCatalogRowProps {
  entry: LoopCatalogEntry;
  bindingKinds: readonly LoopBindingKind[];
  onRun: (entry: LoopCatalogEntry) => void;
}

/**
 * One catalog list row: neutral icon well, name + kind tag + version, one-line
 * goal, meta (inputs / iteration cap / human gate / binding badge), category,
 * last-outcome pill, 30d success-rate, and an inline Run launch. The name/goal
 * area links to the detail page; Run stays a sibling control.
 */
export function LoopCatalogRow({ entry, bindingKinds, onRun }: LoopCatalogRowProps) {
  const category = loopCategory(entry);
  const inputCount = loopInputCount(entry);
  const unbounded = isUnboundedCap(entry);
  const sourceLabel = entry.source === "workspace" ? "Workspace" : "Read-only";
  return (
    <div
      className="grid grid-cols-[34px_minmax(0,1fr)_auto] items-center gap-3.5 border-b border-line-soft px-4 py-3.5 transition-colors last:border-b-0 hover:bg-row-hover"
      data-testid="loop-catalog-row"
      data-loop={entry.name}
    >
      <Link
        to="/loops/$name"
        params={{ name: entry.name }}
        className="col-span-2 grid min-w-0 grid-cols-[34px_minmax(0,1fr)] items-center gap-3.5"
        aria-label={`Open ${entry.name}`}
      >
        <span className="grid size-[34px] place-items-center rounded-md bg-elevated text-muted">
          <Repeat2 aria-hidden="true" className="size-[17px]" />
        </span>
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-2">
            <b className="truncate text-sm font-medium text-fg-strong">{entry.name}</b>
            <Pill size="xs" tone="neutral">
              {sourceLabel}
            </Pill>
            {unbounded ? (
              <Pill size="xs" tone="neutral">
                ∞ cap
              </Pill>
            ) : null}
            <span className="shrink-0 font-mono text-[11px] text-faint">v{entry.version}</span>
          </div>
          {entry.contract.goal ? (
            <p className="mt-1 truncate text-[12.5px] text-muted">{entry.contract.goal}</p>
          ) : null}
          <div className="mt-1.5 flex flex-wrap items-center gap-2 text-[11px] text-faint">
            <span className="font-mono text-[10px] text-subtle">{inputCount} inputs</span>
            <span aria-hidden="true" className="size-0.5 rounded-full bg-faint" />
            <span>iteration cap {iterationCapLabel(entry.contract.iteration_cap)}</span>
            {hasHumanGate(entry) ? (
              <>
                <span aria-hidden="true" className="size-0.5 rounded-full bg-faint" />
                <span>human gate</span>
              </>
            ) : null}
            <LoopBindingBadge kinds={bindingKinds} />
          </div>
        </div>
      </Link>
      <div className="flex items-center gap-4.5">
        {category ? (
          <span className="hidden w-24 text-right text-xs text-subtle sm:block">{category}</span>
        ) : null}
        {entry.last_run ? <LoopStatusPill status={entry.last_run.status} /> : null}
        <div className="flex w-20 flex-col items-end gap-0.5">
          <span className="font-mono text-xs font-semibold tabular-nums text-fg">
            {successRateLabel(entry.success_rate_30d)}
          </span>
          <span className="text-[10px] text-faint">{entry.aggregate_30d.runs} runs · 30d</span>
        </div>
        <button
          type="button"
          data-testid="loop-catalog-run"
          className="inline-flex h-[26px] shrink-0 items-center gap-1.5 rounded-md border border-line bg-btn-default-fill px-2.5 text-xs font-medium text-fg transition-colors hover:border-transparent hover:bg-accent hover:text-accent-ink"
          onClick={() => onRun(entry)}
        >
          <Play aria-hidden="true" className="size-3" />
          Run
        </button>
      </div>
    </div>
  );
}
