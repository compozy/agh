import { useChildMatches, useNavigate, useRouter } from "@tanstack/react-router";

import {
  LoopsApiError,
  readLoopGraph,
  useCreateLoop,
  useLoop,
  useLoopRuns,
  useLoops,
} from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

import { useLoopBindings } from "./use-loop-bindings";

const RECENT_RUNS_LIMIT = 5;

/** View-model for the Loop detail route: definition, 30d aggregate, recent runs, bindings, nav. */
export function useLoopDetail(name: string) {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const navigate = useNavigate();
  const router = useRouter();
  const active = workspaceId !== "" && !hasChildMatch;

  const loopQuery = useLoop(workspaceId, name, active);
  const catalogQuery = useLoops(workspaceId, active);
  const runsQuery = useLoopRuns(workspaceId, { loop: name, limit: RECENT_RUNS_LIMIT }, active);
  const createLoop = useCreateLoop();
  const bindings = useLoopBindings(active ? workspaceId : "", active ? name : "");

  const catalogEntry = catalogQuery.data?.find(entry => entry.name === name) ?? null;

  const handlers = {
    onBack: () => {
      if (router.history.canGoBack()) {
        router.history.back();
        return;
      }
      void navigate({ to: "/loops" });
    },
    onRun: () => void navigate({ to: "/loops/$name/run", params: { name } }),
    onConfigure: () => void navigate({ to: "/loops/$name/configure", params: { name } }),
    onFork: async () => {
      if (workspaceId === "" || createLoop.isPending) return;
      if (loopQuery.data?.source !== "workspace") {
        try {
          await createLoop.mutateAsync({ workspaceId, data: { fork_from_name: name } });
        } catch (error) {
          if (!(error instanceof LoopsApiError && error.status === 409)) {
            return;
          }
        }
      }
      void navigate({ to: "/loops/$name/editor", params: { name } });
    },
    onAddTrigger: () => void navigate({ to: "/triggers", search: { create: "loop", loop: name } }),
    onAddSchedule: () => void navigate({ to: "/jobs", search: { create: "loop", loop: name } }),
  };

  return {
    hasChildMatch,
    workspaceId,
    loopQuery,
    catalogEntry,
    runsQuery,
    bindings,
    readGraph: readLoopGraph,
    handlers,
  };
}
