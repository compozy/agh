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

import { ToolCallRow } from "../tool-call-card";

const meta: Meta<typeof ToolCallRow> = {
  title: "systems/session/components/ToolCallRow",
  component: ToolCallRow,
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

function ToolCallRowFrame({ children }: { children: React.ReactNode }) {
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
    <ToolCallRowFrame>
      <ToolCallRow message={runningBashToolMessageFixture} />
    </ToolCallRowFrame>
  ),
};

/**
 * Completed command row in its collapsed default state.
 */
export const Done: Story = {
  args: {},
  render: () => (
    <ToolCallRowFrame>
      <ToolCallRow message={bashToolMessageFixture} />
    </ToolCallRowFrame>
  ),
};

/**
 * Expanded Bash, Edit, and dynamic MCP payloads keep their existing output renderers inline.
 */
export const ExpandedOutputs: Story = {
  args: {},
  render: () => (
    <ToolCallRowFrame>
      <div className="flex min-w-0 flex-col gap-2">
        <ToolCallRow message={bashToolMessageFixture} defaultExpanded />
        <ToolCallRow message={multiHunkEditToolMessageFixture} defaultExpanded />
        <ToolCallRow message={mcpToolMessageFixture} defaultExpanded />
      </div>
    </ToolCallRowFrame>
  ),
};

/**
 * Completed read row with a path preview.
 */
export const DoneRead: Story = {
  args: {},
  render: () => (
    <ToolCallRowFrame>
      <ToolCallRow message={readToolMessageFixture} />
    </ToolCallRowFrame>
  ),
};

/**
 * Failed command row opens the inline error body by default.
 */
export const Error: Story = {
  args: {},
  render: () => (
    <ToolCallRowFrame>
      <ToolCallRow message={errorToolMessageFixture} />
    </ToolCallRowFrame>
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
    <ToolCallRowFrame>
      <div className="flex min-w-0 flex-col gap-0.5">
        <ToolCallRow message={pendingToolMessageFixture} />
        <ToolCallRow message={runningBashToolMessageFixture} />
        <ToolCallRow message={errorToolMessageFixture} />
        <ToolCallRow message={readToolMessageFixture} turnSettled />
        <ToolCallRow message={emptyToolMessageFixture} turnSettled={false} />
      </div>
    </ToolCallRowFrame>
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
    <ToolCallRowFrame>
      <div className="flex min-w-0 flex-col gap-0.5">
        <ToolCallRow message={runningBashToolMessageFixture} />
        <ToolCallRow message={readToolMessageFixture} />
        <ToolCallRow message={multiHunkEditToolMessageFixture} />
        <ToolCallRow message={searchToolMessageFixture} />
        <ToolCallRow message={globToolMessageFixture} />
        <ToolCallRow message={webSearchToolMessageFixture} />
        <ToolCallRow message={aghMemoryToolMessageFixture} />
        <ToolCallRow message={mcpToolMessageFixture} />
        <ToolCallRow message={errorToolMessageFixture} />
      </div>
    </ToolCallRowFrame>
  ),
};
