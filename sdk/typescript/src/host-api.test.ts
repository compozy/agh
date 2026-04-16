import { describe, expect, it } from "vitest";

import {
  CapabilityDeniedError,
  InvalidParamsError,
  NotInitializedError,
  RateLimitedError,
  RPCError,
} from "./errors.js";
import { HostAPI } from "./host-api.js";
import { createMockTransportPair } from "./testing/mock-transport.js";

describe("HostAPI", () => {
  it("sessions.create sends correct JSON-RPC request", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("sessions/create", async params => {
      expect(params).toEqual({
        agent: "coder",
        prompt: "debug this",
        workspace: "workspace-1",
      });
      return { session_id: "sess-1" };
    });

    await expect(
      host.sessions.create({ agent: "coder", prompt: "debug this", workspace: "workspace-1" })
    ).resolves.toEqual({ session_id: "sess-1" });

    expect(pair.extension.requests[0]).toMatchObject({
      method: "sessions/create",
      params: {
        agent: "coder",
        prompt: "debug this",
        workspace: "workspace-1",
      },
    });
  });

  it("sessions.list parses response array", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("sessions/list", async () => [
      {
        id: "sess-1",
        name: "debug",
        agent: "claude",
        state: "active",
        created_at: "2026-04-10T12:00:00.000Z",
      },
    ]);

    await expect(host.sessions.list()).resolves.toEqual([
      {
        id: "sess-1",
        name: "debug",
        agent: "claude",
        state: "active",
        created_at: "2026-04-10T12:00:00.000Z",
      },
    ]);
  });

  it("memory.store sends correct params", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("memory/store", async params => {
      expect(params).toEqual({
        key: "deploy-script",
        content: "use ./scripts/deploy.sh",
        tags: ["reference"],
      });
      return {};
    });

    await expect(
      host.memory.store({
        key: "deploy-script",
        content: "use ./scripts/deploy.sh",
        tags: ["reference"],
      })
    ).resolves.toEqual({});
  });

  it("observe.events supports since parameter", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("observe/events", async params => {
      expect(params).toEqual({
        since: "2026-04-10T12:00:00.000Z",
        limit: 5,
      });
      return [
        {
          type: "message.end",
          timestamp: "2026-04-10T12:01:00.000Z",
          data: { summary: "done" },
        },
      ];
    });

    await expect(
      host.observe.events({ since: "2026-04-10T12:00:00.000Z", limit: 5 })
    ).resolves.toHaveLength(1);
  });

  it("supports the remaining host api methods", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("sessions/prompt", async params => {
      expect(params).toEqual({ session_id: "sess-1", message: "hello" });
      return { turn_id: "turn-1" };
    });
    pair.host.handle("sessions/stop", async params => {
      expect(params).toEqual({ session_id: "sess-1" });
      return {};
    });
    pair.host.handle("sessions/status", async () => ({
      session_id: "sess-1",
      agent: "claude",
      state: "active",
      created_at: "2026-04-10T12:00:00.000Z",
      updated_at: "2026-04-10T12:01:00.000Z",
    }));
    pair.host.handle("memory/recall", async () => [
      { key: "deploy", content: "run deploy", score: 1 },
    ]);
    pair.host.handle("memory/forget", async params => {
      expect(params).toEqual({ key: "deploy" });
      return {};
    });
    pair.host.handle("observe/health", async () => ({
      status: "ok",
      uptime_seconds: 1,
      active_sessions: 1,
      active_agents: 1,
      global_db_size_bytes: 1,
      session_db_size_bytes: 1,
      version: "0.5.0",
    }));
    pair.host.handle("skills/list", async () => [
      { name: "skill-a", description: "desc", source: "workspace" },
    ]);
    pair.host.handle("bridges/messages/ingest", async params => {
      expect(params).toEqual({
        bridge_instance_id: "chan-1",
        scope: "global",
        event_family: "message",
        platform_message_id: "msg-1",
        received_at: "2026-04-11T12:00:00.000Z",
        sender: { id: "user-1" },
        content: { text: "hello" },
        idempotency_key: "idem-1",
      });
      return {
        session_id: "sess-1",
        route_created: true,
        routing_key: {
          scope: "global",
          bridge_instance_id: "chan-1",
          peer_id: "user-1",
        },
      };
    });
    pair.host.handle("bridges/instances/list", async () => [
      {
        id: "chan-1",
        scope: "global",
        platform: "telegram",
        extension_name: "telegram-adapter",
        display_name: "Telegram",
        enabled: true,
        status: "ready",
        routing_policy: { include_peer: true, include_thread: false, include_group: false },
      },
    ]);
    pair.host.handle("bridges/instances/get", async params => {
      expect(params).toEqual({ bridge_instance_id: "chan-1" });
      return {
        id: "chan-1",
        scope: "global",
        platform: "telegram",
        extension_name: "telegram-adapter",
        display_name: "Telegram",
        enabled: true,
        status: "ready",
        routing_policy: { include_peer: true, include_thread: false, include_group: false },
      };
    });
    pair.host.handle("bridges/instances/report_state", async params => {
      expect(params).toEqual({
        bridge_instance_id: "chan-1",
        status: "auth_required",
        degradation: { reason: "auth_failed", message: "token expired" },
      });
      return {
        id: "chan-1",
        scope: "global",
        platform: "telegram",
        extension_name: "telegram-adapter",
        display_name: "Telegram",
        enabled: true,
        status: "auth_required",
        degradation: { reason: "auth_failed", message: "token expired" },
        routing_policy: { include_peer: true, include_thread: false, include_group: false },
      };
    });

    await expect(host.sessions.prompt({ session_id: "sess-1", message: "hello" })).resolves.toEqual(
      {
        turn_id: "turn-1",
      }
    );
    await expect(host.sessions.stop({ session_id: "sess-1" })).resolves.toEqual({});
    await expect(host.sessions.status({ session_id: "sess-1" })).resolves.toMatchObject({
      session_id: "sess-1",
      state: "active",
    });
    await expect(host.memory.recall({ query: "deploy" })).resolves.toEqual([
      { key: "deploy", content: "run deploy", score: 1 },
    ]);
    await expect(host.memory.forget({ key: "deploy" })).resolves.toEqual({});
    await expect(host.observe.health()).resolves.toMatchObject({ status: "ok" });
    await expect(host.skills.list()).resolves.toEqual([
      { name: "skill-a", description: "desc", source: "workspace" },
    ]);
    await expect(
      host.bridges.ingest({
        bridge_instance_id: "chan-1",
        scope: "global",
        event_family: "message",
        platform_message_id: "msg-1",
        received_at: "2026-04-11T12:00:00.000Z",
        sender: { id: "user-1" },
        content: { text: "hello" },
        idempotency_key: "idem-1",
      })
    ).resolves.toMatchObject({
      session_id: "sess-1",
      route_created: true,
    });
    await expect(host.bridges.list()).resolves.toEqual([
      expect.objectContaining({
        id: "chan-1",
        status: "ready",
      }),
    ]);
    await expect(host.bridges.get({ bridge_instance_id: "chan-1" })).resolves.toMatchObject({
      id: "chan-1",
      status: "ready",
    });
    await expect(
      host.bridges.reportState({
        bridge_instance_id: "chan-1",
        status: "auth_required",
        degradation: { reason: "auth_failed", message: "token expired" },
      })
    ).resolves.toMatchObject({
      id: "chan-1",
      status: "auth_required",
    });
  });

  it("resources helpers validate payload shape and send snapshot requests", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("resources/list", async params => {
      expect(params).toEqual({
        kind: "tool",
        scope: { kind: "workspace", id: "ws-1" },
        limit: 5,
      });
      return [
        {
          kind: "tool",
          id: "grep",
          version: 2,
          scope: { kind: "workspace", id: "ws-1" },
          owner: { kind: "extension", id: "resource-ext" },
          source: { kind: "extension", id: "resource-ext" },
          spec: { command: "rg" },
          created_at: "2026-04-15T12:00:00.000Z",
          updated_at: "2026-04-15T12:01:00.000Z",
        },
      ];
    });
    pair.host.handle("resources/get", async params => {
      expect(params).toEqual({ kind: "tool", id: "grep" });
      return {
        kind: "tool",
        id: "grep",
        version: 2,
        scope: { kind: "workspace", id: "ws-1" },
        owner: { kind: "extension", id: "resource-ext" },
        source: { kind: "extension", id: "resource-ext" },
        spec: { command: "rg" },
        created_at: "2026-04-15T12:00:00.000Z",
        updated_at: "2026-04-15T12:01:00.000Z",
      };
    });
    pair.host.handle("resources/snapshot", async params => {
      expect(params).toEqual({
        source_version: 3,
        records: [
          {
            kind: "tool",
            id: "grep",
            scope: { kind: "workspace", id: "ws-1" },
            spec: { command: "rg" },
          },
        ],
      });
      return {};
    });

    await expect(
      host.resources.list({
        kind: "tool",
        scope: { kind: "workspace", id: "ws-1" },
        limit: 5,
      })
    ).resolves.toHaveLength(1);
    await expect(host.resources.get({ kind: "tool", id: "grep" })).resolves.toMatchObject({
      id: "grep",
      version: 2,
    });
    await expect(
      host.resources.snapshot({
        source_version: 3,
        records: [
          {
            kind: "tool",
            id: "grep",
            scope: { kind: "workspace", id: "ws-1" },
            spec: { command: "rg" },
          },
        ],
      })
    ).resolves.toEqual({});

    await expect(
      host.resources.list({ scope: { kind: "workspace", id: "" } })
    ).rejects.toBeInstanceOf(InvalidParamsError);
    await expect(host.resources.get({ kind: "", id: "grep" })).rejects.toBeInstanceOf(
      InvalidParamsError
    );
    await expect(
      host.resources.snapshot({
        source_version: 0,
        records: [],
      })
    ).rejects.toBeInstanceOf(InvalidParamsError);
  });

  it("resources helpers surface 403, 409, 413, and 429 protocol errors", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("resources/get", async () => {
      throw new RPCError(403, "Forbidden", { error: "same-source only" });
    });
    pair.host.handle("resources/list", async () => {
      throw new RPCError(409, "Conflict", { error: "stale source_version" });
    });
    pair.host.handle("resources/snapshot", async params => {
      const request = params as { source_version: number };
      switch (request.source_version) {
        case 4:
          throw new RPCError(413, "Payload too large", { error: "snapshot too large" });
        case 5:
          throw new RPCError(429, "Rate limited", { error: "snapshot queued" });
        default:
          return {};
      }
    });

    await expect(host.resources.get({ kind: "tool", id: "grep" })).rejects.toMatchObject({
      code: 403,
      message: "Forbidden",
    });
    await expect(host.resources.list()).rejects.toMatchObject({
      code: 409,
      message: "Conflict",
    });
    await expect(
      host.resources.snapshot({
        source_version: 4,
        records: [
          {
            kind: "tool",
            id: "grep",
            scope: { kind: "workspace", id: "ws-1" },
            spec: { command: "rg" },
          },
        ],
      })
    ).rejects.toMatchObject({
      code: 413,
      message: "Payload too large",
    });
    await expect(
      host.resources.snapshot({
        source_version: 5,
        records: [
          {
            kind: "tool",
            id: "grep",
            scope: { kind: "workspace", id: "ws-1" },
            spec: { command: "rg" },
          },
        ],
      })
    ).rejects.toMatchObject({
      code: 429,
      message: "Rate limited",
    });
  });

  it("rejects calls before the session is ready", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => false });

    await expect(host.sessions.list()).rejects.toBeInstanceOf(NotInitializedError);
  });

  it("throws a typed capability denied error with code -32001", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("sessions/create", async () => {
      throw new CapabilityDeniedError({
        method: "sessions/create",
        required: ["session.write"],
        granted: ["session.read"],
      });
    });

    await expect(host.sessions.create({ agent: "coder" })).rejects.toMatchObject({
      code: -32001,
      data: {
        method: "sessions/create",
        required: ["session.write"],
        granted: ["session.read"],
      },
    });
  });

  it("throws a typed rate limited error with retry_after_ms", async () => {
    const pair = createMockTransportPair();
    const host = new HostAPI(pair.extension, { isReady: () => true });

    pair.host.handle("sessions/list", async () => {
      throw new RateLimitedError({
        scope: "host_api.sessions/list",
        retry_after_ms: 1000,
        limit: 10,
        burst: 20,
      });
    });

    await expect(host.sessions.list()).rejects.toMatchObject({
      code: -32002,
      data: {
        scope: "host_api.sessions/list",
        retry_after_ms: 1000,
      },
    });
  });
});
