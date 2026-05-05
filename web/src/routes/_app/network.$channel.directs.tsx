import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/network/$channel/directs")({
  component: NetworkChannelDirectsRoute,
});

function NetworkChannelDirectsRoute() {
  const { channel } = Route.useParams();

  return (
    <section
      aria-label={`Direct rooms in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-directs-tab"
    >
      <div
        aria-live="polite"
        className="flex min-h-40 items-center justify-center px-6 text-center"
      >
        <p className="font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
          Direct rooms list renders in task_14
        </p>
      </div>
    </section>
  );
}
