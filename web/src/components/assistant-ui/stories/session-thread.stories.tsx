import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { SessionChatRuntimeProvider } from "@/systems/session/components/session-chat-runtime-provider";
import { primarySessionFixture } from "@/systems/session/mocks";
import type { TranscriptMessage } from "@/systems/session/types";
import { SessionThread } from "../session-thread";

const mixedStreamingTranscript: TranscriptMessage[] = [
  {
    id: "story_user_mixed",
    role: "user",
    parts: [
      {
        type: "text",
        text: "Check the launch note, then summarize the risk.",
        state: "done",
      },
    ],
  },
  {
    id: "story_assistant_mixed",
    role: "assistant",
    parts: [
      {
        type: "text",
        text: "I will inspect the current launch note first.",
        state: "done",
      },
      {
        type: "reasoning",
        text: "Need the live note and the latest verification output before giving the operator a final answer.",
        state: "streaming",
      },
      {
        type: "data-provider-note",
        data: {
          title: "Provider note",
          detail: "Unregistered data event kept inline.",
        },
      },
      {
        type: "tool-WebSearch",
        toolCallId: "story_tool_web",
        state: "output-available",
        input: {
          query: "launch note risk",
        },
        output: {
          type: "tool_result",
          title: "WebSearch",
          raw: {
            content: "Found launch note reference.",
          },
        },
      },
      {
        type: "tool-Bash",
        toolCallId: "story_tool_bash",
        state: "output-available",
        input: {
          command: "bunx turbo run test --filter=./web",
        },
        output: {
          type: "tool_result",
          title: "Bash",
          raw: {
            stdout: "web tests passed\n",
          },
        },
      },
      {
        type: "text",
        text: [
          "The launch note is present and the web checks are green.",
          "",
          "| Area | Status |",
          "| --- | --- |",
          "| Chat stream | Fixed |",
          "| Tool timeline | Inline |",
          "",
          "```ts",
          'const nextAction = "keep monitoring";',
          "```",
        ].join("\n"),
        state: "streaming",
      },
    ],
  },
];

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
        http.get("/api/workspaces/:workspace_id/sessions/:id/transcript", () =>
          HttpResponse.json({ messages: [] })
        ),
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

/**
 * Mixed streaming turn — reasoning, unregistered and registered tool calls, then later markdown text.
 */
export const MixedStreaming: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        http.get("/api/workspaces/:workspace_id/sessions/:id/transcript", () =>
          HttpResponse.json({ messages: mixedStreamingTranscript })
        ),
      ],
    }),
  },
};

/**
 * Busy turn controls — queue, steer, and interrupt share the running composer surface.
 */
export const BusyInputControls: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
    onQueuePrompt: () => undefined,
    onInterruptPrompt: () => undefined,
    onSteerPrompt: () => undefined,
    isBusyInputPending: true,
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        http.get("/api/workspaces/:workspace_id/sessions/:id/transcript", () =>
          HttpResponse.json({ messages: mixedStreamingTranscript })
        ),
      ],
    }),
  },
};

/**
 * Wide panel — viewport and composer share the same full-width content rail.
 */
export const WidePanel: Story = {
  ...MixedStreaming,
  decorators: [
    Story => (
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={primarySessionFixture.workspace_id}
      >
        <div className="flex h-[640px] w-[1200px] max-w-full flex-col border border-line bg-background">
          <Story />
        </div>
      </SessionChatRuntimeProvider>
    ),
  ],
};

/**
 * Onboarding inset — matches wizard header/footer horizontal padding (px-8).
 */
export const OnboardingInset: Story = {
  ...MixedStreaming,
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    contentInset: "px-8",
    onCancelPrompt: () => undefined,
  },
  decorators: [
    Story => (
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={primarySessionFixture.workspace_id}
      >
        <div className="flex h-[640px] w-[1200px] max-w-full flex-col border border-line bg-background">
          <Story />
        </div>
      </SessionChatRuntimeProvider>
    ),
  ],
};
