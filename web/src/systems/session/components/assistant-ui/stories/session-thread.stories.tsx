import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ComponentProps } from "react";
import { HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import { storybookMswParameters } from "@/storybook/msw";
import { SessionChatRuntimeProvider } from "@/systems/session/components/session-chat-runtime-provider";
import { SessionTranscriptThreadProvider } from "@/systems/session/lib/session-transcript-thread-context";
import { primarySessionFixture } from "@/systems/session/mocks";
import type { TranscriptMessage } from "@/systems/session/types";
import { ScrollToBottomPill, SessionThread } from "@/components/assistant-ui/session-thread";

const storyWorkspaceId = primarySessionFixture.workspace_id ?? "ws_alpha";

type TranscriptPart = NonNullable<TranscriptMessage["parts"]>[number];

function readToolPart(
  index: number,
  options: { turnId?: string; timestamp?: string } = {}
): TranscriptPart {
  return {
    type: "tool-Read",
    toolCallId: `story_tool_read_${index}`,
    state: "output-available",
    ...(options.turnId ? { turnId: options.turnId } : {}),
    ...(options.timestamp ? { timestamp: options.timestamp } : {}),
    input: {
      file_path: `/workspace/app/src/file-${index}.ts`,
    },
    output: {
      type: "tool_result",
      title: "Read",
      raw: {
        stdout: `export const fixture${index} = true;\n`,
      },
    },
  } as unknown as TranscriptPart;
}

function textTurnPart(text: string, turnId: string, timestamp: string): TranscriptPart {
  return {
    type: "text",
    text,
    state: "done",
    turnId,
    timestamp,
  } as unknown as TranscriptPart;
}

function reasoningTurnPart(text: string, turnId: string, timestamp: string): TranscriptPart {
  return {
    type: "reasoning",
    text,
    state: "done",
    turnId,
    timestamp,
  } as unknown as TranscriptPart;
}

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
      ...Array.from({ length: 8 }, (_, index) => readToolPart(index + 1)),
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

const foldedTurnsTranscript: TranscriptMessage[] = [
  {
    id: "story_user_folded",
    role: "user",
    parts: [
      {
        type: "text",
        text: "Review three work turns and leave the final answer readable.",
        state: "done",
      },
    ],
  },
  {
    id: "story_assistant_folded_1",
    role: "assistant",
    parts: [
      reasoningTurnPart(
        "First turn: inspect the launch checklist before touching copy.",
        "story-turn-1",
        "2026-07-07T12:00:00Z"
      ),
      readToolPart(1, {
        turnId: "story-turn-1",
        timestamp: "2026-07-07T12:00:03Z",
      }),
      textTurnPart(
        "Launch checklist is present and current.",
        "story-turn-1",
        "2026-07-07T12:00:05Z"
      ),
    ],
  },
  {
    id: "story_assistant_folded_2",
    role: "assistant",
    parts: [
      ...Array.from({ length: 8 }, (_, index) =>
        readToolPart(index + 2, {
          turnId: "story-turn-2",
          timestamp: `2026-07-07T12:00:${10 + index}Z`,
        })
      ),
      textTurnPart(
        "The eight referenced files agree on the launch-state language.",
        "story-turn-2",
        "2026-07-07T12:00:20Z"
      ),
    ],
  },
  {
    id: "story_assistant_folded_3",
    role: "assistant",
    parts: [
      reasoningTurnPart(
        "Final turn: verify the command output and patch summary.",
        "story-turn-3",
        "2026-07-07T12:01:00Z"
      ),
      {
        type: "tool-Bash",
        toolCallId: "story_tool_bash_folded",
        state: "output-available",
        turnId: "story-turn-3",
        timestamp: "2026-07-07T12:01:05Z",
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
      } as unknown as TranscriptPart,
      {
        type: "tool-Edit",
        toolCallId: "story_tool_edit_folded",
        state: "output-available",
        turnId: "story-turn-3",
        timestamp: "2026-07-07T12:01:08Z",
        input: {
          file_path: "/workspace/app/src/launch-note.ts",
          old_string: "const status = 'pending';",
          new_string: "const status = 'ready';",
        },
        output: {
          type: "tool_result",
          title: "Edit",
          raw: {
            content: "Applied patch successfully.",
          },
        },
      } as unknown as TranscriptPart,
      textTurnPart(
        "All three work turns are complete; the final launch note is ready.",
        "story-turn-3",
        "2026-07-07T12:01:12Z"
      ),
    ],
  },
];

function editToolPart(
  index: number,
  filePath: string,
  oldString: string,
  newString: string,
  timestamp: string
): TranscriptPart {
  return {
    type: "tool-Edit",
    toolCallId: `story_tool_edit_${index}`,
    state: "output-available",
    turnId: "story-turn-edits",
    timestamp,
    input: { file_path: filePath, old_string: oldString, new_string: newString },
    output: { type: "tool_result", title: "Edit", raw: { content: "Applied patch." } },
  } as unknown as TranscriptPart;
}

const changedFilesTranscript: TranscriptMessage[] = [
  {
    id: "story_user_changed_files",
    role: "user",
    parts: [
      {
        type: "text",
        text: "Flip the launch flag to ready and refresh the release notes.",
        state: "done",
      },
    ],
  },
  {
    id: "story_assistant_changed_files",
    role: "assistant",
    parts: [
      editToolPart(
        1,
        "/workspace/app/src/launch-note.ts",
        "const status = 'pending';",
        "const status = 'ready';",
        "2026-07-07T12:00:00Z"
      ),
      editToolPart(
        2,
        "/workspace/app/src/config/flags.ts",
        "launchReady: false,",
        "launchReady: true,\n  launchReadyAt: Date.now(),",
        "2026-07-07T12:00:03Z"
      ),
      {
        type: "tool-Write",
        toolCallId: "story_tool_write_release",
        state: "output-available",
        turnId: "story-turn-edits",
        timestamp: "2026-07-07T12:00:06Z",
        input: {
          file_path: "/workspace/RELEASE.md",
          content: ["# Release", "", "- Launch flag set to ready.", "- Notes refreshed."].join(
            "\n"
          ),
        },
        output: { type: "tool_result", title: "Write", raw: { content: "Wrote RELEASE.md." } },
      } as unknown as TranscriptPart,
      textTurnPart(
        "Launch flag is ready and the release notes are refreshed.",
        "story-turn-edits",
        "2026-07-07T12:00:09Z"
      ),
    ],
  },
];

const hoverToolbarTranscript: TranscriptMessage[] = [
  {
    id: "story_user_hover",
    role: "user",
    parts: [
      {
        type: "text",
        text: "Summarize the launch readiness in one line.",
        state: "done",
      },
    ],
  },
  {
    id: "story_assistant_hover",
    role: "assistant",
    parts: [
      textTurnPart(
        "Launch readiness is green: the launch note is present, web checks pass, and no blocking risks remain.",
        "story-turn-hover",
        "2026-07-07T12:00:05Z"
      ),
    ],
  },
];

function transcriptEntries(messages: TranscriptMessage[]) {
  return messages.map((message, index) => ({ message, sequence: index + 1 }));
}

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
  title: "systems/session/components/assistant-ui/SessionThread",
  component: SessionThread,
  parameters: {
    layout: "fullscreen",
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: [] })
        ),
      ],
    }),
    docs: {
      description: {
        component:
          "Shell wrapping `@assistant-ui/react` ThreadPrimitive + ComposerPrimitive. The composer input box is raised onto `--elevated` with a `--line-strong` ring so it reads distinctly against the `--canvas-soft` shell; focus-within shifts the ring to `--accent`. Idle shows an `--accent` send disc (`--accent-ink` glyph, never raw white); while a turn runs the primary disc becomes a `--danger` Stop, Enter queues the draft (with a visible hint), and queued prompts fuse onto the composer top as steer/edit/remove rows. Clear-conversation lives in the topbar, not the composer.",
      },
    },
  },
  decorators: [
    Story => (
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={storyWorkspaceId}
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

const baseArgs = {
  sessionId: primarySessionFixture.id,
  agentName: primarySessionFixture.agent_name,
  canPrompt: true,
  onCancelPrompt: () => undefined,
};

function renderWithTranscriptState(
  args: ComponentProps<typeof SessionThread>,
  state: {
    status: "pending" | "error" | "success";
    error?: Error | null;
  }
) {
  return (
    <SessionTranscriptThreadProvider
      messages={[]}
      status={state.status}
      isPending={state.status === "pending"}
      isError={state.status === "error"}
      error={state.error ?? null}
      retry={() => undefined}
    >
      <SessionThread {...args} />
    </SessionTranscriptThreadProvider>
  );
}

/**
 * Transcript loading state — shimmer message skeleton, never the empty-state copy.
 */
export const LoadingSkeleton: Story = {
  args: baseArgs,
  render: args => renderWithTranscriptState(args, { status: "pending" }),
};

/**
 * Transcript fetch failure — retryable pane with provider detail surfaced.
 */
export const TranscriptError: Story = {
  args: baseArgs,
  render: args =>
    renderWithTranscriptState(args, {
      status: "error",
      error: new Error("Storybook transcript fetch failed."),
    }),
};

/**
 * Empty thread state — assistant-ui empty slot renders the agent eyebrow + intro copy.
 */
export const Empty: Story = {
  args: baseArgs,
};

/**
 * Running composer with queued follow-ups — the strip fuses onto the composer top,
 * each row exposing steer / edit / remove; the primary disc is the `--danger` Stop
 * and Enter queues (hint visible).
 */
export const QueuedComposer: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    isSessionRunning: true,
    allowBusyInput: true,
    onCancelPrompt: () => undefined,
    onQueuePrompt: () => undefined,
    onInterruptPrompt: () => undefined,
    onSteerPrompt: () => undefined,
    onRemoveQueuedPrompt: () => undefined,
    onSteerQueuedPrompt: () => undefined,
    queuedPrompts: [
      { id: "inq-1", text: "Add a regression test for the reconnect path." },
      { id: "inq-2", text: "Then update the CLI docs for `--frames`." },
    ],
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: transcriptEntries(mixedStreamingTranscript) })
        ),
      ],
    }),
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
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: transcriptEntries(mixedStreamingTranscript) })
        ),
      ],
    }),
  },
};

/**
 * Folded work turns — three settled assistant turns collapse to duration buttons while final text remains visible.
 */
export const FoldedTurns: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: transcriptEntries(foldedTurnsTranscript) })
        ),
      ],
    }),
  },
};

/**
 * Changed-files roll-up — a settled editing turn closes with a collapsed
 * "Edited N files +a/-d" audit summary at its tail (below the folded work and the
 * terminal answer). Display-only: no Undo/Review, since AGH exposes no checkpoint
 * semantics. Expanding it lists each modified file with its diff stats.
 */
export const ChangedFilesRollup: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: transcriptEntries(changedFilesTranscript) })
        ),
      ],
    }),
  },
};

/**
 * Hover toolbar — copy (markdown source) + timestamp reveal on a settled assistant
 * message. The decorator forces the reveal so a static capture shows the hovered
 * state; production keeps the row `opacity-0` until hover / keyboard focus-within.
 */
export const HoverToolbar: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    onCancelPrompt: () => undefined,
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: transcriptEntries(hoverToolbarTranscript) })
        ),
      ],
    }),
  },
  decorators: [
    Story => (
      <>
        <style>
          {`[data-testid$="-message-actions"]{opacity:1 !important;pointer-events:auto !important;}`}
        </style>
        <Story />
      </>
    ),
  ],
};

/**
 * Busy turn controls — queue, steer, and interrupt share the running composer surface.
 */
export const BusyInputControls: Story = {
  args: {
    sessionId: primarySessionFixture.id,
    agentName: primarySessionFixture.agent_name,
    canPrompt: true,
    isSessionRunning: true,
    allowBusyInput: true,
    onCancelPrompt: () => undefined,
    onQueuePrompt: () => undefined,
    onInterruptPrompt: () => undefined,
    onSteerPrompt: () => undefined,
    isBusyInputPending: true,
  },
  parameters: {
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/workspaces/{workspace_id}/sessions/{session_id}/transcript", () =>
          HttpResponse.json({ entries: transcriptEntries(mixedStreamingTranscript) })
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
        workspaceId={storyWorkspaceId}
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
        workspaceId={storyWorkspaceId}
      >
        <div className="flex h-[640px] w-[1200px] max-w-full flex-col border border-line bg-background">
          <Story />
        </div>
      </SessionChatRuntimeProvider>
    ),
  ],
};

/**
 * Scroll-to-bottom pill — the live-follow affordance revealed when the reader
 * scrolls away from the live edge. Neutral `size-8 rounded-full` disc
 * (`bg-canvas-soft` + `border-line`, no glass/backdrop-blur), floating over the
 * transcript above the composer. Interaction-gated in production; rendered here
 * in its visible state over a transcript-like backdrop.
 */
export const ScrollToBottomAffordance: Story = {
  args: baseArgs,
  render: () => (
    <div className="relative flex min-h-0 flex-1 flex-col bg-background">
      <div className="flex-1 space-y-4 overflow-hidden px-4 py-6 text-sm leading-7 text-muted">
        {Array.from({ length: 8 }, (_, index) => (
          <p key={index}>
            Streamed transcript line {index + 1} — the reader has scrolled up while the assistant
            keeps answering below the fold, so the live-follow pill offers a one-click return to the
            newest output.
          </p>
        ))}
      </div>
      <ScrollToBottomPill visible onClick={() => undefined} />
    </div>
  ),
};
