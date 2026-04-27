import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { sessionTranscriptPermissionFixture } from "@/systems/session/mocks";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/agents/session",
    "Nested session chat route under an agent. Mirrors the previous routes/app/session stories — transcript hydration, stopped session, permission prompt, loading and not-found behaviors — through the canonical /agents/$name/sessions/$id URL."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

const codexSessionRoute = "/agents/codex-agent/sessions/sess-storybook";
const claudeStoppedSessionRoute = "/agents/claude-agent/sessions/sess-reviewer";
const missingSessionRoute = "/agents/codex-agent/sessions/sess-missing";

/**
 * Active session transcript route for the primary Storybook session.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters(codexSessionRoute),
};

/**
 * Loading branch before the selected session resource resolves.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(codexSessionRoute),
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
  parameters: appRouteParameters(claudeStoppedSessionRoute),
};

/**
 * Pending permission prompt hydrated from the transcript replay payload.
 */
export const PendingPermission: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(codexSessionRoute),
    ...storybookMswParameters({
      session: [
        http.get("/api/sessions/:id/transcript", () =>
          HttpResponse.json({ messages: sessionTranscriptPermissionFixture })
        ),
      ],
    }),
  },
};

/**
 * Not-found session behavior — toast fires and navigation falls back to the parent agent route.
 */
export const NotFoundRedirect: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(missingSessionRoute),
    ...storybookMswParameters({
      session: [
        http.get("/api/sessions/:id", ({ params }) =>
          HttpResponse.json({ error: `Session not found: ${String(params.id)}` }, { status: 404 })
        ),
      ],
    }),
  },
};
