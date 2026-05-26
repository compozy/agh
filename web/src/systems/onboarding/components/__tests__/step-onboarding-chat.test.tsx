import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { OnboardingChatApi } from "../../hooks/use-onboarding-chat";
import { StepOnboardingChat } from "../step-onboarding-chat";

const mocks = vi.hoisted(() => ({
  append: vi.fn(),
  cancelSessionPrompt: vi.fn(),
  threadListItem: {
    id: "sess_onboarding",
    remoteId: "sess_onboarding",
    status: "regular",
  } as {
    id: string;
    remoteId?: string;
    status: "new" | "regular" | "archived";
  },
}));

vi.mock("@assistant-ui/react", () => ({
  useAui: () => ({
    thread: () => ({ append: mocks.append }),
  }),
  useAuiState: <T,>(selector: (state: { threadListItem: typeof mocks.threadListItem }) => T) =>
    selector({ threadListItem: mocks.threadListItem }),
}));

vi.mock("@/components/assistant-ui/session-thread", () => ({
  SessionThread: ({
    canPrompt,
    contentInset,
    onCancelPrompt,
  }: {
    canPrompt: boolean;
    contentInset?: string;
    onCancelPrompt: () => void;
  }) => (
    <div
      data-can-prompt={String(canPrompt)}
      data-content-inset={contentInset ?? ""}
      data-testid="session-thread"
    >
      <button data-testid="session-thread-cancel" onClick={onCancelPrompt} type="button">
        Cancel prompt
      </button>
    </div>
  ),
}));

vi.mock("@/systems/session", () => ({
  cancelSessionPrompt: mocks.cancelSessionPrompt,
  SessionChatRuntimeProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

function chat(overrides: Partial<OnboardingChatApi> = {}): OnboardingChatApi {
  return {
    session: {
      sessionId: "sess_onboarding",
      workspaceId: "ws_alpha",
      canPrompt: true,
      recoveryMessage: null,
      canRestart: false,
    },
    kickoffSessionId: "",
    isCreating: false,
    error: null,
    ensureSession: vi.fn().mockResolvedValue(undefined),
    retry: vi.fn().mockResolvedValue(undefined),
    markKickoffSent: vi.fn(),
    reportError: vi.fn(),
    ...overrides,
  };
}

describe("StepOnboardingChat", () => {
  beforeEach(() => {
    mocks.append.mockReset();
    mocks.append.mockReturnValue(undefined);
    mocks.cancelSessionPrompt.mockReset();
    mocks.cancelSessionPrompt.mockResolvedValue(undefined);
    mocks.threadListItem = {
      id: "sess_onboarding",
      remoteId: "sess_onboarding",
      status: "regular",
    };
  });

  it("does not send the kickoff while assistant-ui is still on a local new thread", async () => {
    const markKickoffSent = vi.fn();
    mocks.threadListItem = { id: "__LOCALID_pending", status: "new" };

    render(<StepOnboardingChat chat={chat({ markKickoffSent })} />);

    await waitFor(() => expect(screen.getByTestId("session-thread-cancel")).toBeInTheDocument());
    expect(mocks.append).not.toHaveBeenCalled();
    expect(markKickoffSent).not.toHaveBeenCalled();
  });

  it("does not send the kickoff when the assistant-ui thread maps to another session", async () => {
    const markKickoffSent = vi.fn();
    mocks.threadListItem = {
      id: "sess_other",
      remoteId: "sess_other",
      status: "regular",
    };

    render(<StepOnboardingChat chat={chat({ markKickoffSent })} />);

    await waitFor(() => expect(screen.getByTestId("session-thread-cancel")).toBeInTheDocument());
    expect(mocks.append).not.toHaveBeenCalled();
    expect(markKickoffSent).not.toHaveBeenCalled();
  });

  it("passes onboarding content inset to SessionThread", async () => {
    render(<StepOnboardingChat chat={chat()} />);

    await waitFor(() => expect(screen.getByTestId("session-thread")).toBeInTheDocument());
    expect(screen.getByTestId("session-thread")).toHaveAttribute("data-content-inset", "px-8");
  });

  it("sends the kickoff once after assistant-ui maps to the real AGH session", async () => {
    const markKickoffSent = vi.fn();

    const view = render(<StepOnboardingChat chat={chat({ markKickoffSent })} />);

    await waitFor(() => expect(mocks.append).toHaveBeenCalledTimes(1));
    expect(markKickoffSent).toHaveBeenCalledWith("sess_onboarding");

    view.rerender(<StepOnboardingChat chat={chat({ markKickoffSent })} />);
    expect(mocks.append).toHaveBeenCalledTimes(1);
  });

  it("does not resend the kickoff for a persisted seeded session", async () => {
    render(<StepOnboardingChat chat={chat({ kickoffSessionId: "sess_onboarding" })} />);

    await waitFor(() => expect(screen.getByTestId("session-thread-cancel")).toBeInTheDocument());
    expect(mocks.append).not.toHaveBeenCalled();
  });

  it("keeps preserved stopped sessions mounted instead of blocking on an error panel", async () => {
    const retry = vi.fn().mockResolvedValue(undefined);

    render(
      <StepOnboardingChat
        chat={chat({
          retry,
          session: {
            sessionId: "sess_onboarding",
            workspaceId: "ws_alpha",
            canPrompt: false,
            recoveryMessage:
              "This onboarding session stopped before setup finished. The history below is preserved.",
            canRestart: true,
          },
        })}
      />
    );

    await waitFor(() => expect(screen.getByTestId("session-thread")).toBeInTheDocument());
    expect(screen.getByTestId("session-thread")).toHaveAttribute("data-can-prompt", "false");
    expect(screen.getByTestId("onboarding-chat-status")).toHaveTextContent(
      "This onboarding session stopped before setup finished. The history below is preserved."
    );
    expect(screen.queryByTestId("onboarding-chat-retry")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("onboarding-chat-restart"));
    expect(retry).toHaveBeenCalledTimes(1);
    expect(mocks.append).not.toHaveBeenCalled();
  });

  it("reports kickoff append failures", async () => {
    const reportError = vi.fn();
    mocks.append.mockImplementation(() => {
      throw new Error("transport failed");
    });

    render(<StepOnboardingChat chat={chat({ reportError })} />);

    await waitFor(() =>
      expect(reportError).toHaveBeenCalledWith(
        "Failed to send the onboarding kickoff message. transport failed"
      )
    );
  });

  it("reports async kickoff append failures", async () => {
    const reportError = vi.fn();
    mocks.append.mockRejectedValue(new Error("async transport failed"));

    render(<StepOnboardingChat chat={chat({ reportError })} />);

    await waitFor(() =>
      expect(reportError).toHaveBeenCalledWith(
        "Failed to send the onboarding kickoff message. async transport failed"
      )
    );
  });

  it("reports cancel prompt failures", async () => {
    const reportError = vi.fn();
    mocks.cancelSessionPrompt.mockRejectedValue(new Error("cancel failed"));

    render(<StepOnboardingChat chat={chat({ reportError })} />);
    fireEvent.click(screen.getByTestId("session-thread-cancel"));

    await waitFor(() =>
      expect(reportError).toHaveBeenCalledWith(
        "Failed to cancel the onboarding prompt. cancel failed"
      )
    );
  });
});
