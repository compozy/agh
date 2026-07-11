import { Activity, AlertCircle } from "lucide-react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import {
  Empty,
  ListingPage,
  ListingToolbar,
  NativeSelect,
  NativeSelectOption,
  Spinner,
} from "@agh/ui";
import { useLoopRunsRoute, type LoopRunsRouteSearch } from "@/hooks/routes/use-loop-runs-route";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopRunsView } from "@/systems/loops";
import { preloadLoopRunsRoute } from "./-loops-preload";

function validateLoopRunsSearch(search: Record<string, unknown>): LoopRunsRouteSearch {
  const origin =
    search.origin === "catalog" || search.origin === "session" ? search.origin : undefined;
  const originSession =
    typeof search.origin_session === "string" ? search.origin_session.trim() : "";
  return { origin, origin_session: originSession || undefined };
}

export const Route = createFileRoute("/_app/loop-runs")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Runs", icon: Activity },
  }),
  loaderDeps: ({ search }) => ({
    origin: search.origin,
    origin_session: search.origin_session,
  }),
  loader: ({ context, deps, location }) =>
    location.pathname.split("/").filter(Boolean).length === 1
      ? preloadLoopRunsRoute(context.queryClient, deps)
      : Promise.resolve(),
  validateSearch: validateLoopRunsSearch,
  component: LoopRunsRoute,
});

function LoopRunsRoute() {
  const search = Route.useSearch();
  const {
    hasChildMatch,
    outcome,
    runsQuery,
    setOrigin,
    setOriginSession,
    setOutcome,
    workspaceId,
    workspaceLabel,
  } = useLoopRunsRoute(search);

  if (hasChildMatch) {
    return <Outlet />;
  }

  if (workspaceId === "") {
    return (
      <RunsState
        description="Select a workspace to view its Loop runs."
        testId="loop-runs-no-workspace"
        title="No workspace selected"
      />
    );
  }

  if (runsQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-runs-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (runsQuery.error) {
    return (
      <RunsState
        description={runsQuery.error.message ?? "Failed to load loop runs"}
        icon={AlertCircle}
        testId="loop-runs-error"
        title="Unable to load runs"
      />
    );
  }

  const runs = runsQuery.data?.runs ?? [];

  if (runs.length === 0 && !search.origin && !search.origin_session) {
    return (
      <RunsState
        description="No Loop has run in this workspace yet."
        testId="loop-runs-empty"
        title="No runs yet"
      />
    );
  }

  return (
    <ListingPage data-testid="loop-runs">
      <ListingPage.Head
        count={runs.length}
        countTestId="loop-runs-page-count"
        meta={
          <>
            <span>Every execution of a Loop, across the full outcome spectrum.</span>
            <ListingPage.MetaDot />
            <span>{workspaceLabel}</span>
          </>
        }
        title="Runs"
      />
      <ListingToolbar data-testid="loop-runs-origin-toolbar">
        <ListingToolbar.Leading>
          <NativeSelect
            aria-label="Run origin"
            data-testid="loop-runs-origin-filter"
            value={search.origin ?? "all"}
            onChange={event =>
              setOrigin(
                event.target.value === "all"
                  ? undefined
                  : (event.target.value as "catalog" | "session")
              )
            }
          >
            <NativeSelectOption value="all">All origins</NativeSelectOption>
            <NativeSelectOption value="catalog">Catalog</NativeSelectOption>
            <NativeSelectOption value="session">Session</NativeSelectOption>
          </NativeSelect>
          {search.origin === "session" ? (
            <input
              aria-label="Origin session id"
              className="h-8 min-w-search-input rounded-md border border-line bg-input-fill px-2.5 font-mono text-small-body text-fg outline-none placeholder:text-faint focus:border-line-strong"
              data-testid="loop-runs-origin-session-filter"
              onChange={event => setOriginSession(event.target.value.trim())}
              placeholder="Session id"
              value={search.origin_session ?? ""}
            />
          ) : null}
        </ListingToolbar.Leading>
      </ListingToolbar>
      {runs.length === 0 ? (
        <Empty
          className="mx-auto my-16 max-w-md"
          description="Adjust the origin filters to include more runs."
          icon={Activity}
          title="No matching runs"
        />
      ) : (
        <LoopRunsView onOutcomeChange={setOutcome} outcome={outcome} runs={runs} />
      )}
    </ListingPage>
  );
}

interface RunsStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Activity;
}

function RunsState({ title, description, testId, icon = Activity }: RunsStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
