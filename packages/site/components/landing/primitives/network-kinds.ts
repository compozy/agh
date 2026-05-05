/**
 * Wire-protocol kinds rendered on the landing diagrams. Kept as data-only so
 * the chrome (a `Pill mono` from `@agh/ui`) can be composed inline by callers.
 *
 * `direct` is intentionally not a kind. Restricted two-party conversation is
 * modeled with `surface:"direct"` plus `direct_id`; the message kind for that
 * envelope is still `say` (or any other conversation-bearing kind).
 */
export type NetworkKind = "greet" | "whois" | "say" | "capability" | "receipt" | "trace";

/** One-line purpose for every kind — tooltip copy, alt text, and copy audit source. */
export const KIND_MEANING = {
  greet: "Announce presence + capabilities to a channel",
  whois: "Ask the network which peers match a capability",
  say: "Send conversation text inside a public thread or direct room",
  capability: "Transfer a full capability artifact inside a thread or direct room",
  receipt: "Acknowledge work admission, rejection, or cancellation",
  trace: "Stream lifecycle progress for an open work_id",
} as const satisfies Record<NetworkKind, string>;

/**
 * Conversation surfaces. `surface:"thread"` is the public N-to-N container;
 * `surface:"direct"` is the restricted two-party container.
 */
export type NetworkSurface = "thread" | "direct";

export const SURFACE_MEANING = {
  thread: "Public N-to-N conversation container, identified by thread_id",
  direct: "Restricted two-party conversation container, identified by deterministic direct_id",
} as const satisfies Record<NetworkSurface, string>;
