import { UIProvider } from "@agh/ui";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { MemoryDecision, MemoryHeader } from "../../types";

import { KnowledgeDetailPanel } from "../knowledge-detail-panel";

const MEMORY: MemoryHeader = {
  filename: "user-role.md",
  mod_time: "2026-04-09T10:00:00Z",
  name: "User Role",
  scope: "global",
  type: "user",
  recall_count: 4,
  injection: true,
  system_managed: false,
  description: "Guidance for the assistant.",
  last_recalled_at: "2026-04-09T11:00:00Z",
};

const AGENT_MEMORY: MemoryHeader = {
  filename: "cto-tone.md",
  mod_time: "2026-04-09T10:00:00Z",
  name: "CTO Tone",
  scope: "agent",
  agent_name: "cto",
  agent_tier: "workspace",
  workspace_id: "ws_launch",
  type: "user",
  recall_count: 6,
  injection: true,
  system_managed: false,
  staleness_banner: "Updated >7 days after last recall",
  superseded_by: "cto-tone-v2.md",
};

const SAMPLE_DECISION: MemoryDecision = {
  id: "dec_001",
  candidate_hash: "h",
  op: "update",
  scope: "global",
  source: "rule",
  confidence: 0.93,
  decided_at: "2026-04-09T11:00:00Z",
  applied_at: "2026-04-09T11:00:01Z",
  target_filename: "user-role.md",
  reason: "rule:exact-slug-collision",
  frontmatter: {
    filename: "user-role.md",
    mod_time: "2026-04-09T10:00:00Z",
    name: "User Role",
    type: "user",
  },
};

function renderDetail(props: Partial<React.ComponentProps<typeof KnowledgeDetailPanel>> = {}) {
  const merged: React.ComponentProps<typeof KnowledgeDetailPanel> = {
    memory: MEMORY,
    content: "# User Role\n\nBody content.",
    scope: "global",
    isLoading: false,
    error: null,
    onDelete: vi.fn(),
    isDeletePending: false,
    decisions: [],
    isDecisionsLoading: false,
    decisionsError: null,
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgeDetailPanel {...merged} />
    </UIProvider>
  );
}

describe("KnowledgeDetailPanel", () => {
  it("Should render the empty state when no memory is selected", () => {
    renderDetail({ memory: undefined, content: undefined });
    const empty = screen.getByTestId("knowledge-detail-empty");
    expect(empty).toBeInTheDocument();
    expect(
      within(empty).getByText("Select a memory to view details", { selector: "h3" })
    ).toBeInTheDocument();
  });

  it("Should render the loading spinner when isLoading is true", () => {
    renderDetail({ isLoading: true, content: undefined });
    expect(screen.getByTestId("knowledge-detail-loading")).toBeInTheDocument();
  });

  it("Should render the error fallback when error is set", () => {
    renderDetail({ error: new Error("Boom"), content: undefined });
    expect(screen.getByTestId("knowledge-detail-error")).toBeInTheDocument();
    expect(screen.getByText("Boom")).toBeInTheDocument();
  });

  it("Should render the markdown preview inside the CodeBlock primitive", () => {
    renderDetail();
    const preview = screen.getByTestId("content-preview");
    expect(preview).toHaveAttribute("data-slot", "code-block");
  });

  it("Should render type and scope chips with the correct tone", () => {
    renderDetail();
    expect(screen.getByTestId("detail-type-badge")).toHaveAttribute("data-tone", "accent");
    expect(screen.getByTestId("detail-type-badge")).toHaveTextContent("user");
    expect(screen.getByTestId("detail-scope-badge")).toHaveAttribute("data-tone", "neutral");
    expect(screen.getByTestId("detail-scope-badge")).toHaveTextContent("GLOBAL");
  });

  it("Should render Memory v2 metadata rows when present", () => {
    renderDetail({
      memory: AGENT_MEMORY,
      content: "agent body",
      scope: "agent",
    });
    expect(screen.getByTestId("metadata-row-Type")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Scope")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Agent tier")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Agent")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Workspace")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Recalls")).toHaveTextContent(/6/);
    expect(screen.getByTestId("metadata-row-Staleness")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Superseded by")).toBeInTheDocument();
    expect(screen.getByTestId("detail-superseded-badge")).toBeInTheDocument();
    expect(screen.getByTestId("detail-agent-tier-badge")).toBeInTheDocument();
  });

  it("Should hide the agent metadata row when agent_name is absent", () => {
    renderDetail();
    expect(screen.queryByTestId("metadata-row-Agent")).not.toBeInTheDocument();
  });

  it("Should open the delete dialog and emit onDelete when confirmed", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn().mockResolvedValue(undefined);
    renderDetail({ onDelete });

    await user.click(screen.getByTestId("delete-memory-btn"));
    expect(screen.getByTestId("knowledge-delete-dialog")).toBeInTheDocument();

    await user.type(screen.getByTestId("knowledge-delete-confirm-typing"), MEMORY.filename);
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));
    expect(onDelete).toHaveBeenCalledWith(MEMORY);
  });

  it("Should disable the delete button while a delete is pending", () => {
    renderDetail({ isDeletePending: true });
    expect(screen.getByTestId("delete-memory-btn")).toBeDisabled();
  });

  it("Should surface delete failures inline and inside the delete dialog", async () => {
    const user = userEvent.setup();
    renderDetail({ deleteError: "Delete failed" });

    expect(screen.getByTestId("knowledge-delete-error")).toHaveTextContent("Delete failed");
    await user.click(screen.getByTestId("delete-memory-btn"));
    expect(screen.getByTestId("knowledge-delete-dialog-error")).toHaveTextContent("Delete failed");
  });

  it("Should hide the edit button when no edit handler is provided", () => {
    renderDetail();
    expect(screen.queryByTestId("edit-memory-btn")).not.toBeInTheDocument();
  });

  it("Should open the edit dialog and submit the new content via onEdit", async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn().mockResolvedValue(undefined);
    renderDetail({ onEdit });

    await user.click(screen.getByTestId("edit-memory-btn"));
    const contentArea = screen.getByTestId("knowledge-edit-content");
    await user.type(contentArea, "\nMore body");

    await user.click(screen.getByTestId("confirm-edit-memory-btn"));
    expect(onEdit).toHaveBeenCalledWith(MEMORY, {
      content: "# User Role\n\nBody content.\nMore body",
      description: "Guidance for the assistant.",
    });
  });

  it("Should disable the edit button while content is unavailable", () => {
    renderDetail({ onEdit: vi.fn(), content: undefined });
    expect(screen.getByTestId("edit-memory-btn")).toBeDisabled();
  });

  it("Should surface edit failures inline and inside the edit dialog", async () => {
    const user = userEvent.setup();
    renderDetail({ onEdit: vi.fn(), editError: "Edit failed" });

    expect(screen.getByTestId("knowledge-edit-error")).toHaveTextContent("Edit failed");
    await user.click(screen.getByTestId("edit-memory-btn"));
    expect(screen.getByTestId("knowledge-edit-dialog-error")).toHaveTextContent("Edit failed");
  });

  it("Should render the controller decisions section when decisions are present", () => {
    renderDetail({ decisions: [SAMPLE_DECISION] });
    expect(screen.getByTestId("knowledge-decisions-list")).toBeInTheDocument();
    expect(screen.getByTestId(`knowledge-decision-${SAMPLE_DECISION.id}`)).toBeInTheDocument();
  });

  it("Should render the empty decisions fallback when there are no decisions", () => {
    renderDetail();
    expect(screen.getByTestId("knowledge-decisions-empty")).toBeInTheDocument();
  });

  it("Should render the decisions error fallback when the decisions query fails", () => {
    renderDetail({ decisionsError: new Error("Decisions failed") });
    expect(screen.getByTestId("knowledge-decisions-error")).toBeInTheDocument();
    expect(screen.getByText("Decisions failed")).toBeInTheDocument();
  });

  it("Should render the decisions loading state while decisions are loading", () => {
    renderDetail({ isDecisionsLoading: true });
    expect(screen.getByTestId("knowledge-decisions-loading")).toBeInTheDocument();
  });
});
