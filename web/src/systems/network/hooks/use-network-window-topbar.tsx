import { Network as NetworkIcon } from "lucide-react";

import { useTopbarSlot } from "@agh/ui";

import { NetworkHeadStatus } from "../components/shell/network-head-status";
import {
  networkThreadsLocation,
  networkWindowTrail,
  type ParsedNetworkWindowLocation,
} from "../lib/network-window-location";
import type { NetworkWindowNavigation } from "./use-network-route-shell";
import type { OpenWorkEntry } from "./use-work";
import type { NetworkStatus } from "../types";

export interface UseNetworkWindowTopbarArgs {
  location: ParsedNetworkWindowLocation;
  navigation: NetworkWindowNavigation;
  workspaceId: string;
  activeChannelKey: string | null;
  channelCount: number;
  status: NetworkStatus | null;
  workEntries: ReadonlyArray<OpenWorkEntry>;
  createAction: React.ReactNode;
  conversationLabel?: string | null;
}

/**
 * Publishes the network drill-in head (glyph · trail · count · status ·
 * actions) into the window topbar, per the os/pagehead contract. Channel
 * selection keeps the root head; only an open conversation drills in, and
 * conversation verbs (promote, mute) stay in the pane header.
 */
export function useNetworkWindowTopbar({
  location,
  navigation,
  workspaceId,
  activeChannelKey,
  channelCount,
  status,
  workEntries,
  createAction,
  conversationLabel,
}: UseNetworkWindowTopbarArgs) {
  const trail = networkWindowTrail(location, conversationLabel);
  const goToRoot = () => navigation.push({ pathname: "/network", search: {} });
  const goToChannel = () => {
    if (workspaceId && activeChannelKey) {
      navigation.push(networkThreadsLocation(workspaceId, activeChannelKey));
    } else {
      goToRoot();
    }
  };

  useTopbarSlot({
    glyph: <NetworkIcon />,
    crumb: trail.leaf,
    crumbs: trail.parents.map(parent => ({
      id: parent.id,
      label: parent.label,
      onSelect: parent.id === "root" ? goToRoot : goToChannel,
    })),
    onBack: trail.drilledIn ? goToChannel : undefined,
    count: !trail.drilledIn && channelCount > 0 ? channelCount : undefined,
    status: status ? (
      <NetworkHeadStatus
        drilledIntoConversation={trail.drilledIn}
        status={status}
        workEntries={workEntries}
      />
    ) : undefined,
    actions: status && !trail.drilledIn ? createAction : undefined,
  });
}
