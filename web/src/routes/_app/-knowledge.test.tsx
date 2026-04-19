import { UIProvider } from "@agh/ui";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { MemoryHeader } from "@/systems/knowledge/types";

// ---------------------------------------------------------------------------
// Mock state
// ---------------------------------------------------------------------------

let mockMemories: MemoryHeader[] = [];
let mockMemoriesLoading = false;
let mockMemoriesError: Error | null = null;

let mockMemoryContent: string | undefined;
let mockMemoryContentLoading = false;
let mockMemoryContentError: Error | null = null;

const mockDeleteMutate = vi.fn();
let mockDeletePending = false;

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
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

vi.mock("@/systems/knowledge", async () => {
  const actual = await vi.importActual("@/systems/knowledge");
  return {
    ...actual,
    useMemories: () => ({
      data: mockMemories,
      isLoading: mockMemoriesLoading,
      error: mockMemoriesError,
    }),
    useMemory: () => ({
      data: mockMemoryContent,
      isLoading: mockMemoryContentLoading,
      error: mockMemoryContentError,
    }),
    useDeleteMemory: () => ({
      mutate: mockDeleteMutate,
      isPending: mockDeletePending,
    }),
  };
});

import { Route } from "./knowledge";

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

function makeMemory(overrides: Partial<MemoryHeader> = {}): MemoryHeader {
  return {
    filename: "user_role.md",
    mod_time: "2026-04-09T10:00:00Z",
    name: "User Role",
    description: "User is a senior engineer",
    type: "user",
    ...overrides,
  };
}

const GLOBAL_MEMORIES: MemoryHeader[] = [
  makeMemory({
    filename: "user_role.md",
    name: "User Role",
    description: "User is a senior engineer",
    type: "user",
    mod_time: "2026-04-09T10:00:00Z",
  }),
  makeMemory({
    filename: "feedback_testing.md",
    name: "Testing Feedback",
    description: "Always run integration tests",
    type: "feedback",
    mod_time: "2026-04-08T14:00:00Z",
  }),
  makeMemory({
    filename: "project_migration.md",
    name: "Migration Project",
    description: "Database migration in progress",
    type: "project",
    mod_time: "2026-04-07T09:00:00Z",
  }),
];

const WORKSPACE_MEMORIES: MemoryHeader[] = [
  makeMemory({
    filename: "workspace/ref_api.md",
    name: "API Reference",
    description: "REST API docs at docs.internal",
    type: "reference",
    mod_time: "2026-04-06T11:00:00Z",
    agent_name: "coder",
  }),
  makeMemory({
    filename: "workspace/project_sprint.md",
    name: "Sprint Planning",
    description: "Sprint 5 goals and deadlines",
    type: "project",
    mod_time: "2026-04-05T08:00:00Z",
  }),
];

const ALL_MEMORIES = [...GLOBAL_MEMORIES, ...WORKSPACE_MEMORIES];

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const KnowledgePage = (Route as any).component as () => React.ReactNode;

function renderPage() {
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgePage />
    </UIProvider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("KnowledgePage", () => {
  beforeEach(() => {
    mockMemories = ALL_MEMORIES;
    mockMemoriesLoading = false;
    mockMemoriesError = null;
    mockMemoryContent = undefined;
    mockMemoryContentLoading = false;
    mockMemoryContentError = null;
    mockDeletePending = false;
    mockDeleteMutate.mockReset();
  });

  // -----------------------------------------------------------------------
  // Rendering & tabs
  // -----------------------------------------------------------------------

  it("renders ALL tab by default with full memory list", () => {
    renderPage();
    expect(screen.getByTestId("tab-all")).toHaveTextContent("ALL");
    expect(screen.getByTestId("tab-all")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();
  });

  it("shows total memory count badge in header", () => {
    renderPage();
    expect(screen.getByTestId("knowledge-shell")).toBeInTheDocument();
    const header = screen.getByTestId("knowledge-shell-title").closest("header");
    expect(header).not.toBeNull();
    expect(within(header as HTMLElement).getByText("5")).toBeInTheDocument();
  });

  it("GLOBAL tab activates when clicked", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-global"));
    expect(screen.getByTestId("tab-global")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("tab-all")).toHaveAttribute("aria-pressed", "false");
  });

  it("WORKSPACE tab activates when clicked", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-workspace"));
    expect(screen.getByTestId("tab-workspace")).toHaveAttribute("aria-pressed", "true");
  });

  it("clicking ALL tab returns to full list", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-global"));
    await user.click(screen.getByTestId("tab-all"));

    expect(screen.getByTestId("tab-all")).toHaveAttribute("aria-pressed", "true");
  });

  // -----------------------------------------------------------------------
  // Grouping
  // -----------------------------------------------------------------------

  it("groups memories by scope (GLOBAL before WORKSPACE) with counts", () => {
    renderPage();
    const groups = screen.getAllByTestId(/^knowledge-group-/).filter(el => {
      const testId = el.getAttribute("data-testid") ?? "";
      return testId === "knowledge-group-global" || testId === "knowledge-group-workspace";
    });
    expect(groups).toHaveLength(2);
    expect(groups[0]).toHaveAttribute("data-testid", "knowledge-group-global");
    expect(groups[1]).toHaveAttribute("data-testid", "knowledge-group-workspace");
    expect(
      within(screen.getByTestId("knowledge-group-header-global")).getByText("3")
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("knowledge-group-header-workspace")).getByText("2")
    ).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Selection
  // -----------------------------------------------------------------------

  it("selecting a memory highlights it with accent left bar", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("memory-item-feedback_testing.md"));

    const item = screen.getByTestId("memory-item-feedback_testing.md");
    expect(within(item).getByTestId("memory-active-indicator")).toBeInTheDocument();
  });

  it("auto-selects first memory when no selection is made", () => {
    renderPage();
    const item = screen.getByTestId("memory-item-user_role.md");
    expect(within(item).getByTestId("memory-active-indicator")).toBeInTheDocument();
  });

  it("detail panel renders when a memory is selected", () => {
    mockMemoryContent = "Some memory content here";
    renderPage();
    expect(screen.getByTestId("knowledge-detail-panel")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Detail panel
  // -----------------------------------------------------------------------

  it("detail panel shows type + scope MonoBadges for the selected memory", () => {
    mockMemories = [makeMemory({ type: "user", name: "User Role" })];
    mockMemoryContent = "content";
    renderPage();

    const typeBadge = screen.getByTestId("detail-type-badge");
    expect(typeBadge).toHaveTextContent("user");
    expect(typeBadge).toHaveAttribute("data-tone", "accent");

    const scopeBadge = screen.getByTestId("detail-scope-badge");
    expect(scopeBadge).toHaveTextContent("GLOBAL");
  });

  it("detail panel renders the markdown preview inside the CodeBlock primitive", () => {
    mockMemoryContent = "# Heading\n\nline one\nline two";
    renderPage();

    const preview = screen.getByTestId("content-preview");
    expect(preview).toBeInTheDocument();
    expect(preview).toHaveAttribute("data-slot", "code-block");
  });

  it("delete button opens the confirmation dialog without mutating yet", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    renderPage();

    await user.click(screen.getByTestId("delete-memory-btn"));

    expect(screen.getByTestId("knowledge-delete-dialog")).toBeInTheDocument();
    expect(mockDeleteMutate).not.toHaveBeenCalled();
  });

  it("confirming the delete dialog calls useDeleteMemory mutation", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    renderPage();

    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));

    expect(mockDeleteMutate).toHaveBeenCalledWith({
      filename: "user_role.md",
      scope: "global",
      workspace: "ws_test",
    });
  });

  it("cancelling the delete dialog closes it without mutating", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    renderPage();

    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.click(screen.getByTestId("cancel-delete-memory-btn"));

    expect(mockDeleteMutate).not.toHaveBeenCalled();
  });

  it("delete button is disabled while a delete is pending", () => {
    mockDeletePending = true;
    mockMemoryContent = "content";
    renderPage();

    expect(screen.getByTestId("delete-memory-btn")).toBeDisabled();
  });

  // -----------------------------------------------------------------------
  // Metadata table
  // -----------------------------------------------------------------------

  it("metadata rows cover type, scope, agent, and modified", () => {
    mockMemories = [
      makeMemory({
        filename: "workspace/ref_api.md",
        name: "API Reference",
        type: "reference",
        agent_name: "coder",
      }),
    ];
    mockMemoryContent = "content";
    renderPage();

    expect(screen.getByTestId("metadata-table")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Type")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Scope")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Agent")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Modified")).toBeInTheDocument();
  });

  it("metadata Modified row falls back to the original string for invalid dates", () => {
    mockMemories = [makeMemory({ mod_time: "not-a-date" })];
    mockMemoryContent = "content";
    renderPage();

    expect(screen.getByTestId("metadata-row-Modified")).toHaveTextContent("not-a-date");
  });

  // -----------------------------------------------------------------------
  // Type/scope badges
  // -----------------------------------------------------------------------

  it("list items show type MonoBadges (user, feedback, project, reference)", () => {
    renderPage();

    expect(screen.getAllByTestId("type-badge-user").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("type-badge-feedback").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("type-badge-project").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("type-badge-reference").length).toBeGreaterThanOrEqual(1);
  });

  it("list items show scope MonoBadges (GLOBAL, WS)", () => {
    renderPage();

    expect(screen.getAllByTestId("scope-badge-global").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("scope-badge-workspace").length).toBeGreaterThanOrEqual(1);
  });

  // -----------------------------------------------------------------------
  // Dream status
  // -----------------------------------------------------------------------

  it("dream status indicator shows in page header", () => {
    renderPage();
    expect(screen.getByTestId("dream-status")).toBeInTheDocument();
    expect(screen.getByText(/Dream:/)).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Search
  // -----------------------------------------------------------------------

  it("search input filters the memory list (case-insensitive, name/description/type)", async () => {
    const user = userEvent.setup();
    renderPage();

    const searchInput = screen.getByTestId("knowledge-search-input");
    await user.type(searchInput, "api reference");

    expect(screen.getByTestId("memory-item-workspace/ref_api.md")).toBeInTheDocument();
    expect(screen.queryByTestId("memory-item-user_role.md")).not.toBeInTheDocument();
  });

  it("search with no results shows the empty fallback", async () => {
    const user = userEvent.setup();
    renderPage();

    const searchInput = screen.getByTestId("knowledge-search-input");
    await user.type(searchInput, "zzzznotfound");

    expect(screen.getByTestId("knowledge-list-empty")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Loading / Error states
  // -----------------------------------------------------------------------

  it("loading state shows spinner", () => {
    mockMemoriesLoading = true;
    mockMemories = [];
    renderPage();

    expect(screen.getByTestId("knowledge-loading")).toBeInTheDocument();
  });

  it("error state shows the Empty error card", () => {
    mockMemoriesError = new Error("Network failure");
    mockMemories = [];
    renderPage();

    expect(screen.getByTestId("knowledge-error")).toBeInTheDocument();
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });

  it("empty memories list renders an Empty fallback inside the list panel", () => {
    mockMemories = [];
    renderPage();

    expect(screen.getByTestId("knowledge-list-empty")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Detail loading / error
  // -----------------------------------------------------------------------

  it("detail panel shows loading spinner when fetching content", () => {
    mockMemoryContentLoading = true;
    renderPage();
    expect(screen.getByTestId("knowledge-detail-loading")).toBeInTheDocument();
  });

  it("detail panel shows Empty error when content fetch fails", () => {
    mockMemoryContentError = new Error("Content fetch failed");
    renderPage();
    expect(screen.getByTestId("knowledge-detail-error")).toBeInTheDocument();
    expect(screen.getByText("Content fetch failed")).toBeInTheDocument();
  });

  it("detail panel shows Empty state when no memories exist", () => {
    mockMemories = [];
    renderPage();
    const empty = screen.getByTestId("knowledge-detail-empty");
    expect(empty).toBeInTheDocument();
    expect(
      within(empty).getByText("Select a memory to view details", { selector: "h3" })
    ).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Integration: full flow
  // -----------------------------------------------------------------------

  it("full page flow: load memories, select, view detail, confirm delete, switch tabs", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "Full content of the memory file";
    renderPage();

    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();

    await user.click(screen.getByTestId("memory-item-workspace/ref_api.md"));
    expect(screen.getByTestId("content-preview")).toBeInTheDocument();

    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));
    expect(mockDeleteMutate).toHaveBeenCalled();

    await user.click(screen.getByTestId("tab-global"));
    expect(screen.getByTestId("tab-global")).toHaveAttribute("aria-pressed", "true");

    await user.click(screen.getByTestId("tab-all"));
    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();
  });
});
