import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  useMatchRoute: () => () => false,
}));

vi.mock("@/systems/settings/adapters/settings-api", () => ({
  getSettingsRestartStatus: vi.fn(),
  getSettingsNetwork: vi.fn(),
  updateSettingsNetwork: vi.fn(),
  triggerSettingsRestart: vi.fn(),
  SettingsApiError: class SettingsApiError extends Error {
    status = 500;
  },
}));

import {
  getSettingsNetwork,
  updateSettingsNetwork,
} from "@/systems/settings/adapters/settings-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsNetworkSection } from "@/systems/settings";
import { useSettingsNetworkPage } from "../use-settings-network-page";

const networkEnvelope: SettingsNetworkSection = {
  section: "network",
  scope: "global",
  available_scopes: ["global"],
  config: {
    enabled: true,
    port: 4222,
    default_channel: "agh",
    greet_interval: 30,
    max_payload: 131072,
    max_queue_depth: 1024,
    max_replay_age: 86400,
  },
  runtime: {
    available: true,
    enabled: true,
    status: "ready",
    listener_host: "127.0.0.1",
    listener_port: 4222,
    local_peers: 1,
    remote_peers: 0,
    channels: 2,
    queued_messages: 0,
    queued_sessions: 0,
    delivery_workers: 2,
  },
  links: [{ label: "network", path: "/network" }],
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
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
  vi.mocked(getSettingsNetwork).mockResolvedValue(networkEnvelope);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSettingsNetworkPage", () => {
  it("loads the envelope and seeds the draft", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsNetworkPage(), { wrapper });

    await waitFor(() => {
      expect(result.current.envelope).toBeTruthy();
      expect(result.current.draft).toEqual(networkEnvelope.config);
    });
  });

  it("marks the page dirty when the draft diverges and resets on discard", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsNetworkPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.setDraft({ ...networkEnvelope.config, port: 5222 });
    });
    expect(result.current.isDirty).toBe(true);

    act(() => {
      result.current.handleReset();
    });
    expect(result.current.isDirty).toBe(false);
  });

  it("save persists draft via the mutation and stores the restart-required applied label", async () => {
    vi.mocked(updateSettingsNetwork).mockResolvedValue({
      section: "network",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsNetworkPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.setDraft({ ...networkEnvelope.config, port: 5222 });
      result.current.handleSave();
    });

    await waitFor(() => {
      expect(result.current.lastAppliedLabel).toContain("restart required");
    });
    expect(updateSettingsNetwork).toHaveBeenCalled();
  });
});
