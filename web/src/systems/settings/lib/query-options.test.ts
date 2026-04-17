import { describe, expect, it } from "vitest";

import {
  SETTINGS_QUERY_INTERVALS,
  settingsAutomationOptions,
  settingsEnvironmentDetailOptions,
  settingsGeneralOptions,
  settingsMCPServersListOptions,
  settingsProviderDetailOptions,
  settingsRestartStatusOptions,
} from "./query-options";

describe("settings section options", () => {
  it("uses the configured stale and refetch intervals for sections", () => {
    const general = settingsGeneralOptions();
    const automation = settingsAutomationOptions();

    expect(general.staleTime).toBe(SETTINGS_QUERY_INTERVALS.sectionStaleTime);
    expect(general.refetchInterval).toBe(SETTINGS_QUERY_INTERVALS.sectionRefetchInterval);
    expect(automation.queryKey).toEqual(["settings", "section", "automation"]);
  });
});

describe("settings collection options", () => {
  it("disables detail queries when name is empty", () => {
    expect(settingsProviderDetailOptions("").enabled).toBe(false);
    expect(settingsEnvironmentDetailOptions("").enabled).toBe(false);
  });

  it("enables detail queries when name is provided", () => {
    expect(settingsProviderDetailOptions("openai").enabled).toBe(true);
    expect(settingsEnvironmentDetailOptions("cloud").enabled).toBe(true);
  });

  it("includes scope and workspace filters in MCP list query keys", () => {
    const global = settingsMCPServersListOptions({ scope: "global" });
    const scoped = settingsMCPServersListOptions({
      scope: "workspace",
      workspace_id: "ws_alpha",
    });

    expect(global.queryKey).toEqual([
      "settings",
      "collection",
      "mcp-servers",
      "list",
      "global",
      "",
    ]);
    expect(scoped.queryKey).toEqual([
      "settings",
      "collection",
      "mcp-servers",
      "list",
      "workspace",
      "ws_alpha",
    ]);
  });
});

describe("settings restart options", () => {
  it("is disabled while no operation id is active", () => {
    const disabled = settingsRestartStatusOptions(null, true);
    const enabled = settingsRestartStatusOptions("op_1", true);

    expect(disabled.enabled).toBe(false);
    expect(enabled.enabled).toBe(true);
    expect(enabled.queryKey).toEqual(["settings", "restart", "op_1"]);
  });

  it("polls while the status is not terminal and stops on terminal states", () => {
    const options = settingsRestartStatusOptions("op_1", true);
    const refetchInterval = options.refetchInterval as (query: {
      state: { data?: { status: string } };
    }) => number | false;

    expect(refetchInterval({ state: {} })).toBe(SETTINGS_QUERY_INTERVALS.restartPollInterval);
    expect(refetchInterval({ state: { data: { status: "stopping" } } })).toBe(
      SETTINGS_QUERY_INTERVALS.restartPollInterval
    );
    expect(refetchInterval({ state: { data: { status: "ready" } } })).toBe(false);
    expect(refetchInterval({ state: { data: { status: "failed" } } })).toBe(false);
  });
});
