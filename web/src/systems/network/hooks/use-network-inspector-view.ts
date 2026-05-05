import { useChannelMembers, type UseChannelMembersResult } from "./use-channel-members";
import { useNetworkDirects, type UseNetworkDirectsResult } from "./use-directs";
import { useNetworkThreads, type UseNetworkThreadsResult } from "./use-threads";
import { useInspectorState, type UseInspectorStateResult } from "./use-inspector-state";

export interface UseNetworkInspectorViewArgs {
  channel: string | null | undefined;
  /**
   * The view should only fetch members/threads/directs while the inspector is
   * actually visible — when it's collapsed or a thread overlay is taking the
   * right rail, the inspector queries stay quiet.
   */
  enabled: boolean;
}

export interface UseNetworkInspectorViewResult {
  inspector: UseInspectorStateResult;
  members: UseChannelMembersResult;
  threads: UseNetworkThreadsResult;
  directs: UseNetworkDirectsResult;
}

/**
 * Composite view-model for the right-rail Network inspector. Bundles the
 * per-channel inspector state with the three feeds (members, threads,
 * directs) so the route can keep a flat hook surface and avoid blowing the
 * `compozy-react(max-component-complexity)` budget.
 */
export function useNetworkInspectorView({
  channel,
  enabled,
}: UseNetworkInspectorViewArgs): UseNetworkInspectorViewResult {
  const inspector = useInspectorState(channel);
  const queryEnabled = enabled && Boolean(channel);
  const members = useChannelMembers(channel, { enabled: queryEnabled });
  const threads = useNetworkThreads(channel, { enabled: queryEnabled });
  const directs = useNetworkDirects(channel, { enabled: queryEnabled });

  return { inspector, members, threads, directs };
}
