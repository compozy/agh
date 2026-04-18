import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { settingsSkillsSectionFixture } from "@/systems/settings/mocks";
import {
  StorybookRestartBannerSetup,
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/skills",
    "Skills settings route stories covering policy editing, disabled-skill empty states, restart banners, and request failures."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default skills configuration surface with disabled-skill list and marketplace policy.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/skills"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Disabled-skills empty branch when nothing has been opted out.
 */
export const DisabledEmpty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/skills"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/skills", () =>
          HttpResponse.json({
            ...settingsSkillsSectionFixture,
            disabled_count: 0,
            config: {
              ...settingsSkillsSectionFixture.config,
              disabled_skills: [],
            },
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Restart banner after saving marketplace policy that only applies after daemon restart.
 */
export const RestartBanner: Story = {
  args: {},
  parameters: appRouteParameters("/settings/skills"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookRestartBannerSetup section="skills" />
    </>
  ),
};

/**
 * Loading state before the skills settings section resolves.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/skills"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/skills", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Error branch when the skills settings request fails.
 */
export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/settings/skills"),
    ...storybookMswParameters({
      settings: [
        http.get("/api/settings/skills", () =>
          HttpResponse.json({ error: "Failed to load skills settings" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
