import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  useMatchRoute: () => () => false,
}));

vi.mock("@/systems/settings/adapters/settings-api", () => ({
  getSettingsRestartStatus: vi.fn(),
  getSettingsMemory: vi.fn(),
  updateSettingsMemory: vi.fn(),
  triggerSettingsRestart: vi.fn(),
  SettingsApiError: class SettingsApiError extends Error {
    status = 500;
  },
}));

vi.mock("@/systems/knowledge", () => ({
  useConsolidateMemory: () => consolidateMock,
}));

const consolidateMock = {
  mutate: vi.fn(),
  isPending: false,
};

import { getSettingsMemory, updateSettingsMemory } from "@/systems/settings/adapters/settings-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsMemorySection } from "@/systems/settings";
import { useSettingsMemoryPage } from "./use-settings-memory-page";

const memoryEnvelope: SettingsMemorySection = {
  section: "memory",
  scope: "global",
  available_scopes: ["global"],
  actions: {
    consolidate: { available: true, behavior: "action_trigger", name: "consolidate" },
  },
  config: {
    dream: {
      agent: "general",
      check_interval: "30m",
      enabled: true,
      min_hours: 24,
      min_sessions: 3,
    },
    enabled: true,
    global_dir: "~/.agh/memory",
  },
  health: {
    available: true,
    dream_enabled: true,
    file_count: 12,
    last_consolidated_at: "2026-04-17T10:00:00Z",
  },
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);

  return { queryClient, wrapper };
}

beforeEach(() => {
  vi.clearAllMocks();
  consolidateMock.mutate.mockReset();
  consolidateMock.isPending = false;
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
  vi.mocked(getSettingsMemory).mockResolvedValue(memoryEnvelope);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSettingsMemoryPage", () => {
  it("loads the envelope and seeds the draft with the current config", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMemoryPage(), { wrapper });

    await waitFor(() => {
      expect(result.current.envelope).toBeTruthy();
      expect(result.current.draft).toEqual(memoryEnvelope.config);
    });
  });

  it("marks the page dirty when draft diverges from the envelope and resets on discard", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMemoryPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.setDraft({
        ...memoryEnvelope.config,
        enabled: false,
      });
    });

    expect(result.current.isDirty).toBe(true);

    act(() => {
      result.current.handleReset();
    });

    expect(result.current.isDirty).toBe(false);
  });

  it("save persists draft via the mutation and stores the applied label", async () => {
    vi.mocked(updateSettingsMemory).mockResolvedValue({
      section: "memory",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMemoryPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.setDraft({
        ...memoryEnvelope.config,
        enabled: false,
      });
      result.current.handleSave();
    });

    await waitFor(() => {
      expect(result.current.lastAppliedLabel).toContain("restart required");
    });
    expect(updateSettingsMemory).toHaveBeenCalled();
  });

  it("exposes handleConsolidate that delegates to the knowledge consolidate mutation", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMemoryPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.handleConsolidate();
    });

    expect(consolidateMock.mutate).toHaveBeenCalled();
  });
});
