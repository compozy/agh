import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/home",
    "Full app-shell route stories for the root workspace surface, including the default empty state and onboarding branch."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default root route with the real app shell and the empty session placeholder.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Onboarding branch shown when the daemon has no configured workspaces yet.
 */
export const Onboarding: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/"),
    ...storybookMswParameters({
      workspace: [http.get("/api/workspaces", () => HttpResponse.json({ workspaces: [] }))],
    }),
  },
};
