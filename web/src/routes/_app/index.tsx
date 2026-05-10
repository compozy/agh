import { AlertTriangle, Home, ServerOff } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  ConnectionIndicator,
  Empty,
  Metric,
  Pill,
  Section,
  Skeleton,
  StatusCard,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { type HomeMetricEntry, type HomePageView, useHomePage } from "@/hooks/routes/use-home-page";

export const Route = createFileRoute("/_app/")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Home", icon: Home },
  }),
  component: AppHomePage,
});

const METRIC_ORDER: HomeMetricEntry["key"][] = [
  "active-sessions",
  "workspaces",
  "agents",
  "uptime",
];

function AppHomePage() {
  const page = useHomePage();
  useTopbarSlot({
    actions: (
      <ConnectionIndicator data-testid="home-connection-indicator" status={page.connectionStatus} />
    ),
  });

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 flex-col" data-testid="home-shell">
        <div className="flex flex-col gap-6 p-6" data-testid="home-loading">
          <DaemonStatusSkeleton />
          <MetricsSkeleton />
        </div>
      </div>
    );
  }

  if (page.hasFatalError) {
    return (
      <div className="flex min-h-0 flex-1 flex-col" data-testid="home-shell">
        <div className="flex flex-1 items-start p-6" data-testid="home-error">
          <Empty
            className="max-w-xl"
            description={page.errorMessage ?? "Unable to load workspace data from the daemon."}
            icon={AlertTriangle}
            title="Unable to load dashboard"
          />
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="home-shell">
      <div className="flex flex-1 flex-col gap-6 overflow-y-auto p-6" data-testid="home-body">
        <DaemonStatusSection page={page} />
        <OverviewSection page={page} />
      </div>
    </div>
  );
}

function DaemonStatusSection({ page }: { page: HomePageView }) {
  const isDisconnected = page.connectionStatus === "disconnected";

  return (
    <Section
      data-testid="home-section-daemon"
      label="Daemon"
      right={
        page.daemonVersion ? (
          <Pill mono data-testid="home-daemon-version" tone="neutral">
            v{page.daemonVersion}
          </Pill>
        ) : null
      }
    >
      {isDisconnected ? (
        <DisconnectedCard description={page.daemonStatus.description} />
      ) : (
        <StatusCard
          data-testid="home-daemon-card"
          data-status={page.daemonStatus.key}
          tone={page.daemonStatus.tone}
        >
          <StatusCard.Header
            dotProps={{
              "data-testid": "home-daemon-status-dot",
              "data-status": page.daemonStatus.key,
            }}
            label={page.daemonStatus.label}
            labelProps={{ "data-testid": "home-daemon-status-label" }}
          />
          <StatusCard.Body data-testid="home-daemon-status-description">
            {page.daemonStatus.description}
          </StatusCard.Body>
        </StatusCard>
      )}
    </Section>
  );
}

function DisconnectedCard({ description }: { description: string }) {
  return (
    <Empty
      className="max-w-xl"
      data-testid="home-daemon-disconnected"
      description={description}
      icon={ServerOff}
      title={
        <ConnectionIndicator
          data-testid="home-daemon-disconnected-indicator"
          status="disconnected"
        />
      }
    />
  );
}

function OverviewSection({ page }: { page: HomePageView }) {
  const metricsByKey = new Map(page.metrics.map(metric => [metric.key, metric] as const));

  return (
    <Section data-testid="home-section-overview" label="Overview">
      <div
        className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4"
        data-testid="home-metric-grid"
      >
        {METRIC_ORDER.map(key => {
          const metric = metricsByKey.get(key);
          if (!metric) {
            return null;
          }
          return (
            <Metric
              data-testid={`home-metric-${metric.key}`}
              detail={metric.detail}
              key={metric.key}
              label={metric.label}
              value={metric.value}
            />
          );
        })}
      </div>
    </Section>
  );
}

function DaemonStatusSkeleton() {
  return (
    <div className="flex flex-col gap-3" data-testid="home-daemon-skeleton">
      <Skeleton className="h-3 w-24" />
      <div className="flex flex-col gap-3 rounded-(--radius-diagram) border border-(--line) bg-(--canvas-soft) px-5 py-4">
        <div className="flex items-center gap-3">
          <Skeleton className="size-2 rounded-full" />
          <Skeleton className="h-4 w-32" />
        </div>
        <Skeleton className="h-3 w-full max-w-md" />
      </div>
    </div>
  );
}

function MetricsSkeleton() {
  return (
    <div className="flex flex-col gap-3" data-testid="home-metric-skeleton">
      <Skeleton className="h-3 w-24" />
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {METRIC_ORDER.map(key => (
          <div
            className="flex flex-col gap-2 rounded-(--radius-diagram) border border-(--line) bg-(--canvas-soft) px-5 py-4"
            data-testid={`home-metric-skeleton-${key}`}
            key={key}
          >
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-7 w-24" />
          </div>
        ))}
      </div>
    </div>
  );
}
