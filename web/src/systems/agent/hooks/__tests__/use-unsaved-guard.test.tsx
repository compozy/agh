import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockUseBlocker, mockProceed, mockReset } = vi.hoisted(() => {
  const mockProceed = vi.fn();
  const mockReset = vi.fn();
  return {
    mockProceed,
    mockReset,
    mockUseBlocker: vi.fn(() => ({
      status: "idle" as "idle" | "blocked",
      proceed: mockProceed,
      reset: mockReset,
    })),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useBlocker: mockUseBlocker,
}));

import { useUnsavedGuard } from "../use-unsaved-guard";

describe("useUnsavedGuard", () => {
  beforeEach(() => {
    mockProceed.mockReset();
    mockReset.mockReset();
    mockUseBlocker.mockClear();
    mockUseBlocker.mockReturnValue({
      status: "idle",
      proceed: mockProceed,
      reset: mockReset,
    });
  });

  it("Should disable the blocker when pristine", () => {
    renderHook(() => useUnsavedGuard({ dirty: false, entityName: "coder" }));

    expect(mockUseBlocker).toHaveBeenCalledWith(
      expect.objectContaining({
        disabled: true,
        enableBeforeUnload: false,
        withResolver: true,
      })
    );
  });

  it("Should enable blocking when dirty and expose proceed/reset while blocked", () => {
    mockUseBlocker.mockReturnValue({
      status: "blocked",
      proceed: mockProceed,
      reset: mockReset,
    });

    const { result } = renderHook(() => useUnsavedGuard({ dirty: true, entityName: "coder" }));

    expect(mockUseBlocker).toHaveBeenCalledWith(
      expect.objectContaining({
        disabled: false,
        enableBeforeUnload: true,
        withResolver: true,
      })
    );
    expect(result.current.status).toBe("blocked");
    expect(result.current.proceed).toBeTypeOf("function");
    expect(result.current.reset).toBeTypeOf("function");

    act(() => {
      result.current.proceed?.();
      result.current.reset?.();
    });

    expect(mockProceed).toHaveBeenCalledTimes(1);
    expect(mockReset).toHaveBeenCalledTimes(1);
  });

  it("Should keep proceed/reset undefined when idle", () => {
    const { result } = renderHook(() => useUnsavedGuard({ dirty: true, entityName: "coder" }));

    expect(result.current.status).toBe("idle");
    expect(result.current.proceed).toBeUndefined();
    expect(result.current.reset).toBeUndefined();
  });
});
