import { UIProvider } from "@agh/ui";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SkillPayload } from "../../types";

import { SkillListPanel } from "../skill-list-panel";

function makeSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    name: "test-skill",
    description: "desc",
    source: "bundled",
    enabled: true,
    dir: "/path/to/skill",
    ...overrides,
  };
}

const SKILLS: SkillPayload[] = [
  makeSkill({ name: "alpha", source: "bundled", enabled: true, version: "1.0.0" }),
  makeSkill({ name: "beta", source: "bundled", enabled: false }),
  makeSkill({ name: "ws-tool", source: "workspace", enabled: true }),
  makeSkill({ name: "mp-plugin", source: "marketplace", enabled: true }),
];

function renderPanel(props: Partial<React.ComponentProps<typeof SkillListPanel>> = {}) {
  const merged: React.ComponentProps<typeof SkillListPanel> = {
    skills: SKILLS,
    onSearchChange: vi.fn(),
    onSelectSkill: vi.fn(),
    searchQuery: "",
    selectedSkillName: null,
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <SkillListPanel {...merged} />
    </UIProvider>
  );
}

describe("SkillListPanel", () => {
  it("Should group rows by source in BUNDLED → WORKSPACE → MARKETPLACE order", () => {
    renderPanel();
    const groups = screen.getAllByTestId(/^skill-group-(?!header-)/);
    expect(groups.map(element => element.getAttribute("data-testid"))).toEqual([
      "skill-group-bundled",
      "skill-group-workspace",
      "skill-group-marketplace",
    ]);
  });

  it("Should render source count badge in each group header", () => {
    renderPanel();
    const bundledHeader = screen.getByTestId("skill-group-header-bundled");
    expect(within(bundledHeader).getByText("2")).toBeInTheDocument();
  });

  it("Should highlight selected row and emit active indicator", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    renderPanel({ selectedSkillName: "beta", onSelectSkill: onSelect });
    expect(
      within(screen.getByTestId("skill-item-beta")).getByTestId("skill-active-indicator")
    ).toBeInTheDocument();

    await user.click(screen.getByTestId("skill-item-alpha"));
    expect(onSelect).toHaveBeenCalledWith("alpha");
  });

  it("Should filter rows by name, description and tags", () => {
    const withTags = [
      ...SKILLS,
      makeSkill({
        name: "tagged-skill",
        source: "bundled",
        description: "another",
        metadata: { tags: ["DATABASE"] },
      }),
    ];
    const onSearchChange = vi.fn();
    const { rerender } = renderPanel({
      skills: withTags,
      searchQuery: "alpha",
      onSearchChange,
    });

    expect(screen.getByTestId("skill-item-alpha")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-beta")).not.toBeInTheDocument();

    rerender(
      <UIProvider reducedMotion="always">
        <SkillListPanel
          onSearchChange={onSearchChange}
          onSelectSkill={vi.fn()}
          searchQuery="database"
          selectedSkillName={null}
          skills={withTags}
        />
      </UIProvider>
    );
    expect(screen.getByTestId("skill-item-tagged-skill")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-alpha")).not.toBeInTheDocument();
  });

  it("Should show Empty state when nothing matches the query", () => {
    renderPanel({ skills: [] });
    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills found");
  });

  it("Should show loading Empty state when isLoading and list is empty", () => {
    renderPanel({ skills: [], isLoading: true });
    expect(screen.getByTestId("skill-list-loading")).toBeInTheDocument();
  });

  it("Should render error Empty state when errorMessage is provided and list is empty", () => {
    renderPanel({ skills: [], errorMessage: "daemon offline" });
    expect(screen.getByTestId("skill-list-error")).toHaveTextContent("daemon offline");
  });

  it("Should forward SearchInput changes to onSearchChange", () => {
    const onSearchChange = vi.fn();
    renderPanel({ onSearchChange });
    fireEvent.change(screen.getByTestId("skill-search-input"), {
      target: { value: "alpha" },
    });
    expect(onSearchChange).toHaveBeenCalledWith("alpha");
  });
});
