import type { Meta, StoryObj } from "@storybook/react-vite";

import { DesignSystemShowcase } from "../design-system-showcase";

const meta: Meta<typeof DesignSystemShowcase> = {
  title: "components/design-system/DesignSystemShowcase",
  component: DesignSystemShowcase,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "The living preview route for the first-pass AGH design foundations, combining the shared primitives into a command-surface composition.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Full design-system showcase used by the home route.
 */
export const Default: Story = {
  args: {},
};
