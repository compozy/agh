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
    render(<KnowledgePage />);
    expect(screen.getByTestId("tab-all")).toHaveTextContent("ALL");
    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();
  });

  it("shows total memory count badge in header", () => {
    render(<KnowledgePage />);
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("GLOBAL tab filters to show only global-scope memories", async () => {
    const user = userEvent.setup();
    render(<KnowledgePage />);

    await user.click(screen.getByTestId("tab-global"));

    // Tab should be active
    expect(screen.getByTestId("tab-global").className).toContain("bg-[color:var(--color-accent)]");
  });

  it("WORKSPACE tab filters to show only workspace-scope memories", async () => {
    const user = userEvent.setup();
    render(<KnowledgePage />);

    await user.click(screen.getByTestId("tab-workspace"));

    expect(screen.getByTestId("tab-workspace").className).toContain(
      "bg-[color:var(--color-accent)]"
    );
  });

  it("clicking ALL tab returns to full list", async () => {
    const user = userEvent.setup();
    render(<KnowledgePage />);

    await user.click(screen.getByTestId("tab-global"));
    await user.click(screen.getByTestId("tab-all"));

    expect(screen.getByTestId("tab-all").className).toContain("bg-[color:var(--color-accent)]");
  });

  // -----------------------------------------------------------------------
  // Grouping
  // -----------------------------------------------------------------------

  it("groups memories by scope (GLOBAL, WORKSPACE) with counts", () => {
    render(<KnowledgePage />);
    const globalGroup = screen.getByTestId("knowledge-group-global");
    expect(globalGroup).toBeInTheDocument();
    expect(within(globalGroup).getByText("3")).toBeInTheDocument();

    const wsGroup = screen.getByTestId("knowledge-group-workspace");
    expect(wsGroup).toBeInTheDocument();
    expect(within(wsGroup).getByText("2")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Selection
  // -----------------------------------------------------------------------

  it("selecting a memory highlights it with accent left bar", async () => {
    const user = userEvent.setup();
    render(<KnowledgePage />);

    await user.click(screen.getByTestId("memory-item-feedback_testing.md"));

    const item = screen.getByTestId("memory-item-feedback_testing.md");
    expect(within(item).getByTestId("memory-active-indicator")).toBeInTheDocument();
  });

  it("auto-selects first memory when no selection is made", () => {
    render(<KnowledgePage />);
    // First memory in the data array is user_role.md
    const item = screen.getByTestId("memory-item-user_role.md");
    expect(within(item).getByTestId("memory-active-indicator")).toBeInTheDocument();
  });

  it("selecting a memory shows detail panel with correct title and description", () => {
    mockMemoryContent = "Some memory content here";
    render(<KnowledgePage />);

    const detailPanel = screen.getByTestId("knowledge-detail-panel");
    // Auto-selected first memory (sorted alphabetically by name)
    expect(detailPanel).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Detail panel
  // -----------------------------------------------------------------------

  it("detail panel shows USER badge with accent tint for user-type memories", () => {
    mockMemories = [makeMemory({ type: "user", name: "User Role" })];
    mockMemoryContent = "content";
    render(<KnowledgePage />);

    const badge = screen.getByTestId("detail-type-badge");
    expect(badge).toHaveTextContent("user");
    expect(badge.className).toContain("text-[#e8572a]");
  });

  it("detail panel shows content preview card with truncated content", () => {
    const longContent = "A".repeat(400);
    mockMemoryContent = longContent;
    render(<KnowledgePage />);

    expect(screen.getByTestId("content-preview")).toBeInTheDocument();
    expect(screen.getByTestId("view-full-content-link")).toBeInTheDocument();
  });

  it("detail panel 'View full content' link is clickable", () => {
    mockMemoryContent = "A".repeat(400);
    render(<KnowledgePage />);

    const link = screen.getByTestId("view-full-content-link");
    expect(link.tagName).toBe("BUTTON");
    expect(link).toBeDisabled();
  });

  it("detail panel Delete button calls useDeleteMemory mutation", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    render(<KnowledgePage />);

    await user.click(screen.getByTestId("delete-memory-btn"));

    expect(mockDeleteMutate).toHaveBeenCalledWith({
      filename: "user_role.md",
      scope: "global",
      workspace: "ws_test",
    });
  });

  // -----------------------------------------------------------------------
  // Metadata table
  // -----------------------------------------------------------------------

  it("metadata table renders striped rows for type, scope, agent, modified", () => {
    mockMemories = [
      makeMemory({
        filename: "workspace/ref_api.md",
        name: "API Reference",
        type: "reference",
        agent_name: "coder",
      }),
    ];
    mockMemoryContent = "content";
    render(<KnowledgePage />);

    expect(screen.getByTestId("metadata-table")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Type")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Scope")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Agent")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Modified")).toBeInTheDocument();
  });

  it("detail panel falls back to the original timestamp text for invalid dates", () => {
    mockMemories = [makeMemory({ mod_time: "not-a-date" })];
    mockMemoryContent = "content";
    render(<KnowledgePage />);

    expect(screen.getByTestId("metadata-row-Modified")).toHaveTextContent("not-a-date");
  });

  // -----------------------------------------------------------------------
  // Type/scope badges
  // -----------------------------------------------------------------------

  it("list items show type badges (user, feedback, project, reference)", () => {
    render(<KnowledgePage />);

    expect(screen.getAllByTestId("type-badge-user").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("type-badge-feedback").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("type-badge-project").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("type-badge-reference").length).toBeGreaterThanOrEqual(1);
  });

  it("list items show scope badges (GLOBAL, WS)", () => {
    render(<KnowledgePage />);

    expect(screen.getAllByTestId("scope-badge-global").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByTestId("scope-badge-workspace").length).toBeGreaterThanOrEqual(1);
  });

  // -----------------------------------------------------------------------
  // Dream status
  // -----------------------------------------------------------------------

  it("dream status indicator shows in page header", () => {
    render(<KnowledgePage />);
    expect(screen.getByTestId("dream-status")).toBeInTheDocument();
    expect(screen.getByText(/Dream:/)).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Search
  // -----------------------------------------------------------------------

  it("search input filters the memory list", async () => {
    const user = userEvent.setup();
    render(<KnowledgePage />);

    const searchInput = screen.getByTestId("knowledge-search-input");
    await user.type(searchInput, "API Reference");

    expect(screen.getByTestId("memory-item-workspace/ref_api.md")).toBeInTheDocument();
    expect(screen.queryByTestId("memory-item-user_role.md")).not.toBeInTheDocument();
  });

  it("search with no results shows empty message", async () => {
    const user = userEvent.setup();
    render(<KnowledgePage />);

    const searchInput = screen.getByTestId("knowledge-search-input");
    await user.type(searchInput, "zzzznotfound");

    expect(screen.getByTestId("knowledge-list-empty")).toHaveTextContent(
      "No knowledge items found"
    );
  });

  // -----------------------------------------------------------------------
  // Loading / Error states
  // -----------------------------------------------------------------------

  it("loading state shows spinner", () => {
    mockMemoriesLoading = true;
    mockMemories = [];
    render(<KnowledgePage />);

    expect(screen.getByTestId("knowledge-loading")).toBeInTheDocument();
  });

  it("error state shows appropriate message", () => {
    mockMemoriesError = new Error("Network failure");
    mockMemories = [];
    render(<KnowledgePage />);

    expect(screen.getByTestId("knowledge-error")).toBeInTheDocument();
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });

  it("empty memories list shows empty message in list panel", () => {
    mockMemories = [];
    render(<KnowledgePage />);

    expect(screen.getByTestId("knowledge-list-empty")).toHaveTextContent(
      "No knowledge items found"
    );
  });

  // -----------------------------------------------------------------------
  // Detail loading / error
  // -----------------------------------------------------------------------

  it("detail panel shows loading spinner when fetching content", () => {
    mockMemoryContentLoading = true;
    render(<KnowledgePage />);
    expect(screen.getByTestId("knowledge-detail-loading")).toBeInTheDocument();
  });

  it("detail panel shows error when content fetch fails", () => {
    mockMemoryContentError = new Error("Content fetch failed");
    render(<KnowledgePage />);
    expect(screen.getByTestId("knowledge-detail-error")).toHaveTextContent(
      "Failed to load memory details"
    );
  });

  it("detail panel shows empty state when no memories exist", () => {
    mockMemories = [];
    render(<KnowledgePage />);
    expect(screen.getByTestId("knowledge-detail-empty")).toHaveTextContent(
      "Select a memory to view details"
    );
  });

  // -----------------------------------------------------------------------
  // Integration: full flow
  // -----------------------------------------------------------------------

  it("full page flow: load memories, select memory, view detail, delete memory", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "Full content of the memory file";
    render(<KnowledgePage />);

    // Memories are loaded and displayed
    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();

    // Select a different memory
    await user.click(screen.getByTestId("memory-item-workspace/ref_api.md"));

    // Detail panel shows content
    expect(screen.getByTestId("content-preview")).toBeInTheDocument();

    // Delete the memory
    await user.click(screen.getByTestId("delete-memory-btn"));
    expect(mockDeleteMutate).toHaveBeenCalled();

    // Switch tabs
    await user.click(screen.getByTestId("tab-global"));
    expect(screen.getByTestId("tab-global").className).toContain("bg-[color:var(--color-accent)]");

    // Switch back to all
    await user.click(screen.getByTestId("tab-all"));
    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();
  });
});
