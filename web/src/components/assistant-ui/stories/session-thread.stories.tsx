import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { SessionChatRuntimeProvider } from "@/systems/session/components/session-chat-runtime-provider";
import { primarySessionFixture } from "@/systems/session/mocks";
import { SessionThread } from "../session-thread";

/**
 * Storybook stories for the assistant-ui session thread shell.
 *
 * The component depends on `@assistant-ui/react`'s `ThreadPrimitive` /
 * `MessagePrimitive` / `ComposerPrimitive`, which in turn require an active
 * runtime context (`AssistantRuntimeProvider`). These stories use the same
 * session runtime provider as the route shell, while overriding transcript
 * hydration to keep the empty-thread chrome visible.
 */
const meta: Meta<typeof SessionThread> = {
  title: "components/assistant-ui/SessionThread",
  component: SessionThread,
  parameters: {
    layout: "fullscreen",
    ...storybookMswParameters({
      session: [
        http.get("/api/sessions/:id/transcript", () => HttpResponse.json({ messages: [] })),
      ],
    }),
    docs: {
      description: {
        component:
          "Shell wrapping `@assistant-ui/react` ThreadPrimitive + ComposerPrimitive. Composer surface stays flat on `--canvas-soft` with a `--line` ring; focus-within shifts the ring to `--accent`; the send button uses `--accent` fill with `--accent-ink` glyph (warm brand orange + readable ink, never raw white). The clear-conversation flow opens a confirm dialog using the kit's Dialog primitive.",
      },
    },
  },
  decorators: [
    Story => (
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={primarySessionFixture.workspace_id}
      >
        <div className="flex h-[640px] w-full flex-col bg-background border border-line">
          <Story />
        </div>
      </SessionChatRuntimeProvider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Empty thread state — assistant-ui empty slot renders the agent eyebrow + intro copy.
 */
export const Empty: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
  },
};

/**
 * With clear-conversation control enabled — opens the kit's Dialog confirm flow.
 */
export const WithClear: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
    onClearConversation: () => undefined,
    canClearConversation: true,
    isClearingConversation: false,
  },
};

/**
 * Disabled composer (session not active) — verifies placeholder + disabled state.
 */
export const Disabled: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: false,
    onCancelPrompt: () => undefined,
  },
};
