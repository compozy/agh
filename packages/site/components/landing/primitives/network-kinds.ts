/**
 * Wire-protocol kinds rendered on the landing diagrams. Kept as data-only so
 * the chrome (a `Pill mono` from `@agh/ui`) can be composed inline by callers.
 */
export type NetworkKind = "greet" | "whois" | "say" | "direct" | "capability" | "receipt" | "trace";

/** One-line purpose for every kind — tooltip copy, alt text, and copy audit source. */
export const KIND_MEANING = {
  greet: "Announce presence + capabilities to a channel",
  whois: "Ask the network which peers match a capability",
  say: "Free-form operator chat to a channel",
  direct: "Send a structured task to a named peer",
  capability: "Transfer a full capability artifact to a peer",
  receipt: "Confirm completion with status and trace IDs",
  trace: "Stream progress updates during a task",
} as const satisfies Record<NetworkKind, string>;
