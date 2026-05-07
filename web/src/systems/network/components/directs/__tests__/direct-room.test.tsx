// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const directDetailMock = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: React.ReactNode }) => (
    <a {...(rest as Record<string, unknown>)}>{children}</a>
  ),
  useNavigate: () => () => undefined,
}));

vi.mock("../../../hooks/use-network-presence", () => ({
  useNetworkPresence: () => ({ state: "idle" }),
}));

vi.mock("../../../hooks/use-directs", async () => {
  const actual = await vi.importActual<typeof import("../../../hooks/use-directs")>(
    "../../../hooks/use-directs"
  );
  return {
    ...actual,
    useNetworkDirectDetail: () => directDetailMock(),
  };
});

vi.mock("../../../hooks/use-messages", () => ({
  useNetworkMessages: () => ({
    messages: [],
    isLoading: false,
    isFetching: false,
    error: null,
  }),
}));

vi.mock("../../../hooks/use-active-session", () => ({
  useActiveNetworkSession: () => ({
    session: {
      channel: "ops",
      peerId: "peer-self",
      sessionId: "sess-self",
      displayName: "Self",
    },
    disabledReason: null,
    isLoading: false,
  }),
}));

import { DirectRoom } from "../direct-room";

describe("DirectRoom headerless layout", () => {
  beforeEach(() => {
    directDetailMock.mockReturnValue({
      direct: {
        channel: "ops",
        direct_id: "direct_test",
        last_activity_at: "2026-04-17T18:00:00Z",
        last_message_preview: "preview",
        message_count: 0,
        open_work_count: 0,
        opened_at: "2026-04-17T17:00:00Z",
        peer_a: "peer-self",
        peer_b: "peer-remote",
      },
      isLoading: false,
      error: null,
    });
  });

  function renderRoom() {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    return render(
      <QueryClientProvider client={client}>
        <DirectRoom channel="ops" directId="direct_test" selfPeerId="peer-self" />
      </QueryClientProvider>
    );
  }

  it("Should render no #channel name and no member count", () => {
    renderRoom();
    expect(screen.queryByText("#ops")).toBeNull();
    expect(screen.queryByText(/members/i)).toBeNull();
    expect(screen.getByText("@peer-remote")).toBeInTheDocument();
  });

  it("Should render the role chip as agent (mono, no chromatic fill)", () => {
    renderRoom();
    expect(screen.getByText("agent")).toBeInTheDocument();
  });

  it("Should not render a presence dot when state is idle", () => {
    renderRoom();
    expect(screen.queryByTestId("network-direct-presence-dot")).toBeNull();
  });

  it("Should render an unavailable state without composer when the direct detail fails", () => {
    directDetailMock.mockReturnValue({
      direct: null,
      isLoading: false,
      error: new Error("Direct room not found"),
    });

    renderRoom();

    expect(screen.getByTestId("network-direct-room-error")).toHaveTextContent(
      "Direct room unavailable"
    );
    expect(screen.getByTestId("network-direct-room-error")).toHaveTextContent(
      "Could not load direct room direct_test. Choose an existing direct room from #ops."
    );
    expect(screen.queryByRole("textbox", { name: /message @peer/i })).toBeNull();
  });

  it("Should render loading state without composer while direct detail resolves", () => {
    directDetailMock.mockReturnValue({
      direct: null,
      isLoading: true,
      error: null,
    });

    renderRoom();

    expect(screen.getByTestId("network-timeline-skeleton")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: /message @peer/i })).toBeNull();
  });
});
