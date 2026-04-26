import { UIProvider } from "@agh/ui";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { MemoryHeader } from "../types";

import { KnowledgeDetailPanel } from "./knowledge-detail-panel";

const MEMORY: MemoryHeader = {
  filename: "global/user-role.md",
  mod_time: "2026-04-09T10:00:00Z",
  name: "User Role",
  type: "user",
  description: "Guidance for the assistant.",
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
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgeDetailPanel {...merged} />
    </UIProvider>
  );
}

describe("KnowledgeDetailPanel", () => {
  it("renders Empty state when no memory is selected", () => {
    renderDetail({ memory: undefined, content: undefined });
    const empty = screen.getByTestId("knowledge-detail-empty");
    expect(empty).toBeInTheDocument();
    expect(
      within(empty).getByText("Select a memory to view details", { selector: "h3" })
    ).toBeInTheDocument();
  });

  it("renders the loading spinner when isLoading is true", () => {
    renderDetail({ isLoading: true, content: undefined });
    expect(screen.getByTestId("knowledge-detail-loading")).toBeInTheDocument();
  });

  it("renders the error Empty card when error is set", () => {
    renderDetail({ error: new Error("Boom"), content: undefined });
    expect(screen.getByTestId("knowledge-detail-error")).toBeInTheDocument();
    expect(screen.getByText("Boom")).toBeInTheDocument();
  });

  it("renders markdown preview inside the CodeBlock primitive", () => {
    renderDetail();
    const preview = screen.getByTestId("content-preview");
    expect(preview).toHaveAttribute("data-slot", "code-block");
  });

  it("renders type + scope MonoBadge chips with correct tone", () => {
    renderDetail();
    expect(screen.getByTestId("detail-type-badge")).toHaveAttribute("data-tone", "accent");
    expect(screen.getByTestId("detail-type-badge")).toHaveTextContent("user");
    expect(screen.getByTestId("detail-scope-badge")).toHaveAttribute("data-tone", "neutral");
    expect(screen.getByTestId("detail-scope-badge")).toHaveTextContent("GLOBAL");
  });

  it("renders metadata rows for Type, Scope, Agent (when present), and Modified", () => {
    renderDetail({ memory: { ...MEMORY, agent_name: "coder" } });
    expect(screen.getByTestId("metadata-row-Type")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Scope")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Agent")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-Modified")).toBeInTheDocument();
  });

  it("hides the Agent metadata row when agent_name is absent", () => {
    renderDetail();
    expect(screen.queryByTestId("metadata-row-Agent")).not.toBeInTheDocument();
  });

  it("opens the delete confirmation dialog on delete button click", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    renderDetail({ onDelete });

    await user.click(screen.getByTestId("delete-memory-btn"));
    expect(screen.getByTestId("knowledge-delete-dialog")).toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("calls onDelete with the selected memory when confirm is clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn().mockResolvedValue(undefined);
    renderDetail({ onDelete });

    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));

    expect(onDelete).toHaveBeenCalledWith(MEMORY);
  });

  it("does not call onDelete when cancel is clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    renderDetail({ onDelete });

    await user.click(screen.getByTestId("delete-memory-btn"));
    await user.click(screen.getByTestId("cancel-delete-memory-btn"));

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("disables the delete button while a mutation is pending", () => {
    renderDetail({ isDeletePending: true });
    expect(screen.getByTestId("delete-memory-btn")).toBeDisabled();
  });

  it("surfaces delete failures inline and inside the dialog", async () => {
    const user = userEvent.setup();
    renderDetail({ deleteError: "Delete failed" });

    expect(screen.getByTestId("knowledge-delete-error")).toHaveTextContent("Delete failed");

    await user.click(screen.getByTestId("delete-memory-btn"));

    expect(screen.getByTestId("knowledge-delete-dialog-error")).toHaveTextContent("Delete failed");
  });

  it("falls back to deriving scope from filename when scope prop is omitted", () => {
    renderDetail({
      memory: { ...MEMORY, filename: "workspace/foo.md" },
      scope: undefined,
    });
    expect(screen.getByTestId("detail-scope-badge")).toHaveTextContent("WORKSPACE");
  });
});
