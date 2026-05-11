import { UIProvider } from "@agh/ui";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SkillPayload } from "../../types";

import { SkillDetailPanel } from "../skill-detail-panel";

function makeSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    name: "alpha-skill",
    description: "Enforce repo conventions.",
    source: "bundled",
    enabled: true,
    dir: "/path/to/alpha",
    ...overrides,
  };
}

function renderPanel(props: Partial<React.ComponentProps<typeof SkillDetailPanel>> = {}) {
  const merged: React.ComponentProps<typeof SkillDetailPanel> = {
    skill: makeSkill(),
    isLoading: false,
    error: null,
    content: undefined,
    isContentLoading: false,
    contentError: null,
    onViewContent: vi.fn(),
    onRetryContent: vi.fn(),
    onDisable: vi.fn(),
    onEnable: vi.fn(),
    isActionPending: false,
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <SkillDetailPanel {...merged} />
    </UIProvider>
  );
}

describe("SkillDetailPanel", () => {
  it("Should render loading state when isLoading is true", () => {
    renderPanel({ skill: undefined, isLoading: true });
    expect(screen.getByTestId("skill-detail-loading")).toBeInTheDocument();
  });

  it("Should render error Empty state when error is present", () => {
    renderPanel({ skill: undefined, error: new Error("kaboom") });
    expect(screen.getByTestId("skill-detail-error")).toHaveTextContent("kaboom");
  });

  it("Should render empty state when no skill is selected", () => {
    renderPanel({ skill: undefined });
    expect(screen.getByTestId("skill-detail-empty")).toBeInTheDocument();
  });

  it("Should render title + meta pills inside the new DetailHeader anatomy", () => {
    renderPanel({
      skill: makeSkill({
        name: "alpha-skill",
        source: "marketplace",
        version: "3.1.0",
        provenance: { slug: "author", registry: "clawhub", installed_at: "", version: "3.1.0" },
      }),
    });

    const header = screen.getByTestId("skill-detail-header");
    expect(header).toHaveAttribute("data-slot", "detail-header");
    expect(within(header).getByTestId("skill-detail-title")).toHaveTextContent("alpha-skill");
    expect(within(header).getByTestId("detail-version-badge")).toHaveTextContent("v3.1.0");
    expect(within(header).getByTestId("detail-author-badge")).toHaveTextContent("@author");
    expect(within(header).getByTestId("source-badge")).toHaveAttribute("data-tone", "accent");
  });

  it("Should render the enable/disable Switch inside the DetailHeader actions slot", () => {
    renderPanel();
    const header = screen.getByTestId("skill-detail-header");
    const actions = header.querySelector('[data-slot="detail-header-actions"]');
    expect(actions).not.toBeNull();
    expect(actions?.querySelector('[data-testid="skill-enabled-switch"]')).not.toBeNull();
    expect(actions?.querySelector('[data-testid="skill-enabled-toggle"]')).not.toBeNull();
  });

  it("Should fire handleEnable when Switch is toggled on while disabled", async () => {
    const user = userEvent.setup();
    const onEnable = vi.fn();
    const onDisable = vi.fn();
    renderPanel({
      skill: makeSkill({ enabled: false }),
      onEnable,
      onDisable,
    });

    await user.click(screen.getByTestId("skill-enabled-switch"));
    expect(onEnable).toHaveBeenCalledWith("alpha-skill");
    expect(onDisable).not.toHaveBeenCalled();
  });

  it("Should fire handleDisable when Switch is toggled off while enabled", async () => {
    const user = userEvent.setup();
    const onEnable = vi.fn();
    const onDisable = vi.fn();
    renderPanel({
      skill: makeSkill({ enabled: true }),
      onEnable,
      onDisable,
    });

    await user.click(screen.getByTestId("skill-enabled-switch"));
    expect(onDisable).toHaveBeenCalledWith("alpha-skill");
    expect(onEnable).not.toHaveBeenCalled();
  });

  it("Should disable the Switch while isActionPending is true", () => {
    renderPanel({ isActionPending: true });
    const sw = screen.getByTestId("skill-enabled-switch");
    expect(sw).toHaveAttribute("aria-disabled", "true");
    expect(sw).toHaveAttribute("data-disabled");
  });

  it("Should render capabilities as MonoBadge rows when metadata.capabilities is present", () => {
    renderPanel({
      skill: makeSkill({ metadata: { capabilities: ["shell.run", "git.stage"] } }),
    });
    expect(screen.getByTestId("skill-capability-shell.run")).toBeInTheDocument();
    expect(screen.getByTestId("skill-capability-git.stage")).toBeInTheDocument();
  });

  it("Should render recent-calls Table with <Time>-driven When cells", () => {
    const timestamp = new Date("2026-04-17T17:30:00Z").toISOString();
    renderPanel({
      skill: makeSkill({
        metadata: {
          recent_calls: [
            { label: "skill.call(x)", status: "success", timestamp },
            { label: "skill.call(y)", status: "error" },
          ],
        },
      }),
    });

    expect(screen.getByTestId("skill-recent-calls-table")).toBeInTheDocument();
    const firstRow = screen.getByTestId("skill-recent-call-row-0");
    expect(firstRow).toHaveTextContent("skill.call(x)");
    const time = firstRow.querySelector("time[datetime]");
    expect(time).not.toBeNull();
    expect(time?.getAttribute("datetime")).toBe(timestamp);
  });

  it("Should render Empty placeholders for missing capabilities and recent calls", () => {
    renderPanel();
    expect(screen.getByTestId("skill-capabilities-empty")).toBeInTheDocument();
    expect(screen.getByTestId("skill-recent-calls-empty")).toBeInTheDocument();
  });

  it("Should load full content on demand via the view-full-content button", async () => {
    const user = userEvent.setup();
    const onViewContent = vi.fn();
    renderPanel({ onViewContent });
    await user.click(screen.getByTestId("view-full-content-btn"));
    expect(onViewContent).toHaveBeenCalledWith("alpha-skill");
  });

  it("Should render content loading, error and body states", () => {
    const { rerender } = renderPanel({ isContentLoading: true });
    expect(screen.getByTestId("content-loading")).toBeInTheDocument();

    rerender(
      <UIProvider reducedMotion="always">
        <SkillDetailPanel
          content={undefined}
          contentError={new Error("content offline")}
          error={null}
          isActionPending={false}
          isContentLoading={false}
          isLoading={false}
          onDisable={vi.fn()}
          onEnable={vi.fn()}
          onRetryContent={vi.fn()}
          onViewContent={vi.fn()}
          skill={makeSkill()}
        />
      </UIProvider>
    );
    expect(screen.getByTestId("content-error")).toHaveTextContent("content offline");

    rerender(
      <UIProvider reducedMotion="always">
        <SkillDetailPanel
          content="# Hello"
          contentError={null}
          error={null}
          isActionPending={false}
          isContentLoading={false}
          isLoading={false}
          onDisable={vi.fn()}
          onEnable={vi.fn()}
          onRetryContent={vi.fn()}
          onViewContent={vi.fn()}
          skill={makeSkill()}
        />
      </UIProvider>
    );
    expect(screen.getByTestId("content-body")).toHaveTextContent("# Hello");
    expect(
      screen.getByTestId("content-body").querySelector('[data-slot="code-block"]')
    ).toBeInTheDocument();
  });

  it("Should retry content fetch when retry-view-content-btn is clicked", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    renderPanel({ contentError: new Error("network"), onRetryContent: onRetry });
    await user.click(screen.getByTestId("retry-view-content-btn"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("Should not render the legacy CLI placeholder action", () => {
    renderPanel();
    expect(screen.queryByTestId("view-in-cli-btn")).not.toBeInTheDocument();
  });
});
