import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookSessionPermissionSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/session",
    "Session route stories rendered via the real router, including transcript hydration, stopped sessions, permission prompts and not-found redirect behavior."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Active session transcript route for the primary Storybook session.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/session/sess-storybook"),
};

/**
 * Loading branch before the selected session resource resolves.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/session/sess-storybook"),
    ...storybookMswParameters({
      session: [
        http.get("/api/sessions/:id", async () => {
          await delay("infinite");
          return HttpResponse.json({ session: null });
        }),
      ],
    }),
  },
};

/**
 * Stopped-session branch that swaps the header action set and hides the composer.
 */
export const Stopped: Story = {
  args: {},
  parameters: appRouteParameters("/session/sess-reviewer"),
};

/**
 * Pending permission prompt injected into the route's existing session store flow.
 */
export const PendingPermission: Story = {
  args: {},
  parameters: appRouteParameters("/session/sess-storybook"),
  render: () => <StorybookSessionPermissionSetup />,
};

/**
 * Not-found session behavior, which redirects back to the root empty state.
 */
export const NotFoundRedirect: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/session/sess-missing"),
    ...storybookMswParameters({
      session: [
        http.get("/api/sessions/:id", ({ params }) =>
          HttpResponse.json({ error: `Session not found: ${String(params.id)}` }, { status: 404 })
        ),
      ],
    }),
  },
};
