import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storyAgentNames, storySessionIds } from "@/storybook/fintech-scenario";
import { sessionTranscriptPermissionFixture } from "@/systems/session/mocks";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/agents/session",
    "Nested session chat route under an agent. Mirrors the previous routes/app/session stories , transcript hydration, stopped session, permission prompt, loading and not-found behaviors , through the canonical /agents/$name/sessions/$id URL."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

const fraudSessionRoute = `/agents/${storyAgentNames.fraud}/sessions/${storySessionIds.fraud}`;
const marketingStoppedSessionRoute = `/agents/${storyAgentNames.marketing}/sessions/${storySessionIds.marketing}`;
const missingSessionRoute = `/agents/${storyAgentNames.fraud}/sessions/sess-missing`;

/**
 * Active session transcript route for the primary payout-operations session.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters(fraudSessionRoute),
};

/**
 * Loading branch before the selected session resource resolves.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudSessionRoute),
    ...storybookMswParameters({
      session: [
        http.get("/api/workspaces/:workspace_id/sessions/:id", async () => {
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
  parameters: appRouteParameters(marketingStoppedSessionRoute),
};

/**
 * Pending permission prompt hydrated from the transcript replay payload.
 */
export const PendingPermission: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudSessionRoute),
    ...storybookMswParameters({
      session: [
        http.get("/api/workspaces/:workspace_id/sessions/:id/transcript", () =>
          HttpResponse.json({ messages: sessionTranscriptPermissionFixture })
        ),
      ],
    }),
  },
};

/**
 * Not-found session behavior , toast fires and navigation falls back to the parent agent route.
 */
export const NotFoundRedirect: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(missingSessionRoute),
    ...storybookMswParameters({
      session: [
        http.get("/api/workspaces/:workspace_id/sessions/:id", ({ params }) =>
          HttpResponse.json({ error: `Session not found: ${String(params.id)}` }, { status: 404 })
        ),
      ],
    }),
  },
};
