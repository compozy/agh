import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { AgentCategoryTree } from "../agent-category-tree";
import type { AgentPayload } from "../../types";

type MatchRouteParams = Record<string, string>;
let matchedRouteExact: Record<string, boolean> = {};
let matchedRouteFuzzy: Record<string, boolean> = {};

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: ReactNode;
    to: string;
    params?: MatchRouteParams;
    [key: string]: unknown;
  }) => {
    const href = params
      ? Object.entries(params).reduce((acc, [key, value]) => acc.replace(`$${key}`, value), to)
      : to;
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  },
  useMatchRoute: () => (opts: { to: string; params?: MatchRouteParams; fuzzy?: boolean }) => {
    const key = opts.params
      ? `${opts.to}?${Object.entries(opts.params)
          .map(([k, v]) => `${k}=${v}`)
          .join("&")}`
      : opts.to;
    if (opts.fuzzy) return matchedRouteFuzzy[key] ?? false;
    return matchedRouteExact[key] ?? false;
  },
}));

function makeAgent(overrides: Partial<AgentPayload> & { name: string }): AgentPayload {
  return {
    provider: overrides.provider ?? "claude",
    prompt: overrides.prompt ?? `prompt for ${overrides.name}`,
    ...overrides,
  } as AgentPayload;
}

function renderTree(props: Parameters<typeof AgentCategoryTree>[0]) {
  return render(
    <UIProvider reducedMotion="always">
      <AgentCategoryTree {...props} />
    </UIProvider>
  );
}

describe("AgentCategoryTree", () => {
  beforeEach(() => {
    matchedRouteExact = {};
    matchedRouteFuzzy = {};
  });

  it("Should render category folders and agent leaves", () => {
    renderTree({
      agents: [
        makeAgent({ name: "deals", category_path: ["Sales"] }),
        makeAgent({ name: "writer" }),
      ],
      agentsLoading: false,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agent-category-Sales")).toBeInTheDocument();
    expect(screen.getByTestId("agent-row-deals")).toHaveAttribute("href", "/agents/deals");
    expect(screen.getByTestId("agent-row-writer")).toHaveAttribute("href", "/agents/writer");
  });

  it("Should expand ancestors of the active agent on initial render", () => {
    matchedRouteFuzzy["/agents/$name?name=deals"] = true;
    renderTree({
      agents: [
        makeAgent({ name: "deals", category_path: ["Marketing", "Sales"] }),
        makeAgent({ name: "outreach", category_path: ["Operations"] }),
      ],
      agentsLoading: false,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agent-category-Marketing")).toHaveAttribute("data-expanded", "true");
    expect(screen.getByTestId("agent-category-Marketing/Sales")).toHaveAttribute(
      "data-expanded",
      "true"
    );
    expect(screen.getByTestId("agent-category-Operations")).toHaveAttribute(
      "data-expanded",
      "false"
    );
  });

  it("Should expand top-level categories on initial render when no agent is active", () => {
    renderTree({
      agents: [
        makeAgent({ name: "deals", category_path: ["Marketing", "Sales"] }),
        makeAgent({ name: "outreach", category_path: ["Operations"] }),
      ],
      agentsLoading: false,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agent-category-Marketing")).toHaveAttribute("data-expanded", "true");
    expect(screen.getByTestId("agent-category-Operations")).toHaveAttribute(
      "data-expanded",
      "true"
    );
    expect(screen.getByTestId("agent-category-Marketing/Sales")).toHaveAttribute(
      "data-expanded",
      "false"
    );
  });

  it("Should mark the active agent row with data-active=true and render the indicator", () => {
    matchedRouteFuzzy["/agents/$name?name=writer"] = true;
    renderTree({
      agents: [makeAgent({ name: "writer" }), makeAgent({ name: "coder" })],
      agentsLoading: false,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agent-row-writer")).toHaveAttribute("data-active", "true");
    expect(screen.getByTestId("agent-active-writer")).toBeInTheDocument();
    expect(screen.getByTestId("agent-row-coder")).toHaveAttribute("data-active", "false");
  });

  it("Should render the active-session dot for agents with active sessions", () => {
    renderTree({
      agents: [makeAgent({ name: "coder" }), makeAgent({ name: "writer" })],
      agentsLoading: false,
      agentsError: false,
      sessions: [
        {
          id: "s1",
          name: "Session",
          agent_name: "coder",
          provider: "claude",
          workspace_id: "ws_alpha",
          workspace_path: "/workspace/alpha",
          state: "active",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
        {
          id: "s2",
          name: "Session",
          agent_name: "writer",
          provider: "claude",
          workspace_id: "ws_alpha",
          workspace_path: "/workspace/alpha",
          state: "stopped",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
      ],
    });
    expect(screen.getByTestId("agent-status-dot-coder")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-status-dot-writer")).not.toBeInTheDocument();
  });

  it("Should render the loading state with agents-loading test ID", () => {
    renderTree({
      agents: undefined,
      agentsLoading: true,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agents-loading")).toHaveAttribute("data-fill", "false");
  });

  it("Should expand top-level categories after agents load from an initial loading state", () => {
    const view = renderTree({
      agents: undefined,
      agentsLoading: true,
      agentsError: false,
      sessions: [],
    });

    view.rerender(
      <UIProvider reducedMotion="always">
        <AgentCategoryTree
          agents={[
            makeAgent({ name: "deals", category_path: ["Marketing", "Sales"] }),
            makeAgent({ name: "support", category_path: ["Operations"] }),
          ]}
          agentsLoading={false}
          agentsError={false}
          sessions={[]}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("agent-category-Marketing")).toHaveAttribute("data-expanded", "true");
    expect(screen.getByTestId("agent-category-Operations")).toHaveAttribute(
      "data-expanded",
      "true"
    );
    expect(screen.getByTestId("agent-category-Marketing/Sales")).toHaveAttribute(
      "data-expanded",
      "false"
    );
  });

  it("Should render the empty state with agents-empty test ID when there are no agents", () => {
    renderTree({
      agents: [],
      agentsLoading: false,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agents-empty")).toHaveAttribute("data-fill", "false");
  });

  it("Should render the empty state with agents-empty test ID on error", () => {
    renderTree({
      agents: undefined,
      agentsLoading: false,
      agentsError: true,
      sessions: [],
    });
    expect(screen.getByTestId("agents-error")).toHaveAttribute("data-fill", "false");
  });

  it("Should keep rendering stale agents when a refresh error occurs after data loads", () => {
    renderTree({
      agents: [makeAgent({ name: "writer" })],
      agentsLoading: false,
      agentsError: true,
      sessions: [],
    });
    expect(screen.getByTestId("agent-row-writer")).toBeInTheDocument();
    expect(screen.queryByTestId("agents-error")).not.toBeInTheDocument();
  });

  it("Should not auto-expand a different category when the active route changes after mount", () => {
    matchedRouteFuzzy["/agents/$name?name=outreach"] = true;
    const view = renderTree({
      agents: [
        makeAgent({ name: "deals", category_path: ["Marketing", "Sales"] }),
        makeAgent({ name: "outreach", category_path: ["Operations"] }),
      ],
      agentsLoading: false,
      agentsError: false,
      sessions: [],
    });
    expect(screen.getByTestId("agent-category-Operations")).toHaveAttribute(
      "data-expanded",
      "true"
    );
    expect(screen.getByTestId("agent-category-Marketing")).toHaveAttribute(
      "data-expanded",
      "false"
    );

    matchedRouteFuzzy = { "/agents/$name?name=deals": true };
    view.rerender(
      <UIProvider reducedMotion="always">
        <AgentCategoryTree
          agents={[
            makeAgent({ name: "deals", category_path: ["Marketing", "Sales"] }),
            makeAgent({ name: "outreach", category_path: ["Operations"] }),
          ]}
          agentsLoading={false}
          agentsError={false}
          sessions={[]}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("agent-category-Operations")).toHaveAttribute(
      "data-expanded",
      "true"
    );
    expect(screen.getByTestId("agent-category-Marketing")).toHaveAttribute(
      "data-expanded",
      "false"
    );
    // Marketing/Sales is not in the DOM because Marketing remained collapsed:
    // proving the route change did not silently auto-expand a different branch.
    expect(screen.queryByTestId("agent-category-Marketing/Sales")).not.toBeInTheDocument();
  });
});
