import * as React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SkillMarketplaceListingPayload, SkillPayload } from "@/systems/skill/types";
import { renderWithTopbar } from "@/test/render-with-topbar";

function render(ui: React.ReactElement) {
  return renderWithTopbar(ui, { title: "Skills" });
}

let mockSkills: SkillPayload[] = [];
let mockSkillsLoading = false;
let mockSkillsError: Error | null = null;
const mockRefetchSkills = vi.fn();

const mockDisableMutate = vi.fn();
const mockEnableMutate = vi.fn();
const mockInstallMutate = vi.fn();
const mockUpdateMutate = vi.fn();
const mockRemoveMutate = vi.fn();
let mockDisablePending = false;
let mockEnablePending = false;
let mockInstallPending = false;
let mockUpdatePending = false;
let mockRemovePending = false;

let mockMarketplaceListings: SkillMarketplaceListingPayload[] = [];
let mockMarketplaceSearching = false;
let mockMarketplaceError: Error | null = null;
const mockRefetchMarketplace = vi.fn();

const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
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
  useChildMatches: () => [],
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
  Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
    <a
      href={typeof to === "string" ? to : "#"}
      data-params={JSON.stringify(params)}
      {...(props as Record<string, unknown>)}
    >
      {children as React.ReactNode}
    </a>
  ),
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
    useSkillMarketplaceSearch: () => ({
      data: mockMarketplaceListings,
      isFetching: mockMarketplaceSearching,
      error: mockMarketplaceError,
      refetch: mockRefetchMarketplace,
    }),
    useInstallSkillMarketplace: () => ({
      mutate: mockInstallMutate,
      isPending: mockInstallPending,
    }),
    useUpdateSkillMarketplace: () => ({
      mutate: mockUpdateMutate,
      isPending: mockUpdatePending,
    }),
    useRemoveSkillMarketplace: () => ({
      mutate: mockRemoveMutate,
      isPending: mockRemovePending,
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

const MARKETPLACE_LISTINGS: SkillMarketplaceListingPayload[] = [
  {
    name: "mp-plugin",
    slug: "@compozy/mp-plugin",
    author: "compozy",
    description: "An installable marketplace plugin",
    downloads: 1234,
    source: "clawhub",
    version: "3.1.0",
  },
  {
    name: "remote-only",
    slug: "@community/remote-only",
    author: "community",
    description: "Not yet installed",
    downloads: 42,
    source: "clawhub",
    version: "0.1.0",
  },
];

describe("SkillsPage", () => {
  beforeEach(() => {
    mockSkills = ALL_SKILLS;
    mockSkillsLoading = false;
    mockSkillsError = null;
    mockRefetchSkills.mockReset();
    mockDisablePending = false;
    mockEnablePending = false;
    mockInstallPending = false;
    mockUpdatePending = false;
    mockRemovePending = false;
    mockDisableMutate.mockReset();
    mockEnableMutate.mockReset();
    mockInstallMutate.mockReset();
    mockUpdateMutate.mockReset();
    mockRemoveMutate.mockReset();
    mockMarketplaceListings = MARKETPLACE_LISTINGS;
    mockMarketplaceSearching = false;
    mockMarketplaceError = null;
    mockRefetchMarketplace.mockReset();
    routerState.searchListeners.clear();
    routerState.searchParams = {};
    routerState.navigateMock.mockReset();
  });

  it("renders Installed tab by default with listing shell", () => {
    render(<SkillsPage />);
    expect(screen.getByTestId("tab-installed")).toHaveTextContent("Installed");
    expect(screen.getByTestId("skills-page-head")).toBeInTheDocument();
    expect(screen.getByTestId("listing-toolbar")).toBeInTheDocument();
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("skills-split-pane")).not.toBeInTheDocument();
  });

  it("clicking MARKETPLACE tab switches to marketplace view", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));

    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();
    expect(getValidatedSearch()).toMatchObject({ tab: "marketplace" });
    expect(screen.queryByTestId("skill-list-panel")).not.toBeInTheDocument();
  });

  it("clicking back to INSTALLED tab shows list panel again", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));
    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();

    await user.click(screen.getByTestId("tab-installed"));
    expect(getValidatedSearch().tab).toBeUndefined();
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
  });

  it("restores tab state from URL search", () => {
    routerState.searchParams = { q: "mp-plugin", tab: "marketplace" };

    render(<SkillsPage />);

    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-search-input")).toHaveValue("mp-plugin");
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

  it("Browse marketplace topbar action switches tab", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("skills-browse-marketplace"));
    expect(getValidatedSearch()).toMatchObject({ tab: "marketplace" });
    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();
  });

  it("marketplace tab shows a search prompt with no query", async () => {
    const user = userEvent.setup();
    mockMarketplaceListings = [];
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));

    expect(screen.getByTestId("marketplace-search-prompt")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-grid")).not.toBeInTheDocument();
  });

  it("marketplace search query renders remote listings and installed state", () => {
    routerState.searchParams = { q: "plugin", tab: "marketplace" };
    render(<SkillsPage />);

    expect(screen.getByTestId("marketplace-row-mp-plugin")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-remote-only")).toBeInTheDocument();
    expect(screen.getByTestId("installed-pill-mp-plugin")).toBeInTheDocument();
    expect(screen.getByTestId("install-btn-remote-only")).toBeInTheDocument();
  });

  it("marketplace install button triggers the install mutation with the slug", async () => {
    const user = userEvent.setup();
    routerState.searchParams = { q: "plugin", tab: "marketplace" };
    render(<SkillsPage />);

    await user.click(screen.getByTestId("install-btn-remote-only"));

    expect(mockInstallMutate).toHaveBeenCalledWith({
      body: { slug: "@community/remote-only" },
      workspace: "ws_test",
    });
  });

  it("marketplace update button triggers the update mutation", async () => {
    const user = userEvent.setup();
    routerState.searchParams = { q: "plugin", tab: "marketplace" };
    render(<SkillsPage />);

    await user.click(screen.getByTestId("update-btn-mp-plugin"));

    expect(mockUpdateMutate).toHaveBeenCalledWith({
      body: { name: "mp-plugin" },
      workspace: "ws_test",
    });
  });

  it("marketplace remove requires explicit confirmation before mutating", async () => {
    const user = userEvent.setup();
    routerState.searchParams = { q: "plugin", tab: "marketplace" };
    render(<SkillsPage />);

    await user.click(screen.getByTestId("remove-btn-mp-plugin"));
    expect(mockRemoveMutate).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("confirm-remove-mp-plugin"));
    expect(mockRemoveMutate).toHaveBeenCalledWith({
      name: "mp-plugin",
      workspace: "ws_test",
    });
  });

  it("marketplace shows empty state when remote search returns nothing", () => {
    routerState.searchParams = { q: "no-match", tab: "marketplace" };
    mockMarketplaceListings = [];
    render(<SkillsPage />);

    expect(screen.getByTestId("marketplace-empty")).toBeInTheDocument();
  });

  it("marketplace surfaces remote search errors inline", () => {
    routerState.searchParams = { q: "boom", tab: "marketplace" };
    mockMarketplaceListings = [];
    mockMarketplaceError = new Error("clawhub unavailable");
    render(<SkillsPage />);

    expect(screen.getByTestId("marketplace-error")).toHaveTextContent("clawhub unavailable");
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

    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills yet");
  });

  it("full page flow: load skills, switch tabs, toggle view", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-alpha-skill")).toBeInTheDocument();

    await user.click(screen.getByTestId("listing-view-cards"));
    expect(screen.getByTestId("skill-list-card-grid")).toBeInTheDocument();

    await user.click(screen.getByTestId("tab-marketplace"));
    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();

    await user.click(screen.getByTestId("tab-installed"));
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
  });
});
