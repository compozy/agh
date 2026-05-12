import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button, DetailHeader, Pill } from "@agh/ui";

const meta: Meta<typeof DetailHeader> = {
  title: "components/custom/DetailHeader",
  component: DetailHeader,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Six-row detail hero — crumbs / pre-title / 24 px H1 / pills / meta / actions. Bottom hairline on --line; surface stays canvas. Pass `back` to wire the chevron back affordance (router.history.back with parent-route fallback).",
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

/** Full anatomy with every row populated. */
export const Full: Story = {
  args: {},
  render: () => (
    <DetailHeader
      crumbs={[{ label: "Workspaces" }, { label: "personal" }, { label: "Sessions" }]}
      preTitle="Run #42"
      title="Refactor internal/network for the new agh-network/v0 contract"
      back={() => undefined}
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

/** Title-only — verifies the gap structure when optional rows are omitted. */
export const TitleOnly: Story = {
  args: {},
  render: () => <DetailHeader title="Untitled session" />,
};

/** Crumbs + back affordance only. */
export const WithBack: Story = {
  args: {},
  render: () => (
    <DetailHeader
      title="Provider configuration"
      crumbs="Settings / Providers"
      back={() => undefined}
    />
  ),
};

/** Plain ReactNode crumbs (legacy bag of children). */
export const NodeCrumbs: Story = {
  args: {},
  render: () => <DetailHeader title="Knowledge entry" crumbs="Knowledge / Notes / 2026-05-11" />,
};
