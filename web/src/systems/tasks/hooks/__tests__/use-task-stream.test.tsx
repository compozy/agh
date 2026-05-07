import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { buildTaskStreamUrl, useTaskStream } from "@/systems/tasks";
import type { TaskStreamEventSource, TaskStreamPayload } from "@/systems/tasks";

type TaskStreamEventSourceLike = TaskStreamEventSource;

class FakeTaskStreamEventSource {
  public close = vi.fn();
  public onmessage: ((event: MessageEvent) => void) | null = null;
  public onerror: ((event: Event) => void) | null = null;
  public readonly addEventListener = vi.fn(this.recordAddListener.bind(this));
  public readonly removeEventListener = vi.fn(this.recordRemoveListener.bind(this));
  private readonly listeners = new Map<string, EventListenerOrEventListenerObject[]>();

  private recordAddListener(type: string, listener: EventListenerOrEventListenerObject) {
    const current = this.listeners.get(type) ?? [];
    current.push(listener);
    this.listeners.set(type, current);
  }

  private recordRemoveListener(type: string, listener: EventListenerOrEventListenerObject) {
    const current = this.listeners.get(type) ?? [];
    this.listeners.set(
      type,
      current.filter(candidate => candidate !== listener)
    );
  }

  hasListener(type: string): boolean {
    const current = this.listeners.get(type) ?? [];
    return current.length > 0;
  }

  emitMessage(data: unknown) {
    if (!this.onmessage) {
      return;
    }
    const event = new MessageEvent("message", {
      data: typeof data === "string" ? data : JSON.stringify(data),
    });
    this.onmessage(event);
  }

  emitNamed(type: string, data: unknown) {
    const current = this.listeners.get(type) ?? [];
    if (current.length === 0) {
      return;
    }
    const event = new MessageEvent(type, {
      data: typeof data === "string" ? data : JSON.stringify(data),
    });
    for (const listener of current) {
      if (typeof listener === "function") {
        listener(event);
      } else if (listener && typeof listener.handleEvent === "function") {
        listener.handleEvent(event);
      }
    }
  }

  emitError(event?: Event) {
    if (!this.onerror) {
      return;
    }
    this.onerror(event ?? new Event("error"));
  }
}

function createNamedOnlySource(): {
  source: TaskStreamEventSourceLike;
  hasListener: (type: string) => boolean;
} {
  const listeners = new Map<string, EventListenerOrEventListenerObject[]>();
  const source: TaskStreamEventSourceLike = {
    onmessage: null,
    onerror: null,
    close: vi.fn(),
    addEventListener: (type, listener) => {
      const current = listeners.get(type) ?? [];
      current.push(listener);
      listeners.set(type, current);
    },
    // Intentionally omit removeEventListener to exercise the cleanup-without-removeEventListener
    // branch of useTaskStream. Cleanup must still close the source and clear onmessage/onerror.
  };
  return {
    source,
    hasListener: type => (listeners.get(type) ?? []).length > 0,
  };
}

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

function buildStreamPayload(overrides: Partial<TaskStreamPayload> = {}): TaskStreamPayload {
  const base: TaskStreamPayload = {
    sequence: 25,
    type: "task.run_started",
    timeline: {
      event_id: "evt_25",
      sequence: 25,
      timestamp: "2026-04-17T18:05:00Z",
      event_type: "task.run_started",
      origin: { kind: "agent_session", ref: "session_a" },
      actor: { kind: "agent_session", ref: "session_a" },
    },
  } as TaskStreamPayload;
  return { ...base, ...overrides };
}

describe("buildTaskStreamUrl", () => {
  it("Should build a stream URL with after_sequence query when seed is provided", () => {
    expect(buildTaskStreamUrl("task_001", { after_sequence: 14 })).toBe(
      "/api/tasks/task_001/stream?after_sequence=14"
    );
  });

  it("Should include after_sequence=0 to opt into deterministic Last-Event-ID:0 precedence", () => {
    expect(buildTaskStreamUrl("task_001", { after_sequence: 0 })).toBe(
      "/api/tasks/task_001/stream?after_sequence=0"
    );
  });

  it("Should omit query string when no seed is provided", () => {
    expect(buildTaskStreamUrl("task_001")).toBe("/api/tasks/task_001/stream");
  });

  it("Should encode unsafe characters in the task id segment", () => {
    expect(buildTaskStreamUrl("task with spaces", { after_sequence: 1 })).toBe(
      "/api/tasks/task%20with%20spaces/stream?after_sequence=1"
    );
  });

  it("Should reject empty task ids", () => {
    expect(() => buildTaskStreamUrl("")).toThrow(/task id is required/);
    expect(() => buildTaskStreamUrl("   ")).toThrow(/task id is required/);
  });
});

describe("useTaskStream", () => {
  it("Should open a stream URL seeded with after_sequence and parse named SSE events", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const eventSource = new FakeTaskStreamEventSource();
    const factory = vi.fn(() => eventSource);
    const onEvent = vi.fn();

    const { unmount } = renderHook(
      () =>
        useTaskStream("task_001", {
          afterSequence: 14,
          eventSourceFactory: factory,
          onEvent,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    expect(factory).toHaveBeenCalledTimes(1);
    expect(factory).toHaveBeenCalledWith("/api/tasks/task_001/stream?after_sequence=14");

    expect(eventSource.hasListener("task.run_started")).toBe(true);
    expect(eventSource.hasListener("task.run_review_requested")).toBe(true);
    expect(eventSource.hasListener("task.notification_delivered")).toBe(true);

    const payload = buildStreamPayload();

    act(() => {
      eventSource.emitNamed("task.run_started", payload);
    });

    await waitFor(() => {
      expect(onEvent).toHaveBeenCalledWith(payload);
    });

    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["tasks", "detail", "task_001"],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["tasks", "timeline"],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["tasks", "context-bundle"],
    });

    unmount();
    expect(eventSource.close).toHaveBeenCalledTimes(1);
    expect(eventSource.onmessage).toBeNull();
    expect(eventSource.onerror).toBeNull();
    expect(eventSource.removeEventListener).toHaveBeenCalledWith(
      "task.run_started",
      expect.any(Function)
    );
    expect(eventSource.removeEventListener).toHaveBeenCalledWith(
      "task.run_review_requested",
      expect.any(Function)
    );
    expect(eventSource.removeEventListener).toHaveBeenCalledWith(
      "task.notification_delivered",
      expect.any(Function)
    );
    expect(eventSource.hasListener("task.run_started")).toBe(false);
    expect(eventSource.hasListener("task.run_review_requested")).toBe(false);
    expect(eventSource.hasListener("task.notification_delivered")).toBe(false);
  });

  it("Should still parse defensive unnamed message frames via onmessage", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const eventSource = new FakeTaskStreamEventSource();
    const factory = vi.fn(() => eventSource);
    const onEvent = vi.fn();

    renderHook(
      () =>
        useTaskStream("task_001", {
          afterSequence: 14,
          eventSourceFactory: factory,
          onEvent,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    const payload = buildStreamPayload();

    act(() => {
      eventSource.emitMessage(payload);
    });

    await waitFor(() => {
      expect(onEvent).toHaveBeenCalledWith(payload);
    });
  });

  it("Should close the source on cleanup even when removeEventListener is missing", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const namedOnly = createNamedOnlySource();
    const factory = vi.fn(() => namedOnly.source);

    const { unmount } = renderHook(
      () =>
        useTaskStream("task_001", {
          afterSequence: 4,
          eventSourceFactory: factory,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    expect(namedOnly.hasListener("task.run_started")).toBe(true);

    expect(() => unmount()).not.toThrow();
    expect(namedOnly.source.close).toHaveBeenCalledTimes(1);
    expect(namedOnly.source.onmessage).toBeNull();
    expect(namedOnly.source.onerror).toBeNull();
  });

  it("Should not open a source when disabled", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const factory = vi.fn(() => new FakeTaskStreamEventSource());

    renderHook(
      () =>
        useTaskStream("task_001", {
          enabled: false,
          afterSequence: 1,
          eventSourceFactory: factory,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    expect(factory).not.toHaveBeenCalled();
  });

  it("Should not open a source for an empty task id", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const factory = vi.fn(() => new FakeTaskStreamEventSource());

    renderHook(
      () =>
        useTaskStream("   ", {
          afterSequence: 1,
          eventSourceFactory: factory,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    expect(factory).not.toHaveBeenCalled();
  });

  it("Should call onError when a named SSE payload cannot be parsed", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const eventSource = new FakeTaskStreamEventSource();
    const factory = vi.fn(() => eventSource);
    const onError = vi.fn();
    const onEvent = vi.fn();

    renderHook(
      () =>
        useTaskStream("task_001", {
          afterSequence: 7,
          eventSourceFactory: factory,
          onEvent,
          onError,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    expect(() => {
      act(() => {
        eventSource.emitNamed("task.run_started", "{not valid json");
      });
    }).not.toThrow();

    expect(onError).toHaveBeenCalledTimes(1);
    expect(onEvent).not.toHaveBeenCalled();
  });

  it("Should forward connection errors to onError", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const eventSource = new FakeTaskStreamEventSource();
    const factory = vi.fn(() => eventSource);
    const onError = vi.fn();

    renderHook(
      () =>
        useTaskStream("task_001", {
          afterSequence: 1,
          eventSourceFactory: factory,
          onError,
        }),
      { wrapper: createWrapper(queryClient) }
    );

    const errorEvent = new Event("error");
    act(() => {
      eventSource.emitError(errorEvent);
    });

    expect(onError).toHaveBeenCalledWith(errorEvent);
  });

  it("Should reuse the same source when re-rendered with the same inputs", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const factory = vi.fn(() => new FakeTaskStreamEventSource());

    const { rerender } = renderHook(
      ({ seq }: { seq: number }) =>
        useTaskStream("task_001", {
          afterSequence: seq,
          eventSourceFactory: factory,
        }),
      {
        wrapper: createWrapper(queryClient),
        initialProps: { seq: 14 },
      }
    );

    expect(factory).toHaveBeenCalledTimes(1);

    rerender({ seq: 14 });
    expect(factory).toHaveBeenCalledTimes(1);

    rerender({ seq: 22 });
    expect(factory).toHaveBeenCalledTimes(2);
  });
});
