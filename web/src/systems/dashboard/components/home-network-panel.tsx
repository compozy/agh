import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Button, Panel, Pill, Section } from "@agh/ui";

export interface HomeNetworkRow {
  key: string;
  label: string;
  value: string;
  tone?: "success" | "warning";
  mono?: boolean;
}

export interface HomeNetworkBudget {
  used: number;
  limit: number;
  resetNote: string | null;
}

export interface HomeNetworkPanelProps {
  rows: HomeNetworkRow[];
  budget: HomeNetworkBudget | null;
}

/**
 * Zone 3b — network coordination at a glance. Rows render only for counters
 * the daemon actually reports; the wake budget meter appears when a workspace
 * budget exists.
 */
export function HomeNetworkPanel({ rows, budget }: HomeNetworkPanelProps) {
  const budgetShare =
    budget && budget.limit > 0 ? Math.min(1, Math.max(0, budget.used / budget.limit)) : 0;

  return (
    <Section
      bodyClassName="flex flex-1 flex-col"
      className="flex min-h-full flex-col"
      label="Network"
    >
      <Panel
        bodyClassName="flex flex-1 flex-col p-0"
        className="flex-1"
        foot={
          <Button render={<Link to="/network" />} size="sm" variant="ghost">
            View network
            <ChevronRight aria-hidden="true" />
          </Button>
        }
      >
        <div className="flex flex-1 flex-col divide-y divide-line-soft">
          {rows.length === 0 ? (
            <p className="px-4 py-3.5 text-small-body text-subtle">Network coordination is idle.</p>
          ) : (
            rows.map(row => (
              <div
                className="flex min-h-9 items-center justify-between gap-3 px-4 py-2"
                data-slot="home-network-row"
                key={row.key}
              >
                <span className="text-small-body text-muted">{row.label}</span>
                <span className="flex items-center gap-2 text-small-body font-medium text-fg">
                  {row.tone ? <Pill.Dot tone={row.tone} /> : null}
                  <span
                    className={
                      row.mono ? "font-mono text-small-body tabular-nums text-subtle" : undefined
                    }
                  >
                    {row.value}
                  </span>
                </span>
              </div>
            ))
          )}
          {budget ? (
            <div className="px-4 py-3" data-slot="home-network-budget">
              <div className="mb-2 flex items-center justify-between gap-3 text-small-body text-muted">
                <span>
                  Wakes this week
                  {budget.resetNote ? (
                    <span className="text-faint"> · {budget.resetNote}</span>
                  ) : null}
                </span>
                <span className="font-mono text-mono-id tabular-nums text-subtle">
                  {budget.used} / {budget.limit}
                </span>
              </div>
              <div
                aria-label={`${budget.used} of ${budget.limit} network wakes used`}
                className="h-1.5 overflow-hidden rounded-pill bg-canvas-tint"
                role="img"
              >
                <div
                  className="h-full rounded-pill bg-viz-bar"
                  style={{ width: `${budgetShare * 100}%` }}
                />
              </div>
            </div>
          ) : null}
        </div>
      </Panel>
    </Section>
  );
}
