import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/session/$id")({
  component: SessionPage,
});

function SessionPage() {
  const { id } = Route.useParams();

  return (
    <div className="flex flex-1 flex-col p-6">
      <div className="flex items-center gap-3">
        <h1 className="font-display text-lg font-semibold text-[color:var(--ds-text-primary)]">
          Session
        </h1>
        <span className="font-mono text-[0.64rem] uppercase tracking-[0.2em] text-[color:var(--ds-text-mono)]">
          {id}
        </span>
      </div>
      <div className="mt-6 flex flex-1 items-center justify-center text-sm text-[color:var(--ds-text-muted)]">
        Chat view will be implemented here.
      </div>
    </div>
  );
}
