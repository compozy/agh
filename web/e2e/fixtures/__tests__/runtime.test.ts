// @vitest-environment node

import { describe, expect, it } from "vitest";

import {
  assertDaemonServedHTML,
  buildResolveWorkspaceRequest,
  normalizeBaseURL,
  requiresHTTPAPIReadinessProbe,
  renderRuntimeConfig,
  resolveRuntimeMode,
  runtimeURL,
} from "../runtime-helpers";

describe("runtime helpers", () => {
  it("defaults to launch mode when no attach URL is configured", () => {
    expect(resolveRuntimeMode({})).toEqual({ kind: "launch" });
  });

  it("normalizes attach mode base URLs", () => {
    expect(resolveRuntimeMode({ AGH_E2E_BASE_URL: "http://127.0.0.1:4213/" })).toEqual({
      kind: "attach",
      baseURL: "http://127.0.0.1:4213",
    });
  });

  it("rejects attach URLs that target a non-root path", () => {
    expect(() => normalizeBaseURL("http://127.0.0.1:4213/ui")).toThrow(
      /AGH_E2E_BASE_URL must point at the daemon root/
    );
  });

  it("renders the seeded daemon config with socket and HTTP bindings", () => {
    expect(
      renderRuntimeConfig({
        host: "127.0.0.1",
        port: 4321,
        socketPath: "/tmp/agh.sock",
      })
    ).toBe(
      [
        "[daemon]",
        'socket = "/tmp/agh.sock"',
        "",
        "[http]",
        'host = "127.0.0.1"',
        "port = 4321",
        "",
      ].join("\n")
    );
  });

  it("renders network enablement when requested by the browser runtime", () => {
    expect(
      renderRuntimeConfig({
        host: "127.0.0.1",
        networkEnabled: true,
        port: 4321,
        socketPath: "/tmp/agh.sock",
      })
    ).toContain("[network]\nenabled = true\n");
  });

  it("renders network disablement when requested by the browser runtime", () => {
    expect(
      renderRuntimeConfig({
        host: "127.0.0.1",
        networkEnabled: false,
        port: 4321,
        socketPath: "/tmp/agh.sock",
      })
    ).toContain("[network]\nenabled = false\n");
  });

  it("renders a seeded skill marketplace base URL for launch-mode browser E2E", () => {
    expect(
      renderRuntimeConfig({
        host: "127.0.0.1",
        port: 4321,
        skillsMarketplaceBaseURL: "http://127.0.0.1:9876",
        socketPath: "/tmp/agh.sock",
      })
    ).toContain(
      [
        "[skills.marketplace]",
        'registry = "clawhub"',
        'base_url = "http://127.0.0.1:9876"',
        "",
      ].join("\n")
    );
  });

  it("renders the auth-free acpmock provider when browser E2E seeds mock agents", () => {
    expect(
      renderRuntimeConfig({
        host: "127.0.0.1",
        includeMockAgentProvider: true,
        port: 4321,
        socketPath: "/tmp/agh.sock",
      })
    ).toContain(
      [
        "[providers.acpmock]",
        'command = "acpmock-driver"',
        'display_name = "ACP Mock"',
        'harness = "acp"',
        'auth_mode = "none"',
        'none_security = "local_transport"',
        "",
      ].join("\n")
    );
  });

  it("requires API readiness probes only for loopback HTTP bindings", () => {
    expect(requiresHTTPAPIReadinessProbe("")).toBe(true);
    expect(requiresHTTPAPIReadinessProbe("localhost")).toBe(true);
    expect(requiresHTTPAPIReadinessProbe("127.0.0.1")).toBe(true);
    expect(requiresHTTPAPIReadinessProbe("[::1]")).toBe(true);
    expect(requiresHTTPAPIReadinessProbe("0.0.0.0")).toBe(false);
    expect(requiresHTTPAPIReadinessProbe("192.168.1.10")).toBe(false);
  });

  it("joins runtime URLs against the daemon origin", () => {
    expect(runtimeURL("http://127.0.0.1:4317", "/api/status")).toBe(
      "http://127.0.0.1:4317/api/status"
    );
    expect(runtimeURL("http://127.0.0.1:4317/", "api/workspaces")).toBe(
      "http://127.0.0.1:4317/api/workspaces"
    );
  });

  it("encodes workspace resolution requests with the public path contract", () => {
    expect(buildResolveWorkspaceRequest("/tmp/agh-home")).toEqual({
      path: "/tmp/agh-home",
    });
  });

  it("rejects vite dev HTML when enforcing daemon-served assets", () => {
    expect(() =>
      assertDaemonServedHTML(
        '<!doctype html><html><head><script type="module" src="/@vite/client"></script></head></html>',
        "http://127.0.0.1:3000"
      )
    ).toThrow(/daemon-served embedded assets/);
  });

  it("accepts built embedded HTML without vite dev markers", () => {
    expect(() =>
      assertDaemonServedHTML(
        '<!doctype html><html><head><script type="module" src="/assets/index-abc123.js"></script></head></html>',
        "http://127.0.0.1:4213"
      )
    ).not.toThrow();
  });
});
