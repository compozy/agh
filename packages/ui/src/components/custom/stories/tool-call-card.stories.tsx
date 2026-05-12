import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../../button";
import { ToolCallCard, type ToolCallStatus } from "../tool-call-card";

const meta: Meta<typeof ToolCallCard> = {
  title: "components/custom/ToolCallCard",
  component: ToolCallCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Inline tool-execution card per DESIGN.md §4. Surface bg + 1 px divider between header and body. Header: terminal icon + tool name + optional file path, status pill + optional timestamp + actions slot pinned right. Compose `<ToolCallCard.Input>` and `<ToolCallCard.Output>` for collapsible argument/result regions (closed by default).",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[720px] bg-background p-6">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

const INPUT_JSON = `{
  "path": "internal/api/handlers/sessions.go",
  "encoding": "utf-8"
}`;

const SHORT_OUTPUT = `package handlers

func ListSessions(w http.ResponseWriter, r *http.Request) { /* … */ }`;

const LONG_OUTPUT = Array.from({ length: 240 }, (_, index) => `line ${index + 1}`).join("\n");

const STATUSES: ToolCallStatus[] = ["pending", "in_progress", "completed", "failed"];

export const Running: Story = {
  args: {
    toolName: "shell.safe-run",
    filePath: "packages/runtime/src/session/stream.ts",
    status: "in_progress",
  },
};

export const Done: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
  },
};

export const FailureWithError: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "failed",
    errorMessage: "ENOENT: no such file or directory",
    actions: (
      <Button size="sm" variant="outline">
        Retry
      </Button>
    ),
  },
};

export const WithInput: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "in_progress",
    timestamp: "2026-05-11T12:00:00Z",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" />
    </ToolCallCard>
  ),
};

export const WithInputAndOutput: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
    timestamp: "2026-05-11T12:00:00Z",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" />
      <ToolCallCard.Output source={SHORT_OUTPUT} format="code" />
    </ToolCallCard>
  ),
};

export const LargeOutputCollapsed: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
    timestamp: "2026-05-11T12:00:00Z",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" />
      <ToolCallCard.Output source={LONG_OUTPUT} format="code" />
    </ToolCallCard>
  ),
};

export const MarkdownInput: Story = {
  args: {
    toolName: "review.summarize",
    status: "completed",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input
        defaultOpen
        format="markdown"
        source={[
          "Summarize **only** the failing checks and ignore passing rows.",
          "",
          "- Highlight regressions vs main",
          "- Suggest a one-line revert candidate",
        ].join("\n")}
      />
      <ToolCallCard.Output
        format="markdown"
        source="_No regressions vs `main` — all 12 checks green._"
      />
    </ToolCallCard>
  ),
};

export const AllStatuses: Story = {
  render: () => (
    <div className="flex flex-col gap-3" data-testid="all-statuses">
      {STATUSES.map(status => (
        <ToolCallCard
          key={status}
          toolName="file.read"
          filePath="packages/runtime/src/session/stream.ts"
          status={status}
          data-status-key={status}
        />
      ))}
    </div>
  ),
};
