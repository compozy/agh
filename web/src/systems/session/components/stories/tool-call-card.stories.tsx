import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import {
  bashToolMessageFixture,
  errorToolMessageFixture,
  multiHunkEditToolMessageFixture,
  readToolMessageFixture,
  runningBashToolMessageFixture,
  searchToolMessageFixture,
} from "@/systems/session/mocks";
import type { UIMessage } from "@/systems/session/types";

import { SessionToolCallRow } from "../tool-call-card";

const meta: Meta<typeof SessionToolCallRow> = {
  title: "systems/session/components/SessionToolCallRow",
  component: SessionToolCallRow,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const mcpToolMessageFixture: UIMessage = {
  id: "tool_mcp_context",
  role: "tool_result",
  content: "",
  toolName: "mcp__context7__resolve-library-id",
  toolInput: {
    libraryName: "react",
  },
  toolResult: {
    content: "/websites/react_dev",
  },
  timestamp: Date.parse("2026-04-17T16:07:00Z"),
};

const globToolMessageFixture: UIMessage = {
  id: "tool_glob",
  role: "tool_result",
  content: "",
  toolName: "Glob",
  toolInput: {
    pattern: "apps/launch-site/**/*.test.tsx",
  },
  toolResult: {
    stdout: "apps/launch-site/src/components/hero-banner.test.tsx",
  },
  timestamp: Date.parse("2026-04-17T16:07:30Z"),
};

const webSearchToolMessageFixture: UIMessage = {
  id: "tool_web_search",
  role: "tool_result",
  content: "",
  toolName: "WebSearch",
  toolInput: {
    query: "BR partner-bank checkout timeout copy guidance",
  },
  toolResult: {
    content: "3 results reviewed.",
  },
  timestamp: Date.parse("2026-04-17T16:08:00Z"),
};

const aghMemoryToolMessageFixture: UIMessage = {
  id: "tool_agh_memory",
  role: "tool_result",
  content: "",
  toolName: "agh__memory_note",
  toolInput: {
    note: "Launch cutover blocked on partner-bank timeout copy sign-off.",
  },
  toolResult: {
    content: "Recorded memory note.",
  },
  timestamp: Date.parse("2026-04-17T16:08:30Z"),
};

const pendingToolMessageFixture: UIMessage = {
  id: "tool_pending",
  role: "tool_call",
  content: "",
  toolName: "Read",
  toolInput: {},
  timestamp: Date.parse("2026-04-17T16:09:00Z"),
};

const emptyToolMessageFixture: UIMessage = {
  id: "tool_empty",
  role: "tool_result",
  content: "",
  toolName: "Grep",
  toolInput: {
    pattern: "TODO\\(launch\\)",
  },
  toolResult: {},
  timestamp: Date.parse("2026-04-17T16:09:30Z"),
};

function SessionToolCallRowFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl">{children}</div>
    </CenteredSurface>
  );
}

/**
 * Running command row with compact progress status.
 */
export const Running: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <SessionToolCallRow message={runningBashToolMessageFixture} />
    </SessionToolCallRowFrame>
  ),
};

/**
 * Completed command row in its collapsed default state.
 */
export const Done: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <SessionToolCallRow message={bashToolMessageFixture} />
    </SessionToolCallRowFrame>
  ),
};

/**
 * Expanded Bash, Edit, and dynamic MCP payloads keep their existing output renderers inline.
 */
export const ExpandedOutputs: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <div className="flex min-w-0 flex-col gap-2">
        <SessionToolCallRow message={bashToolMessageFixture} defaultExpanded />
        <SessionToolCallRow message={multiHunkEditToolMessageFixture} defaultExpanded />
        <SessionToolCallRow message={mcpToolMessageFixture} defaultExpanded />
      </div>
    </SessionToolCallRowFrame>
  ),
};

/**
 * Completed read row with a path preview.
 */
export const DoneRead: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <SessionToolCallRow message={readToolMessageFixture} />
    </SessionToolCallRowFrame>
  ),
};

/**
 * Failed command row opens the inline error body by default.
 */
export const Error: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <SessionToolCallRow message={errorToolMessageFixture} />
    </SessionToolCallRowFrame>
  ),
};

/**
 * The unified status matrix: one row-state language across every tool state —
 * `pending` (muted, no glyph), `running` (Spinner), `failed` (X danger, danger
 * heading for the true runtime error), `success` (Check), and `empty` (Minus,
 * neutral mid-stream). No special-case bordered/danger boxes remain.
 */
export const StatusMatrix: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <div className="flex min-w-0 flex-col gap-0.5">
        <SessionToolCallRow message={pendingToolMessageFixture} />
        <SessionToolCallRow message={runningBashToolMessageFixture} />
        <SessionToolCallRow message={errorToolMessageFixture} />
        <SessionToolCallRow message={readToolMessageFixture} turnSettled />
        <SessionToolCallRow message={emptyToolMessageFixture} turnSettled={false} />
      </div>
    </SessionToolCallRowFrame>
  ),
};

/**
 * Mixed-tool batch: each row shows its per-tool glyph and a visible tense-aware
 * verb + target — terminal (running), file-text, file-pen, search, folder-search,
 * globe, the AGH-native `memory` family glyph, the MCP connector, and a failed
 * command. No two known tools share the generic terminal icon.
 */
export const MixedToolBatch: Story = {
  args: {},
  render: () => (
    <SessionToolCallRowFrame>
      <div className="flex min-w-0 flex-col gap-0.5">
        <SessionToolCallRow message={runningBashToolMessageFixture} />
        <SessionToolCallRow message={readToolMessageFixture} />
        <SessionToolCallRow message={multiHunkEditToolMessageFixture} />
        <SessionToolCallRow message={searchToolMessageFixture} />
        <SessionToolCallRow message={globToolMessageFixture} />
        <SessionToolCallRow message={webSearchToolMessageFixture} />
        <SessionToolCallRow message={aghMemoryToolMessageFixture} />
        <SessionToolCallRow message={mcpToolMessageFixture} />
        <SessionToolCallRow message={errorToolMessageFixture} />
      </div>
    </SessionToolCallRowFrame>
  ),
};
