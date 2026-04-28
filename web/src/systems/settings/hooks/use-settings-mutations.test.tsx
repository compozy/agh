import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../adapters/settings-api", () => ({
  deleteSettingsSandbox: vi.fn(),
  deleteSettingsHook: vi.fn(),
  deleteSettingsMCPServer: vi.fn(),
  deleteSettingsProvider: vi.fn(),
  disableSettingsExtension: vi.fn(),
  enableSettingsExtension: vi.fn(),
  putSettingsSandbox: vi.fn(),
  putSettingsHook: vi.fn(),
  putSettingsMCPServer: vi.fn(),
  putSettingsProvider: vi.fn(),
  updateSettingsAutomation: vi.fn(),
  updateSettingsGeneral: vi.fn(),
  updateSettingsHooksExtensions: vi.fn(),
  updateSettingsMemory: vi.fn(),
  updateSettingsNetwork: vi.fn(),
  updateSettingsObservability: vi.fn(),
  updateSettingsSkills: vi.fn(),
}));

import {
  deleteSettingsMCPServer,
  deleteSettingsProvider,
  disableSettingsExtension,
  enableSettingsExtension,
  putSettingsMCPServer,
  updateSettingsGeneral,
  updateSettingsMemory,
} from "../adapters/settings-api";
import { settingsKeys } from "../lib/query-keys";
import { initialSettingsRestartState } from "../stores/settings-restart-store";
import { useSettingsRestartStore } from "../stores/use-settings-restart-store";
import {
  useDeleteSettingsMCPServer,
  useDeleteSettingsProvider,
  useDisableSettingsExtension,
  useEnableSettingsExtension,
  usePutSettingsMCPServer,
  useUpdateSettingsGeneral,
  useUpdateSettingsMemory,
} from "./use-settings-mutations";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });

  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);

  return { queryClient, wrapper };
}

const generalMutation = {
  section: "general" as const,
  scope: "global" as const,
  behavior: "restart_required" as const,
  applied: true,
  restart_required: true,
  restart_scope: "daemon",
  warnings: ["restart the daemon"],
  write_target: "global-config" as const,
};

beforeEach(() => {
  vi.clearAllMocks();
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useUpdateSettingsGeneral", () => {
  it("records mutation state and invalidates only the general section", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(updateSettingsGeneral).mockResolvedValue(generalMutation);

    const { result } = renderHook(() => useUpdateSettingsGeneral(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync({
        config: {
          daemon: { socket: "/tmp/a.sock" },
          defaults: { agent: "claude-code" },
          http: { host: "127.0.0.1", port: 2123 },
          limits: { max_concurrent_agents: 4, max_sessions: 16 },
          permissions: { mode: "approve-reads" as const },
          session_timeout: "30m",
        },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.section("general"),
      });
    });

    expect(useSettingsRestartStore.getState().lastMutation?.restartRequired).toBe(true);
    expect(useSettingsRestartStore.getState().lastMutation?.warnings).toEqual([
      "restart the daemon",
    ]);

    const memoryInvalidations = invalidateSpy.mock.calls.filter(([arg]) =>
      JSON.stringify(arg?.queryKey).includes("memory")
    );
    expect(memoryInvalidations).toHaveLength(0);
  });
});

describe("useUpdateSettingsMemory", () => {
  it("invalidates only memory section queries", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(updateSettingsMemory).mockResolvedValue({
      ...generalMutation,
      section: "memory" as const,
    });

    const { result } = renderHook(() => useUpdateSettingsMemory(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync({
        config: {
          dream: {
            agent: "dreamer",
            check_interval: "30m",
            enabled: true,
            min_hours: 1,
            min_sessions: 2,
          },
          enabled: true,
        },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.section("memory") });
    });
  });
});

describe("provider mutations", () => {
  it("invalidates provider detail and list on delete", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(deleteSettingsProvider).mockResolvedValue({
      ...generalMutation,
      section: "general" as const,
    });

    const { result } = renderHook(() => useDeleteSettingsProvider(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync("openai");
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.providersRoot() });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.providerDetail("openai"),
      });
    });
  });
});

describe("mcp server mutations", () => {
  it("invalidates the entire mcp-server root on put", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      ...generalMutation,
      section: "general" as const,
    });

    const { result } = renderHook(() => usePutSettingsMCPServer(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync({
        name: "github",
        body: { server: { name: "github", command: "gh" } },
        filter: { scope: "global", target: "sidecar" },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.mcpRoot() });
    });

    expect(putSettingsMCPServer).toHaveBeenCalledWith(
      "github",
      { server: { name: "github", command: "gh" } },
      { scope: "global", target: "sidecar" }
    );
  });

  it("forwards scope and target filters on delete", async () => {
    const { wrapper } = createWrapper();
    vi.mocked(deleteSettingsMCPServer).mockResolvedValue({
      ...generalMutation,
      section: "general" as const,
    });

    const { result } = renderHook(() => useDeleteSettingsMCPServer(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync({
        name: "github",
        filter: { scope: "workspace", workspace_id: "ws_alpha", target: "auto" },
      });
    });

    expect(deleteSettingsMCPServer).toHaveBeenCalledWith("github", {
      scope: "workspace",
      workspace_id: "ws_alpha",
      target: "auto",
    });
  });
});

describe("extension action mutations", () => {
  const extension = {
    name: "daytona",
    enabled: true,
    version: "1.2.3",
    state: "running",
    source: "marketplace",
    type: "backend",
    daemon_running: true,
  };

  it("enables an extension and invalidates extension + section caches", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(enableSettingsExtension).mockResolvedValue(extension);

    const { result } = renderHook(() => useEnableSettingsExtension(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync("daytona");
    });

    expect(enableSettingsExtension).toHaveBeenCalledWith("daytona");

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.extensionsRoot(),
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.section("hooks-extensions"),
      });
    });

    // Extension toggles must not leak into the restart banner.
    expect(useSettingsRestartStore.getState().lastMutation).toBeNull();
  });

  it("disables an extension and reuses the same invalidation path", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(disableSettingsExtension).mockResolvedValue({ ...extension, enabled: false });

    const { result } = renderHook(() => useDisableSettingsExtension(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync("daytona");
    });

    expect(disableSettingsExtension).toHaveBeenCalledWith("daytona");
    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.extensionsRoot(),
      });
    });
  });
});
