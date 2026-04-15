import { describe, expect, it } from "vitest";

import { CapabilityDeniedError, NotInitializedError, RateLimitedError } from "./errors.js";
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
    pair.host.handle("bridges/instances/get", async () => ({
      id: "chan-1",
      scope: "global",
      platform: "telegram",
      extension_name: "telegram-adapter",
      display_name: "Telegram",
      enabled: true,
      status: "ready",
      routing_policy: { include_peer: true, include_thread: false, include_group: false },
    }));
    pair.host.handle("bridges/instances/report_state", async params => {
      expect(params).toEqual({ status: "auth_required" });
      return {
        id: "chan-1",
        scope: "global",
        platform: "telegram",
        extension_name: "telegram-adapter",
        display_name: "Telegram",
        enabled: true,
        status: "auth_required",
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
    await expect(host.bridges.get()).resolves.toMatchObject({
      id: "chan-1",
      status: "ready",
    });
    await expect(host.bridges.reportState({ status: "auth_required" })).resolves.toMatchObject({
      id: "chan-1",
      status: "auth_required",
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
