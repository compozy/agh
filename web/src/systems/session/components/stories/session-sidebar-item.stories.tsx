import type { Meta, StoryObj } from "@storybook/react-vite";

import { SidebarSurface } from "@/storybook/story-layout";
import { primarySessionFixture } from "@/systems/session/mocks";

import { SessionSidebarItem } from "../session-sidebar-item";

const meta: Meta<typeof SessionSidebarItem> = {
  title: "systems/session/SessionSidebarItem",
  component: SessionSidebarItem,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function SessionSidebarItemFrame({
  hasPendingPermission = false,
}: {
  hasPendingPermission?: boolean;
}) {
  return (
    <SidebarSurface>
      <div className="p-3">
        <SessionSidebarItem
          hasPendingPermission={hasPendingPermission}
          session={primarySessionFixture}
          workspaceName="agh2"
        />
      </div>
    </SidebarSurface>
  );
}

export const Default: Story = {
  render: () => <SessionSidebarItemFrame />,
};

export const Selected: Story = {
  parameters: {
    router: {
      initialEntries: [`/session/${primarySessionFixture.id}`],
    },
  },
  render: () => <SessionSidebarItemFrame />,
};

export const Unread: Story = {
  render: () => <SessionSidebarItemFrame hasPendingPermission />,
};
