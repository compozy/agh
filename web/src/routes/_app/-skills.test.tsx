import * as React from "react";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SkillPayload } from "@/systems/skill/types";

// ---------------------------------------------------------------------------
// Mock state
// ---------------------------------------------------------------------------

let mockSkills: SkillPayload[] = [];
let mockSkillsLoading = false;
let mockSkillsError: Error | null = null;

let mockSkillDetail: SkillPayload | undefined;
let mockSkillDetailLoading = false;
let mockSkillDetailError: Error | null = null;
let mockSkillContent: string | undefined;
let mockSkillContentLoading = false;
let mockSkillContentError: Error | null = null;
const mockRefetchSkillContent = vi.fn();
const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
  validateSearch: undefined as
    | ((search: Record<string, unknown>) => Record<string, unknown>)
    | undefined,
}));

const mockDisableMutate = vi.fn();
const mockEnableMutate = vi.fn();
let mockDisablePending = false;
let mockEnablePending = false;

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

function getValidatedSearch() {
  return routerState.validateSearch
    ? routerState.validateSearch(routerState.searchParams)
    : routerState.searchParams;
}

vi.mock("@tanstack/react-router", () => ({
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
  useNavigate:
    () =>
    async (options: {
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
    }),
    useSkill: () => ({
      data: mockSkillDetail,
      isLoading: mockSkillDetailLoading,
      error: mockSkillDetailError,
    }),
    useSkillContent: (_name: string, _workspace: string, enabled = false) => ({
      data: enabled ? mockSkillContent : undefined,
      isLoading: enabled && mockSkillContentLoading,
      error: enabled ? mockSkillContentError : null,
      refetch: mockRefetchSkillContent,
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

import { Route } from "./skills";

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

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

const BUNDLED_SKILLS: SkillPayload[] = [
  makeSkill({ name: "alpha-skill", source: "bundled", enabled: true, version: "1.0.0" }),
  makeSkill({ name: "beta-skill", source: "bundled", enabled: false }),
];

const WORKSPACE_SKILLS: SkillPayload[] = [
  makeSkill({ name: "ws-tool", source: "workspace", enabled: true, version: "0.2.0" }),
];

const MARKETPLACE_SKILLS: SkillPayload[] = [
  makeSkill({
    name: "mp-plugin",
    source: "marketplace",
    enabled: true,
    version: "3.1.0",
    metadata: { tags: ["testing", "ai"], downloads: 1234 },
    provenance: { slug: "author", registry: "clawhub", version: "3.1.0", installed_at: "" },
  }),
];

const ALL_SKILLS = [...BUNDLED_SKILLS, ...WORKSPACE_SKILLS, ...MARKETPLACE_SKILLS];

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const SkillsPage = (Route as any).component as () => React.ReactNode;

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("SkillsPage", () => {
  beforeEach(() => {
    mockSkills = ALL_SKILLS;
    mockSkillsLoading = false;
    mockSkillsError = null;
    mockSkillDetail = undefined;
    mockSkillDetailLoading = false;
    mockSkillDetailError = null;
    mockSkillContent = undefined;
    mockSkillContentLoading = false;
    mockSkillContentError = null;
    mockRefetchSkillContent.mockReset();
    mockDisablePending = false;
    mockEnablePending = false;
    mockDisableMutate.mockReset();
    mockEnableMutate.mockReset();
    routerState.searchListeners.clear();
    routerState.searchParams = {};
    routerState.navigateMock.mockReset();
  });

  // -----------------------------------------------------------------------
  // Rendering & tabs
  // -----------------------------------------------------------------------

  it("renders Installed tab by default with skill list", () => {
    render(<SkillsPage />);
    expect(screen.getByTestId("tab-installed")).toHaveTextContent("Installed");
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
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

  it("shows total skill count badge in header", () => {
    render(<SkillsPage />);
    // 4 total skills
    expect(screen.getByText("4")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Skill list grouping
  // -----------------------------------------------------------------------

  it("groups skills by source (BUNDLED, WORKSPACE, MARKETPLACE)", () => {
    render(<SkillsPage />);
    expect(screen.getByTestId("skill-group-bundled")).toBeInTheDocument();
    expect(screen.getByTestId("skill-group-workspace")).toBeInTheDocument();
    expect(screen.getByTestId("skill-group-marketplace")).toBeInTheDocument();
  });

  it("shows section count for each group", () => {
    render(<SkillsPage />);
    const bundledGroup = screen.getByTestId("skill-group-bundled");
    expect(within(bundledGroup).getByText("2")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Skill selection
  // -----------------------------------------------------------------------

  it("selecting a skill highlights it with accent left bar", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("skill-item-beta-skill"));

    expect(getValidatedSearch()).toMatchObject({ skill: "beta-skill" });
    const item = screen.getByTestId("skill-item-beta-skill");
    const indicator = within(item).getByTestId("skill-active-indicator");
    expect(indicator).toBeInTheDocument();
  });

  it("auto-selects first skill when no selection is made", () => {
    render(<SkillsPage />);
    // First skill alphabetically in bundled is alpha-skill
    const item = screen.getByTestId("skill-item-alpha-skill");
    expect(within(item).getByTestId("skill-active-indicator")).toBeInTheDocument();
  });

  it("shows detail panel with correct name when skill is selected", () => {
    mockSkillDetail = ALL_SKILLS[0];
    render(<SkillsPage />);
    const detailPanel = screen.getByTestId("skill-detail-panel");
    expect(within(detailPanel).getByTestId("skill-detail-title")).toHaveTextContent("alpha-skill");
  });

  it("restores selected skill, requested content, and query from URL search", () => {
    routerState.searchParams = {
      content: "beta-skill",
      q: "beta",
      skill: "beta-skill",
    };
    mockSkillDetail = makeSkill({ name: "beta-skill", source: "bundled", enabled: false });
    mockSkillContent = "## Beta instructions";

    render(<SkillsPage />);

    expect(screen.getByTestId("skill-search-input")).toHaveValue("beta");
    expect(
      within(screen.getByTestId("skill-item-beta-skill")).getByTestId("skill-active-indicator")
    ).toBeInTheDocument();
    expect(screen.getByTestId("content-body")).toHaveTextContent("Beta instructions");
  });

  // -----------------------------------------------------------------------
  // Skill detail panel
  // -----------------------------------------------------------------------

  it("detail panel renders source as MonoBadge with accent tone", () => {
    mockSkillDetail = makeSkill({ name: "mp-plugin", source: "marketplace" });
    render(<SkillsPage />);

    const badge = screen.getByTestId("source-badge");
    expect(badge).toHaveTextContent("marketplace");
    expect(badge).toHaveAttribute("data-tone", "accent");
  });

  it("detail panel renders version and author as MonoBadge meta", () => {
    mockSkillDetail = makeSkill({
      name: "mp-plugin",
      source: "marketplace",
      version: "3.1.0",
      provenance: { slug: "author", registry: "clawhub", version: "3.1.0", installed_at: "" },
    });
    render(<SkillsPage />);

    expect(screen.getByTestId("detail-version-badge")).toHaveTextContent("v3.1.0");
    expect(screen.getByTestId("detail-author-badge")).toHaveTextContent("@author");
  });

  it("detail panel shows empty state when no skill selected and no skills exist", () => {
    mockSkills = [];
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-detail-empty")).toHaveTextContent(
      "Select a skill to view details"
    );
  });

  it("detail panel Switch toggles the disable mutation when enabled", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled", enabled: true });
    render(<SkillsPage />);

    await user.click(screen.getByTestId("skill-enabled-switch"));

    expect(mockDisableMutate).toHaveBeenCalledWith({
      name: "alpha-skill",
      workspace: "ws_test",
    });
  });

  it("detail panel Switch toggles the enable mutation when disabled", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "beta-skill", source: "bundled", enabled: false });
    mockSkills = [makeSkill({ name: "beta-skill", source: "bundled", enabled: false })];
    render(<SkillsPage />);

    await user.click(screen.getByTestId("skill-enabled-switch"));

    expect(mockEnableMutate).toHaveBeenCalledWith({
      name: "beta-skill",
      workspace: "ws_test",
    });
  });

  it("detail panel Switch is disabled while an action is pending", () => {
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled", enabled: true });
    mockEnablePending = true;
    render(<SkillsPage />);

    const sw = screen.getByTestId("skill-enabled-switch");
    expect(sw).toHaveAttribute("aria-disabled", "true");
    expect(sw).toHaveAttribute("data-disabled");
  });

  // -----------------------------------------------------------------------
  // Skill list search
  // -----------------------------------------------------------------------

  it("search input filters displayed skills", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    const searchInput = screen.getByTestId("skill-search-input");
    await user.type(searchInput, "alpha");

    expect(getValidatedSearch()).toMatchObject({ q: "alpha" });
    expect(screen.getByTestId("skill-item-alpha-skill")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-beta-skill")).not.toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-ws-tool")).not.toBeInTheDocument();
  });

  it("search with no results shows empty message", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    const searchInput = screen.getByTestId("skill-search-input");
    await user.type(searchInput, "zzzznotfound");

    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills found");
  });

  // -----------------------------------------------------------------------
  // Status dots
  // -----------------------------------------------------------------------

  it("shows success status dot for enabled skills", () => {
    render(<SkillsPage />);
    const dot = screen.getByTestId("skill-status-dot-alpha-skill");
    expect(dot).toHaveAttribute("data-tone", "success");
  });

  it("shows neutral status dot for disabled skills", () => {
    render(<SkillsPage />);
    const dot = screen.getByTestId("skill-status-dot-beta-skill");
    expect(dot).toHaveAttribute("data-tone", "neutral");
  });

  // -----------------------------------------------------------------------
  // Marketplace view
  // -----------------------------------------------------------------------

  it("marketplace search input filters displayed skills", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));

    expect(screen.getByTestId("marketplace-readonly-notice")).toHaveTextContent(
      "The daemon API only exposes metadata for already installed marketplace skills here."
    );
    expect(screen.getByTestId("marketplace-row-mp-plugin")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-row-alpha-skill")).not.toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-row-ws-tool")).not.toBeInTheDocument();

    const searchInput = screen.getByTestId("marketplace-search-input");
    await user.type(searchInput, "mp-plugin");

    expect(getValidatedSearch()).toMatchObject({ q: "mp-plugin", tab: "marketplace" });
    expect(screen.getByTestId("marketplace-row-mp-plugin")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-row-alpha-skill")).not.toBeInTheDocument();
  });

  it("category filter chips toggle active state and filter results", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));
    await user.click(screen.getByTestId("category-chip-TESTING"));

    // mp-plugin has "testing" tag
    expect(screen.getByTestId("marketplace-row-mp-plugin")).toBeInTheDocument();
  });

  it("marketplace row shows INSTALLED pill for already-installed skills", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));

    expect(screen.getByTestId("installed-pill-mp-plugin")).toHaveTextContent("INSTALLED");
    expect(screen.queryByTestId("installed-pill-alpha-skill")).not.toBeInTheDocument();
  });

  it("marketplace category filter shows Empty when nothing matches", async () => {
    const user = userEvent.setup();
    mockSkills = BUNDLED_SKILLS; // no tags, so DATABASE matches nothing
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));
    await user.click(screen.getByTestId("category-chip-DATABASE"));

    expect(screen.getByTestId("marketplace-empty")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-empty")).toHaveTextContent(
      "No installed marketplace skills match the current filters."
    );
  });

  // -----------------------------------------------------------------------
  // Loading / Error states
  // -----------------------------------------------------------------------

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

    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills found");
  });

  // -----------------------------------------------------------------------
  // Detail loading / error
  // -----------------------------------------------------------------------

  it("detail panel shows loading spinner when fetching detail", () => {
    mockSkillDetailLoading = true;
    render(<SkillsPage />);
    expect(screen.getByTestId("skill-detail-loading")).toBeInTheDocument();
  });

  it("detail panel shows error when detail fetch fails", () => {
    mockSkillDetailError = new Error("Detail fetch failed");
    render(<SkillsPage />);
    expect(screen.getByTestId("skill-detail-error")).toHaveTextContent(
      "Failed to load skill details"
    );
  });

  it("detail panel shows capabilities and recent calls from metadata", () => {
    mockSkillDetail = makeSkill({
      name: "alpha-skill",
      source: "bundled",
      metadata: {
        capabilities: ["shell.run", "git.stage"],
        recent_calls: [
          { label: "skill.run", status: "success", timestamp: new Date().toISOString() },
        ],
      },
    });
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-capability-shell.run")).toBeInTheDocument();
    expect(screen.getByTestId("skill-capability-git.stage")).toBeInTheDocument();
    expect(screen.getByTestId("skill-recent-call-row-0")).toHaveTextContent("skill.run");
  });

  it("detail panel shows Empty state when no capabilities or recent calls exist", () => {
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled" });
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-capabilities-empty")).toBeInTheDocument();
    expect(screen.getByTestId("skill-recent-calls-empty")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Content preview
  // -----------------------------------------------------------------------

  it("detail panel loads full content only after clicking view full content", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({
      name: "alpha-skill",
      source: "bundled",
    });
    mockSkillContent = "## Skill instructions\nDo things.";
    render(<SkillsPage />);

    expect(screen.queryByTestId("content-body")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("view-full-content-btn"));

    expect(getValidatedSearch()).toMatchObject({
      content: "alpha-skill",
      skill: "alpha-skill",
    });
    expect(screen.getByTestId("content-body")).toBeInTheDocument();
    expect(screen.getByText(/Skill instructions/)).toBeInTheDocument();
  });

  it("detail panel shows content loading state after content is requested", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled" });
    mockSkillContentLoading = true;
    render(<SkillsPage />);

    await user.click(screen.getByTestId("view-full-content-btn"));

    expect(screen.getByTestId("content-loading")).toBeInTheDocument();
  });

  it("detail panel shows content error state after failed content request", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled" });
    mockSkillContentError = new Error("Content fetch failed");
    render(<SkillsPage />);

    await user.click(screen.getByTestId("view-full-content-btn"));

    expect(screen.getByTestId("content-error")).toBeInTheDocument();
  });

  it("detail panel retries content fetch after a failed request", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled" });
    mockSkillContentError = new Error("Content fetch failed");
    render(<SkillsPage />);

    await user.click(screen.getByTestId("view-full-content-btn"));
    await user.click(screen.getByTestId("retry-view-content-btn"));

    expect(mockRefetchSkillContent).toHaveBeenCalledTimes(1);
  });

  // -----------------------------------------------------------------------
  // Integration: full flow
  // -----------------------------------------------------------------------

  it("full page flow: load skills, select skill, view detail, toggle tab", async () => {
    const user = userEvent.setup();
    mockSkillDetail = ALL_SKILLS[0];
    render(<SkillsPage />);

    // Skills are loaded and displayed
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-alpha-skill")).toBeInTheDocument();

    // Select a different skill
    await user.click(screen.getByTestId("skill-item-ws-tool"));

    // Switch to marketplace
    await user.click(screen.getByTestId("tab-marketplace"));
    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();

    // Switch back to installed
    await user.click(screen.getByTestId("tab-installed"));
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
  });
});
