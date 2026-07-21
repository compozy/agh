import type { components } from "@/generated/agh-openapi";

import {
  OS_RECT_DEBOUNCE_MS,
  type OsHydration,
  type OsStateEntry,
  type OsStateEvent,
} from "./os-types";

type SnapshotFrame = components["schemas"]["DesktopStateSnapshotFrame"];
type EventFrame = components["schemas"]["DesktopStateEventFrame"];
type AckFrame = components["schemas"]["DesktopStateAckFrame"];
type ErrorFrame = components["schemas"]["DesktopStateErrorFrame"];
type ApplyOp = components["schemas"]["DesktopStateApplyOp"];

type ServerFrame = SnapshotFrame | EventFrame | AckFrame | ErrorFrame | { op: "pong" };

/** Injectable socket surface (mirrors the DOM WebSocket the shell provides). */
export interface OsSocket {
  send(data: string): void;
  close(): void;
  onopen: (() => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: (() => void) | null;
  onerror: (() => void) | null;
}

export type OsSocketFactory = (url: string) => OsSocket;

export interface OsStateClientCallbacks {
  /** Snapshot adoption; `preserveKeys` are keys locally modified while offline. */
  onSnapshot(entries: OsStateEntry[], asOfSeq: number, preserveKeys: ReadonlySet<string>): void;
  onEvent(event: OsStateEvent): void;
  onHydration(state: OsHydration): void;
  /** Settles a key at its acked commit seq (LWW guard feed). */
  onSettled(key: string, seq: number): void;
  /** Current local payload for a key; null when the key no longer exists. */
  readLocal(key: string): Record<string, unknown> | null;
}

export interface OsStateClientOptions {
  workspaceId: string;
  socketFactory: OsSocketFactory;
  callbacks: OsStateClientCallbacks;
  debounceMs?: number;
  /** Backoff schedule in ms; the last value repeats. */
  backoffMs?: number[];
}

const DEFAULT_BACKOFF_MS = [1000, 2000, 4000, 8000, 16000, 30000];

interface PendingApply {
  keys: string[];
  kinds: Map<string, "put" | "delete">;
}

/**
 * WebSocket client for the `os_shell` desktop-state domain. Owns the snapshot
 * fence (invariant 3), `req`-tracked pending mutations with seq-settled LWW
 * (invariant 6), the 250ms trailing rect debounce with explicit flush
 * (invariant 15), degraded mode with backoff (invariant 16), and
 * snapshot-authority reconnect with degraded-keys replay (invariant 10).
 */
export class OsStateClient {
  private readonly options: Required<Pick<OsStateClientOptions, "debounceMs" | "backoffMs">> &
    OsStateClientOptions;
  private socket: OsSocket | null = null;
  private phase: "idle" | "connecting" | "syncing" | "live" | "stopped" = "idle";
  private asOfSeq = 0;
  private reqCounter = 0;
  private retryAttempt = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;
  private readonly dirtyKeys = new Set<string>();
  private readonly pendingApplies = new Map<string, PendingApply>();
  private readonly pendingKeyCounts = new Map<string, number>();
  private readonly bufferedEvents = new Map<string, OsStateEvent>();
  private readonly offlineDirty = new Map<string, "put" | "delete">();

  constructor(options: OsStateClientOptions) {
    this.options = {
      debounceMs: OS_RECT_DEBOUNCE_MS,
      backoffMs: DEFAULT_BACKOFF_MS,
      ...options,
    };
  }

  start(): void {
    if (this.phase !== "idle" && this.phase !== "stopped") return;
    this.phase = "idle";
    this.connect();
  }

  stop(): void {
    this.phase = "stopped";
    this.clearTimers();
    this.teardownSocket();
  }

  get live(): boolean {
    return this.phase === "live";
  }

  /** Debounced put (250ms trailing); the value is read from local state at send time. */
  schedulePut(key: string): void {
    if (this.phase !== "live") {
      this.offlineDirty.set(key, "put");
      return;
    }
    this.dirtyKeys.add(key);
    if (this.debounceTimer !== null) clearTimeout(this.debounceTimer);
    this.debounceTimer = setTimeout(() => {
      this.debounceTimer = null;
      this.sendDirty();
    }, this.options.debounceMs);
  }

  /** Immediate atomic batch for invariant-spanning actions (never split writes). */
  applyNow(ops: ApplyOp[]): void {
    if (ops.length === 0) return;
    if (this.phase !== "live") {
      for (const op of ops) this.offlineDirty.set(op.key, op.kind);
      return;
    }
    for (const op of ops) this.dirtyKeys.delete(op.key);
    this.sendApply(ops);
  }

  /** Gesture-end / beforeunload flush: pending debounced writes send now. */
  flush(): void {
    if (this.debounceTimer !== null) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }
    this.sendDirty();
  }

  private sendDirty(): void {
    if (this.phase !== "live" || this.dirtyKeys.size === 0) return;
    const ops: ApplyOp[] = [];
    for (const key of this.dirtyKeys) {
      const value = this.options.callbacks.readLocal(key);
      if (value !== null) ops.push({ kind: "put", key, value });
    }
    this.dirtyKeys.clear();
    if (ops.length > 0) this.sendApply(ops);
  }

  private sendApply(ops: ApplyOp[]): void {
    const socket = this.socket;
    if (!socket) return;
    this.reqCounter += 1;
    const req = `req-${this.reqCounter}`;
    const kinds = new Map<string, "put" | "delete">();
    for (const op of ops) {
      kinds.set(op.key, op.kind);
      this.pendingKeyCounts.set(op.key, (this.pendingKeyCounts.get(op.key) ?? 0) + 1);
    }
    this.pendingApplies.set(req, { keys: ops.map(op => op.key), kinds });
    socket.send(JSON.stringify({ op: "apply", req, ops }));
  }

  private connect(): void {
    if (this.phase === "stopped") return;
    this.phase = "connecting";
    let socket: OsSocket;
    try {
      socket = this.options.socketFactory(this.streamUrl());
    } catch {
      this.enterDegraded();
      return;
    }
    this.socket = socket;
    socket.onopen = () => {
      if (this.socket !== socket) return;
      this.phase = "syncing";
      socket.send(JSON.stringify({ op: "sub" }));
    };
    socket.onmessage = event => {
      if (this.socket !== socket) return;
      this.handleFrame(event.data);
    };
    const onDrop = () => {
      if (this.socket !== socket) return;
      this.handleDisconnect();
    };
    socket.onclose = onDrop;
    socket.onerror = onDrop;
  }

  private streamUrl(): string {
    return `/api/workspaces/${encodeURIComponent(this.options.workspaceId)}/desktop-state/stream`;
  }

  private handleFrame(data: unknown): void {
    if (typeof data !== "string") return;
    let frame: ServerFrame;
    try {
      frame = JSON.parse(data) as ServerFrame;
    } catch {
      return;
    }
    switch (frame.op) {
      case "snapshot":
        this.handleSnapshot(frame);
        return;
      case "event":
        this.handleEvent(frame);
        return;
      case "ack":
        this.handleAck(frame);
        return;
      case "error":
        this.handleError(frame);
        return;
      case "pong":
        return;
    }
  }

  private handleSnapshot(frame: SnapshotFrame): void {
    this.asOfSeq = frame.as_of_seq;
    this.phase = "live";
    this.retryAttempt = 0;
    const preserve = new Set(this.offlineDirty.keys());
    this.options.callbacks.onSnapshot(frame.entries, frame.as_of_seq, preserve);
    this.options.callbacks.onHydration("live");
    this.replayOfflineDirty();
  }

  /** Re-applies keys modified while offline as ONE batch (invariant 10). */
  private replayOfflineDirty(): void {
    if (this.offlineDirty.size === 0) return;
    const ops: ApplyOp[] = [];
    for (const [key, kind] of this.offlineDirty) {
      if (kind === "delete") {
        ops.push({ kind: "delete", key });
        continue;
      }
      const value = this.options.callbacks.readLocal(key);
      if (value !== null) ops.push({ kind: "put", key, value });
    }
    this.offlineDirty.clear();
    if (ops.length > 0) this.sendApply(ops);
  }

  private handleEvent(frame: EventFrame): void {
    if (frame.entry.seq <= this.asOfSeq) return;
    const key = frame.entry.key;
    const event: OsStateEvent = { entry: frame.entry, origin: frame.origin };
    if ((this.pendingKeyCounts.get(key) ?? 0) > 0) {
      const buffered = this.bufferedEvents.get(key);
      if (!buffered || frame.entry.seq > buffered.entry.seq) {
        this.bufferedEvents.set(key, event);
      }
      return;
    }
    this.options.callbacks.onEvent(event);
  }

  private handleAck(frame: AckFrame): void {
    const pending = this.pendingApplies.get(frame.req);
    if (!pending) return;
    this.pendingApplies.delete(frame.req);
    const ackSeqs = new Map(frame.results.map(result => [result.key, result.seq]));
    for (const key of pending.keys) {
      this.releasePendingKey(key);
      const seq = ackSeqs.get(key);
      if (seq !== undefined) this.options.callbacks.onSettled(key, seq);
      this.settleBuffered(key, seq ?? 0);
    }
  }

  private handleError(frame: ErrorFrame): void {
    if (!frame.req) return;
    const pending = this.pendingApplies.get(frame.req);
    if (!pending) return;
    this.pendingApplies.delete(frame.req);
    for (const key of pending.keys) {
      this.releasePendingKey(key);
      this.settleBuffered(key, 0);
    }
  }

  /** Delivers a buffered remote event only when it out-sequences the acked write. */
  private settleBuffered(key: string, ackSeq: number): void {
    if ((this.pendingKeyCounts.get(key) ?? 0) > 0) return;
    const buffered = this.bufferedEvents.get(key);
    if (!buffered) return;
    this.bufferedEvents.delete(key);
    if (buffered.entry.seq > ackSeq) this.options.callbacks.onEvent(buffered);
  }

  private releasePendingKey(key: string): void {
    const count = this.pendingKeyCounts.get(key) ?? 0;
    if (count <= 1) this.pendingKeyCounts.delete(key);
    else this.pendingKeyCounts.set(key, count - 1);
  }

  private handleDisconnect(): void {
    this.teardownSocket();
    if (this.phase === "stopped") return;
    // Unacked in-flight writes may or may not have committed; replaying them
    // after the next snapshot keeps "what I see is what persists" (US-013).
    for (const [, pending] of this.pendingApplies) {
      for (const key of pending.keys) {
        this.offlineDirty.set(key, pending.kinds.get(key) ?? "put");
      }
    }
    for (const key of this.dirtyKeys) this.offlineDirty.set(key, "put");
    this.dirtyKeys.clear();
    this.pendingApplies.clear();
    this.pendingKeyCounts.clear();
    this.bufferedEvents.clear();
    this.enterDegraded();
  }

  private enterDegraded(): void {
    this.phase = "idle";
    this.options.callbacks.onHydration("degraded");
    const schedule = this.options.backoffMs;
    const delay = schedule[Math.min(this.retryAttempt, schedule.length - 1)];
    this.retryAttempt += 1;
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      this.connect();
    }, delay);
  }

  private clearTimers(): void {
    if (this.retryTimer !== null) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
    if (this.debounceTimer !== null) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }
  }

  private teardownSocket(): void {
    const socket = this.socket;
    if (!socket) return;
    this.socket = null;
    socket.onopen = null;
    socket.onmessage = null;
    socket.onclose = null;
    socket.onerror = null;
    try {
      socket.close();
    } catch {
      // A socket failing to close while tearing down has nothing left to leak.
    }
  }
}
