import { UIProvider } from "@agh/ui";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
      <a
        href={typeof to === "string" ? to : "#"}
        data-params={JSON.stringify(params)}
        {...(props as Record<string, unknown>)}
      >
        {children as React.ReactNode}
      </a>
    ),
  };
});

const { SkillListPanel } = await import("../skill-list-panel");
type SkillPayload = import("../../types").SkillPayload;

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
    searchQuery: "",
    view: "rows",
    sourceFilter: null,
    enabledFilter: null,
    onClearFilters: vi.fn(),
    onDisable: vi.fn(),
    onEnable: vi.fn(),
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <SkillListPanel {...merged} />
    </UIProvider>
  );
}

describe("SkillListPanel", () => {
  it("Should render a flat listing of rows sorted by name", () => {
    renderPanel();
    expect(screen.getByTestId("skill-list-rows")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-beta")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-mp-plugin")).toBeInTheDocument();
    expect(screen.getByTestId("skill-item-ws-tool")).toBeInTheDocument();
    expect(screen.queryByTestId(/^skill-group-/)).not.toBeInTheDocument();
  });

  it("Should link each row to /skills/$name", () => {
    renderPanel();
    const link = screen.getByRole("link", { name: "Open alpha" });
    expect(link).toHaveAttribute("href", "/skills/$name");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ name: "alpha" }));
  });

  it("Should render an enabled switch in the trail instead of status pill + button", () => {
    renderPanel();
    expect(screen.getByTestId("skill-enabled-switch-alpha")).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByTestId("skill-enabled-switch-beta")).toHaveAttribute(
      "aria-checked",
      "false"
    );
    expect(screen.queryByTestId("skill-status-pill-alpha")).not.toBeInTheDocument();
    expect(screen.queryByTestId("skill-disable-alpha")).not.toBeInTheDocument();
    expect(screen.queryByTestId("skill-enable-beta")).not.toBeInTheDocument();
  });

  it("Should call onDisable / onEnable from the trail switch", async () => {
    const user = userEvent.setup();
    const onDisable = vi.fn();
    const onEnable = vi.fn();
    renderPanel({ onDisable, onEnable });

    await user.click(screen.getByTestId("skill-enabled-switch-alpha"));
    expect(onDisable).toHaveBeenCalledWith("alpha");

    await user.click(screen.getByTestId("skill-enabled-switch-beta"));
    expect(onEnable).toHaveBeenCalledWith("beta");
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
    const { rerender } = renderPanel({
      skills: withTags,
      searchQuery: "alpha",
    });

    expect(screen.getByTestId("skill-item-alpha")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-beta")).not.toBeInTheDocument();

    rerender(
      <UIProvider reducedMotion="always">
        <SkillListPanel
          enabledFilter={null}
          onClearFilters={vi.fn()}
          onDisable={vi.fn()}
          onEnable={vi.fn()}
          searchQuery="database"
          sourceFilter={null}
          skills={withTags}
          view="rows"
        />
      </UIProvider>
    );
    expect(screen.getByTestId("skill-item-tagged-skill")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-alpha")).not.toBeInTheDocument();
  });

  it("Should filter by source and enabled chips", () => {
    renderPanel({ sourceFilter: "workspace" });
    expect(screen.getByTestId("skill-item-ws-tool")).toBeInTheDocument();
    expect(screen.queryByTestId("skill-item-alpha")).not.toBeInTheDocument();
  });

  it("Should render cards when view is cards", () => {
    renderPanel({ view: "cards" });
    expect(screen.getByTestId("skill-list-card-grid")).toBeInTheDocument();
    expect(screen.getByTestId("skill-card-alpha")).toBeInTheDocument();
  });

  it("Should show Empty state when nothing matches the query", () => {
    renderPanel({ skills: [], searchQuery: "zzz" });
    expect(screen.getByTestId("skill-list-empty")).toHaveTextContent("No skills match");
  });

  it("Should show Clear filters action when filters are active and list is empty", async () => {
    const user = userEvent.setup();
    const onClearFilters = vi.fn();
    renderPanel({ skills: [], searchQuery: "zzz", onClearFilters });
    await user.click(screen.getByTestId("skill-list-clear-filters"));
    expect(onClearFilters).toHaveBeenCalled();
  });

  it("Should show loading Empty state when isLoading and list is empty", () => {
    renderPanel({ skills: [], isLoading: true });
    expect(screen.getByTestId("skill-list-loading")).toBeInTheDocument();
  });

  it("Should render error Empty state when errorMessage is provided and list is empty", () => {
    renderPanel({ skills: [], errorMessage: "daemon offline" });
    expect(screen.getByTestId("skill-list-error")).toHaveTextContent("daemon offline");
  });
});
