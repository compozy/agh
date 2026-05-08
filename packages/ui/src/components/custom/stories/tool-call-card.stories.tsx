import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, within } from "storybook/test";

import { ToolCallCard, type ToolCallStatus } from "../tool-call-card";

const meta: Meta<typeof ToolCallCard> = {
  title: "components/custom/ToolCallCard",
  component: ToolCallCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Inline tool-execution card per DESIGN.md §4. Surface bg + 1px divider border, terminal icon + tool name + optional file path, status badge pinned right.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const STATUSES: ToolCallStatus[] = ["running", "done", "error"];

const STATUS_EXPECT: Record<ToolCallStatus, { tone: string; label: string }> = {
  running: { tone: "accent", label: "RUNNING" },
  done: { tone: "success", label: "DONE" },
  error: { tone: "danger", label: "ERROR" },
};

export const Running: Story = {
  args: {
    toolName: "shell.safe-run",
    filePath: "packages/runtime/src/session/stream.ts",
    status: "running",
  },
};

export const Done: Story = {
  args: {
    toolName: "file.read",
    filePath: "packages/runtime/src/session/stream.ts",
    status: "done",
  },
};

export const Error: Story = {
  args: {
    toolName: "file.write",
    filePath: "packages/runtime/src/session/stream.ts",
    status: "error",
  },
};

export const WithOutputBody: Story = {
  args: {
    toolName: "shell.safe-run",
    filePath: "packages/runtime",
    status: "done",
    children: (
      <pre className="font-mono text-[12px] leading-[1.6] text-[color:var(--color-text-secondary)]">
        $ rg &quot;onToolCall&quot; packages/runtime -l{"\n"}
        packages/runtime/src/session/stream.ts{"\n"}
        packages/runtime/src/session/replay.ts
      </pre>
    ),
  },
};

export const NoFilePath: Story = {
  args: {
    toolName: "agent.thinking",
    status: "running",
  },
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

export const StatusCycleInteraction: Story = {
  render: () => (
    <div className="flex flex-col gap-3" data-testid="status-cycle">
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
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const wrapper = await canvas.findByTestId("status-cycle");
    for (const status of STATUSES) {
      const card = wrapper.querySelector<HTMLElement>(`[data-status-key="${status}"]`);
      await expect(card).not.toBeNull();
      await expect(card?.getAttribute("data-status")).toBe(status);
      const badge = card?.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
      await expect(badge?.textContent).toBe(STATUS_EXPECT[status].label);
      await expect(badge?.getAttribute("data-tone")).toBe(STATUS_EXPECT[status].tone);
    }
  },
};
