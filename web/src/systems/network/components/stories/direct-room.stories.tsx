import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { networkDirectRoomDetailFixture, networkPeersFixture } from "@/systems/network/mocks";

import { DirectRoom } from "../directs/direct-room";
import { NewDirectDialog } from "../directs/new-direct-dialog";

const meta: Meta<typeof DirectRoom> = {
  title: "systems/network/DirectRoom",
  component: DirectRoom,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Direct room conversation surface plus new-direct peer picker dialog.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[640px] p-0">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Direct room uses network MSW handlers for detail, messages, and work state.
 */
export const Default: Story = {
  args: {
    channel: networkDirectRoomDetailFixture.channel,
    directId: networkDirectRoomDetailFixture.direct_id,
    selfPeerId: networkDirectRoomDetailFixture.peer_a,
  },
};

/**
 * Peer picker dialog loads channel peers through MSW and resolves a room on selection.
 */
export const NewDirect: Story = {
  args: {},
  render: () => (
    <NewDirectDialog
      open
      onOpenChange={() => undefined}
      channel={networkDirectRoomDetailFixture.channel}
      selfPeerId={networkPeersFixture[0]?.peer_id}
      sessionId="session_launch_coordination"
    />
  ),
};
