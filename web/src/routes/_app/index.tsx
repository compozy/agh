import { AlertTriangle, Home, ServerOff } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty, Metric, Pill, PageHeader, Section, Skeleton } from "@agh/ui";

import { ConnectionIndicator } from "@/components/connection-indicator";
import { type HomeMetricEntry, type HomePageView, useHomePage } from "@/hooks/routes/use-home-page";

export const Route = createFileRoute("/_app/")({
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
  const headerMeta = (
    <ConnectionIndicator data-testid="home-connection-indicator" status={page.connectionStatus} />
  );

  const header = (
    <PageHeader
      data-testid="home-page-header"
      icon={Home}
      meta={headerMeta}
      title={<span data-testid="home-page-title">Home</span>}
    />
  );

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 flex-col" data-testid="home-shell">
        {header}
        <div className="flex flex-col gap-6 px-6 py-6" data-testid="home-loading">
          <DaemonStatusSkeleton />
          <MetricsSkeleton />
        </div>
      </div>
    );
  }

  if (page.hasFatalError) {
    return (
      <div className="flex min-h-0 flex-1 flex-col" data-testid="home-shell">
        {header}
        <div className="flex flex-1 items-start px-6 py-6" data-testid="home-error">
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
      {header}
      <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-6" data-testid="home-body">
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
        <div
          className="flex flex-col gap-3 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-5 py-4"
          data-testid="home-daemon-card"
          data-status={page.daemonStatus.key}
        >
          <div className="flex items-center gap-3">
            <Pill.Dot
              data-testid="home-daemon-status-dot"
              data-status={page.daemonStatus.key}
              size="md"
              tone={page.daemonStatus.tone}
            />
            <span
              className="text-[15px] font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]"
              data-testid="home-daemon-status-label"
            >
              {page.daemonStatus.label}
            </span>
          </div>
          <p
            className="text-[13px] leading-5 text-[color:var(--color-text-secondary)]"
            data-testid="home-daemon-status-description"
          >
            {page.daemonStatus.description}
          </p>
        </div>
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
      <div className="flex flex-col gap-3 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-5 py-4">
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
            className="flex flex-col gap-2 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-5 py-4"
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
