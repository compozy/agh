import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/network/$channel/directs/$directId")({
  component: NetworkChannelDirectDetailRoute,
});

function NetworkChannelDirectDetailRoute() {
  const { channel, directId } = Route.useParams();

  return (
    <section
      aria-label={`Direct room ${directId} in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-direct-detail"
    >
      <div className="flex min-h-40 items-center justify-center px-6 text-center">
        <p className="font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
          Direct room detail renders in task_14
        </p>
      </div>
    </section>
  );
}
