import type { Meta, StoryObj } from "@storybook/react-vite";

import { Avatar, AvatarFallback, Button, Pill } from "@agh/ui";
import { ReviewRow } from "../review-row";

const meta: Meta<typeof ReviewRow> = {
  title: "components/custom/ReviewRow",
  component: ReviewRow,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Compact review/decision row with leading avatar, title + description, secondary metadata, and trailing actions. Tone modifies only the border alpha (`--accent/40`, `--success/40`, …); the surface stays `--canvas-soft` to avoid signal-banner anti-patterns.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[640px] bg-background p-4 flex flex-col gap-2">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

const SAMPLE_AVATAR = (initials: string) => (
  <Avatar shape="circle" size="sm">
    <AvatarFallback>{initials}</AvatarFallback>
  </Avatar>
);

/**
 * Three review rows with different tones — the title hierarchy stays consistent.
 */
export const Tones: Story = {
  args: {},
  render: () => (
    <>
      <ReviewRow
        leading={SAMPLE_AVATAR("AC")}
        title="Anthropic Claude approved migration"
        description="Diff of 3 files in internal/network."
        meta={<Pill tone="accent">2 minutes ago</Pill>}
        tone="accent"
        actions={
          <Button size="xs" variant="outline">
            Open
          </Button>
        }
      />
      <ReviewRow
        leading={SAMPLE_AVATAR("OP")}
        title="OpenAI raised a regression risk"
        description="Suggests adding a coverage test before merging."
        meta={<Pill tone="warning">Risk</Pill>}
        tone="warning"
        actions={
          <Button size="xs" variant="outline">
            Reply
          </Button>
        }
      />
      <ReviewRow
        leading={SAMPLE_AVATAR("LL")}
        title="Local llama-3.3 confirmed"
        description="No further changes requested."
        meta={<Pill tone="success">Pass</Pill>}
        tone="success"
      />
    </>
  ),
};
