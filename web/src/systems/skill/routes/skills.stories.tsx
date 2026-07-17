import type { Meta, StoryObj } from "@storybook/react-vite";
import { HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { primarySkillFixture } from "@/systems/skill/mocks/fixtures";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/skill/routes/Skills",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Full-shell route stories for installed skill management and detail routes.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

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
      skill: [aghApiMock.get("/api/skills", () => HttpResponse.json({ skills: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Skill detail route (`/skills/$name`) rendered for the primary fixture skill.
 */
export const DetailOpen: Story = {
  args: {},
  parameters: appRouteParameters(`/skills/${primarySkillFixture.name}`),
  render: () => <StorybookWorkspaceSetup />,
};
