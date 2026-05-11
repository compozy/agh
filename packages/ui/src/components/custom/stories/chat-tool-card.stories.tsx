import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../../button";
import { ChatToolCard } from "../chat-tool-card";

const meta: Meta<typeof ChatToolCard> = {
  title: "components/custom/ChatToolCard",
  component: ChatToolCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Tool-call surface per ADR-014 §6 — head row with `<MonoId>` tool name + status pill + optional relative `<Time>`, plus collapsible input + output regions, an actions slot, and a `--danger-tint` failure state with inline error message. Outputs longer than 200 lines collapse by default.",
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

export const Running: Story = {
  args: {
    toolName: "fs.read_file",
    status: "in_progress",
    timestamp: "2026-05-11T12:00:00Z",
    input: { source: INPUT_JSON, format: "code" },
  },
};

export const Success: Story = {
  args: {
    toolName: "fs.read_file",
    status: "completed",
    timestamp: "2026-05-11T12:00:00Z",
    input: { source: INPUT_JSON, format: "code" },
    output: { source: SHORT_OUTPUT, format: "code" },
  },
};

export const Failure: Story = {
  args: {
    toolName: "fs.read_file",
    status: "failed",
    timestamp: "2026-05-11T12:00:00Z",
    input: { source: INPUT_JSON, format: "code" },
    errorMessage: "ENOENT: no such file or directory",
    actions: (
      <Button size="sm" variant="outline">
        Retry
      </Button>
    ),
  },
};

export const LargeOutput: Story = {
  args: {
    toolName: "fs.read_file",
    status: "completed",
    timestamp: "2026-05-11T12:00:00Z",
    input: { source: INPUT_JSON, format: "code" },
    output: { source: LONG_OUTPUT, format: "code" },
  },
};

export const MarkdownArgs: Story = {
  args: {
    toolName: "review.summarize",
    status: "completed",
    input: {
      source: [
        "Summarize **only** the failing checks and ignore passing rows.",
        "",
        "- Highlight regressions vs main",
        "- Suggest a one-line revert candidate",
      ].join("\n"),
      format: "markdown",
    },
    output: {
      source: "_No regressions vs `main` — all 12 checks green._",
      format: "markdown",
    },
  },
};
