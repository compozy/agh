import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createMswFetch } from "@/test/msw-fetch";
import { handlers } from "@/systems/loops/mocks";
import {
  useLoop,
  useLoopAnnotations,
  useLoopConfig,
  useLoopRun,
  useLoopRuns,
  useLoops,
} from "@/systems/loops";

const WS = "ws_1";

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("loop read hooks", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      createMswFetch(() => handlers)
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("Should fetch the catalog through useLoops", async () => {
    const { result } = renderHook(() => useLoops(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.map(loop => loop.name)).toEqual([
      "software-delivery",
      "reviews-watch",
    ]);
  });

  it("Should fetch a definition, config and annotations for one loop", async () => {
    const wrapper = createWrapper();
    const detail = renderHook(() => useLoop(WS, "software-delivery"), { wrapper });
    await waitFor(() => expect(detail.result.current.isSuccess).toBe(true));
    expect(detail.result.current.data?.definition.meta.name).toBe("software-delivery");

    const config = renderHook(() => useLoopConfig(WS, "software-delivery"), { wrapper });
    await waitFor(() => expect(config.result.current.isSuccess).toBe(true));
    expect(config.result.current.data?.iteration_cap).toBe(16);

    const annotations = renderHook(() => useLoopAnnotations(WS, "software-delivery"), { wrapper });
    await waitFor(() => expect(annotations.result.current.isSuccess).toBe(true));
    expect(annotations.result.current.data).toHaveLength(2);
  });

  it("Should fetch the workspace runs list and a single run detail", async () => {
    const wrapper = createWrapper();
    const runs = renderHook(() => useLoopRuns(WS, { loop: "software-delivery" }), { wrapper });
    await waitFor(() => expect(runs.result.current.isSuccess).toBe(true));
    expect(runs.result.current.data?.runs.every(run => run.loop_name === "software-delivery")).toBe(
      true
    );

    const run = renderHook(() => useLoopRun(WS, "looprun_running"), { wrapper });
    await waitFor(() => expect(run.result.current.isSuccess).toBe(true));
    expect(run.result.current.data?.run.status).toBe("running");
  });

  it("Should stay idle when the workspace is not resolved yet", () => {
    const { result } = renderHook(() => useLoops(""), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
    expect(result.current.data).toBeUndefined();
  });
});
