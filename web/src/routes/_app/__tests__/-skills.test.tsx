import * as React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SkillPayload } from "@/systems/skill/types";
import { renderWithTopbar } from "@/test/render-with-topbar";

function render(ui: React.ReactElement) {
  return renderWithTopbar(ui);
}

let mockSkills: SkillPayload[] = [];
let mockSkillsLoading = false;
let mockSkillsError: Error | null = null;
const mockRefetchSkills = vi.fn();

const mockDisableMutate = vi.fn();
const mockEnableMutate = vi.fn();
let mockDisablePending = false;
let mockEnablePending = false;

const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
  childMatches: [] as unknown[],
  validateSearch: undefined as
    | ((search: Record<string, unknown>) => Record<string, unknown>)
    | undefined,
}));

function getValidatedSearch() {
  return routerState.validateSearch
    ? routerState.validateSearch(routerState.searchParams)
    : routerState.searchParams;
}

vi.mock("@tanstack/react-router", () => ({
  Outlet: () => <div data-testid="skills-outlet" />,
  createFileRoute:
    () =>
    (opts: {
      component: () => React.ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => {
      routerState.validateSearch = opts.validateSearch;

      return {
        component: opts.component,
        useSearch: () => {
          const [search, setSearch] = React.useState(getValidatedSearch());

          React.useEffect(() => {
            routerState.searchListeners.add(setSearch);
            return () => {
              routerState.searchListeners.delete(setSearch);
            };
          }, []);

          return search;
        },
      };
    },
  useChildMatches: () => routerState.childMatches,
  useNavigate:
    () =>
    async (options: {
      params?: Record<string, unknown>;
      search?:
        | Record<string, unknown>
        | ((current: Record<string, unknown>) => Record<string, unknown>);
      to: string;
    }) => {
      if (typeof options.search === "function") {
        routerState.searchParams = options.search(getValidatedSearch());
      } else if (options.search) {
        routerState.searchParams = options.search;
      }

      const nextSearch = getValidatedSearch();
      for (const listener of routerState.searchListeners) {
        listener(nextSearch);
      }

      routerState.navigateMock(options);
    },
  Link: ({ to, params, search, children, ...props }: Record<string, unknown>) => {
    const path = typeof to === "string" ? to : "#";
    const query =
      search && typeof search === "object"
        ? new URLSearchParams(
            Object.entries(search as Record<string, string>).filter(([, value]) => Boolean(value))
          ).toString()
        : "";
    return (
      <a
        href={`${path}${query ? `?${query}` : ""}`}
        data-params={JSON.stringify(params)}
        {...(props as Record<string, unknown>)}
      >
        {children as React.ReactNode}
      </a>
    );
  },
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: [
      {
        id: "ws_test",
        root_dir: "/workspace",
        add_dirs: [],
        name: "test-workspace",
        created_at: "2026-04-03T12:00:00Z",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
    hasWorkspaces: true,
    activeWorkspace: {
      id: "ws_test",
      root_dir: "/workspace",
      add_dirs: [],
      name: "test-workspace",
      created_at: "2026-04-03T12:00:00Z",
      updated_at: "2026-04-03T12:00:00Z",
    },
    activeWorkspaceId: "ws_test",
    setActiveWorkspaceId: vi.fn(),
    clearActiveWorkspaceSelection: vi.fn(),
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/systems/skill", async () => {
  const actual = await vi.importActual("@/systems/skill");
  return {
    ...actual,
    useSkills: () => ({
      data: mockSkills,
      isLoading: mockSkillsLoading,
      error: mockSkillsError,
      refetch: mockRefetchSkills,
    }),
    useDisableSkill: () => ({
      mutate: mockDisableMutate,
      isPending: mockDisablePending,
    }),
    useEnableSkill: () => ({
      mutate: mockEnableMutate,
      isPending: mockEnablePending,
    }),
  };
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../skills";

const SkillsPage = routeComponent(Route);

function makeSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    name: "test-skill",
    description: "A test skill for unit testing",
    source: "bundled",
    enabled: true,
    dir: "/path/to/skill",
    ...overrides,
  };
}

const ALL_SKILLS: SkillPayload[] = [
  makeSkill({ name: "alpha-skill", source: "bundled", enabled: true, version: "1.0.0" }),
  makeSkill({ name: "beta-skill", source: "bundled", enabled: false }),
  makeSkill({ name: "ws-tool", source: "workspace", enabled: true, version: "0.2.0" }),
  makeSkill({
    name: "mp-plugin",
    source: "marketplace",
    enabled: true,
    version: "3.1.0",
    provenance: {
      slug: "author",
      registry: "clawhub",
      version: "3.1.0",
      installed_at: "",
      precedence_tier: "marketplace",
    },
  }),
];

describe("SkillsPage", () => {
  beforeEach(() => {
    mockSkills = ALL_SKILLS;
    mockSkillsLoading = false;
    mockSkillsError = null;
    mockRefetchSkills.mockReset();
    mockDisablePending = false;
    mockEnablePending = false;
    mockDisableMutate.mockReset();
    mockEnableMutate.mockReset();
    routerState.searchListeners.clear();
    routerState.searchParams = {};
    routerState.childMatches = [];
    routerState.navigateMock.mockReset();
  });

  it("renders the installed skills listing shell", () => {
    render(<SkillsPage />);
    expect(screen.queryByTestId("skills-tabs")).not.toBeInTheDocument();
    expect(screen.getByTestId("skills-page-head")).toBeInTheDocument();
    expect(screen.getByTestId("listing-toolbar")).toBeInTheDocument();
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("skills-split-pane")).not.toBeInTheDocument();
  });

  it("Should expose no tab control and link Browse marketplace to the plural Skills kind", () => {
    render(<SkillsPage />);

    expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
    expect(screen.getByTestId("skills-browse-marketplace")).toHaveAttribute(
      "href",
      "/marketplace?kind=skills"
    );
  });

  it("renders Outlet and hides tabs/actions when a child route is active", () => {
    routerState.childMatches = [{}];
    render(<SkillsPage />);
    expect(screen.getByTestId("skills-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("skills-tabs")).not.toBeInTheDocument();
    expect(screen.queryByTestId("skills-topbar-actions")).not.toBeInTheDocument();
    expect(screen.queryByTestId("skill-list-panel")).not.toBeInTheDocument();
  });

  it("shows total skill count badge in page head", () => {
    render(<SkillsPage />);
    expect(screen.getByTestId("skills-page-count")).toHaveTextContent("4");
  });

  it("renders a flat listing without source group headers", () => {
    render(<SkillsPage />);
    expect(screen.getByTestId("skill-list-rows")).toBeInTheDocument();
    expect(screen.queryByTestId(/^skill-group-/)).not.toBeInTheDocument();
  });

  it("links skill rows to /skills/$name", () => {
    render(<SkillsPage />);
    const link = screen.getByRole("link", { name: "Open alpha-skill" });
    expect(link).toHaveAttribute("href", "/skills/$name");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ name: "alpha-skill" }));
  });

  it("persists view=cards in URL search", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("listing-view-cards"));
    expect(getValidatedSearch()).toMatchObject({ view: "cards" });
    expect(screen.getByTestId("skill-list-card-grid")).toBeInTheDocument();
  });

  it("search input filters displayed skills", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    const searchInput = screen.getByTestId("skill-search-input");
    await user.type(searchInput, "alpha");

    expect(getValidatedSearch()).toMatchObject({ q: "alpha" });
    expect(screen.getByTestId("skill-item-alpha-skill")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-beta-skill")).not.toBeInTheDocument();
  });

  it("search with no results shows empty message", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    const searchInput = screen.getByTestId("skill-search-input");
    await user.type(searchInput, "zzzznotfound");

    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills match");
  });

  it("trail enabled switch calls disable/enable mutations", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("skill-enabled-switch-alpha-skill"));
    expect(mockDisableMutate).toHaveBeenCalledWith({
      name: "alpha-skill",
      workspace: "ws_test",
    });

    await user.click(screen.getByTestId("skill-enabled-switch-beta-skill"));
    expect(mockEnableMutate).toHaveBeenCalledWith({
      name: "beta-skill",
      workspace: "ws_test",
    });
  });

  it("loading state shows spinner", () => {
    mockSkillsLoading = true;
    mockSkills = [];
    render(<SkillsPage />);

    expect(screen.getByTestId("skills-loading")).toBeInTheDocument();
  });

  it("error state shows appropriate message", () => {
    mockSkillsError = new Error("Network failure");
    mockSkills = [];
    render(<SkillsPage />);

    expect(screen.getByTestId("skills-error")).toBeInTheDocument();
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });

  it("keeps stale skills visible when a background refresh fails", () => {
    mockSkillsError = new Error("Refresh failed");
    mockSkills = ALL_SKILLS;

    render(<SkillsPage />);

    expect(screen.queryByTestId("skills-error")).not.toBeInTheDocument();
    expect(screen.getByTestId("skills-background-error")).toHaveTextContent("Refresh failed");
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
  });

  it("empty skills list shows empty message in list panel", () => {
    mockSkills = [];
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills installed");
  });

  it("full page flow loads skills and toggles the listing view", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-alpha-skill")).toBeInTheDocument();

    await user.click(screen.getByTestId("listing-view-cards"));
    expect(screen.getByTestId("skill-list-card-grid")).toBeInTheDocument();

    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
  });
});
