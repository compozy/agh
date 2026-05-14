// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const navigateMock = vi.fn();
const threadDetailMock = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: React.ReactNode }) => (
    <a {...(rest as Record<string, unknown>)}>{children}</a>
  ),
  useNavigate: () => navigateMock,
}));

const messages = [
  {
    body: { text: "Root message" },
    channel: "ops",
    direction: "sent" as const,
    display_name: "Codex",
    kind: "say",
    local: true,
    message_id: "msg-root",
    peer_from: "peer-codex",
    preview_text: "Root",
    session_id: "sess-1",
    text: "Root message",
    timestamp: "2026-04-17T14:32:00Z",
  },
  {
    body: { text: "First reply" },
    channel: "ops",
    direction: "sent" as const,
    display_name: "Codex",
    kind: "say",
    local: true,
    message_id: "msg-reply-1",
    peer_from: "peer-codex",
    preview_text: "First reply",
    session_id: "sess-1",
    text: "First reply",
    timestamp: "2026-04-17T14:33:00Z",
  },
];

vi.mock("../../../hooks/use-threads", async () => {
  const actual = await vi.importActual<typeof import("../../../hooks/use-threads")>(
    "../../../hooks/use-threads"
  );
  return {
    ...actual,
    useNetworkThreadDetail: () => threadDetailMock(),
  };
});

vi.mock("../../../hooks/use-messages", () => ({
  useNetworkMessages: () => ({
    messages,
    isLoading: false,
    isFetching: false,
    error: null,
  }),
}));

vi.mock("../../../hooks/use-active-session", () => ({
  useActiveNetworkSession: () => ({
    session: {
      channel: "ops",
      peerId: "peer-codex",
      sessionId: "sess-1",
      displayName: "Codex",
    },
    disabledReason: null,
    isLoading: false,
  }),
}));

import { ThreadOverlay } from "../thread-overlay";

const WORKSPACE_ID = "ws_alpha";

function renderOverlay({ fullPage = false }: { fullPage?: boolean } = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ThreadOverlay
        workspaceId={WORKSPACE_ID}
        channel="ops"
        fullPage={fullPage}
        threadId="thread-test"
      />
    </QueryClientProvider>
  );
}

describe("ThreadOverlay", () => {
  beforeEach(() => {
    threadDetailMock.mockReturnValue({
      thread: {
        channel: "ops",
        last_activity_at: "2026-04-17T14:33:00Z",
        last_message_preview: "First reply",
        message_count: 2,
        open_work_count: 0,
        opened_at: "2026-04-17T14:32:00Z",
        opened_by_peer_id: "peer-codex",
        opened_session_id: "sess-1",
        participant_count: 2,
        root_message_id: "msg-root",
        thread_id: "thread-test",
        title: "Launch command brief",
      },
      isLoading: false,
      error: null,
    });
  });

  it("Should render the thread title, root, and reply count", () => {
    renderOverlay();
    expect(screen.getByText("Launch command brief")).toBeInTheDocument();
    expect(screen.getByTestId("network-thread-overlay-root-badge")).toHaveTextContent("ROOT");
    expect(screen.getByTestId("network-thread-overlay-replies-divider")).toHaveTextContent(
      "1 reply"
    );
  });

  it("Should close on the X button (overlay mode)", async () => {
    navigateMock.mockClear();
    renderOverlay({ fullPage: false });
    const user = userEvent.setup();
    await user.click(screen.getByTestId("network-thread-overlay-close"));
    expect(navigateMock).toHaveBeenCalledWith({
      params: { workspaceId: WORKSPACE_ID, channel: "ops" },
      to: "/network/$workspaceId/$channel/threads",
    });
  });

  it("Should close on Escape key when in overlay mode", () => {
    navigateMock.mockClear();
    renderOverlay({ fullPage: false });
    fireEvent.keyDown(window, { key: "Escape" });
    expect(navigateMock).toHaveBeenCalledWith({
      params: { workspaceId: WORKSPACE_ID, channel: "ops" },
      to: "/network/$workspaceId/$channel/threads",
    });
  });

  it("Should NOT close on Escape when in fullPage mode", () => {
    navigateMock.mockClear();
    renderOverlay({ fullPage: true });
    fireEvent.keyDown(window, { key: "Escape" });
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it("Should expose data-fullpage attribute reflecting the mode", () => {
    renderOverlay({ fullPage: true });
    expect(screen.getByTestId("network-thread-overlay")).toHaveAttribute("data-fullpage", "true");
  });

  it("Should not declare any box-shadow on the overlay subtree", () => {
    renderOverlay();
    const root = screen.getByTestId("network-thread-overlay");
    const all = root.querySelectorAll("*");
    for (const element of [root, ...all]) {
      expect(element.getAttribute("style") ?? "").not.toContain("box-shadow");
    }
  });

  it("Should render an unavailable state without composer when the thread detail fails", () => {
    threadDetailMock.mockReturnValue({
      thread: null,
      isLoading: false,
      error: new Error("Thread not found"),
    });

    renderOverlay();

    expect(screen.getByTestId("network-thread-overlay-error")).toHaveTextContent(
      "Thread unavailable"
    );
    expect(screen.queryByRole("textbox", { name: /reply/i })).toBeNull();
  });

  it("Should render loading state without composer while thread detail resolves", () => {
    threadDetailMock.mockReturnValue({
      thread: null,
      isLoading: true,
      error: null,
    });

    renderOverlay();

    expect(screen.getByTestId("network-thread-overlay-root-loading")).toHaveTextContent(
      "Loading root"
    );
    expect(screen.getByTestId("network-timeline-skeleton")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: /reply/i })).toBeNull();
  });
});
