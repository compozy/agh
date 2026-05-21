import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  useMatchRoute: () => () => false,
}));

vi.mock("@/systems/settings/adapters/settings-api", () => ({
  getSettingsHooksExtensions: vi.fn(),
  getSettingsExtensionProvenance: vi.fn(),
  installSettingsExtension: vi.fn(),
  listSettingsExtensions: vi.fn(),
  removeSettingsExtension: vi.fn(),
  searchSettingsExtensionMarketplace: vi.fn(),
  updateSettingsExtension: vi.fn(),
  updateSettingsHooksExtensions: vi.fn(),
  putSettingsHook: vi.fn(),
  enableSettingsExtension: vi.fn(),
  disableSettingsExtension: vi.fn(),
  getSettingsRestartStatus: vi.fn(),
  triggerSettingsRestart: vi.fn(),
  SettingsApiError: class SettingsApiError extends Error {
    status = 500;
  },
}));

import {
  disableSettingsExtension,
  enableSettingsExtension,
  getSettingsExtensionProvenance,
  getSettingsHooksExtensions,
  installSettingsExtension,
  listSettingsExtensions,
  putSettingsHook,
  removeSettingsExtension,
  searchSettingsExtensionMarketplace,
  updateSettingsExtension,
  updateSettingsHooksExtensions,
} from "@/systems/settings/adapters/settings-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsExtensionEntry, SettingsHooksExtensionsSection } from "@/systems/settings";
import { useSettingsHooksExtensionsPage } from "../use-settings-hooks-extensions-page";

const envelope: SettingsHooksExtensionsSection = {
  section: "hooks-extensions",
  scope: "global",
  available_scopes: ["global"],
  config: {
    marketplace: { registry: "github", base_url: "https://api.github.com" },
    resources: {
      allowed_kinds: ["snapshot", "artifact"],
      max_scope: "workspace",
      snapshot_rate_limit: { queue: 100, requests: 30, window: "5m" },
      operator_write_rate_limit: { queue: 20, requests: 10, window: "1m" },
    },
  },
  hooks: [
    {
      name: "pre-commit-lint",
      declaration: {
        name: "pre-commit-lint",
        event: "tool.pre_call",
        mode: "sync",
        command: "make",
        args: ["lint"],
        matcher: { tool_name: "Bash" },
        required: true,
      },
      source_metadata: {
        available_targets: ["global-config"],
        effective_source: { kind: "global-config", scope: "global" },
      },
    },
  ],
  installed: [
    {
      name: "daytona",
      enabled: true,
      version: "1.2.3",
      state: "running",
      requires_env: ["DAYTONA_TOKEN"],
      missing_env: ["DAYTONA_TOKEN"],
    },
  ],
  transport_parity: {
    known: true,
    settings_http: true,
    settings_uds: true,
    extensions_http: true,
    extensions_uds: true,
  },
};

const extensionEntry: SettingsExtensionEntry = {
  name: "daytona",
  enabled: true,
  version: "1.2.3",
  state: "running",
  source: "marketplace",
  type: "backend",
  daemon_running: true,
  health: "healthy",
  requires_env: ["DAYTONA_TOKEN"],
  missing_env: ["DAYTONA_TOKEN"],
  trust: {
    decision: "allowed_unverified",
    registry_tier: "community",
    checksum_verified: false,
    allow_unverified: true,
  },
  provenance: {
    slug: "daytona/daytona-extension",
    installed_from: "marketplace_registry",
    source_url: "https://registry.example.com/daytona/daytona-extension",
    checksum_sha256: "sha256:fixture-daytona",
    checksum_verified: false,
    registry_tier: "community",
    permissions: ["logs.read"],
    installed_at: "2026-05-21T10:00:00Z",
    installed_by: "operator:web",
    allow_unverified: true,
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
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
  vi.mocked(getSettingsHooksExtensions).mockResolvedValue(envelope);
  vi.mocked(listSettingsExtensions).mockResolvedValue([extensionEntry]);
  vi.mocked(searchSettingsExtensionMarketplace).mockResolvedValue([
    {
      slug: "daytona/daytona-extension",
      name: "daytona",
      source: "github",
      type: "backend",
      version: "1.2.4",
      trust: extensionEntry.trust,
    },
  ]);
  vi.mocked(getSettingsExtensionProvenance).mockResolvedValue(
    extensionEntry.provenance as NonNullable<SettingsExtensionEntry["provenance"]>
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSettingsHooksExtensionsPage", () => {
  it("loads combined hook declarations, installed extensions, and policy draft", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => {
      expect(result.current.envelope).toBeTruthy();
      expect(result.current.draft).toEqual(envelope.config);
    });

    expect(result.current.hooks).toHaveLength(1);
    expect(result.current.hooksCounts.enabled).toBe(1);
    expect(result.current.canMutateExtensions).toBe(true);
  });

  it("falls back to envelope.installed summaries when the live extensions query is empty", async () => {
    vi.mocked(listSettingsExtensions).mockResolvedValue([]);
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.envelope).toBeTruthy());
    await waitFor(() => expect(result.current.extensionsLoading).toBe(false));

    expect(result.current.extensions).toHaveLength(1);
    expect(result.current.extensions[0]?.name).toBe("daytona");
    expect(result.current.extensions[0]?.missing_env).toEqual(["DAYTONA_TOKEN"]);
  });

  it("records a restart-required last action after the policy save mutation succeeds", async () => {
    vi.mocked(updateSettingsHooksExtensions).mockResolvedValue({
      section: "hooks-extensions",
      scope: "global",
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.updatePolicyDraft(current => ({
        ...current,
        marketplace: { ...current.marketplace, registry: "gitlab" },
      }));
      result.current.handleSavePolicy();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });
    expect(
      result.current.lastAction?.kind === "saved" &&
        result.current.lastAction.result.restart_required
    ).toBe(true);
  });

  it("drives immediate-apply semantics for extension toggles without touching restart state", async () => {
    vi.mocked(disableSettingsExtension).mockResolvedValue({ ...extensionEntry, enabled: false });
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.extensions).toHaveLength(1));

    await act(async () => {
      result.current.toggleExtensionEnabled(extensionEntry, false);
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("extension-toggled");
    });
    expect(disableSettingsExtension).toHaveBeenCalledWith("daytona");
    // Extension toggles never publish to the restart banner.
    expect(useSettingsRestartStore.getState().lastMutation).toBeNull();
  });

  it("installs marketplace extensions with the explicit trust decision", async () => {
    vi.mocked(installSettingsExtension).mockResolvedValue(extensionEntry);
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.marketplaceEntries).toHaveLength(1));

    act(() => {
      result.current.setMarketplaceAllowUnverified(true);
    });

    await act(async () => {
      result.current.installMarketplaceExtension(result.current.marketplaceEntries[0]);
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("extension-installed");
    });
    expect(installSettingsExtension).toHaveBeenCalledWith({
      slug: "daytona/daytona-extension",
      source: "github",
      version: "1.2.4",
      allow_unverified: true,
    });
  });

  it("loads provenance and routes update/remove through daemon mutations", async () => {
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
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.extensions).toHaveLength(1));

    act(() => {
      result.current.openExtensionProvenance(extensionEntry);
    });

    await waitFor(() => {
      expect(result.current.selectedProvenance?.installed_from).toBe("marketplace_registry");
    });

    await act(async () => {
      result.current.updateExtension(extensionEntry);
    });
    await waitFor(() => expect(result.current.lastAction?.kind).toBe("extension-updated"));
    expect(updateSettingsExtension).toHaveBeenCalledWith("daytona", {});

    await act(async () => {
      result.current.removeExtension(extensionEntry);
    });
    await waitFor(() => expect(result.current.lastAction?.kind).toBe("extension-removed"));
    expect(removeSettingsExtension).toHaveBeenCalledWith("daytona");
  });

  it("drives the hook toggle through putSettingsHook and tracks pending state", async () => {
    vi.mocked(putSettingsHook).mockResolvedValue({
      section: "hooks-extensions",
      scope: "global",
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.hooks).toHaveLength(1));

    await act(async () => {
      result.current.toggleHookEnabled(result.current.hooks[0], false);
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("hook-toggled");
    });
    expect(putSettingsHook).toHaveBeenCalledTimes(1);
    const [, body] = vi.mocked(putSettingsHook).mock.calls[0];
    expect(body.declaration.required).toBe(false);
  });

  it("flags canMutateExtensions=false when the transport parity exposes the restriction", async () => {
    vi.mocked(getSettingsHooksExtensions).mockResolvedValue({
      ...envelope,
      transport_parity: {
        known: true,
        settings_http: true,
        settings_uds: true,
        extensions_http: false,
        extensions_uds: true,
      },
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.envelope).toBeTruthy());
    expect(result.current.canMutateExtensions).toBe(false);
  });

  it("toggles allowed_kinds in the draft without mutating the envelope config", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.draft).toBeTruthy());

    act(() => {
      result.current.toggleAllowedKind("memory");
    });

    expect(result.current.draft?.resources.allowed_kinds).toEqual([
      "artifact",
      "memory",
      "snapshot",
    ]);
    expect(envelope.config.resources.allowed_kinds).toEqual(["snapshot", "artifact"]);
    expect(result.current.isPolicyDirty).toBe(true);

    act(() => {
      result.current.handleResetPolicy();
    });
    expect(result.current.draft?.resources.allowed_kinds).toEqual(["snapshot", "artifact"]);
    expect(result.current.isPolicyDirty).toBe(false);
  });

  it("reports extension action errors separately from policy save errors", async () => {
    vi.mocked(enableSettingsExtension).mockRejectedValue(new Error("remote denied"));

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsHooksExtensionsPage(), { wrapper });

    await waitFor(() => expect(result.current.extensions).toHaveLength(1));

    await act(async () => {
      result.current.toggleExtensionEnabled({ ...extensionEntry, enabled: false }, true);
    });

    await waitFor(() => {
      expect(result.current.extensionActionError).toBe("remote denied");
    });
    expect(result.current.savePolicyError).toBeNull();
  });
});
