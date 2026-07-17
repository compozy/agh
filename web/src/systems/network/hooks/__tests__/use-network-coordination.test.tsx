import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

const putNetworkCoordination = vi.hoisted(() => vi.fn());

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({ activeWorkspaceId: "ws-1" }),
}));

vi.mock("../../adapters/network-coordination-api", async importOriginal => ({
  ...(await importOriginal<typeof import("../../adapters/network-coordination-api")>()),
  putNetworkCoordination,
}));

import { useAcceptNetworkCoordinationInvitation } from "../use-network-coordination";
import { networkKeys } from "../../lib/query-keys";

describe("useNetworkCoordination mutations", () => {
  it("Should preserve task and run scope then invalidate every workspace coordination view", async () => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const ref = { scope: "task" as const, taskId: "task-1", runId: "run-1" };
    const committed = {
      workspace_id: "ws-1",
      scope: "task",
      task_id: "task-1",
      enabled: true,
      revision: 8,
      eligibility: {
        eligible: false,
        reason: "already_enabled",
        run_id: "run-1",
        coordinator: true,
        worker_count: 2,
      },
    };
    putNetworkCoordination.mockResolvedValueOnce(committed);
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useAcceptNetworkCoordinationInvitation(ref), { wrapper });

    await act(async () => {
      await result.current.mutateAsync(7);
    });

    expect(putNetworkCoordination).toHaveBeenCalledWith("ws-1", {
      scope: "task",
      task_id: "task-1",
      run_id: "run-1",
      enabled: true,
      expected_revision: 7,
    });
    expect(queryClient.getQueryData(networkKeys.coordination("ws-1", ref))).toEqual(committed);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: networkKeys.coordinationRoot("ws-1"),
    });
  });
});
