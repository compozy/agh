import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button, Pill } from "@agh/ui";
import { DetailHeader } from "../detail-header";

const meta: Meta<typeof DetailHeader> = {
  title: "components/custom/DetailHeader",
  component: DetailHeader,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Detail-page hero header. Mono UPPERCASE crumbs row above an Inter 24px / 510-weight title, optional pill row, optional dense meta row, and a trailing action cluster. Bottom rule on `--line`; surface stays canvas (no extra fill).",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[960px] bg-background">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Full header with crumbs, title, pills, meta, and an action cluster.
 */
export const Full: Story = {
  args: {},
  render: () => (
    <DetailHeader
      crumbs="Workspaces / personal / sessions"
      title="Refactor internal/network for the new agh-network/v0 contract"
      pills={
        <>
          <Pill tone="accent">In progress</Pill>
          <Pill tone="neutral">Anthropic Claude</Pill>
        </>
      }
      meta={
        <>
          <span>Started 04:21 UTC</span>
          <span>Owner pedronauck</span>
        </>
      }
      actions={
        <>
          <Button size="sm" variant="outline">
            Inspect
          </Button>
          <Button size="sm">Resume</Button>
        </>
      }
    />
  ),
};

/**
 * Title-only minimal — verifies the gap structure when the optional rows are omitted.
 */
export const TitleOnly: Story = {
  args: {},
  render: () => <DetailHeader title="Untitled session" />,
};
