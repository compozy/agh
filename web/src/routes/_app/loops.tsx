import { AlertCircle, Activity, Repeat2 } from "lucide-react";
import { Link, Outlet, createFileRoute } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopCatalog } from "@/systems/loops";
import { useLoopsCatalog } from "@/hooks/routes/use-loops-catalog";

export const Route = createFileRoute("/_app/loops")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Loops", icon: Repeat2 },
  }),
  component: LoopsRoute,
});

function LoopsRoute() {
  const { hasChildMatch, workspaceId, loopsQuery, bindingIndex, filter, setFilter, handleRun } =
    useLoopsCatalog();

  if (hasChildMatch) {
    return <Outlet />;
  }
  if (workspaceId === "") {
    return (
      <CatalogState
        description="Select a workspace to browse its Loops."
        testId="loops-no-workspace"
        title="No workspace selected"
      />
    );
  }
  if (loopsQuery.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="loops-loading">
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }
  if (loopsQuery.error) {
    return (
      <CatalogState
        description={loopsQuery.error.message ?? "Failed to load loops"}
        icon={AlertCircle}
        testId="loops-error"
        title="Unable to load loops"
      />
    );
  }

  const loops = loopsQuery.data ?? [];
  if (loops.length === 0) {
    return (
      <CatalogState
        description="No Loop definitions are available in this workspace yet."
        testId="loops-empty"
        title="No loops yet"
      />
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto" data-testid="loops-catalog">
      <div className="mx-auto w-full max-w-[1320px] px-9 py-7">
        <header className="mb-4 flex items-start gap-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2.5">
              <h1 className="text-detail-h1 font-medium tracking-detail-h1 text-fg-strong">
                Loops
              </h1>
              <span className="inline-flex min-h-5 items-center rounded-xs border border-line-soft bg-canvas-soft px-1.5 font-mono text-eyebrow tabular-nums text-faint">
                {loops.length}
              </span>
            </div>
            <p className="mt-1.5 text-xs text-subtle">
              Reusable, guardrailed cycles that pursue a goal until it is verified.
            </p>
          </div>
          <Link
            to="/loop-runs"
            className="ml-auto inline-flex h-7 shrink-0 items-center gap-1.5 rounded-md border border-transparent px-2.5 text-[12.5px] font-medium text-muted transition-colors hover:bg-row-hover hover:text-fg-strong"
            data-testid="loops-runs-link"
          >
            <Activity aria-hidden="true" className="size-3.5" />
            Runs
          </Link>
        </header>
        <LoopCatalog
          entries={loops}
          filter={filter}
          onFilterChange={setFilter}
          boundLoops={bindingIndex.byLoop}
          onRun={handleRun}
        />
      </div>
    </div>
  );
}

interface CatalogStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Repeat2;
}

function CatalogState({ title, description, testId, icon = Repeat2 }: CatalogStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
