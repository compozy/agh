import type { Meta, StoryObj } from "@storybook/react-vite";
import { BookOpen, Repeat2, Workflow } from "lucide-react";

import { Pill, PillDot } from "../pill";
import { PageHead } from "../page-head";

const meta: Meta<typeof PageHead> = {
  title: "components/custom/PageHead",
  component: PageHead,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Route identity block (route chrome §08). Every variant starts with the 24px elevated icon well and owns the route's single focusable H1 (`tabIndex={-1}`). PH1 index: icon · H1 · count · meta. PH2 detail: icon · H1 · pills · meta, hairline base. PH3 compact: pre-title · compact H1 · short meta for split-pane surfaces.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Index: Story = {
  render: () => (
    <PageHead
      count={12}
      icon={Workflow}
      meta={
        <>
          <span>Messaging bridges that connect AGH to external platforms.</span>
          <PageHead.MetaDot />
          <span>launch-hq</span>
        </>
      }
      title="Bridges"
    />
  ),
};

export const Detail: Story = {
  render: () => (
    <PageHead
      icon={Repeat2}
      meta={
        <>
          <span>agh.loop/v1</span>
          <PageHead.MetaDot />
          <span>8 nodes</span>
          <PageHead.MetaDot />
          <span>Canonical ship-a-change delivery loop.</span>
        </>
      }
      pills={
        <>
          <Pill tone="success">
            <PillDot />
            ready
          </Pill>
          <Pill tone="neutral">workspace</Pill>
          <Pill mono tone="neutral">
            v4
          </Pill>
        </>
      }
      title="software-delivery"
      variant="detail"
    />
  ),
};

export const DetailWithActions: Story = {
  render: () => (
    <PageHead
      actions={
        <span className="inline-flex h-7 items-center rounded-md border border-line bg-canvas-soft px-3 text-small-body text-muted">
          Provider · Model · Reasoning
        </span>
      }
      icon={Repeat2}
      meta={<span>Engineering / Release</span>}
      pills={
        <Pill tone="success">
          <PillDot />
          Active
        </Pill>
      }
      title="release-captain"
      variant="detail"
    />
  ),
};

export const Compact: Story = {
  render: () => (
    <PageHead
      icon={BookOpen}
      meta={
        <>
          <span>Global</span>
          <PageHead.MetaDot />
          <span>Apr 17, 2026, 14:30</span>
        </>
      }
      pretitle="operator-style.md"
      title="Operator Style"
      variant="compact"
    />
  ),
};
