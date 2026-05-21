import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../../adapters/settings-api", () => ({
  deleteSettingsSandbox: vi.fn(),
  deleteSettingsHook: vi.fn(),
  deleteSettingsMCPServer: vi.fn(),
  deleteSettingsProvider: vi.fn(),
  disableSettingsExtension: vi.fn(),
  enableSettingsExtension: vi.fn(),
  installSettingsExtension: vi.fn(),
  putSettingsSandbox: vi.fn(),
  putSettingsHook: vi.fn(),
  putSettingsMCPServer: vi.fn(),
  putSettingsProvider: vi.fn(),
  reloadSettings: vi.fn(),
  removeSettingsExtension: vi.fn(),
  updateSettingsExtension: vi.fn(),
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
  installSettingsExtension,
  putSettingsMCPServer,
  reloadSettings,
  removeSettingsExtension,
  updateSettingsExtension,
  updateSettingsGeneral,
  updateSettingsMemory,
} from "../../adapters/settings-api";
import { settingsKeys } from "../../lib/query-keys";
import { settingsMemoryConfigFixture } from "../../mocks/fixtures";
import { initialSettingsRestartState } from "../../stores/settings-restart-store";
import { useSettingsRestartStore } from "../../stores/use-settings-restart-store";
import {
  useDeleteSettingsMCPServer,
  useDeleteSettingsProvider,
  useDisableSettingsExtension,
  useEnableSettingsExtension,
  useInstallSettingsExtension,
  usePutSettingsMCPServer,
  useReloadSettings,
  useRemoveSettingsExtension,
  useUpdateSettingsExtension,
  useUpdateSettingsGeneral,
  useUpdateSettingsMemory,
} from "../use-settings-mutations";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });

  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);

  return { queryClient, wrapper };
}

const generalMutation = {
  active_config_hash: "sha256:active-live",
  active_generation: 42,
  section: "general" as const,
  scope: "global" as const,
  applied: true,
  apply_record_id: "cfg_apply_001",
  lifecycle: "restart-required" as const,
  next_action: "restart-daemon" as const,
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
  it("records mutation state and invalidates the general section plus apply records", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(updateSettingsGeneral).mockResolvedValue(generalMutation);

    const { result } = renderHook(() => useUpdateSettingsGeneral(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync({
        config: {
          daemon: {
            reload_timeouts: { bridges: "30s", mcp: "10s", providers: "5s" },
            socket: "/tmp/a.sock",
          },
          defaults: { agent: "claude-code" },
          http: { host: "127.0.0.1", port: 2123 },
          limits: { max_concurrent_agents: 4 },
          permissions: { mode: "approve-reads" as const },
          session_timeout: "30m",
        },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.section("general"),
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.applyRoot(),
      });
    });

    expect(useSettingsRestartStore.getState().lastMutation?.restartRequired).toBe(true);
    expect(useSettingsRestartStore.getState().lastMutation?.warnings).toEqual([
      "restart the daemon",
    ]);
    expect(useSettingsRestartStore.getState().lastMutation?.nextAction).toBe("restart-daemon");
    expect(useSettingsRestartStore.getState().lastMutation?.applyRecordId).toBe("cfg_apply_001");

    const memoryInvalidations = invalidateSpy.mock.calls.filter(([arg]) =>
      JSON.stringify(arg?.queryKey).includes("memory")
    );
    expect(memoryInvalidations).toHaveLength(0);
  });
});

describe("useUpdateSettingsMemory", () => {
  it("invalidates memory section and apply records", async () => {
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
          ...settingsMemoryConfigFixture,
          dream: { ...settingsMemoryConfigFixture.dream, agent: "dreamer", min_hours: 1 },
        },
      });
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.section("memory") });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.applyRoot() });
    });
  });
});

describe("useReloadSettings", () => {
  it("records reload state and invalidates all settings queries", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(reloadSettings).mockResolvedValue(generalMutation);

    const { result } = renderHook(() => useReloadSettings(), { wrapper });

    await act(async () => {
      await result.current.mutateAsync();
    });

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.all });
    });

    expect(useSettingsRestartStore.getState().lastMutation?.activeGeneration).toBe(42);
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
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.applyRoot() });
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
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: settingsKeys.applyRoot() });
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

  it("installs, updates, and removes extensions through the shared invalidation path", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(installSettingsExtension).mockResolvedValue(extension);
    vi.mocked(updateSettingsExtension).mockResolvedValue({
      name: "daytona",
      slug: "daytona/daytona-extension",
      registry: "github",
      path: "/tmp/agh/extensions/daytona",
      current_version: "1.2.3",
      latest_version: "1.2.4",
      status: "available",
    });
    vi.mocked(removeSettingsExtension).mockResolvedValue({
      name: "daytona",
      path: "/tmp/agh/extensions/daytona",
      status: "removed",
    });

    const install = renderHook(() => useInstallSettingsExtension(), { wrapper });
    await act(async () => {
      await install.result.current.mutateAsync({
        slug: "daytona/daytona-extension",
        source: "github",
        allow_unverified: true,
      });
    });
    expect(installSettingsExtension).toHaveBeenCalledWith({
      slug: "daytona/daytona-extension",
      source: "github",
      allow_unverified: true,
    });

    const update = renderHook(() => useUpdateSettingsExtension(), { wrapper });
    await act(async () => {
      await update.result.current.mutateAsync({
        name: "daytona",
        body: { version: "1.2.4" },
      });
    });
    expect(updateSettingsExtension).toHaveBeenCalledWith("daytona", { version: "1.2.4" });

    const remove = renderHook(() => useRemoveSettingsExtension(), { wrapper });
    await act(async () => {
      await remove.result.current.mutateAsync("daytona");
    });
    expect(removeSettingsExtension).toHaveBeenCalledWith("daytona");

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.extensionsRoot(),
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: settingsKeys.section("hooks-extensions"),
      });
    });
  });
});
