import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";

import { LoopStatusPill } from "../loop-status-pill";
import type { LoopRunStatus } from "../../types";

const meta: Meta<typeof LoopStatusPill> = {
  title: "systems/loops/LoopStatusPill",
  component: LoopStatusPill,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// The 11 first-class loop_run statuses (ADR-013/017/021/022): 5 live, then 6 terminal.
const LIVE_STATUSES: LoopRunStatus[] = [
  "queued",
  "running",
  "watching",
  "needs-approval",
  "paused",
];
const TERMINAL_STATUSES: LoopRunStatus[] = [
  "done",
  "no_op",
  "blocked",
  "failed",
  "exhausted",
  "stalled",
];

export const AllStatuses: Story = {
  render: () => (
    <CenteredSurface>
      <div className="flex flex-col gap-6">
        <div className="flex flex-col gap-3">
          <span className="eyebrow text-(--muted)">Live</span>
          <div className="flex flex-wrap items-center gap-2">
            {LIVE_STATUSES.map(status => (
              <LoopStatusPill key={status} status={status} />
            ))}
          </div>
        </div>
        <div className="flex flex-col gap-3">
          <span className="eyebrow text-(--muted)">Terminal</span>
          <div className="flex flex-wrap items-center gap-2">
            {TERMINAL_STATUSES.map(status => (
              <LoopStatusPill key={status} status={status} />
            ))}
          </div>
        </div>
      </div>
    </CenteredSurface>
  ),
};

export const Running: Story = {
  render: () => (
    <CenteredSurface>
      <LoopStatusPill status="running" />
    </CenteredSurface>
  ),
};

export const Unknown: Story = {
  render: () => (
    <CenteredSurface>
      <LoopStatusPill status="something-else" />
    </CenteredSurface>
  ),
};
