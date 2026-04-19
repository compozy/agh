import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { CenteredSurface } from "@/storybook/story-layout";
import type { UIMessage } from "@/systems/session/types";

import {
  SessionInspector,
  SessionInspectorDrawer,
  type InspectorFileEntry,
  type InspectorMemoryDoc,
  type InspectorUsage,
  type SessionInspectorProps,
} from "../session-inspector";

function ts(offsetMin: number): number {
  return Date.parse("2026-04-18T14:00:00Z") + offsetMin * 60 * 1000;
}

const mixedMessages: UIMessage[] = [
  {
    id: "m-start",
    role: "system",
    content: "Session resumed from checkpoint 8471.",
    timestamp: ts(0),
  },
  { id: "m-user", role: "user", content: "Refactor the event mapper grouping.", timestamp: ts(1) },
  {
    id: "m-assistant",
    role: "assistant",
    content: "I'll extract `groupToolCallsByTurn` from stream.ts.",
    timestamp: ts(2),
  },
  {
    id: "m-tool-shell",
    role: "tool_call",
    content: "",
    toolName: "Bash",
    toolInput: { command: "rg 'onToolCall' packages/runtime -l" },
    toolResult: { stdout: "packages/runtime/src/session/stream.ts" },
    timestamp: ts(3),
  },
  {
    id: "m-tool-read",
    role: "tool_call",
    content: "",
    toolName: "Read",
    toolInput: { file_path: "packages/runtime/src/session/stream.ts" },
    toolResult: {
      filePath: "packages/runtime/src/session/stream.ts",
      stdout: "// source snippet",
    },
    timestamp: ts(4),
  },
  {
    id: "m-tool-read-2",
    role: "tool_call",
    content: "",
    toolName: "Read",
    toolInput: { file_path: "packages/runtime/src/session/stream.ts" },
    toolResult: {
      filePath: "packages/runtime/src/session/stream.ts",
      stdout: "// source snippet 2",
    },
    timestamp: ts(5),
  },
  {
    id: "m-tool-read-3",
    role: "tool_call",
    content: "",
    toolName: "Read",
    toolInput: { file_path: "packages/runtime/src/session/replay.ts" },
    toolResult: {
      filePath: "packages/runtime/src/session/replay.ts",
      stdout: "// replay snippet",
    },
    timestamp: ts(6),
  },
  {
    id: "m-tool-error",
    role: "tool_call",
    content: "",
    toolName: "Bash",
    toolError: true,
    toolInput: { command: "make verify" },
    toolResult: { stderr: "exit 1" },
    timestamp: ts(7),
  },
  {
    id: "m-diff",
    role: "diff",
    content: "",
    diff: {
      path: "packages/runtime/src/session/stream.ts",
      additions: 4,
      removals: 38,
      content: "diff",
    },
    timestamp: ts(8),
  },
  {
    id: "m-assistant-2",
    role: "assistant",
    content: "Ready to apply. Approve to continue.",
    timestamp: ts(9),
  },
];

const usageFixture: InspectorUsage = {
  tokensIn: 12_481,
  tokensOut: 2_108,
  costUsd: 0.048,
  ratePerSecond: 128.4,
  tokensInDelta: 540,
  tokensOutDelta: -128,
  costDelta: 0.012,
};

const memoryFixture: InspectorMemoryDoc[] = [
  { id: "doc-1", kind: "ws", title: "agh-architecture.md", bytes: 4_820 },
  { id: "doc-2", kind: "ws", title: "operator-voice.md", bytes: 12_014 },
  { id: "doc-3", kind: "repo", title: "sessions-model.md", bytes: 6_412 },
];

const filesFixture: InspectorFileEntry[] = [
  { path: "packages/runtime/src/session/stream.ts", readCount: 2 },
  { path: "packages/runtime/src/session/replay.ts", readCount: 1 },
  { path: "packages/runtime/src/events/map.ts", readCount: 1 },
];

function InspectorFrame({
  children,
  height = 720,
}: {
  children: React.ReactNode;
  height?: number;
}) {
  return (
    <CenteredSurface>
      <div
        className="flex overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
        style={{ height, width: 480 }}
      >
        {children}
      </div>
    </CenteredSurface>
  );
}

const baseArgs: SessionInspectorProps = {
  messages: mixedMessages,
  usage: usageFixture,
  memoryDocs: memoryFixture,
  files: filesFixture,
  totalTraceEvents: mixedMessages.length,
  onViewAllTrace: () => undefined,
};

const meta: Meta<typeof SessionInspector> = {
  title: "systems/session/SessionInspector",
  component: SessionInspector,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const AllSections: Story = {
  render: () => (
    <InspectorFrame>
      <SessionInspector {...baseArgs} />
    </InspectorFrame>
  ),
};

export const EmptyTrace: Story = {
  render: () => (
    <InspectorFrame>
      <SessionInspector {...baseArgs} messages={[]} files={filesFixture} totalTraceEvents={0} />
    </InspectorFrame>
  ),
};

export const EmptyUsage: Story = {
  render: () => (
    <InspectorFrame>
      <SessionInspector {...baseArgs} usage={null} />
    </InspectorFrame>
  ),
};

export const EmptyMemory: Story = {
  render: () => (
    <InspectorFrame>
      <SessionInspector {...baseArgs} memoryDocs={[]} />
    </InspectorFrame>
  ),
};

export const EmptyFiles: Story = {
  render: () => (
    <InspectorFrame>
      <SessionInspector {...baseArgs} files={[]} />
    </InspectorFrame>
  ),
};

export const AllEmpty: Story = {
  render: () => (
    <InspectorFrame>
      <SessionInspector messages={[]} usage={null} memoryDocs={[]} files={[]} />
    </InspectorFrame>
  ),
};

export const Compact: Story = {
  parameters: {
    viewport: {
      defaultViewport: "sessionInspectorCompact",
      viewports: {
        sessionInspectorCompact: {
          name: "Compact (short viewport)",
          styles: { width: "480px", height: "620px" },
        },
      },
    },
  },
  render: () => (
    <InspectorFrame height={620}>
      <SessionInspector {...baseArgs} />
    </InspectorFrame>
  ),
};

export const Drawer: Story = {
  tags: ["play-fn"],
  parameters: {
    viewport: {
      defaultViewport: "sessionInspectorNarrow",
      viewports: {
        sessionInspectorNarrow: {
          name: "Narrow (drawer)",
          styles: { width: "1100px", height: "720px" },
        },
      },
    },
  },
  render: () => (
    <CenteredSurface>
      <div
        className="flex overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
        style={{ height: 720, width: 960 }}
      >
        <div className="flex min-h-0 flex-1 items-center justify-center">
          <SessionInspectorDrawer {...baseArgs} />
        </div>
      </div>
    </CenteredSurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const trigger = canvas.getByTestId("session-inspector-drawer-trigger");
    await userEvent.click(trigger);
    await waitFor(() =>
      expect(within(document.body).getByTestId("session-inspector-drawer")).toBeInTheDocument()
    );
  },
};

export const CompactTabSwitch: Story = {
  tags: ["play-fn"],
  parameters: {
    viewport: {
      defaultViewport: "sessionInspectorCompact",
      viewports: {
        sessionInspectorCompact: {
          name: "Compact (short viewport)",
          styles: { width: "480px", height: "620px" },
        },
      },
    },
  },
  render: () => (
    <InspectorFrame height={620}>
      <SessionInspector {...baseArgs} />
    </InspectorFrame>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const usageTab = canvas.getByTestId("session-inspector-tab-usage");
    await userEvent.click(usageTab);
    await waitFor(() =>
      expect(canvas.getByTestId("session-inspector-usage-grid")).toBeInTheDocument()
    );
    const filesTab = canvas.getByTestId("session-inspector-tab-files");
    await userEvent.click(filesTab);
    await waitFor(() =>
      expect(canvas.getByTestId("session-inspector-files-list")).toBeInTheDocument()
    );
  },
};
