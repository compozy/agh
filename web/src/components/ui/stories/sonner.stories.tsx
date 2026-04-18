import type { Meta, StoryObj } from "@storybook/react-vite";
import { toast } from "sonner";
import { Button } from "@agh/ui";

import { Toaster } from "../sonner";

const meta: Meta<typeof Toaster> = {
  title: "components/ui/Sonner",
  component: Toaster,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Sonner toaster. Mount a Toaster locally in each story render so notifications stay inside the story iframe and do not leak between stories.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Success: Story = {
  args: {},
  render: () => (
    <div className="flex flex-col items-start gap-3">
      <Button onClick={() => toast.success("Session saved successfully.")}>Trigger toast</Button>
      <Toaster />
    </div>
  ),
};

export const Variants: Story = {
  args: {},
  render: () => (
    <div className="flex flex-col items-start gap-3">
      <div className="flex gap-2">
        <Button variant="outline" onClick={() => toast("Event recorded.")}>
          Default
        </Button>
        <Button variant="outline" onClick={() => toast.info("Dream consolidation scheduled.")}>
          Info
        </Button>
        <Button variant="outline" onClick={() => toast.warning("Approaching token budget.")}>
          Warning
        </Button>
        <Button
          variant="outline"
          onClick={() => toast.error("Daemon disconnected from the UDS socket.")}
        >
          Error
        </Button>
      </div>
      <Toaster />
    </div>
  ),
};

export const WithAction: Story = {
  args: {},
  render: () => (
    <div className="flex flex-col items-start gap-3">
      <Button
        onClick={() =>
          toast("Session archived.", {
            description: "It will be purged in 14 days.",
            action: { label: "Undo", onClick: () => toast.success("Session restored.") },
          })
        }
      >
        Archive session
      </Button>
      <Toaster />
    </div>
  ),
};
