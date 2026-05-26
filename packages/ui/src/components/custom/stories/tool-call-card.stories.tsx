import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../../button";
import { CodeBlock } from "../code-block";
import { ToolCallCard, type ToolCallStatus } from "../tool-call-card";

const meta: Meta<typeof ToolCallCard> = {
  title: "components/custom/ToolCallCard",
  component: ToolCallCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Inline tool-execution card. Collapsed cards are a single header row: terminal icon + tool name + optional file path, Input/Output disclosure chips, signal-toned status icon + optional actions. Body appears only when a chip is open, on error, or when raw children are passed. Compose `<ToolCallCard.Input>` and `<ToolCallCard.Output>` for collapsible argument/result regions (closed by default).",
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

export const ClosedSingleRow: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" language="json" />
      <ToolCallCard.Output source={SHORT_OUTPUT} format="code" language="go" />
    </ToolCallCard>
  ),
};

export const WithInput: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "in_progress",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" language="json" defaultOpen />
    </ToolCallCard>
  ),
};

export const WithInputAndOutput: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" language="json" defaultOpen />
      <ToolCallCard.Output source={SHORT_OUTPUT} format="code" language="go" defaultOpen />
    </ToolCallCard>
  ),
};

export const LargeOutputCollapsed: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={INPUT_JSON} format="code" language="json" />
      <ToolCallCard.Output source={LONG_OUTPUT} format="code" language="plaintext" />
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

export const LiveStack: Story = {
  render: () => (
    <div className="flex flex-col gap-1.5" data-testid="live-stack">
      <ToolCallCard toolName="Bash" filePath="agh agent validate ./agent.toml" status="completed">
        <ToolCallCard.Input
          source="agh agent validate ./agent.toml"
          format="code"
          language="bash"
        />
        <ToolCallCard.Output
          source="✓ schema valid\n✓ tools resolved (4)"
          format="code"
          language="plaintext"
        />
      </ToolCallCard>
      <ToolCallCard
        toolName="Bash"
        filePath="rm -f .claude/agents/wrong-name.md"
        status="completed"
      >
        <ToolCallCard.Input
          source="rm -f .claude/agents/wrong-name.md"
          format="code"
          language="bash"
        />
        <ToolCallCard.Output source="" format="code" language="plaintext" />
      </ToolCallCard>
      <ToolCallCard toolName="Read" filePath="agh.config.toml" status="completed">
        <ToolCallCard.Input
          source='{"file_path":"agh.config.toml"}'
          format="code"
          language="json"
        />
        <ToolCallCard.Output source="[runtime]\nmode = local" format="code" language="toml" />
      </ToolCallCard>
      <ToolCallCard toolName="Grep" filePath='pattern: "INPUT|OUTPUT"' status="in_progress">
        <ToolCallCard.Input source='{"pattern":"INPUT|OUTPUT"}' format="code" language="json" />
        <ToolCallCard.Output source="" format="code" language="plaintext" />
      </ToolCallCard>
      <ToolCallCard toolName="Write" filePath="redesign-toolcall-card.html" status="completed">
        <ToolCallCard.Input
          source='{"file_path":"redesign-toolcall-card.html"}'
          format="code"
          language="json"
        />
        <ToolCallCard.Output source="✓ file written" format="code" language="plaintext" />
      </ToolCallCard>
    </div>
  ),
};

/**
 * Mounts a shiki-highlighted `<CodeBlock>` directly as children — mirrors the
 * web session wrapper's explicit-CodeBlock composition. Regression guard for
 * the children-pass-through render path.
 */
export const CodeBlockInsideInput: Story = {
  args: {
    toolName: "fs.read_file",
    filePath: "internal/api/handlers/sessions.go",
    status: "completed",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input defaultOpen>
        <CodeBlock language="json" code={INPUT_JSON} />
      </ToolCallCard.Input>
    </ToolCallCard>
  ),
};

const LONG_HEADER_COMMAND =
  'agh tool invoke agh__tool_info --input \'{"tool_id":"agh__skill_view","workspace_id":"ws-demo"}\' -o json';

export const LongHeaderTitle: Story = {
  args: {
    toolName: "Bash",
    filePath: LONG_HEADER_COMMAND,
    status: "completed",
  },
  render: args => (
    <ToolCallCard {...args}>
      <ToolCallCard.Input source={LONG_HEADER_COMMAND} format="code" language="bash" />
      <ToolCallCard.Output source="ok" format="code" language="plaintext" />
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
