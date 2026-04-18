import type { Meta, StoryObj } from "@storybook/react-vite";

import { Panel, PanelBody, PanelHeader, PanelTitle } from "../panel";
import { TextureCanvas } from "../texture-canvas";

const meta: Meta<typeof TextureCanvas> = {
  title: "components/design-system/TextureCanvas",
  component: TextureCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "The full-page graphite canvas with subtle striping, glow, and vignette used as the atmospheric base for AGH command surfaces.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default textured canvas with a single anchored panel to show depth against the background.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <TextureCanvas>
      <div className="relative z-10 flex min-h-dvh items-center justify-center p-8">
        <Panel className="w-full max-w-xl">
          <PanelHeader>
            <PanelTitle>Graphite texture, edge vignette, and soft accent glow.</PanelTitle>
          </PanelHeader>
          <PanelBody>
            <p className="text-sm leading-6 text-[color:var(--color-text-secondary)]">
              The canvas exists to make centered control surfaces feel intentional instead of
              floating on flat black.
            </p>
          </PanelBody>
        </Panel>
      </div>
    </TextureCanvas>
  ),
};
