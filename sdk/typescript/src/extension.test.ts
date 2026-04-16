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
    session_nonce: "session-nonce-test",
    extension: {
      name: extension.definition.name,
      version: extension.definition.version,
      source_tier: "user",
    },
    capabilities: {
      provides: extension.definition.capabilities?.provides ?? [],
      granted_actions: extension.definition.actions?.requires ?? [],
      granted_security: extension.definition.security?.capabilities ?? [],
      granted_resource_kinds: [],
      granted_resource_scopes: [],
    },
    methods: {
      daemon_requests: methods.filter(method =>
        ["execute_hook", "health_check", "shutdown"].includes(method)
      ),
      extension_services: methods.filter(
        method => !["execute_hook", "health_check", "shutdown"].includes(method)
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
          granted_resource_kinds: [],
          granted_resource_scopes: [],
        },
      })
    ).rejects.toBeInstanceOf(CapabilityDeniedError);
  });

  it("serves default health checks", async () => {
    const harness = new TestHarness();
    const extension = new Extension({
      name: "tools",
      version: "0.1.0",
    });

    await harness.loadExtension(extension, {
      daemonRequests: ["health_check", "shutdown"],
    });
    await expect(harness.call("health_check", {})).resolves.toMatchObject({ healthy: true });
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
      ready(session.initializeRequest.runtime.bridge?.managed_instances?.[0]?.instance.id);
    });

    await harness.loadExtension(extension, {
      grantedActions: ["bridges/instances/get"],
      grantedSecurity: ["bridge.read"],
      runtime: {
        bridge: {
          runtime_version: "1",
          provider: "bridge-adapter",
          platform: "telegram",
          managed_instances: [
            {
              instance: {
                id: "chan-1",
                scope: "global",
                platform: "telegram",
                extension_name: "bridge-adapter",
                display_name: "Telegram",
                enabled: true,
                status: "ready",
                routing_policy: {
                  include_peer: true,
                  include_thread: false,
                  include_group: false,
                },
                created_at: "2026-04-11T12:00:00.000Z",
                updated_at: "2026-04-11T12:00:00.000Z",
              },
              bound_secrets: [{ binding_name: "bot_token", kind: "bot_token", value: "secret" }],
            },
          ],
        },
      },
    });

    expect(harness.getLastInitializeRequest()).toMatchObject({
      session_nonce: "session-nonce-test",
      capabilities: {
        granted_resource_kinds: [],
        granted_resource_scopes: [],
      },
      methods: { extension_services: ["bridges/deliver"] },
    });
    expect(ready).toHaveBeenCalledWith("chan-1");
  });

  it("captures session nonce and resource grants during initialize", async () => {
    const ready = vi.fn();
    const harness = new TestHarness();
    const extension = new Extension({
      name: "resource-ext",
      version: "0.1.0",
    });

    extension.onReady((_host, session) => {
      ready({
        session_nonce: session.initializeRequest.session_nonce,
        granted_resource_kinds: session.initializeRequest.capabilities.granted_resource_kinds,
        granted_resource_scopes: session.initializeRequest.capabilities.granted_resource_scopes,
      });
    });

    await harness.loadExtension(extension, {
      sessionNonce: "nonce-resource",
      grantedResourceKinds: ["tool", "skill"],
      grantedResourceScopes: ["workspace"],
    });

    expect(ready).toHaveBeenCalledWith({
      session_nonce: "nonce-resource",
      granted_resource_kinds: ["tool", "skill"],
      granted_resource_scopes: ["workspace"],
    });
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
