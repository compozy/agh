import { useMemo, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import {
  NetworkWorkspaceShell,
  buildPeerCapabilityViews,
  filterNetworkMessagesByKind,
  formatChannelMemberCount,
  formatNetworkDateTime,
  formatNetworkRelativeTime,
  getChannelRecencyAt,
  getNetworkRoomKey,
  getPeerDisplayName,
  getPeerPresenceTone,
  getPeerRecencyAt,
  summarizeChannelMeta,
  summarizeChannelPreview,
  summarizeChannelSubtitle,
} from "@/systems/network";
import type {
  NetworkActiveRoom,
  NetworkChannelSummary,
  NetworkDetailsTab,
  NetworkKindFilter,
  NetworkPeerSummary,
  NetworkRoomListItem,
  NetworkRoomMember,
  NetworkTimelineMessage,
} from "@/systems/network";
import {
  networkChannelFixture,
  networkChannelMessagesFixture,
  networkChannelsFixture,
  networkPeerFixture,
  networkPeerMessagesFixture,
  networkPeersFixture,
  networkStatusFixture,
} from "@/systems/network/mocks";

const meta: Meta<typeof NetworkWorkspaceShell> = {
  title: "systems/network/NetworkWorkspaceShell",
  component: NetworkWorkspaceShell,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Presentational network workspace shell with mocked channel, direct-message, detail rail, and empty/error states.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function requireFixture<T>(value: T | undefined, name: string): T {
  if (!value) {
    throw new Error(`Network Storybook fixture is missing: ${name}.`);
  }
  return value;
}

const storybookChannel = requireFixture(
  networkChannelsFixture.channels.find(channel => channel.channel === "storybook"),
  "storybook channel"
);
const releaseChannel = requireFixture(
  networkChannelsFixture.channels.find(channel => channel.channel === "release"),
  "release channel"
);
const storybookPeer = requireFixture(
  networkPeersFixture.find(peer => peer.peer_id === "peer_storybook_local"),
  "storybook peer"
);

function makeChannelRoom(
  channel: NetworkChannelSummary,
  starredChannelIds: string[],
  selectedRoomKey: string | null
): NetworkRoomListItem {
  const key = getNetworkRoomKey("channel", channel.channel);
  const lastSeenAt = getChannelRecencyAt(channel);

  return {
    id: channel.channel,
    isStarred: starredChannelIds.includes(channel.channel),
    key,
    lastActivityAt: lastSeenAt,
    meta: summarizeChannelMeta(channel),
    preview: summarizeChannelPreview(channel),
    roomType: "channel",
    subtitle: summarizeChannelSubtitle(channel),
    title: channel.channel,
    tone: (channel.message_count ?? 0) > 0 ? "accent" : "neutral",
    unreadCount: selectedRoomKey === key ? 0 : lastSeenAt ? 1 : 0,
  };
}

function makePeerRoom(
  peer: NetworkPeerSummary,
  selectedRoomKey: string | null
): NetworkRoomListItem {
  const key = getNetworkRoomKey("peer", peer.peer_id);
  const lastSeenAt = getPeerRecencyAt(peer);

  return {
    id: peer.peer_id,
    isStarred: false,
    key,
    lastActivityAt: lastSeenAt,
    meta: lastSeenAt ? formatNetworkRelativeTime(lastSeenAt) : "offline",
    preview: `#${peer.channel}`,
    roomType: "peer",
    subtitle: peer.local ? "Local peer" : "Remote peer",
    title: getPeerDisplayName(peer),
    tone: getPeerPresenceTone(peer),
    unreadCount: selectedRoomKey === key ? 0 : lastSeenAt ? 1 : 0,
  };
}

function makeMember(peer: NetworkPeerSummary): NetworkRoomMember {
  return {
    id: peer.peer_id,
    lastSeen: getPeerRecencyAt(peer),
    local: peer.local,
    sessionId: peer.session_id ?? null,
    subtitle: peer.local ? `Local · #${peer.channel}` : `Remote · #${peer.channel}`,
    title: getPeerDisplayName(peer),
    tone: getPeerPresenceTone(peer),
  };
}

function summarizeKinds(messages: NetworkTimelineMessage[]): NetworkActiveRoom["kindCounts"] {
  const counts = new Map<Exclude<NetworkKindFilter, "all">, number>();

  for (const message of messages) {
    if (
      message.kind === "say" ||
      message.kind === "direct" ||
      message.kind === "receipt" ||
      message.kind === "capability" ||
      message.kind === "greet" ||
      message.kind === "whois" ||
      message.kind === "trace"
    ) {
      counts.set(message.kind, (counts.get(message.kind) ?? 0) + 1);
    }
  }

  return [...counts.entries()].map(([kind, count]) => ({ kind, count }));
}

function findLastNonPresenceTimestamp(messages: NetworkTimelineMessage[]): string | null {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message && message.kind !== "greet") {
      return message.timestamp;
    }
  }

  return null;
}

function countVisibleConversationMessages(messages: NetworkTimelineMessage[]): number {
  return messages.filter(message => message.kind !== "greet").length;
}

function makeChannelActiveRoom(
  channel: NetworkChannelSummary,
  messages: NetworkTimelineMessage[],
  activeKind: NetworkKindFilter,
  isStarred: boolean
): NetworkActiveRoom {
  const filteredMessages = filterNetworkMessagesByKind(messages, activeKind);
  const members = (networkChannelFixture.peers ?? []).map(makeMember);
  const lastActivityAt = channel.last_activity_at ?? findLastNonPresenceTimestamp(messages);
  const visibleMessageCount = countVisibleConversationMessages(messages);
  const purpose = channel.purpose ?? null;

  return {
    aboutFields: [
      { label: "Purpose", value: purpose ?? "No purpose has been recorded yet." },
      { label: "Created", value: formatNetworkDateTime(channel.created_at) },
    ],
    canCompose: channel.channel === "storybook",
    canStar: true,
    capabilities: [],
    channel: channel.channel,
    composeHint:
      channel.channel === "storybook"
        ? "Broadcasts send through the first local session in this channel."
        : "This channel has no local session available for composing yet.",
    composePlaceholder: `Send a broadcast to #${channel.channel}`,
    description: purpose ?? `Coordination room for #${channel.channel}.`,
    id: channel.channel,
    introBody:
      purpose ??
      "Materialize this room with a short operator note so other agents know how to use it.",
    introTitle: `Welcome to #${channel.channel}`,
    isStarred,
    key: getNetworkRoomKey("channel", channel.channel),
    kindCounts: summarizeKinds(messages),
    lastActivityAt,
    lastPresenceAt: channel.last_presence_at ?? null,
    memberCount: channel.peer_count ?? members.length,
    members,
    messageCount: channel.message_count ?? visibleMessageCount,
    messages: filteredMessages,
    presenceCount: channel.presence_count ?? 0,
    preview: summarizeChannelPreview(channel),
    purpose,
    roomType: "channel",
    subtitle: summarizeChannelSubtitle(channel),
    title: channel.channel,
    wireFields: [
      { label: "Workspace", mono: true, value: channel.workspace_id ?? "unassigned" },
      { label: "Created By", mono: true, value: channel.created_by ?? "system" },
      { label: "Members", mono: true, value: formatChannelMemberCount(networkChannelFixture) },
      { label: "Last Activity", value: formatNetworkDateTime(lastActivityAt) },
    ],
  };
}

function makePeerActiveRoom(activeKind: NetworkKindFilter): NetworkActiveRoom {
  const filteredMessages = filterNetworkMessagesByKind(networkPeerMessagesFixture, activeKind);
  const channelPeers = networkPeersFixture.filter(peer => peer.channel === storybookPeer.channel);
  const lastActivityAt = findLastNonPresenceTimestamp(networkPeerMessagesFixture);
  const lastSeenAt = getPeerRecencyAt(storybookPeer);
  const visibleMessageCount = countVisibleConversationMessages(networkPeerMessagesFixture);

  return {
    aboutFields: [
      { label: "Peer ID", mono: true, value: storybookPeer.peer_id },
      { label: "Channel", mono: true, value: storybookPeer.channel },
      { label: "Last Seen", value: formatNetworkDateTime(lastSeenAt) },
    ],
    canCompose: true,
    canStar: false,
    capabilities: buildPeerCapabilityViews(
      networkPeerFixture.peer_card.capabilities,
      networkPeerFixture.capability_catalog
    ),
    channel: storybookPeer.channel,
    composeHint: `Direct messages send through the first local session in #${storybookPeer.channel}.`,
    composePlaceholder: `Send a direct message to ${getPeerDisplayName(storybookPeer)}`,
    description: `Directed timeline for ${getPeerDisplayName(storybookPeer)} on #${storybookPeer.channel}.`,
    id: storybookPeer.peer_id,
    introBody:
      "This is a local peer lane. Use it for targeted coordination and handoff acknowledgements.",
    introTitle: `Direct thread with ${getPeerDisplayName(storybookPeer)}`,
    isStarred: false,
    key: getNetworkRoomKey("peer", storybookPeer.peer_id),
    kindCounts: summarizeKinds(networkPeerMessagesFixture),
    lastActivityAt,
    lastPresenceAt: lastSeenAt,
    memberCount: channelPeers.length,
    members: channelPeers.map(makeMember),
    messageCount: visibleMessageCount,
    messages: filteredMessages,
    presenceCount: 0,
    preview: `#${storybookPeer.channel}`,
    purpose: null,
    roomType: "peer",
    subtitle: "Local peer",
    title: getPeerDisplayName(storybookPeer),
    wireFields: [
      { label: "Sent", mono: true, value: String(networkPeerFixture.metrics.sent ?? 0) },
      { label: "Received", mono: true, value: String(networkPeerFixture.metrics.received ?? 0) },
      {
        label: "Delivered",
        mono: true,
        tone: "success",
        value: String(networkPeerFixture.metrics.delivered ?? 0),
      },
      {
        label: "Rejected",
        mono: true,
        tone: "danger",
        value: String(networkPeerFixture.metrics.rejected ?? 0),
      },
    ],
  };
}

function NetworkWorkspaceShellHarness({
  empty = false,
  initialRoomKey = getNetworkRoomKey("channel", storybookChannel.channel),
  roomError = null,
}: {
  empty?: boolean;
  initialRoomKey?: string;
  roomError?: Error | null;
}) {
  const [activeKind, setActiveKind] = useState<NetworkKindFilter>("all");
  const [activeRoomKey, setActiveRoomKey] = useState(initialRoomKey);
  const [composeDraft, setComposeDraft] = useState("Can someone pick up the docs route next?");
  const [detailsTab, setDetailsTab] = useState<NetworkDetailsTab>("about");
  const [isDetailsOpen, setDetailsOpen] = useState(true);
  const [showPresence, setShowPresence] = useState(false);
  const [sidebarQuery, setSidebarQuery] = useState("");
  const [starredChannelIds, setStarredChannelIds] = useState<string[]>(["storybook"]);

  const selectedRoomKey = empty ? null : activeRoomKey;
  const channelRooms = empty
    ? []
    : [storybookChannel, releaseChannel]
        .filter(channel => !starredChannelIds.includes(channel.channel))
        .map(channel => makeChannelRoom(channel, starredChannelIds, selectedRoomKey));
  const starredChannelRooms = empty
    ? []
    : [storybookChannel, releaseChannel]
        .filter(channel => starredChannelIds.includes(channel.channel))
        .map(channel => makeChannelRoom(channel, starredChannelIds, selectedRoomKey));
  const directRooms = empty
    ? []
    : networkPeersFixture.map(peer => makePeerRoom(peer, selectedRoomKey));

  const activeRoom = useMemo(() => {
    if (empty) {
      return null;
    }
    if (activeRoomKey === getNetworkRoomKey("peer", storybookPeer.peer_id)) {
      return makePeerActiveRoom(activeKind);
    }
    if (activeRoomKey === getNetworkRoomKey("channel", releaseChannel.channel)) {
      return makeChannelActiveRoom(
        releaseChannel,
        [],
        activeKind,
        starredChannelIds.includes(releaseChannel.channel)
      );
    }
    return makeChannelActiveRoom(
      storybookChannel,
      networkChannelMessagesFixture,
      activeKind,
      starredChannelIds.includes(storybookChannel.channel)
    );
  }, [activeKind, activeRoomKey, empty, starredChannelIds]);

  return (
    <PanelSurface className="min-h-[760px]">
      <NetworkWorkspaceShell
        activeKind={activeKind}
        activeRoom={activeRoom}
        channelRooms={channelRooms}
        composeDraft={composeDraft}
        detailsTab={detailsTab}
        directRooms={directRooms}
        isComposePending={false}
        isDetailsOpen={isDetailsOpen}
        isRoomLoading={false}
        isTimelineLoading={false}
        onComposeDraftChange={setComposeDraft}
        onComposeSubmit={() => setComposeDraft("")}
        onOpenCreateDialog={() => undefined}
        onSelectDetailsTab={setDetailsTab}
        onSelectKind={setActiveKind}
        onSelectRoom={room => setActiveRoomKey(room.key)}
        onSidebarQueryChange={setSidebarQuery}
        onToggleDetails={() => setDetailsOpen(current => !current)}
        onTogglePresence={() => setShowPresence(current => !current)}
        onToggleStarChannel={channel =>
          setStarredChannelIds(current =>
            current.includes(channel)
              ? current.filter(candidate => candidate !== channel)
              : [channel, ...current]
          )
        }
        roomError={roomError}
        selectedRoomKey={selectedRoomKey}
        showPresence={showPresence}
        sidebarQuery={sidebarQuery}
        starredChannelRooms={starredChannelRooms}
        status={networkStatusFixture}
      />
    </PanelSurface>
  );
}

/**
 * Default shell layout with a channel selected, populated timeline, and detail rail open.
 */
export const Default: Story = {
  args: {},
  render: () => <NetworkWorkspaceShellHarness />,
};

/**
 * Direct-message room state with peer capabilities and peer-level metrics visible.
 */
export const DirectRoom: Story = {
  args: {},
  render: () => (
    <NetworkWorkspaceShellHarness
      initialRoomKey={getNetworkRoomKey("peer", storybookPeer.peer_id)}
    />
  ),
};

/**
 * Empty shell branch before the route has any channels or peers to select.
 */
export const EmptySelection: Story = {
  args: {},
  render: () => <NetworkWorkspaceShellHarness empty />,
};

/**
 * Room-level error branch after a selected room detail request fails.
 */
export const RoomError: Story = {
  args: {},
  render: () => (
    <NetworkWorkspaceShellHarness roomError={new globalThis.Error("Channel detail unavailable")} />
  ),
};
