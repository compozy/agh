import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../button";
import { StatusCard, type StatusCardTone } from "../custom/status-card";

const TONES: StatusCardTone[] = ["success", "warning", "danger", "info", "neutral"];

const meta: Meta<typeof StatusCard> = {
  title: "components/custom/StatusCard",
  component: StatusCard,
  args: {
    tone: "success",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: args => (
    <StatusCard {...args}>
      <StatusCard.Header label="Healthy" />
      <StatusCard.Body>All daemon subsystems are responding.</StatusCard.Body>
    </StatusCard>
  ),
};

export const AllTones: Story = {
  args: {},
  render: () => (
    <div className="grid max-w-3xl grid-cols-1 gap-3 sm:grid-cols-2">
      {TONES.map(tone => (
        <StatusCard key={tone} tone={tone}>
          <StatusCard.Header label={tone} />
          <StatusCard.Body>Daemon status card rendered with {tone} signal.</StatusCard.Body>
        </StatusCard>
      ))}
    </div>
  ),
};

export const WithAction: Story = {
  args: {},
  render: () => (
    <StatusCard tone="danger">
      <StatusCard.Header label="Disconnected" />
      <StatusCard.Body>The daemon is unreachable from the operator UI.</StatusCard.Body>
      <StatusCard.Footer>
        <StatusCard.Action>
          <Button size="sm" type="button" variant="outline">
            Retry
          </Button>
        </StatusCard.Action>
      </StatusCard.Footer>
    </StatusCard>
  ),
};
