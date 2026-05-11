import type { Meta, StoryObj } from "@storybook/react-vite";

import { SessionThread } from "../session-thread";

/**
 * Storybook stories for the assistant-ui session thread shell.
 *
 * The component depends on `@assistant-ui/react`'s `ThreadPrimitive` /
 * `MessagePrimitive` / `ComposerPrimitive`, which in turn require an active
 * runtime context (`AssistantRuntimeProvider`). Storybook does not bootstrap
 * that runtime, so these stories exercise the chrome (composer + clear button
 * + empty state) without actually streaming messages — they verify that the
 * SessionThread layout renders and the composer chrome stays on-token (`--accent`,
 * `--accent-ink`, `--canvas-soft`, `--line`, `--danger`/8|12|18 alpha mods).
 */
const meta: Meta<typeof SessionThread> = {
  title: "components/assistant-ui/SessionThread",
  component: SessionThread,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shell wrapping `@assistant-ui/react` ThreadPrimitive + ComposerPrimitive. Composer surface stays flat on `--canvas-soft` with a `--line` ring; focus-within shifts the ring to `--accent`; the send button uses `--accent` fill with `--accent-ink` glyph (warm brand orange + readable ink, never raw white). The clear-conversation flow opens a confirm dialog using the kit's Dialog primitive.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="flex h-[640px] w-full flex-col bg-background border border-line">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Empty thread state — assistant-ui empty slot renders the agent eyebrow + intro copy.
 * Composer is enabled but unable to actually stream messages without a runtime context.
 */
export const Empty: Story = {
  args: {
    sessionId: "sess_storybook_demo",
    agentName: "anthropic-claude",
    canPrompt: true,
    onCancelPrompt: () => undefined,
  },
};

/**
 * With clear-conversation control enabled — opens the kit's Dialog confirm flow.
 */
export const WithClear: Story = {
  args: {
    sessionId: "sess_storybook_demo",
    agentName: "openai",
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
    sessionId: "sess_storybook_demo",
    agentName: "local-llama",
    canPrompt: false,
    onCancelPrompt: () => undefined,
  },
};
