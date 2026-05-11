import { UIProvider } from "@agh/ui";
import { screen, within } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { MemoryDecision, MemoryHeader, MemorySearchResponse } from "@/systems/knowledge/types";

// ---------------------------------------------------------------------------
// Mock state
// ---------------------------------------------------------------------------

interface SelectorLike {
  scope?: string;
  workspaceId?: string;
  agentName?: string;
  agentTier?: string;
}

let mockGlobalMemories: MemoryHeader[] = [];
let mockWorkspaceMemories: MemoryHeader[] = [];
let mockAgentMemories: MemoryHeader[] = [];
let mockGlobalMemoriesLoading = false;
let mockWorkspaceMemoriesLoading = false;
let mockAgentMemoriesLoading = false;
let mockGlobalMemoriesError: Error | null = null;
let mockWorkspaceMemoriesError: Error | null = null;
let mockAgentMemoriesError: Error | null = null;

let mockMemoryContent: string | undefined;
let mockMemoryContentLoading = false;
let mockMemoryContentError: Error | null = null;

let mockSearchResponse: MemorySearchResponse | undefined;
let mockSearchLoading = false;
let mockSearchError: Error | null = null;

let mockDecisions: MemoryDecision[] = [];
let mockDecisionsLoading = false;
let mockDecisionsError: Error | null = null;

const mockDeleteMutateAsync = vi.fn();
const mockDeleteReset = vi.fn();
let mockDeletePending = false;
let mockDeleteError: Error | null = null;

const mockEditMutateAsync = vi.fn();
const mockEditReset = vi.fn();
let mockEditPending = false;
let mockEditError: Error | null = null;

const mockWriteMutateAsync = vi.fn();
const mockWriteReset = vi.fn();
let mockWritePending = false;
let mockWriteError: Error | null = null;

const mockRevertMutateAsync = vi.fn();
const mockRevertReset = vi.fn();
let mockRevertPending = false;
let mockRevertError: Error | null = null;

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
    useMemories: (selector?: SelectorLike) => {
      if (!selector) {
        return { data: [], isLoading: false, error: null };
      }
      if (selector.scope === "workspace") {
        return {
          data: mockWorkspaceMemories,
          isLoading: mockWorkspaceMemoriesLoading,
          error: mockWorkspaceMemoriesError,
        };
      }
      if (selector.scope === "agent") {
        return {
          data: mockAgentMemories,
          isLoading: mockAgentMemoriesLoading,
          error: mockAgentMemoriesError,
        };
      }
      return {
        data: mockGlobalMemories,
        isLoading: mockGlobalMemoriesLoading,
        error: mockGlobalMemoriesError,
      };
    },
    useMemory: () => ({
      data:
        mockMemoryContent === undefined
          ? undefined
          : { content: mockMemoryContent, filename: "user_role.md" },
      isLoading: mockMemoryContentLoading,
      error: mockMemoryContentError,
    }),
    useMemorySearch: () => ({
      data: mockSearchResponse,
      isLoading: mockSearchLoading,
      error: mockSearchError,
    }),
    useMemoryDecisions: () => ({
      data: { decisions: mockDecisions },
      isLoading: mockDecisionsLoading,
      error: mockDecisionsError,
    }),
    useDeleteMemory: () => ({
      mutateAsync: mockDeleteMutateAsync,
      reset: mockDeleteReset,
      isPending: mockDeletePending,
      error: mockDeleteError,
    }),
    useEditMemory: () => ({
      mutateAsync: mockEditMutateAsync,
      reset: mockEditReset,
      isPending: mockEditPending,
      error: mockEditError,
    }),
    useWriteMemory: () => ({
      mutateAsync: mockWriteMutateAsync,
      reset: mockWriteReset,
      isPending: mockWritePending,
      error: mockWriteError,
    }),
    useRevertMemoryDecision: () => ({
      mutateAsync: mockRevertMutateAsync,
      reset: mockRevertReset,
      isPending: mockRevertPending,
      error: mockRevertError,
    }),
  };
});

import { KnowledgePage } from "../knowledge";

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

function makeMemory(overrides: Partial<MemoryHeader> = {}): MemoryHeader {
  return {
    filename: "user_role.md",
    mod_time: "2026-04-09T10:00:00Z",
    name: "User Role",
    description: "User is a senior engineer",
    scope: "global",
    type: "user",
    recall_count: 0,
    injection: true,
    system_managed: false,
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
    filename: "ref_api.md",
    name: "API Reference",
    description: "REST API docs at docs.internal",
    type: "reference",
    mod_time: "2026-04-06T11:00:00Z",
    scope: "workspace",
    workspace_id: "ws_test",
  }),
  makeMemory({
    filename: "project_sprint.md",
    name: "Sprint Planning",
    description: "Sprint 5 goals and deadlines",
    type: "project",
    mod_time: "2026-04-05T08:00:00Z",
    scope: "workspace",
    workspace_id: "ws_test",
  }),
];

const AGENT_MEMORIES: MemoryHeader[] = [
  makeMemory({
    filename: "cto_tone.md",
    name: "CTO Tone",
    description: "Direct, calm tone for CTO summaries",
    type: "user",
    mod_time: "2026-04-09T11:00:00Z",
    scope: "agent",
    agent_name: "cto",
    agent_tier: "workspace",
    workspace_id: "ws_test",
  }),
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
    mockGlobalMemories = GLOBAL_MEMORIES;
    mockWorkspaceMemories = WORKSPACE_MEMORIES;
    mockAgentMemories = AGENT_MEMORIES;
    mockGlobalMemoriesLoading = false;
    mockWorkspaceMemoriesLoading = false;
    mockAgentMemoriesLoading = false;
    mockGlobalMemoriesError = null;
    mockWorkspaceMemoriesError = null;
    mockAgentMemoriesError = null;
    mockMemoryContent = undefined;
    mockMemoryContentLoading = false;
    mockMemoryContentError = null;
    mockSearchResponse = undefined;
    mockSearchLoading = false;
    mockSearchError = null;
    mockDecisions = [];
    mockDecisionsLoading = false;
    mockDecisionsError = null;
    mockDeletePending = false;
    mockDeleteError = null;
    mockEditPending = false;
    mockEditError = null;
    mockWritePending = false;
    mockWriteError = null;
    mockRevertPending = false;
    mockRevertError = null;
    mockDeleteMutateAsync.mockReset();
    mockDeleteMutateAsync.mockResolvedValue(undefined);
    mockDeleteReset.mockReset();
    mockEditMutateAsync.mockReset();
    mockEditMutateAsync.mockResolvedValue(undefined);
    mockEditReset.mockReset();
    mockWriteMutateAsync.mockReset();
    mockWriteMutateAsync.mockResolvedValue({
      applied: true,
      decision: {
        id: "dec_write",
        candidate_hash: "h",
        op: "add",
        scope: "global",
        source: "rule",
        confidence: 1,
        decided_at: "2026-04-25T21:03:00Z",
        target_filename: "new-entry.md",
        frontmatter: {
          filename: "new-entry.md",
          mod_time: "2026-04-25T21:03:00Z",
          name: "New Entry",
          type: "user",
        },
      },
    });
    mockWriteReset.mockReset();
    mockRevertMutateAsync.mockReset();
    mockRevertMutateAsync.mockResolvedValue({ reverted: true });
    mockRevertReset.mockReset();
  });

  it("Should default to the GLOBAL scope and render the global memory list", () => {
    renderPage();
    expect(screen.getByTestId("tab-global")).toHaveTextContent("GLOBAL");
    expect(screen.getByTestId("tab-global")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("knowledge-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("memory-item-global:user_role.md")).toBeInTheDocument();
  });

  it("Should switch to the WORKSPACE scope when clicked", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-workspace"));
    expect(screen.getByTestId("tab-workspace")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("memory-item-workspace:ref_api.md")).toBeInTheDocument();
  });

  it("Should reveal agent inputs and require an agent name on the AGENT scope", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-agent"));
    expect(screen.getByTestId("agent-name-input")).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-guard")).toBeInTheDocument();

    await user.type(screen.getByTestId("agent-name-input"), "cto");
    expect(screen.queryByTestId("knowledge-guard")).not.toBeInTheDocument();
    expect(screen.getByTestId("memory-item-agent:cto_tone.md")).toBeInTheDocument();
  });

  it("Should render scope-aware metadata badges (agent tier, recall count, system flag)", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-agent"));
    await user.type(screen.getByTestId("agent-name-input"), "cto");

    expect(screen.getByTestId("agent-tier-badge-workspace")).toBeInTheDocument();
    expect(screen.getByTestId("agent-name-badge")).toHaveTextContent("cto");
  });

  it("Should switch to server-backed search when a query is entered", async () => {
    const user = userEvent.setup();
    mockSearchResponse = {
      results: [
        {
          memory: {
            ...AGENT_MEMORIES[0],
          },
          score: 0.92,
          snippet: "match",
          why_recalled: ["fts5"],
        },
      ],
      recall: { blocks: [], header: { content_hash: "h", text: "" } },
    };

    renderPage();
    await user.click(screen.getByTestId("tab-agent"));
    await user.type(screen.getByTestId("agent-name-input"), "cto");
    await user.type(screen.getByTestId("knowledge-search-input"), "tone");

    expect(screen.getByTestId("knowledge-search-info")).toHaveTextContent(/Recall/);
    expect(screen.getByTestId("memory-item-agent:cto_tone.md")).toBeInTheDocument();
  });

  it("Should render the detail panel with the Overview ContextBox metadata grid", () => {
    mockMemoryContent = "# Memory content";
    renderPage();

    expect(screen.getByTestId("knowledge-detail-context")).toBeInTheDocument();
    expect(screen.getByTestId("context-type-value")).toBeInTheDocument();
    expect(screen.getByTestId("context-tier-value")).toBeInTheDocument();
    expect(screen.getByTestId("context-recalls-value")).toBeInTheDocument();
    expect(screen.getByTestId("context-injection-value")).toBeInTheDocument();
  });

  it("Should open the delete dialog and call the delete mutation with the full selector", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    renderPage();

    await user.click(screen.getByTestId("memory-item-global:user_role.md"));
    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.type(screen.getByTestId("knowledge-delete-confirm-typing"), "user_role.md");
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));

    expect(mockDeleteMutateAsync).toHaveBeenCalledWith({
      selector: {
        scope: "global",
        workspaceId: undefined,
        agentName: undefined,
        agentTier: undefined,
      },
      filename: "user_role.md",
    });
  });

  it("Should open the edit dialog and submit the controller edit body", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "# original content\n";
    renderPage();

    await user.click(screen.getByTestId("memory-item-global:user_role.md"));
    await user.click(screen.getByTestId("edit-memory-btn"));

    const contentInput = screen.getByTestId("knowledge-edit-content");
    await user.type(contentInput, " edited");

    await user.click(screen.getByTestId("confirm-edit-memory-btn"));

    expect(mockEditMutateAsync).toHaveBeenCalledWith({
      filename: "user_role.md",
      body: expect.objectContaining({
        content: "# original content\n edited",
        scope: "global",
        type: "user",
        name: "User Role",
      }),
    });
  });

  it("Should open the create dialog and submit a scoped controller write", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("tab-workspace"));
    await user.click(screen.getByTestId("create-memory-btn"));
    await user.click(screen.getByTestId("knowledge-create-type-project"));
    await user.type(screen.getByTestId("knowledge-create-name"), "Launch Memory");
    await user.type(screen.getByTestId("knowledge-create-description"), "workspace contract");
    await user.type(screen.getByTestId("knowledge-create-content"), "Use the launch playbook.");
    await user.click(screen.getByTestId("confirm-create-memory-btn"));

    expect(mockWriteMutateAsync).toHaveBeenCalledWith({
      scope: "workspace",
      workspace_id: "ws_test",
      agent_name: undefined,
      agent_tier: undefined,
      type: "project",
      name: "Launch Memory",
      description: "workspace contract",
      content: "Use the launch playbook.",
    });
  });

  it("Should show the controller decisions section with returned decisions", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    mockDecisions = [
      {
        id: "dec_1",
        candidate_hash: "h",
        op: "update",
        scope: "global",
        source: "rule",
        confidence: 0.9,
        decided_at: "2026-04-25T21:03:00Z",
        target_filename: "user_role.md",
        frontmatter: {
          filename: "user_role.md",
          mod_time: "2026-04-25T21:00:00Z",
          name: "User Role",
          type: "user",
        },
      },
    ];

    renderPage();

    await user.click(screen.getByTestId("memory-item-global:user_role.md"));

    expect(screen.getByTestId("knowledge-decisions-list")).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-decision-dec_1")).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-decision-op-dec_1")).toHaveTextContent("update");
  });

  it("Should revert an applied controller decision from the detail panel", async () => {
    const user = userEvent.setup();
    mockMemoryContent = "content";
    mockDecisions = [
      {
        id: "dec_1",
        candidate_hash: "h",
        op: "update",
        scope: "global",
        source: "rule",
        confidence: 0.9,
        decided_at: "2026-04-25T21:03:00Z",
        applied_at: "2026-04-25T21:03:02Z",
        target_filename: "user_role.md",
        frontmatter: {
          filename: "user_role.md",
          mod_time: "2026-04-25T21:00:00Z",
          name: "User Role",
          type: "user",
        },
      },
    ];

    renderPage();

    await user.click(screen.getByTestId("memory-item-global:user_role.md"));
    await user.click(screen.getByTestId("revert-memory-decision-dec_1"));

    expect(mockRevertMutateAsync).toHaveBeenCalledWith({
      decisionID: "dec_1",
      body: { reason: "operator reverted from Knowledge" },
    });
  });

  it("Should show empty decisions state when no decisions are returned", () => {
    mockMemoryContent = "content";
    renderPage();
    expect(screen.getByTestId("knowledge-decisions-empty")).toBeInTheDocument();
  });

  it("Should show the loading spinner when the list query is loading", () => {
    mockGlobalMemoriesLoading = true;
    mockGlobalMemories = [];
    renderPage();
    expect(screen.getByTestId("knowledge-loading")).toBeInTheDocument();
  });

  it("Should show an Empty error card when the list query fails", () => {
    mockGlobalMemoriesError = new Error("Network failure");
    mockGlobalMemories = [];
    renderPage();
    expect(screen.getByTestId("knowledge-error")).toBeInTheDocument();
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });

  it("Should show the empty list fallback when there are no memories", () => {
    mockGlobalMemories = [];
    renderPage();
    expect(screen.getByTestId("knowledge-list-empty")).toBeInTheDocument();
  });

  it("Should show the detail loading spinner while content fetches", () => {
    mockMemoryContentLoading = true;
    renderPage();
    expect(screen.getByTestId("knowledge-detail-loading")).toBeInTheDocument();
  });

  it("Should surface a detail error when the content fetch fails", () => {
    mockMemoryContentError = new Error("Content fetch failed");
    renderPage();
    expect(screen.getByTestId("knowledge-detail-error")).toBeInTheDocument();
    expect(screen.getByText("Content fetch failed")).toBeInTheDocument();
  });

  it("Should surface a detail empty state when no memory is selected", () => {
    mockGlobalMemories = [];
    renderPage();
    const empty = screen.getByTestId("knowledge-detail-empty");
    expect(empty).toBeInTheDocument();
    expect(
      within(empty).getByText("Select a memory to view details", { selector: "h3" })
    ).toBeInTheDocument();
  });

  it("Should surface a search error when the recall query fails", async () => {
    const user = userEvent.setup();
    mockSearchError = new Error("Recall failed");
    mockSearchResponse = {
      results: [],
      recall: { blocks: [], header: { content_hash: "h", text: "" } },
    };
    renderPage();

    await user.type(screen.getByTestId("knowledge-search-input"), "anything");

    expect(screen.getByTestId("knowledge-error")).toBeInTheDocument();
    expect(screen.getByText("Recall failed")).toBeInTheDocument();
  });

  it("Should surface a delete failure inline when the mutation rejects", async () => {
    const user = userEvent.setup();
    mockDeleteMutateAsync.mockImplementation(async () => {
      mockDeleteError = new Error("Delete failed");
      throw mockDeleteError;
    });
    mockMemoryContent = "content";
    renderPage();

    await user.click(screen.getByTestId("memory-item-global:user_role.md"));
    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.type(screen.getByTestId("knowledge-delete-confirm-typing"), "user_role.md");
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));

    expect(await screen.findByTestId("knowledge-delete-error")).toHaveTextContent("Delete failed");
  });
});
