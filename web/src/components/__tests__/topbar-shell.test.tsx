import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useTopbarSlot } from "@agh/ui";

import { TopbarShell } from "@/components/topbar-shell";

const matchesMock = vi.fn();
const subscribeMock = vi.fn();
const pathnameMock = vi.fn(() => "/agents");

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
  useRouterState: ({
    select,
  }: {
    select: (state: { location: { pathname: string } }) => unknown;
  }) => select({ location: { pathname: pathnameMock() } }),
  Link: ({ to, children, ...props }: Record<string, unknown>) => (
    <a href={typeof to === "string" ? to : "#"} {...(props as Record<string, unknown>)}>
      {children as ReactNode}
    </a>
  ),
}));

function getLatestOnResolvedHandler(): OnResolvedHandler {
  const handler = subscribeMock.mock.calls.at(-1)?.[1];
  if (typeof handler !== "function") {
    throw new Error("TopbarShell did not subscribe to the onResolved event.");
  }

  return handler;
}

function matchWithCrumb(label: string, to?: string) {
  return { context: { topbar: { crumb: { label, ...(to ? { to } : {}) } } } };
}

describe("TopbarShell", () => {
  beforeEach(() => {
    matchesMock.mockReset();
    subscribeMock.mockClear();
    pathnameMock.mockReturnValue("/agents");
  });

  it("Should collect every match level's topbar.crumb into the breadcrumb trail", () => {
    matchesMock.mockReturnValue([
      { context: {} },
      matchWithCrumb("Agents", "/agents"),
      matchWithCrumb("release-captain"),
    ]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Agents" })).toHaveAttribute("href", "/agents");
    expect(screen.getByRole("heading", { level: 1, name: "release-captain" })).toBeInTheDocument();
  });

  it("Should render exactly Marketplace and the kind crumb after redirect-mediated entry", () => {
    const marketplaceTopbar = { crumb: { label: "Marketplace", to: "/marketplace" } };
    const skillsTopbar = { crumb: { label: "Skills" } };
    matchesMock.mockReturnValue([
      { context: {} },
      { context: { topbar: marketplaceTopbar } },
      { context: { topbar: marketplaceTopbar } },
      { context: { topbar: skillsTopbar } },
    ]);
    pathnameMock.mockReturnValue("/marketplace/skills");
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    expect(screen.getByRole("link", { name: "Marketplace" })).toHaveAttribute(
      "href",
      "/marketplace"
    );
    expect(screen.getAllByText("Marketplace")).toHaveLength(1);
    expect(screen.getByRole("heading", { level: 1, name: "Skills" })).toBeInTheDocument();
  });

  it("Should show zero Marketplace crumbs on a sibling route after leaving marketplace", () => {
    const marketplaceTopbar = { crumb: { label: "Marketplace", to: "/marketplace" } };
    matchesMock.mockReturnValue([
      { context: {} },
      { context: { topbar: marketplaceTopbar } },
      { context: { topbar: marketplaceTopbar } },
      { context: { topbar: { crumb: { label: "Skills" } } } },
    ]);
    pathnameMock.mockReturnValue("/marketplace/skills");
    const { rerender } = render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );
    expect(screen.getByText("Marketplace")).toBeInTheDocument();

    pathnameMock.mockReturnValue("/triggers");
    matchesMock.mockReturnValue([
      { context: {} },
      { context: { topbar: { crumb: { label: "Triggers", to: "/triggers" } } } },
    ]);
    rerender(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    expect(screen.queryByText("Marketplace")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "Triggers" })).toBeInTheDocument();
  });

  it("Should still render the home breadcrumb when no match exposes a topbar crumb", () => {
    matchesMock.mockReturnValue([{ context: {} }, { context: {} }]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    expect(document.querySelector("[data-slot='breadcrumb']")).not.toBeNull();
    expect(screen.getByTestId("topbar-breadcrumb-home")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("heading", { level: 1, name: "Untitled" })).toBeInTheDocument();
    expect(document.querySelector("[data-slot='breadcrumb-separator']")).toBeNull();
  });

  it("Should render the home icon as the current page on the dashboard", () => {
    pathnameMock.mockReturnValue("/");
    matchesMock.mockReturnValue([matchWithCrumb("Home", "/")]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    const home = screen.getByTestId("topbar-breadcrumb-home");
    expect(home).toHaveAttribute("aria-current", "page");
    expect(home).toHaveAttribute("aria-label", "Dashboard");
    expect(screen.queryByRole("link", { name: "Dashboard" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "Home" })).toBeInTheDocument();
  });

  it("Should override the leaf breadcrumb label via the published slot crumb", () => {
    function DestinationRoute() {
      useTopbarSlot({ crumb: "Loop A" });
      return null;
    }

    matchesMock.mockReturnValue([matchWithCrumb("Loops", "/loops")]);
    render(
      <TopbarShell>
        <DestinationRoute />
        <main id="app-content" />
      </TopbarShell>
    );

    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("heading", { level: 1, name: "Loop A" })).toBeInTheDocument();
    expect(screen.queryByText("Loops")).not.toBeInTheDocument();
  });

  it("Should render the route H1 inside the topbar", () => {
    matchesMock.mockReturnValue([matchWithCrumb("Tasks")]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    const topbar = document.querySelector("[data-slot='topbar']");
    expect(topbar?.querySelector("h1")).toHaveTextContent("Tasks");
    expect(topbar?.querySelector("[data-slot='page-head-count']")).toBeNull();
  });

  it("Should subscribe to onResolved so route navigation can refocus the page title", () => {
    matchesMock.mockReturnValue([matchWithCrumb("Home")]);
    render(
      <TopbarShell>
        <main id="app-content" />
      </TopbarShell>
    );

    expect(subscribeMock).toHaveBeenCalledTimes(1);
    expect(subscribeMock).toHaveBeenCalledWith("onResolved", expect.any(Function));
  });

  it("Should keep focus on the stable Topbar H1 while destination content mounts later", () => {
    matchesMock.mockReturnValue([matchWithCrumb("Home")]);

    const { rerender } = render(
      <TopbarShell>
        <main id="app-content">
          <label htmlFor="task-filter">Filter tasks</label>
          <input id="task-filter" />
        </main>
      </TopbarShell>
    );

    screen.getByLabelText("Filter tasks").focus();
    getLatestOnResolvedHandler()({ pathChanged: true });

    expect(screen.getByRole("heading", { level: 1, name: "Home" })).toHaveFocus();

    matchesMock.mockReturnValue([matchWithCrumb("Tasks")]);
    rerender(
      <TopbarShell>
        <main id="app-content">
          <div data-testid="tasks-loading">Loading tasks</div>
        </main>
      </TopbarShell>
    );

    expect(screen.getByRole("heading", { level: 1, name: "Tasks" })).toHaveFocus();
    expect(screen.getByTestId("tasks-loading")).toBeInTheDocument();
  });

  it("Should preserve field focus when route resolution only changes search params", () => {
    matchesMock.mockReturnValue([matchWithCrumb("Skills")]);

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

  it("Should preserve the destination slot when path resolution fires after it publishes", () => {
    function DestinationRoute() {
      useTopbarSlot({ actions: <button type="button">New session</button> });
      return null;
    }

    matchesMock.mockReturnValue([matchWithCrumb("Agent")]);
    render(
      <TopbarShell>
        <DestinationRoute />
        <main id="app-content" />
      </TopbarShell>
    );

    expect(screen.getByRole("button", { name: "New session" })).toBeVisible();
    getLatestOnResolvedHandler()({ pathChanged: true });
    expect(screen.getByRole("button", { name: "New session" })).toBeVisible();
  });
});
