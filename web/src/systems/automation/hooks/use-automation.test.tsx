import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  useAutomationJob,
  useAutomationJobs,
  useAutomationJobRuns,
  useAutomationRuns,
  useAutomationTrigger,
  useAutomationTriggers,
  useAutomationTriggerRuns,
} from "./use-automation";

vi.mock("../adapters/automation-api", () => ({
  getAutomationJob: vi.fn(),
  listAutomationJobRuns: vi.fn(),
  listAutomationJobs: vi.fn(),
  listAutomationRuns: vi.fn(),
  getAutomationTrigger: vi.fn(),
  listAutomationTriggerRuns: vi.fn(),
  listAutomationTriggers: vi.fn(),
}));

import {
  getAutomationJob,
  getAutomationTrigger,
  listAutomationJobRuns,
  listAutomationJobs,
  listAutomationRuns,
  listAutomationTriggerRuns,
  listAutomationTriggers,
} from "../adapters/automation-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const jobFixture = {
  id: "job_daily_review",
  name: "daily-review",
  agent_name: "reviewer",
  prompt: "Review recent changes.",
};

const triggerFixture = {
  id: "trg_push_review",
  name: "push-review",
  agent_name: "reviewer",
  prompt: "Review push event {{ .Data.branch }}.",
  event: "ext.github.push",
};

const runFixture = {
  id: "run_001",
  attempt: 1,
  status: "running",
};

describe("useAutomation query hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads jobs with the provided filters", async () => {
    vi.mocked(listAutomationJobs).mockResolvedValue([jobFixture] as never);

    const { result } = renderHook(
      () => useAutomationJobs({ scope: "workspace", workspace_id: "ws_alpha" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listAutomationJobs).toHaveBeenCalledWith(
      { scope: "workspace", workspace_id: "ws_alpha" },
      expect.any(AbortSignal)
    );
  });

  it("loads a job detail and respects empty ids", async () => {
    vi.mocked(getAutomationJob).mockResolvedValue(jobFixture as never);

    const { result } = renderHook(() => useAutomationJob("job_daily_review"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.id).toBe("job_daily_review");
    });

    expect(getAutomationJob).toHaveBeenCalledWith("job_daily_review", expect.any(AbortSignal));

    renderHook(() => useAutomationJob(""), { wrapper: createWrapper() });
    expect(getAutomationJob).toHaveBeenCalledTimes(1);
  });

  it("loads job run history and respects explicit disable flags", async () => {
    vi.mocked(listAutomationJobRuns).mockResolvedValue([runFixture] as never);

    const { result } = renderHook(
      () => useAutomationJobRuns("job_daily_review", { status: "running" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listAutomationJobRuns).toHaveBeenCalledWith(
      "job_daily_review",
      { status: "running" },
      expect.any(AbortSignal)
    );

    renderHook(() => useAutomationJobRuns("job_daily_review", {}, { enabled: false }), {
      wrapper: createWrapper(),
    });
    expect(listAutomationJobRuns).toHaveBeenCalledTimes(1);
  });

  it("loads trigger lists, details, and trigger run history", async () => {
    vi.mocked(listAutomationTriggers).mockResolvedValue([triggerFixture] as never);
    vi.mocked(getAutomationTrigger).mockResolvedValue(triggerFixture as never);
    vi.mocked(listAutomationTriggerRuns).mockResolvedValue([runFixture] as never);

    const triggers = renderHook(() => useAutomationTriggers({ event: "ext.github.push" }), {
      wrapper: createWrapper(),
    });
    const trigger = renderHook(() => useAutomationTrigger("trg_push_review"), {
      wrapper: createWrapper(),
    });
    const triggerRuns = renderHook(
      () => useAutomationTriggerRuns("trg_push_review", { limit: 5 }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(triggers.result.current.data).toHaveLength(1);
      expect(trigger.result.current.data?.id).toBe("trg_push_review");
      expect(triggerRuns.result.current.data).toHaveLength(1);
    });

    expect(listAutomationTriggers).toHaveBeenCalledWith(
      { event: "ext.github.push" },
      expect.any(AbortSignal)
    );
    expect(getAutomationTrigger).toHaveBeenCalledWith("trg_push_review", expect.any(AbortSignal));
    expect(listAutomationTriggerRuns).toHaveBeenCalledWith(
      "trg_push_review",
      { limit: 5 },
      expect.any(AbortSignal)
    );
  });

  it("loads run lists and respects disabled state", async () => {
    vi.mocked(listAutomationRuns).mockResolvedValue([runFixture] as never);

    const { result } = renderHook(
      () => useAutomationRuns({ job_id: "job_daily_review", status: "running" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listAutomationRuns).toHaveBeenCalledWith(
      { job_id: "job_daily_review", status: "running" },
      expect.any(AbortSignal)
    );

    renderHook(() => useAutomationRuns({}, { enabled: false }), {
      wrapper: createWrapper(),
    });
    expect(listAutomationRuns).toHaveBeenCalledTimes(1);
  });
});
