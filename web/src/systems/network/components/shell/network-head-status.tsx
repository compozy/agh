import { Pill } from "@agh/ui";

import type { OpenWorkEntry } from "../../hooks/use-work";
import type { NetworkStatus } from "../../types";

/**
 * Head status for the network window (prototype w2-status contract): the
 * listener state + live-peer count at root/channel level; the open-work state
 * takes over while drilled into a thread with work in flight.
 */
export function NetworkHeadStatus({
  status,
  workEntries,
  drilledIntoConversation,
}: {
  status: NetworkStatus;
  workEntries: ReadonlyArray<OpenWorkEntry>;
  drilledIntoConversation: boolean;
}) {
  if (drilledIntoConversation && workEntries.length > 0) {
    const needsInput = workEntries.some(entry => entry.state === "needs_input");
    const working = workEntries.some(entry => entry.state === "working");
    if (needsInput || working) {
      return (
        <Pill data-testid="network-head-status" tone="warning">
          <Pill.Dot pulse={needsInput} tone="warning" />
          {needsInput ? "needs input" : "working"} · {workEntries.length} work open
        </Pill>
      );
    }
  }

  if (!status.enabled) return null;
  const live = status.local_peers ?? 0;

  return (
    <Pill data-testid="network-head-status" tone="success">
      <Pill.Dot tone="success" />
      Active{live > 0 ? ` · ${live} live` : ""}
    </Pill>
  );
}
