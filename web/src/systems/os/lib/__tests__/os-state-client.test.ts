import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createDesktopPersistence } from "../desktop-persistence";
import { OsStateClient, type OsSocket } from "../os-state-client";
import type { OsStateEntry } from "../os-types";
import { createDesktopStore } from "../../stores/desktop-store";

interface FakeSocket extends OsSocket {
  sent: string[];
  open(): void;
  receive(frame: unknown): void;
  drop(): void;
}

function createFakeSocket(): FakeSocket {
  const socket: FakeSocket = {
    sent: [],
    onopen: null,
    onmessage: null,
    onclose: null,
    onerror: null,
    send(data: string) {
      socket.sent.push(data);
    },
    close() {},
    open() {
      socket.onopen?.();
    },
    receive(frame: unknown) {
      socket.onmessage?.({ data: JSON.stringify(frame) });
    },
    drop() {
      socket.onclose?.();
    },
  };
  return socket;
}

function sentFrames(socket: FakeSocket): Array<Record<string, unknown>> {
  return socket.sent.map(raw => JSON.parse(raw) as Record<string, unknown>);
}

interface ApplyFrame extends Record<string, unknown> {
  op: "apply";
  req: string;
  ops: Array<{ key: string; value?: unknown }>;
}

function isApplyFrame(frame: Record<string, unknown>): frame is ApplyFrame {
  return (
    frame.op === "apply" &&
    typeof frame.req === "string" &&
    Array.isArray(frame.ops) &&
    frame.ops.every(op => {
      return typeof op === "object" && op !== null && "key" in op && typeof op.key === "string";
    })
  );
}

function applyFrames(socket: FakeSocket): ApplyFrame[] {
  return sentFrames(socket).filter(isApplyFrame);
}

function requireLatestApplyFrame(socket: FakeSocket): ApplyFrame {
  const frame = applyFrames(socket).at(-1);
  if (!frame) throw new Error("expected the client to send an apply frame");
  return frame;
}

function entry(
  key: string,
  seq: number,
  value: Record<string, unknown> | null,
  overrides: Partial<OsStateEntry> = {}
): OsStateEntry {
  return {
    key,
    value,
    rev: 1,
    seq,
    deleted: value === null,
    updated_at: "2026-07-20T00:00:00Z",
    ...overrides,
  };
}

function windowValue(x: number): Record<string, unknown> {
  return {
    v: 1,
    app: "tasks",
    instanceKey: null,
    location: { pathname: "/tasks", search: {} },
    rect: { x, y: 10, w: 400, h: 300 },
    prevRect: null,
    z: 1,
    minimized: false,
    maximized: false,
  };
}

describe("OsStateClient", () => {
  let sockets: FakeSocket[];
  let store: ReturnType<typeof createDesktopStore>;
  let client: OsStateClient;

  beforeEach(() => {
    vi.useFakeTimers();
    sockets = [];
    store = createDesktopStore();
    const persistence = createDesktopPersistence(store);
    client = new OsStateClient({
      workspaceId: "w1",
      socketFactory: () => {
        const socket = createFakeSocket();
        sockets.push(socket);
        return socket;
      },
      callbacks: persistence.callbacks,
    });
    persistence.bind(client);
  });

  afterEach(() => {
    client.stop();
    vi.useRealTimers();
  });

  function bootLive(entries: OsStateEntry[] = [], asOfSeq = 10): FakeSocket {
    client.start();
    const socket = sockets.at(-1);
    if (!socket) throw new Error("no socket created");
    socket.open();
    socket.receive({ op: "snapshot", as_of_seq: asOfSeq, entries });
    return socket;
  }

  it("Should drop events at or below the snapshot fence and apply later ones (UT-052, invariant 3)", () => {
    const socket = bootLive([entry("win:app:tasks", 8, windowValue(10))], 10);
    expect(store.getState().windows["app:tasks"].rect.x).toBe(10);

    socket.receive({
      op: "event",
      entry: entry("win:app:tasks", 10, windowValue(55), { rev: 2 }),
      origin: "conn-b",
    });
    expect(store.getState().windows["app:tasks"].rect.x).toBe(10);

    socket.receive({
      op: "event",
      entry: entry("win:app:tasks", 11, windowValue(77), { rev: 3 }),
      origin: "conn-b",
    });
    expect(store.getState().windows["app:tasks"].rect.x).toBe(77);
  });

  it("Should settle an acked own write and ignore its stale echo (UT-053, invariant 6)", () => {
    const socket = bootLive();
    store.getState().openOrFocus({ app: "tasks" });
    const apply = requireLatestApplyFrame(socket);

    socket.receive({
      op: "ack",
      req: apply.req,
      results: apply.ops.map(op => ({
        key: op.key,
        rev: 2,
        seq: 12,
      })),
    });

    const localRect = store.getState().windows["app:tasks"].rect;
    // A late echo of our own write (same commit seq, origin tagged) must not
    // reapply over the settled state.
    socket.receive({
      op: "event",
      entry: entry("win:app:tasks", 12, windowValue(999), { rev: 2 }),
      origin: "conn-self",
    });
    expect(store.getState().windows["app:tasks"].rect).toEqual(localRect);
  });

  it("Should send exactly one trailing apply for rapid rect commits and flush immediately on demand (UT-054, invariant 15)", () => {
    const socket = bootLive([entry("win:app:tasks", 5, windowValue(10))], 10);
    const before = applyFrames(socket).length;

    store.getState().commitRect("app:tasks", { x: 20, y: 10, w: 400, h: 300 });
    vi.advanceTimersByTime(100);
    store.getState().commitRect("app:tasks", { x: 30, y: 10, w: 400, h: 300 });
    vi.advanceTimersByTime(100);
    store.getState().commitRect("app:tasks", { x: 40, y: 10, w: 400, h: 300 });
    expect(applyFrames(socket).length).toBe(before);

    vi.advanceTimersByTime(250);
    const frames = applyFrames(socket);
    expect(frames.length).toBe(before + 1);
    const ops = requireLatestApplyFrame(socket).ops;
    expect(ops).toHaveLength(1);
    expect(ops[0].key).toBe("win:app:tasks");
    expect(ops[0].value).toMatchObject({ rect: { x: 40 } });

    store.getState().commitRect("app:tasks", { x: 50, y: 10, w: 400, h: 300 });
    client.flush();
    const afterFlush = applyFrames(socket);
    expect(afterFlush.length).toBe(before + 2);
  });

  it("Should re-sub after a drop and adopt the fresh snapshot as authority (UT-055, invariant 10)", () => {
    const socket = bootLive([entry("win:app:tasks", 5, windowValue(10))], 10);
    expect(sentFrames(socket).some(frame => frame.op === "sub")).toBe(true);

    socket.drop();
    expect(store.getState().hydration).toBe("degraded");

    vi.advanceTimersByTime(1000);
    const next = sockets.at(-1);
    expect(next).not.toBe(socket);
    next?.open();
    expect(sentFrames(next as FakeSocket).some(frame => frame.op === "sub")).toBe(true);

    next?.receive({
      op: "snapshot",
      as_of_seq: 20,
      entries: [entry("win:app:tasks", 18, windowValue(300), { rev: 4 })],
    });
    expect(store.getState().hydration).toBe("live");
    expect(store.getState().windows["app:tasks"].rect.x).toBe(300);
  });

  it("Should degrade when the socket factory rejects, keep local mutations working, and retry with backoff (UT-056, invariant 16)", () => {
    const failing = new OsStateClient({
      workspaceId: "w1",
      socketFactory: () => {
        throw new Error("connection refused");
      },
      callbacks: createDesktopPersistence(store).callbacks,
    });
    failing.start();

    expect(store.getState().hydration).toBe("degraded");
    store.getState().openOrFocus({ app: "tasks" });
    expect(store.getState().windows["app:tasks"]).toBeDefined();

    // Backoff schedule keeps retrying (1s, then 2s).
    const attempts: number[] = [];
    const probe = new OsStateClient({
      workspaceId: "w1",
      socketFactory: () => {
        attempts.push(Date.now());
        throw new Error("still down");
      },
      callbacks: createDesktopPersistence(createDesktopStore()).callbacks,
    });
    probe.start();
    expect(attempts.length).toBe(1);
    vi.advanceTimersByTime(1000);
    expect(attempts.length).toBe(2);
    vi.advanceTimersByTime(2000);
    expect(attempts.length).toBe(3);
    probe.stop();
    failing.stop();
  });

  it("Should buffer remote events on pending keys and settle by seq after the ack (UT-074, invariant 6)", () => {
    const socket = bootLive([entry("win:app:tasks", 5, windowValue(10))], 10);
    store.getState().commitRect("app:tasks", { x: 20, y: 10, w: 400, h: 300 });
    client.flush();
    const apply = requireLatestApplyFrame(socket);
    expect(apply).toBeDefined();

    // A remote event on the pending key arrives before our ack: buffered.
    socket.receive({
      op: "event",
      entry: entry("win:app:tasks", 12, windowValue(500), { rev: 3 }),
      origin: "conn-b",
    });
    expect(store.getState().windows["app:tasks"].rect.x).toBe(20);

    // Ack lands with seq 11 < buffered 12 → the buffered remote wins (LWW by seq).
    socket.receive({
      op: "ack",
      req: apply.req,
      results: [{ key: "win:app:tasks", rev: 2, seq: 11 }],
    });
    expect(store.getState().windows["app:tasks"].rect.x).toBe(500);
  });

  it("Should drop a buffered remote event that the ack out-sequences (no oscillation)", () => {
    const socket = bootLive([entry("win:app:tasks", 5, windowValue(10))], 10);
    store.getState().commitRect("app:tasks", { x: 20, y: 10, w: 400, h: 300 });
    client.flush();
    const apply = requireLatestApplyFrame(socket);

    socket.receive({
      op: "event",
      entry: entry("win:app:tasks", 11, windowValue(500), { rev: 3 }),
      origin: "conn-b",
    });
    socket.receive({
      op: "ack",
      req: apply.req,
      results: [{ key: "win:app:tasks", rev: 4, seq: 13 }],
    });

    expect(store.getState().windows["app:tasks"].rect.x).toBe(20);
  });

  it("Should replay keys modified while degraded as one batch after snapshot adoption (UT-075, invariant 10)", () => {
    const socket = bootLive(
      [
        entry("win:app:settings", 4, {
          ...windowValue(60),
          app: "settings",
          location: { pathname: "/settings", search: {} },
        }),
      ],
      10
    );
    socket.drop();
    expect(store.getState().hydration).toBe("degraded");

    // Local changes while degraded: open tasks + move it.
    store.getState().openOrFocus({ app: "tasks" });
    store.getState().commitRect("app:tasks", { x: 111, y: 22, w: 400, h: 300 });

    vi.advanceTimersByTime(1000);
    const next = sockets.at(-1) as FakeSocket;
    next.open();
    next.receive({
      op: "snapshot",
      as_of_seq: 30,
      entries: [
        entry("win:app:settings", 28, {
          ...windowValue(90),
          app: "settings",
          location: { pathname: "/settings", search: {} },
        }),
      ],
    });

    // Untouched keys adopt daemon truth; touched keys keep the local value.
    expect(store.getState().windows["app:settings"].rect.x).toBe(90);
    expect(store.getState().windows["app:tasks"].rect.x).toBe(111);

    // The degraded keys replay as ONE apply batch.
    const replay = requireLatestApplyFrame(next);
    const keys = replay.ops.map(op => op.key).sort();
    expect(keys).toContain("win:app:tasks");
    expect(new Set(keys).size).toBe(keys.length);
  });
});
