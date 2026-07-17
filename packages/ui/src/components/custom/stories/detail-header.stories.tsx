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
          "Detail hero — crumbs / pre-title / title+pills / meta / actions. Bottom hairline on --line; surface stays canvas. When `pills` is set they share the H1 flex row. Pass `back` to wire the chevron back affordance (router.history.back with parent-route fallback).",
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
      crumbs={[
        { id: "workspaces", label: "Workspaces" },
        { id: "personal", label: "personal" },
        { id: "sessions", label: "Sessions" },
      ]}
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

/** Leading mark beside the title block (management detail chrome). */
export const WithLeading: Story = {
  args: {},
  render: () => (
    <DetailHeader
      crumbs={[
        { id: "extensions", label: "Extensions", to: "/extensions" },
        { id: "cost-guard", label: "cost-guard" },
      ]}
      leading={
        <span className="grid size-(--size-provider-logo-well) place-items-center rounded-lg bg-elevated text-muted">
          <span className="text-small-body">◇</span>
        </span>
      }
      meta={<span>registry · installed 2d ago</span>}
      pills={
        <>
          <Pill mono size="xs">
            hook
          </Pill>
          <Pill mono size="xs">
            v0.3.2
          </Pill>
          <Pill mono size="xs" tone="success">
            healthy
          </Pill>
        </>
      }
      title="cost-guard"
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
