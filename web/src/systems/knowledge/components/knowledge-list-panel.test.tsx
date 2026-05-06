import { UIProvider } from "@agh/ui";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { KnowledgeMemoryItem } from "../types";

import { KnowledgeListPanel } from "./knowledge-list-panel";

const GLOBAL: KnowledgeMemoryItem = {
  filename: "user-role.md",
  key: "global:user-role.md",
  mod_time: "2026-04-09T10:00:00Z",
  name: "User Role",
  scope: "global",
  type: "user",
  recall_count: 2,
  injection: true,
  system_managed: false,
  description: "Guidance that shapes the assistant's tone and ownership.",
};

const WORKSPACE: KnowledgeMemoryItem = {
  filename: "project-context.md",
  key: "workspace:project-context.md",
  mod_time: "2026-04-09T08:00:00Z",
  name: "Project Context",
  scope: "workspace",
  type: "project",
  recall_count: 0,
  injection: true,
  system_managed: false,
  description: "Workspace-local notes about rollout.",
  workspace_id: "ws_launch",
};

const AGENT: KnowledgeMemoryItem = {
  filename: "cto-tone.md",
  key: "agent:cto-tone.md",
  mod_time: "2026-04-09T09:00:00Z",
  name: "CTO Tone",
  scope: "agent",
  agent_name: "cto",
  agent_tier: "workspace",
  workspace_id: "ws_launch",
  type: "user",
  recall_count: 5,
  injection: true,
  system_managed: false,
  staleness_banner: "Updated >7 days after last recall",
};

const ALL: KnowledgeMemoryItem[] = [GLOBAL, WORKSPACE, AGENT];

function renderPanel(props: Partial<React.ComponentProps<typeof KnowledgeListPanel>> = {}) {
  const merged: React.ComponentProps<typeof KnowledgeListPanel> = {
    memories: ALL,
    onSearchChange: vi.fn(),
    onSelectMemory: vi.fn(),
    searchQuery: "",
    selectedMemoryKey: null,
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgeListPanel {...merged} />
    </UIProvider>
  );
}

describe("KnowledgeListPanel", () => {
  it("Should render groups in scope order with count chips", () => {
    renderPanel();
    const groups = screen.getAllByTestId(/^knowledge-group-/).filter(element => {
      const id = element.getAttribute("data-testid") ?? "";
      return [
        "knowledge-group-global",
        "knowledge-group-workspace",
        "knowledge-group-agent",
      ].includes(id);
    });
    expect(groups[0]).toHaveAttribute("data-testid", "knowledge-group-global");
    expect(groups[1]).toHaveAttribute("data-testid", "knowledge-group-workspace");
    expect(groups[2]).toHaveAttribute("data-testid", "knowledge-group-agent");
    expect(
      within(screen.getByTestId("knowledge-group-header-global")).getByText("1")
    ).toBeInTheDocument();
  });

  it("Should render type, scope, and agent tier badges per row", () => {
    renderPanel();
    const userBadges = screen.getAllByTestId("type-badge-user");
    expect(userBadges.length).toBeGreaterThanOrEqual(2);
    userBadges.forEach(badge => {
      expect(badge).toHaveAttribute("data-tone", "accent");
    });
    expect(screen.getByTestId("type-badge-project")).toHaveAttribute("data-tone", "success");
    expect(screen.getByTestId("scope-badge-global")).toHaveAttribute("data-tone", "neutral");
    expect(screen.getByTestId("scope-badge-workspace")).toHaveAttribute("data-tone", "info");
    expect(screen.getByTestId("scope-badge-agent")).toHaveAttribute("data-tone", "warning");
    expect(screen.getByTestId("agent-tier-badge-workspace")).toBeInTheDocument();
    expect(screen.getByTestId("agent-name-badge")).toHaveTextContent("cto");
    const recallBadges = screen.getAllByTestId("recall-count-badge");
    expect(recallBadges).toHaveLength(2);
    expect(recallBadges.some(badge => badge.textContent?.includes("↻ 5"))).toBe(true);
    expect(screen.getByTestId("staleness-badge")).toBeInTheDocument();
  });

  it("Should show the empty fallback when there are no memories", () => {
    renderPanel({ memories: [] });
    const empty = screen.getByTestId("knowledge-list-empty");
    expect(empty).toBeInTheDocument();
    expect(
      within(empty).getByText("No knowledge items found", { selector: "h3" })
    ).toBeInTheDocument();
  });

  it("Should show the loading state while loading and the list is empty", () => {
    renderPanel({ isLoading: true, memories: [] });
    expect(screen.getByTestId("knowledge-list-loading")).toBeInTheDocument();
  });

  it("Should show the error fallback when errorMessage is set and the list is empty", () => {
    renderPanel({ errorMessage: "Network failure", memories: [] });
    expect(screen.getByTestId("knowledge-list-error")).toBeInTheDocument();
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });

  it("Should expose recall mode messaging through searchInfo and search-mode placeholder", () => {
    renderPanel({ searchMode: true, searchInfo: "Recall 2 of top-K", memories: [] });
    expect(screen.getByTestId("knowledge-search-info")).toHaveTextContent("Recall 2 of top-K");
    expect(screen.getByTestId("knowledge-list-empty")).toHaveTextContent(/recall query/i);
  });

  it("Should emit onSearchChange with the typed query", () => {
    const onSearchChange = vi.fn();
    renderPanel({ onSearchChange });
    const input = screen.getByLabelText("Search knowledge");
    expect(input).toHaveAttribute("data-testid", "knowledge-search-input");
    fireEvent.change(input, { target: { value: "alpha" } });
    expect(onSearchChange).toHaveBeenCalledWith("alpha");
  });

  it("Should emit onSelectMemory with the canonical key when a row is clicked", async () => {
    const user = userEvent.setup();
    const onSelectMemory = vi.fn();
    renderPanel({ onSelectMemory });
    await user.click(screen.getByTestId("memory-item-workspace:project-context.md"));
    expect(onSelectMemory).toHaveBeenCalledWith("workspace:project-context.md");
  });

  it("Should fall back to scope:filename when memory.key is missing", () => {
    renderPanel({
      memories: [
        GLOBAL,
        {
          ...WORKSPACE,
          key: undefined,
        },
      ],
    });

    expect(screen.getByTestId("memory-item-workspace:project-context.md")).toBeInTheDocument();
  });

  it("Should render the selection indicator only on the selected row", () => {
    renderPanel({ selectedMemoryKey: "workspace:project-context.md" });
    const selected = screen.getByTestId("memory-item-workspace:project-context.md");
    expect(within(selected).getByTestId("memory-active-indicator")).toBeInTheDocument();
    const unselected = screen.getByTestId("memory-item-global:user-role.md");
    expect(within(unselected).queryByTestId("memory-active-indicator")).not.toBeInTheDocument();
  });
});
