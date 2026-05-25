import { ListChecksIcon } from "lucide-react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TopbarShell } from "@/components/topbar-shell";

const matchesMock = vi.fn();
const subscribeMock = vi.fn();

interface OnResolvedEvent {
  pathChanged: boolean;
}

type OnResolvedHandler = (event: OnResolvedEvent) => void;

vi.mock("@tanstack/react-router", () => ({
  useRouter: () => ({
    subscribe: (event: string, handler: () => void) => {
      subscribeMock(event, handler);
      return () => undefined;
    },
  }),
  useMatches: () => matchesMock(),
}));

vi.mock("@/systems/runtime", () => ({
  useNavCounts: () => ({ counts: {}, refresh: vi.fn(), status: "ready" }),
}));

function getLatestOnResolvedHandler(): OnResolvedHandler {
  const handler = subscribeMock.mock.calls.at(-1)?.[1];
  if (typeof handler !== "function") {
    throw new Error("TopbarShell did not subscribe to the onResolved event.");
  }

  return handler;
}

describe("TopbarShell", () => {
  it("Should render route icon, title, and count from the deepest match's topbar context", () => {
    matchesMock.mockReturnValue([
      { context: {} },
      {
        context: {
          topbar: {
            title: "Tasks",
            icon: ListChecksIcon,
            getCount: () => 12,
          },
        },
      },
    ]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );
    expect(screen.getByText("Tasks")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
  });

  it("Should render the fallback Untitled when no match exposes a topbar context", () => {
    matchesMock.mockReturnValue([{ context: {} }, { context: {} }]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );
    expect(screen.getByText("Untitled")).toBeInTheDocument();
  });

  it("Should subscribe to onResolved so route navigation can clear the slot and refocus", () => {
    matchesMock.mockReturnValue([{ context: { topbar: { title: "Home" } } }]);
    subscribeMock.mockClear();
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );
    expect(subscribeMock).toHaveBeenCalledTimes(1);
    expect(subscribeMock).toHaveBeenCalledWith("onResolved", expect.any(Function));
  });

  it("Should expose a focusable topbar h1 so the shell can move focus on route resolution", () => {
    matchesMock.mockReturnValue([{ context: { topbar: { title: "Home" } } }]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );
    const heading = screen.getByTestId("topbar-title-text");
    expect(heading.tagName).toBe("H1");
    expect(heading.getAttribute("tabindex")).toBe("-1");
  });

  it("Should move focus to the topbar title when route resolution changes path", () => {
    matchesMock.mockReturnValue([{ context: { topbar: { title: "Tasks" } } }]);
    subscribeMock.mockClear();

    render(
      <TopbarShell>
        <main id="app-content">
          <label htmlFor="task-filter">Filter tasks</label>
          <input id="task-filter" />
        </main>
      </TopbarShell>
    );

    screen.getByLabelText("Filter tasks").focus();
    getLatestOnResolvedHandler()({ pathChanged: true });

    expect(screen.getByTestId("topbar-title-text")).toHaveFocus();
  });

  it("Should preserve field focus when route resolution only changes search params", () => {
    matchesMock.mockReturnValue([{ context: { topbar: { title: "Skills" } } }]);
    subscribeMock.mockClear();

    render(
      <TopbarShell>
        <main id="app-content">
          <label htmlFor="marketplace-search">Search marketplace skills</label>
          <input id="marketplace-search" />
        </main>
      </TopbarShell>
    );

    const input = screen.getByLabelText("Search marketplace skills");
    input.focus();
    getLatestOnResolvedHandler()({ pathChanged: false });

    expect(input).toHaveFocus();
  });
});
