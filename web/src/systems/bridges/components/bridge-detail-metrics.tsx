import { Metric, Section } from "@agh/ui";

import { formatBridgeRelativeTime } from "../lib/bridge-formatters";
import type { BridgeHealth, BridgeRoute } from "../types";

interface BridgeMetrics {
  activeRoutes: string;
  eventsTotal: string;
  lastDelivery: string;
  successRate: string;
  successTone: "default" | "accent" | "success" | "warning" | "danger";
}

function computeBridgeMetrics(
  health: BridgeHealth | undefined,
  routes: BridgeRoute[]
): BridgeMetrics {
  const backlog = health?.delivery_backlog ?? 0;
  const failures = health?.delivery_failures_total ?? 0;
  const dropped = health?.delivery_dropped_total ?? 0;
  const active = health?.route_count ?? routes.length;
  const total = backlog + failures + dropped + active;
  let successRate = "--";
  let successTone: BridgeMetrics["successTone"] = "default";

  if (total > 0) {
    const percentage = (active / total) * 100;
    successRate = `${Math.round(percentage)}%`;
    successTone = percentage >= 90 ? "success" : percentage >= 70 ? "default" : "warning";
  }

  return {
    activeRoutes: String(active),
    eventsTotal: String(total),
    lastDelivery: formatBridgeRelativeTime(health?.last_success_at),
    successRate,
    successTone,
  };
}

export function BridgeDetailMetrics({
  health,
  routes,
}: {
  health: BridgeHealth | undefined;
  routes: BridgeRoute[];
}) {
  const metrics = computeBridgeMetrics(health, routes);

  return (
    <Section label="Delivery metrics">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <Metric
          data-testid="bridge-metric-events-24h"
          label="Events (24h)"
          subtext="backlog + failures + active"
          value={metrics.eventsTotal}
        />
        <Metric
          data-testid="bridge-metric-success-rate"
          label="Success rate"
          subtext="active vs. backlog"
          tone={metrics.successTone}
          value={metrics.successRate}
        />
        <Metric
          data-testid="bridge-metric-last-delivery"
          label="Last delivery"
          subtext="most recent success"
          value={metrics.lastDelivery}
        />
        <Metric
          data-testid="bridge-metric-active-routes"
          label="Active routes"
          subtext="sessions mapped"
          tone="accent"
          value={metrics.activeRoutes}
        />
      </div>
    </Section>
  );
}
