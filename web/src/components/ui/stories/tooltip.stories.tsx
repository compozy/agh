import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button, Kbd, KbdGroup } from "@agh/ui";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../tooltip";

const meta: Meta<typeof Tooltip> = {
  title: "components/ui/Tooltip",
  component: Tooltip,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Keyboard-friendly tooltip. Wrap the story in a TooltipProvider and drive the trigger from any @agh/ui primitive.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <TooltipProvider delay={150}>
      <Tooltip>
        <TooltipTrigger render={<Button variant="outline">Hover me</Button>} />
        <TooltipContent>Renames this session locally</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  ),
};

export const WithShortcut: Story = {
  args: {},
  render: () => (
    <TooltipProvider delay={150}>
      <Tooltip>
        <TooltipTrigger render={<Button variant="ghost">Run command</Button>} />
        <TooltipContent>
          Open palette
          <KbdGroup>
            <Kbd>⌘</Kbd>
            <Kbd>K</Kbd>
          </KbdGroup>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  ),
};
