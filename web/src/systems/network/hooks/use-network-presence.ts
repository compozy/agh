export type NetworkPresenceState = "idle" | "running" | "needs_input" | "errored";

export interface NetworkPresenceArgs {
  channel?: string | null;
  peerId?: string | null;
}

export interface NetworkPresence {
  state: NetworkPresenceState;
}

/**
 * Placeholder presence hook (post-MVP per `_design.md` §5.6 and §11.3).
 * Returns a static idle state so direct-room headers and presence dots can
 * compose against a stable shape today; replace once the protocol exposes
 * presence telemetry.
 */
export function useNetworkPresence(args: NetworkPresenceArgs = {}): NetworkPresence {
  // Placeholder: presence telemetry lands post-MVP; keep args in signature so
  // callers wire up channel/peer once the real source exists.
  void args;
  return { state: "idle" };
}
