import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { useEffect } from "react";
import { expect, userEvent, within } from "storybook/test";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/skills",
    "Full-shell route stories for installed and marketplace skill browsing, including detail and content branches."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

function MarketplaceTabAutoClick() {
  useEffect(() => {
    let raf = 0;
    const tryClick = () => {
      const tab = document.querySelector<HTMLButtonElement>("[data-testid='tab-marketplace']");
      if (tab) {
        tab.click();
        return;
      }
      raf = window.requestAnimationFrame(tryClick);
    };
    tryClick();
    return () => {
      if (raf !== 0) window.cancelAnimationFrame(raf);
    };
  }, []);
  return null;
}

/**
 * Populated installed-skills branch with list + auto-selected detail.
 */
export const InstalledPopulated: Story = {
  args: {},
  parameters: appRouteParameters("/skills"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Installed tab when no skills exist.
 */
export const InstalledEmpty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/skills"),
    ...storybookMswParameters({
      skill: [http.get("/api/skills", () => HttpResponse.json({ skills: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Detail panel populated — visually identical to InstalledPopulated thanks to
 * auto-selection, but retained as a distinct baseline for the detail state.
 */
export const DetailOpen: Story = {
  args: {},
  parameters: appRouteParameters("/skills"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Marketplace tab active. Uses a minimal auto-click wrapper (not a Storybook
 * play function) so the visual-snapshot suite still captures the card grid.
 */
export const MarketplaceGrid: Story = {
  args: {},
  parameters: appRouteParameters("/skills"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <MarketplaceTabAutoClick />
    </>
  ),
};

/**
 * Interaction test: clicking the marketplace tab surfaces the card grid.
 */
export const MarketplaceInteraction: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: appRouteParameters("/skills"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("tab-marketplace"));
    await expect(canvas.findByTestId("marketplace-view")).resolves.toBeDefined();
  },
};
