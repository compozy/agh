import { UIProvider } from "@agh/ui";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SkillPayload } from "../types";

import { MarketplaceView } from "./marketplace-view";

function makeSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    name: "test-skill",
    description: "description",
    source: "marketplace",
    enabled: true,
    dir: "/path",
    ...overrides,
  };
}

const SKILLS: SkillPayload[] = [
  makeSkill({ name: "alpha", metadata: { tags: ["TESTING"], downloads: 10 } }),
  makeSkill({ name: "beta", metadata: { tags: ["DATABASE"] } }),
  makeSkill({ name: "gamma", metadata: { tags: ["AI"] } }),
];

function renderView(props: Partial<React.ComponentProps<typeof MarketplaceView>> = {}) {
  const merged: React.ComponentProps<typeof MarketplaceView> = {
    skills: SKILLS,
    installedSkillNames: new Set<string>(),
    isInstalling: false,
    onInstall: vi.fn(),
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <MarketplaceView {...merged} />
    </UIProvider>
  );
}

describe("MarketplaceView", () => {
  it("Should render a responsive grid of Cards", () => {
    renderView();
    const grid = screen.getByTestId("marketplace-grid");
    expect(grid).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-beta")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-gamma")).toBeInTheDocument();
  });

  it("Should call onInstall with the skill name when the install button is clicked", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn();
    renderView({ onInstall });

    await user.click(screen.getByTestId("install-btn-alpha"));
    expect(onInstall).toHaveBeenCalledWith("alpha");
  });

  it("Should show the Installed MonoBadge for skills already installed", () => {
    renderView({ installedSkillNames: new Set(["alpha"]) });
    expect(screen.getByTestId("installed-pill-alpha")).toBeInTheDocument();
    expect(screen.queryByTestId("install-btn-alpha")).not.toBeInTheDocument();
  });

  it("Should disable install button when isInstalling is true", () => {
    renderView({ isInstalling: true });
    expect(screen.getByTestId("install-btn-alpha")).toBeDisabled();
  });

  it("Should switch to a browse-only catalog when installs are unavailable", () => {
    renderView({
      onInstall: undefined,
      installUnavailableReason: "Installs are not exposed by the daemon yet.",
    });

    expect(screen.getByTestId("marketplace-readonly-notice")).toHaveTextContent(
      "Installs are not exposed by the daemon yet."
    );
    expect(screen.getByTestId("marketplace-search-input")).toHaveAttribute(
      "aria-label",
      "Filter installed marketplace skills"
    );
    expect(screen.getByTestId("marketplace-search-input")).toHaveAttribute(
      "placeholder",
      "Filter installed marketplace skills…"
    );
    expect(screen.getByTestId("marketplace-readonly-notice")).toHaveTextContent(
      "Installed marketplace metadata only"
    );
    expect(screen.queryByTestId("install-btn-alpha")).not.toBeInTheDocument();
  });

  it("Should use browse-only empty copy when no installed marketplace skills are available", () => {
    renderView({
      onInstall: undefined,
      skills: [],
    });

    expect(screen.getByTestId("marketplace-empty")).toHaveTextContent(
      "No marketplace-installed skills"
    );
    expect(screen.getByTestId("marketplace-empty")).toHaveTextContent(
      "No marketplace-installed skills are available in this workspace yet."
    );
  });

  it("Should filter by category (pills) and show Empty when nothing matches", async () => {
    const user = userEvent.setup();
    renderView();

    await user.click(screen.getByTestId("category-chip-DATABASE"));
    expect(screen.getByTestId("marketplace-row-beta")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-row-alpha")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("category-chip-SECURITY"));
    expect(screen.getByTestId("marketplace-empty")).toBeInTheDocument();
  });

  it("Should filter by search query across name and tags", () => {
    renderView();
    const input = screen.getByTestId("marketplace-search-input");
    fireEvent.change(input, { target: { value: "alpha" } });
    expect(screen.getByTestId("marketplace-row-alpha")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-row-beta")).not.toBeInTheDocument();

    fireEvent.change(input, { target: { value: "AI" } });
    expect(screen.getByTestId("marketplace-row-gamma")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-row-alpha")).not.toBeInTheDocument();
  });
});
