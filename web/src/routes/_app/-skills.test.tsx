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

const mockDisableMutate = vi.fn();
const mockEnableMutate = vi.fn();
let mockDisablePending = false;
let mockEnablePending = false;

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/systems/workspace", () => ({
  useWorkspaces: () => ({
    data: [
      {
        id: "ws_test",
        root_dir: "/workspace",
        add_dirs: [],
        name: "test-workspace",
        created_at: "2026-04-03T12:00:00Z",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
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
    mockDisablePending = false;
    mockEnablePending = false;
    mockDisableMutate.mockReset();
    mockEnableMutate.mockReset();
  });

  // -----------------------------------------------------------------------
  // Rendering & tabs
  // -----------------------------------------------------------------------

  it("renders INSTALLED tab by default with skill list", () => {
    render(<SkillsPage />);
    expect(screen.getByTestId("tab-installed")).toHaveTextContent("INSTALLED");
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
  });

  it("clicking MARKETPLACE tab switches to marketplace view", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));

    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-list-panel")).not.toBeInTheDocument();
  });

  it("clicking back to INSTALLED tab shows list panel again", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));
    expect(screen.getByTestId("marketplace-view")).toBeInTheDocument();

    await user.click(screen.getByTestId("tab-installed"));
    expect(screen.getByTestId("skill-list-panel")).toBeInTheDocument();
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
    expect(within(detailPanel).getByText("alpha-skill")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Skill detail panel
  // -----------------------------------------------------------------------

  it("detail panel shows BUNDLED badge for bundled skills", () => {
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled" });
    render(<SkillsPage />);

    const badge = screen.getByTestId("source-badge");
    expect(badge).toHaveTextContent("bundled");
  });

  it("detail panel shows empty state when no skill selected and no skills exist", () => {
    mockSkills = [];
    render(<SkillsPage />);

    expect(screen.getByTestId("skill-detail-empty")).toHaveTextContent(
      "Select a skill to view details"
    );
  });

  it("detail panel Disable button calls useDisableSkill mutation", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "alpha-skill", source: "bundled", enabled: true });
    render(<SkillsPage />);

    await user.click(screen.getByTestId("disable-skill-btn"));

    expect(mockDisableMutate).toHaveBeenCalledWith({
      name: "alpha-skill",
      workspace: "ws_test",
    });
  });

  it("detail panel Enable button calls useEnableSkill mutation", async () => {
    const user = userEvent.setup();
    mockSkillDetail = makeSkill({ name: "beta-skill", source: "bundled", enabled: false });
    // Select the disabled skill
    mockSkills = [makeSkill({ name: "beta-skill", source: "bundled", enabled: false })];
    render(<SkillsPage />);

    await user.click(screen.getByTestId("enable-skill-btn"));

    expect(mockEnableMutate).toHaveBeenCalledWith({
      name: "beta-skill",
      workspace: "ws_test",
    });
  });

  // -----------------------------------------------------------------------
  // Skill list search
  // -----------------------------------------------------------------------

  it("search input filters displayed skills", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    const searchInput = screen.getByTestId("skill-search-input");
    await user.type(searchInput, "alpha");

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

  it("shows green status dot for enabled skills", () => {
    render(<SkillsPage />);
    const dot = screen.getByTestId("skill-status-dot-alpha-skill");
    expect(dot.className).toContain("bg-[color:var(--color-success)]");
  });

  it("shows gray status dot for disabled skills", () => {
    render(<SkillsPage />);
    const dot = screen.getByTestId("skill-status-dot-beta-skill");
    expect(dot.className).toContain("bg-[color:var(--color-text-tertiary)]");
  });

  // -----------------------------------------------------------------------
  // Marketplace view
  // -----------------------------------------------------------------------

  it("marketplace search input filters displayed skills", async () => {
    const user = userEvent.setup();
    render(<SkillsPage />);

    await user.click(screen.getByTestId("tab-marketplace"));

    const searchInput = screen.getByTestId("marketplace-search-input");
    await user.type(searchInput, "mp-plugin");

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

    // All skills in our mock are "installed"
    expect(screen.getByTestId("installed-pill-alpha-skill")).toHaveTextContent("INSTALLED");
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

  it("detail panel shows metadata table when skill has metadata", () => {
    mockSkillDetail = makeSkill({
      name: "alpha-skill",
      source: "bundled",
      metadata: { author: "team", category: "testing" },
    });
    render(<SkillsPage />);
    expect(screen.getByText("author")).toBeInTheDocument();
    expect(screen.getByText("team")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Content preview
  // -----------------------------------------------------------------------

  it("detail panel shows content preview when skill has content", () => {
    mockSkillDetail = makeSkill({
      name: "alpha-skill",
      source: "bundled",
      content: "## Skill instructions\nDo things.",
    });
    render(<SkillsPage />);

    expect(screen.getByTestId("content-preview")).toBeInTheDocument();
    expect(screen.getByText(/Skill instructions/)).toBeInTheDocument();
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
