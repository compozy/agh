import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";

import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../dialog";
import { Button } from "../button";
import { Input } from "../input";
import { Label } from "../label";
import { UIProvider } from "../ui-provider";

const meta: Meta<typeof Dialog> = {
  title: "ui/Dialog",
  component: Dialog,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Modal dialog built on Base UI with motion-driven enter/exit animations via `AnimatePresence`. Respects the `reducedMotion` setting from `UIProvider`.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

function BasicDialog() {
  return (
    <Dialog>
      <DialogTrigger render={<Button>Rename task</Button>} />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename task</DialogTitle>
          <DialogDescription>
            Update the display name of the currently selected task.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-2 px-0 py-2">
          <Label htmlFor="task-name">Name</Label>
          <Input id="task-name" defaultValue="Spin up agent" />
        </div>
        <DialogFooter>
          <DialogClose render={<Button variant="ghost">Cancel</Button>} />
          <DialogClose render={<Button>Save</Button>} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export const Default: Story = {
  render: () => <BasicDialog />,
};

export const ReducedMotion: Story = {
  parameters: {
    docs: {
      description: {
        story:
          "With `UIProvider reducedMotion='always'`, motion drops the scale transform and animates only opacity on open/close.",
      },
    },
  },
  render: () => (
    <UIProvider reducedMotion="always">
      <BasicDialog />
    </UIProvider>
  ),
};

export const OpenAndFocus: Story = {
  render: () => <BasicDialog />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const trigger = await canvas.findByRole("button", { name: "Rename task" });
    await userEvent.click(trigger);
    const dialog = await waitFor(() => within(document.body).getByRole("dialog"));
    await expect(dialog).toBeInTheDocument();
    const input = await within(document.body).findByLabelText("Name");
    await expect(input).toHaveValue("Spin up agent");
    await userEvent.keyboard("{Escape}");
    await waitFor(
      () => expect(within(document.body).queryByRole("dialog")).not.toBeInTheDocument(),
      { timeout: 2000 }
    );
  },
};

export const HiddenCloseButton: Story = {
  render: () => (
    <Dialog>
      <DialogTrigger render={<Button variant="outline">Open</Button>} />
      <DialogContent showCloseButton={false}>
        <DialogTitle>Confirm</DialogTitle>
        <DialogDescription>The close button is hidden.</DialogDescription>
        <DialogFooter>
          <DialogClose render={<Button>Got it</Button>} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  ),
};
