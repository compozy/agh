import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const routeHookMocks = vi.hoisted(() => ({
  auiState: {
    thread: {
      isRunning: false,
      messages: [] as Array<{ id: string }>,
    },
  },
  resetThread: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  cancelSessionPrompt: vi.fn(),
  clearMutation: {
    isPending: false,
    mutate: vi.fn(),
  },
  deleteMutation: {
    isPending: false,
    mutate: vi.fn(),
  },
  resumeMutation: {
    isPending: false,
    mutate: vi.fn(),
  },
  queuePromptMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  interruptPromptMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  steerPromptMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  stopMutation: {
    isPending: false,
    mutate: vi.fn(),
  },
}));

vi.mock("@assistant-ui/react", () => ({
  useAui: () => ({
    thread: () => ({
      reset: routeHookMocks.resetThread,
    }),
  }),
  useAuiState: (selector: (state: typeof routeHookMocks.auiState) => unknown) =>
    selector(routeHookMocks.auiState),
}));

vi.mock("sonner", () => ({
  toast: {
    error: routeHookMocks.toastError,
    success: routeHookMocks.toastSuccess,
  },
}));

vi.mock("@/systems/session", () => ({
  cancelSessionPrompt: routeHookMocks.cancelSessionPrompt,
  useClearSessionConversation: () => routeHookMocks.clearMutation,
  useDeleteSession: () => routeHookMocks.deleteMutation,
  useInterruptSessionPrompt: () => routeHookMocks.interruptPromptMutation,
  useQueueSessionPrompt: () => routeHookMocks.queuePromptMutation,
  useResumeSession: () => routeHookMocks.resumeMutation,
  useSteerSessionPrompt: () => routeHookMocks.steerPromptMutation,
  useStopSession: () => routeHookMocks.stopMutation,
}));

import { useSessionPageControls } from "../use-session-page-controls";

const WORKSPACE_ID = "ws_alpha";

function renderControls(
  state: Parameters<typeof useSessionPageControls>[1],
  options: Parameters<typeof useSessionPageControls>[2] = {}
) {
  return renderHook(() =>
    useSessionPageControls("sess-1", state, { workspaceId: WORKSPACE_ID, ...options })
  );
}

function createDeferredPromise<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, reject, resolve };
}

describe("useSessionPageControls", () => {
  beforeEach(() => {
    routeHookMocks.auiState.thread.isRunning = false;
    routeHookMocks.auiState.thread.messages = [];
    routeHookMocks.resetThread.mockReset();
    routeHookMocks.toastError.mockReset();
    routeHookMocks.toastSuccess.mockReset();
    routeHookMocks.cancelSessionPrompt.mockReset();
    routeHookMocks.clearMutation.isPending = false;
    routeHookMocks.clearMutation.mutate.mockReset();
    routeHookMocks.deleteMutation.isPending = false;
    routeHookMocks.deleteMutation.mutate.mockReset();
    routeHookMocks.resumeMutation.isPending = false;
    routeHookMocks.resumeMutation.mutate.mockReset();
    routeHookMocks.queuePromptMutation.isPending = false;
    routeHookMocks.queuePromptMutation.mutateAsync.mockReset();
    routeHookMocks.interruptPromptMutation.isPending = false;
    routeHookMocks.interruptPromptMutation.mutateAsync.mockReset();
    routeHookMocks.steerPromptMutation.isPending = false;
    routeHookMocks.steerPromptMutation.mutateAsync.mockReset();
    routeHookMocks.stopMutation.isPending = false;
    routeHookMocks.stopMutation.mutate.mockReset();
  });

  it("blocks delete while prompt cancellation is in flight", async () => {
    const cancelPrompt = createDeferredPromise<void>();
    routeHookMocks.auiState.thread.isRunning = true;
    routeHookMocks.cancelSessionPrompt.mockReturnValue(cancelPrompt.promise);

    const { result } = renderControls("active");

    await act(async () => {
      result.current.handleCancelPrompt();
    });

    act(() => {
      result.current.handleDelete();
    });

    expect(routeHookMocks.deleteMutation.mutate).not.toHaveBeenCalled();
    expect(routeHookMocks.cancelSessionPrompt).toHaveBeenCalledWith(WORKSPACE_ID, "sess-1");

    await act(async () => {
      cancelPrompt.resolve();
      await cancelPrompt.promise;
    });
  });

  it("blocks clear while another control mutation is pending", () => {
    routeHookMocks.auiState.thread.messages = [{ id: "message-1" }];
    routeHookMocks.resumeMutation.isPending = true;

    const { result } = renderControls("active");

    act(() => {
      result.current.handleClear();
    });

    expect(routeHookMocks.clearMutation.mutate).not.toHaveBeenCalled();
  });

  it("blocks clear while a prompt is still running", () => {
    routeHookMocks.auiState.thread.isRunning = true;
    routeHookMocks.auiState.thread.messages = [{ id: "message-1" }];

    const { result } = renderControls("active");

    act(() => {
      result.current.handleClear();
    });

    expect(routeHookMocks.clearMutation.mutate).not.toHaveBeenCalled();
  });

  it("blocks stop while another control action is pending", () => {
    routeHookMocks.auiState.thread.isRunning = true;
    routeHookMocks.clearMutation.isPending = true;

    const { result } = renderControls("active");

    act(() => {
      result.current.handleStop();
    });

    expect(routeHookMocks.cancelSessionPrompt).not.toHaveBeenCalled();
    expect(routeHookMocks.stopMutation.mutate).not.toHaveBeenCalled();
  });

  it("blocks resume while another control action is pending", () => {
    routeHookMocks.deleteMutation.isPending = true;

    const { result } = renderControls("stopped");

    act(() => {
      result.current.handleResume();
    });

    expect(routeHookMocks.resumeMutation.mutate).not.toHaveBeenCalled();
  });

  it("runs delete success side effects when controls are idle", () => {
    const onDeleteSuccess = vi.fn();
    const { result } = renderControls("active", { onDeleteSuccess });

    act(() => {
      result.current.handleDelete();
    });

    expect(routeHookMocks.deleteMutation.mutate).toHaveBeenCalledTimes(1);
    const [, options] = routeHookMocks.deleteMutation.mutate.mock.calls[0] ?? [];
    expect(options).toEqual(
      expect.objectContaining({
        onError: expect.any(Function),
        onSuccess: expect.any(Function),
      })
    );

    act(() => {
      options.onSuccess();
    });

    expect(routeHookMocks.resetThread).toHaveBeenCalledTimes(1);
    expect(routeHookMocks.toastSuccess).toHaveBeenCalledWith("Session deleted.");
    expect(onDeleteSuccess).toHaveBeenCalledTimes(1);
  });

  it("shows the delete error toast from the mutation callback", () => {
    const { result } = renderControls("active");

    act(() => {
      result.current.handleDelete();
    });

    const [, options] = routeHookMocks.deleteMutation.mutate.mock.calls[0] ?? [];

    act(() => {
      options.onError(new Error("delete failed"));
    });

    expect(routeHookMocks.toastError).toHaveBeenCalledWith("delete failed");
  });
});
