import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

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

export const Error: Story = {
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
