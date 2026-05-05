import { useActiveNetworkSession, type UseActiveNetworkSessionResult } from "./use-active-session";
import { useChannelMembers, type UseChannelMembersResult } from "./use-channel-members";
import { useNetworkDirects, type UseNetworkDirectsResult } from "./use-directs";

export interface UseNetworkChannelDirectsRouteResult {
  directs: UseNetworkDirectsResult;
  session: UseActiveNetworkSessionResult;
  members: UseChannelMembersResult;
}

/**
 * Composite view-model for the `/network/:channel/directs` route. Consolidates
 * the directs list, the active session lookup, and the channel members feed so
 * the route component stays under the
 * `compozy-react(max-component-complexity)` hook budget.
 */
export function useNetworkChannelDirectsRoute(
  channel: string
): UseNetworkChannelDirectsRouteResult {
  const directs = useNetworkDirects(channel);
  const session = useActiveNetworkSession(channel);
  const members = useChannelMembers(channel);

  return { directs, session, members };
}
