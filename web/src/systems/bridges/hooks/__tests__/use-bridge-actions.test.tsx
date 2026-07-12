import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { resolveBridgeTarget } from "../../adapters/bridges-api";
import { bridgeKeys } from "../../lib/query-keys";
import { useResolveBridgeTarget } from "../use-bridge-actions";

vi.mock("../../adapters/bridges-api", async importOriginal => {
  const original = await importOriginal<typeof import("../../adapters/bridges-api")>();
  return { ...original, resolveBridgeTarget: vi.fn() };
});

describe("useResolveBridgeTarget", () => {
  beforeEach(() => {
    vi.mocked(resolveBridgeTarget).mockReset();
  });

  it("Should invalidate every cached target list for the resolved bridge", async () => {
    vi.mocked(resolveBridgeTarget).mockResolvedValue({
      result: { ambiguous: false, match: null, step: 1 },
    });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useResolveBridgeTarget(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync({ id: "brg_support", data: { name: "launch" } });
    });

    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: bridgeKeys.targetsForBridge("brg_support"),
    });
  });

  it("Should expose resolution errors without invalidating unchanged target lists", async () => {
    const resolutionError = new Error("target resolution failed");
    vi.mocked(resolveBridgeTarget).mockRejectedValue(resolutionError);
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useResolveBridgeTarget(), { wrapper });

    await act(async () => {
      await expect(
        result.current.mutateAsync({ id: "brg_support", data: { name: "launch" } })
      ).rejects.toBe(resolutionError);
    });

    await waitFor(() => {
      expect(result.current.error).toBe(resolutionError);
    });
    expect(invalidateQueries).not.toHaveBeenCalled();
  });
});
