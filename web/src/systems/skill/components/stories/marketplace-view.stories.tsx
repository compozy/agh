import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { storybookMswParameters } from "@/storybook/msw";
import { PanelSurface } from "@/storybook/story-layout";

import { MarketplaceView } from "../marketplace-view";

const meta: Meta<typeof MarketplaceView> = {
  title: "systems/skill/MarketplaceView",
  component: MarketplaceView,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function MarketplaceViewFromPage() {
  const page = useSkillsPage();
  return (
    <PanelSurface>
      <MarketplaceView
        installUnavailableReason="Marketplace install is not implemented yet"
        installedSkillNames={new Set(page.skills.slice(0, 1).map(skill => skill.name))}
        isInstalling={false}
        onInstall={() => undefined}
        skills={page.error ? [] : page.skills}
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <MarketplaceViewFromPage />,
};

export const ErrorState: Story = {
  parameters: {
    ...storybookMswParameters({
      skill: [
        http.get("/api/skills", () =>
          HttpResponse.json({ error: "marketplace unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <MarketplaceViewFromPage />,
};

export const DisabledInstall: Story = {
  render: () => {
    const page = useSkillsPage();
    return (
      <PanelSurface>
        <MarketplaceView
          installUnavailableReason="Daemon does not support marketplace installs"
          installedSkillNames={new Set()}
          isInstalling={false}
          onInstall={undefined}
          skills={page.skills}
        />
      </PanelSurface>
    );
  },
};

export const AllInstalled: Story = {
  render: () => {
    const page = useSkillsPage();
    return (
      <PanelSurface>
        <MarketplaceView
          installUnavailableReason="Marketplace install is not implemented yet"
          installedSkillNames={new Set(page.skills.map(skill => skill.name))}
          isInstalling={false}
          onInstall={() => undefined}
          skills={page.skills}
        />
      </PanelSurface>
    );
  },
};

/**
 * Interaction test — filter marketplace by a non-matching category shows Empty.
 */
export const FilterToEmpty: Story = {
  tags: ["play-fn"],
  render: () => <MarketplaceViewFromPage />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const chip = await canvas.findByTestId("category-chip-SECURITY");
    await userEvent.click(chip);
    await expect(canvas.findByTestId("marketplace-empty")).resolves.toBeDefined();
  },
};
