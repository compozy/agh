import { describe, expect, it, vi } from "vitest";

import { Extension } from "./extension.js";
import {
  CapabilityDeniedError,
  MethodNotFoundError,
  NotInitializedError,
  ShutdownInProgressError,
} from "./errors.js";
import { TestHarness } from "./testing/harness.js";
import { createMockTransportPair } from "./testing/mock-transport.js";
import type { InitializeRequest } from "./types.js";

function initializeFor(extension: Extension): InitializeRequest {
  const methods = extension.getImplementedMethods();
  return {
    protocol_version: "1",
    supported_protocol_versions: ["1"],
    agh_version: "0.5.0",
    extension: {
      name: extension.definition.name,
      version: extension.definition.version,
      source_tier: "user",
    },
    capabilities: {
      provides: extension.definition.capabilities?.provides ?? [],
      granted_actions: extension.definition.actions?.requires ?? [],
      granted_security: extension.definition.security?.capabilities ?? [],
    },
    methods: {
      daemon_requests: methods.filter(method =>
        ["execute_hook", "health_check", "shutdown", "provide_tools"].includes(method)
      ),
      extension_services: methods.filter(
        method => !["execute_hook", "health_check", "shutdown", "provide_tools"].includes(method)
      ),
    },
    runtime: {
      health_check_interval_ms: 30_000,
      health_check_timeout_ms: 5_000,
      shutdown_timeout_ms: 10_000,
      default_hook_timeout_ms: 5_000,
    },
  };
}

describe("Extension", () => {
  it("start performs initialize handshake first", async () => {
    const pair = createMockTransportPair();
    const extension = new Extension(
      {
        name: "test-ext",
        version: "0.1.0",
      },
      { transport: pair.extension }
    );

    const startPromise = extension.start();

    await expect(pair.host.call("health_check", {})).rejects.toBeInstanceOf(NotInitializedError);
    await expect(pair.host.call("initialize", initializeFor(extension))).resolves.toMatchObject({
      protocol_version: "1",
    });
    await expect(startPromise).resolves.toBeDefined();
  });

  it("handle routes inbound requests to the correct handler", async () => {
    const harness = new TestHarness();
    const extension = new Extension({
      name: "router",
      version: "0.1.0",
    });

    extension.handle("memory/store", async (_ctx, params: { key: string }) => ({
      stored: params.key,
    }));
    await harness.loadExtension(extension);

    await expect(harness.call("memory/store", { key: "alpha" })).resolves.toEqual({
      stored: "alpha",
    });
  });

  it("returns method not found when no handler is registered", async () => {
    const harness = new TestHarness();
    const extension = new Extension({
      name: "missing",
      version: "0.1.0",
    });

    await harness.loadExtension(extension);
    await expect(harness.call("memory/store", { key: "x" })).rejects.toBeInstanceOf(
      MethodNotFoundError
    );
  });

  it("onReady fires after successful handshake", async () => {
    const ready = vi.fn(async () => {});
    const harness = new TestHarness();
    const extension = new Extension({
      name: "ready",
      version: "0.1.0",
    });

    extension.onReady(ready);
    await harness.loadExtension(extension);

    expect(ready).toHaveBeenCalledTimes(1);
  });

  it("rejects initialize when required grants are missing", async () => {
    const pair = createMockTransportPair();
    const extension = new Extension(
      {
        name: "denied",
        version: "0.1.0",
        actions: { requires: ["sessions/list"] },
      },
      { transport: pair.extension }
    );

    void extension.start();
    await expect(
      pair.host.call("initialize", {
        ...initializeFor(extension),
        capabilities: {
          provides: [],
          granted_actions: [],
          granted_security: [],
        },
      })
    ).rejects.toBeInstanceOf(CapabilityDeniedError);
  });

  it("serves provide_tools and default health checks", async () => {
    const harness = new TestHarness();
    const extension = new Extension({
      name: "tools",
      version: "0.1.0",
    });

    extension.handle("provide_tools", async () => ({
      tools: [
        {
          name: "lookup",
          description: "finds things",
          input_schema: { type: "object" },
          read_only: true,
          source: "extension",
        },
      ],
    }));

    await harness.loadExtension(extension, {
      daemonRequests: ["health_check", "shutdown", "provide_tools"],
    });
    await expect(harness.call("health_check", {})).resolves.toMatchObject({ healthy: true });
    await expect(harness.call("provide_tools", {})).resolves.toMatchObject({
      tools: [expect.objectContaining({ name: "lookup" })],
    });
  });

  it("negotiates bridges/deliver for bridge adapters and exposes scoped runtime data", async () => {
    const ready = vi.fn();
    const harness = new TestHarness();
    const extension = new Extension({
      name: "bridge-adapter",
      version: "0.1.0",
      capabilities: { provides: ["bridge.adapter"] },
      actions: { requires: ["bridges/instances/get"] },
      security: { capabilities: ["bridge.read"] },
    });

    extension.handle("bridges/deliver", async () => ({
      acknowledged: true,
    }));
    extension.onReady((_host, session) => {
      ready(session.initializeRequest.runtime.bridge?.instance.id);
    });

    await harness.loadExtension(extension, {
      grantedActions: ["bridges/instances/get"],
      grantedSecurity: ["bridge.read"],
      runtime: {
        bridge: {
          instance: {
            id: "chan-1",
            scope: "global",
            platform: "telegram",
            extension_name: "bridge-adapter",
            display_name: "Telegram",
            enabled: true,
            status: "ready",
            routing_policy: { include_peer: true, include_thread: false, include_group: false },
            created_at: "2026-04-11T12:00:00.000Z",
            updated_at: "2026-04-11T12:00:00.000Z",
          },
          bound_secrets: [{ binding_name: "bot_token", kind: "bot_token", value: "secret" }],
        },
      },
    });

    expect(harness.getLastInitializeRequest()).toMatchObject({
      methods: { extension_services: ["bridges/deliver"] },
    });
    expect(ready).toHaveBeenCalledWith("chan-1");
  });

  it("rejects bridge.adapter initialize when bridges/deliver is not implemented", async () => {
    const pair = createMockTransportPair();
    const extension = new Extension(
      {
        name: "bridge-denied",
        version: "0.1.0",
        capabilities: { provides: ["bridge.adapter"] },
      },
      { transport: pair.extension }
    );

    void extension.start();
    await expect(pair.host.call("initialize", initializeFor(extension))).rejects.toMatchObject({
      data: {
        error: expect.stringContaining("bridge.adapter"),
      },
    });
  });

  it("rejects new work after shutdown begins", async () => {
    const harness = new TestHarness();
    const extension = new Extension({
      name: "shutdown",
      version: "0.1.0",
    });

    await harness.loadExtension(extension);
    await expect(harness.call("shutdown", { reason: "test", deadline_ms: 1000 })).resolves.toEqual({
      acknowledged: true,
    });
    await expect(harness.call("health_check", {})).rejects.toBeInstanceOf(ShutdownInProgressError);
  });
});
