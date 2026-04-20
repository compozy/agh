import { UIProvider } from "@agh/ui";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { MemoryHeader } from "../types";

import { KnowledgeListPanel } from "./knowledge-list-panel";

const GLOBAL: MemoryHeader = {
  filename: "global/user-role.md",
  mod_time: "2026-04-09T10:00:00Z",
  name: "User Role",
  type: "user",
  description: "Guidance that shapes the assistant's tone and ownership.",
};

const WORKSPACE: MemoryHeader = {
  filename: "workspace/project-context.md",
  mod_time: "2026-04-09T08:00:00Z",
  name: "Project Context",
  type: "project",
  description: "Workspace-local notes about rollout.",
  agent_name: "codex-agent",
};

const ALL: MemoryHeader[] = [GLOBAL, WORKSPACE];

function renderPanel(props: Partial<React.ComponentProps<typeof KnowledgeListPanel>> = {}) {
  const merged: React.ComponentProps<typeof KnowledgeListPanel> = {
    memories: ALL,
    onSearchChange: vi.fn(),
    onSelectMemory: vi.fn(),
    searchQuery: "",
    selectedFilename: null,
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgeListPanel {...merged} />
    </UIProvider>
  );
}

describe("KnowledgeListPanel", () => {
  it("renders groups with GLOBAL before WORKSPACE and renders the group count badge", () => {
    renderPanel();
    const groups = screen.getAllByTestId(/^knowledge-group-/).filter(element => {
      const id = element.getAttribute("data-testid") ?? "";
      return id === "knowledge-group-global" || id === "knowledge-group-workspace";
    });
    expect(groups[0]).toHaveAttribute("data-testid", "knowledge-group-global");
    expect(groups[1]).toHaveAttribute("data-testid", "knowledge-group-workspace");
    expect(
      within(screen.getByTestId("knowledge-group-header-global")).getByText("1")
    ).toBeInTheDocument();
  });

  it("renders MonoBadge chips for type and scope on each row", () => {
    renderPanel();
    expect(screen.getByTestId("type-badge-user")).toHaveAttribute("data-tone", "accent");
    expect(screen.getByTestId("type-badge-project")).toHaveAttribute("data-tone", "success");
    expect(screen.getByTestId("scope-badge-global")).toHaveAttribute("data-tone", "neutral");
    expect(screen.getByTestId("scope-badge-workspace")).toHaveAttribute("data-tone", "info");
  });

  it("filters rows by name (case-insensitive)", () => {
    renderPanel({ searchQuery: "project" });
    expect(screen.getByTestId("memory-item-workspace/project-context.md")).toBeInTheDocument();
    expect(screen.queryByTestId("memory-item-global/user-role.md")).not.toBeInTheDocument();
  });

  it("filters rows by description (case-insensitive)", () => {
    renderPanel({ searchQuery: "rollout" });
    expect(screen.getByTestId("memory-item-workspace/project-context.md")).toBeInTheDocument();
    expect(screen.queryByTestId("memory-item-global/user-role.md")).not.toBeInTheDocument();
  });

  it("filters rows by type", () => {
    renderPanel({ searchQuery: "USER" });
    expect(screen.getByTestId("memory-item-global/user-role.md")).toBeInTheDocument();
    expect(
      screen.queryByTestId("memory-item-workspace/project-context.md")
    ).not.toBeInTheDocument();
  });

  it("shows the filtered-empty Empty card when the query matches nothing", () => {
    renderPanel({ searchQuery: "zzzzz" });
    expect(screen.getByTestId("knowledge-list-empty")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("knowledge-list-empty")).getByText(
        /different search term or adjust the scope filter/i
      )
    ).toBeInTheDocument();
  });

  it("shows the no-items Empty card when the memories array is empty", () => {
    renderPanel({ memories: [] });
    const empty = screen.getByTestId("knowledge-list-empty");
    expect(empty).toBeInTheDocument();
    expect(
      within(empty).getByText("No knowledge items found", { selector: "h3" })
    ).toBeInTheDocument();
  });

  it("shows the loading fallback while loading and list is empty", () => {
    renderPanel({ isLoading: true, memories: [] });
    expect(screen.getByTestId("knowledge-list-loading")).toBeInTheDocument();
  });

  it("shows the error Empty card when errorMessage is set and list is empty", () => {
    renderPanel({ errorMessage: "Network failure", memories: [] });
    expect(screen.getByTestId("knowledge-list-error")).toBeInTheDocument();
    expect(screen.getByText("Network failure")).toBeInTheDocument();
  });

  it("emits onSearchChange with the typed query", () => {
    const onSearchChange = vi.fn();
    renderPanel({ onSearchChange });
    const input = screen.getByLabelText("Search knowledge");
    expect(input).toHaveAttribute("data-testid", "knowledge-search-input");
    fireEvent.change(input, { target: { value: "alpha" } });
    expect(onSearchChange).toHaveBeenCalledWith("alpha");
  });

  it("emits onSelectMemory with the clicked filename", async () => {
    const user = userEvent.setup();
    const onSelectMemory = vi.fn();
    renderPanel({ onSelectMemory });
    await user.click(screen.getByTestId("memory-item-workspace/project-context.md"));
    expect(onSelectMemory).toHaveBeenCalledWith("workspace/project-context.md");
  });

  it("renders the 3px accent indicator on the selected row only", () => {
    renderPanel({ selectedFilename: "workspace/project-context.md" });
    const selected = screen.getByTestId("memory-item-workspace/project-context.md");
    expect(within(selected).getByTestId("memory-active-indicator")).toBeInTheDocument();
    const unselected = screen.getByTestId("memory-item-global/user-role.md");
    expect(within(unselected).queryByTestId("memory-active-indicator")).not.toBeInTheDocument();
  });
});
