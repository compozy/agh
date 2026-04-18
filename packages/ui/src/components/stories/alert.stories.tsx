import type { Meta, StoryObj } from "@storybook/react-vite";
import { TriangleAlertIcon } from "lucide-react";

import { Alert, AlertAction, AlertDescription, AlertTitle } from "../alert";
import { Button } from "../button";

const meta: Meta<typeof Alert> = {
  title: "ui/Alert",
  component: Alert,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Static notification surface with title, description, optional icon, and action.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[480px] bg-background p-4 text-foreground">
        <Story />
      </div>
    ),
  ],
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <Alert>
      <AlertTitle>Session paused</AlertTitle>
      <AlertDescription>
        The orchestrator halted execution; resume to continue from the last checkpoint.
      </AlertDescription>
    </Alert>
  ),
};

export const Destructive: Story = {
  args: {},
  render: () => (
    <Alert variant="destructive">
      <TriangleAlertIcon />
      <AlertTitle>Connection lost</AlertTitle>
      <AlertDescription>
        The daemon did not respond within 30 seconds. Check that it is running locally.
      </AlertDescription>
      <AlertAction>
        <Button size="sm" variant="outline">
          Retry
        </Button>
      </AlertAction>
    </Alert>
  ),
};
