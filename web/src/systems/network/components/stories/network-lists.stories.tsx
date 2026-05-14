import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import {
  networkDirectRoomsFixture,
  networkThreadsFixture,
  networkWorkFixture,
} from "@/systems/network/mocks";
import type { OpenWorkEntry } from "@/systems/network/hooks/use-work";
import type { ChannelMember } from "@/systems/network/hooks/use-channel-members";

import { ActivityFeed } from "../activity/activity-feed";
import { DirectsList } from "../directs/directs-list";
import { InspectorActivityFeed } from "../shell/inspector-activity-feed";
import { InspectorMembersList } from "../shell/inspector-members-list";
import { NetworkInspector } from "../shell/network-inspector";
import { ThreadsList } from "../threads/threads-list";
import { WorkInspectorRow } from "../work/work-inspector-row";

const channel = "launch-war-room";
const workEntry: OpenWorkEntry = {
  workId: networkWorkFixture.work_id,
  state: networkWorkFixture.state,
  messageId: "msg_launch_work",
  targetPeerId: networkWorkFixture.target_peer_id ?? null,
  openedAt: networkWorkFixture.opened_at ?? null,
  lastActivityAt: networkWorkFixture.last_activity_at ?? null,
};
const members: ChannelMember[] = [
  { peerId: "northstar-local", displayName: "Northstar Local", role: "agent", local: true },
  { peerId: "partner-settlement", displayName: "Partner Settlement", role: "agent", local: false },
  { peerId: "ops-human", displayName: "Ops Human", role: "human", local: false },
];

const meta: Meta<typeof ThreadsList> = {
  title: "systems/network/NetworkLists",
  component: ThreadsList,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Thread, direct, activity, and open-work list surfaces for network navigation.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[520px] p-0">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Threads list with active row and open-work chip.
 */
export const Threads: Story = {
  args: {},
  render: () => (
    <ThreadsList
      workspaceId={storyDefaultWorkspaceId}
      channel={channel}
      threads={networkThreadsFixture}
      activeThreadId={networkThreadsFixture[0]?.thread_id ?? null}
      isLoading={false}
      onStartThread={fn()}
    />
  ),
};

/**
 * Direct rooms list resolves the opposite peer and member role.
 */
export const Directs: Story = {
  args: {},
  render: () => (
    <DirectsList
      workspaceId={storyDefaultWorkspaceId}
      channel={channel}
      directs={networkDirectRoomsFixture}
      activeDirectId={networkDirectRoomsFixture[0]?.direct_id ?? null}
      isLoading={false}
      selfPeerId="northstar-local"
      members={[
        {
          peerId: "partner-settlement",
          displayName: "Partner Settlement",
          role: "agent",
          local: false,
        },
        { peerId: "northstar-growth", displayName: "Growth Desk", role: "human", local: false },
      ]}
      onNewDirect={fn()}
    />
  ),
};

/**
 * Activity feed merges thread and direct activity by last timestamp.
 */
export const Activity: Story = {
  args: {},
  render: () => (
    <ActivityFeed
      workspaceId={storyDefaultWorkspaceId}
      channel={channel}
      threads={networkThreadsFixture}
      directs={networkDirectRoomsFixture}
      isLoading={false}
    />
  ),
};

/**
 * Inspector lists cover member identity, activity ordering, tabs, and close
 * affordances without mounting the full route shell.
 */
export const Inspector: StoryObj<typeof NetworkInspector> = {
  args: {},
  render: () => (
    <div className="grid min-h-[520px] gap-4 md:grid-cols-[360px_360px]">
      <NetworkInspector
        channel={channel}
        activeTab="members"
        onTabChange={fn()}
        onClose={fn()}
        members={members}
        isMembersLoading={false}
        workEntries={[workEntry]}
        isWorkLoading={false}
        workCount={1}
        onWorkJump={fn()}
        threads={networkThreadsFixture}
        directs={networkDirectRoomsFixture}
        isActivityLoading={false}
      />
      <div className="grid min-h-0 gap-4">
        <InspectorMembersList members={members} />
        <InspectorActivityFeed
          channel={channel}
          threads={networkThreadsFixture}
          directs={networkDirectRoomsFixture}
        />
      </div>
    </div>
  ),
};

/**
 * Open-work row shows target, age, state chip, and jump action.
 */
export const WorkRow: Story = {
  args: {},
  render: () => (
    <ul className="w-full max-w-md">
      <WorkInspectorRow entry={workEntry} onJump={fn()} />
    </ul>
  ),
};
