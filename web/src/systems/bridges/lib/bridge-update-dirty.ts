import { parseBridgeProviderConfig } from "./bridge-drafts";
import { compactBridgeDeliveryDefaults } from "./bridge-formatters";
import type { BridgeUpdateDraft } from "../types";

/** Stable serialization so key order can never read as a change. */
function stableStringify(value: unknown): string {
  if (value === null || typeof value !== "object") return JSON.stringify(value) ?? "null";
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(",")}]`;
  const entries = Object.entries(value as Record<string, unknown>)
    .filter(([, entryValue]) => entryValue !== undefined)
    .sort(([left], [right]) => (left < right ? -1 : left > right ? 1 : 0));
  return `{${entries.map(([key, entryValue]) => `${JSON.stringify(key)}:${stableStringify(entryValue)}`).join(",")}}`;
}

/**
 * Whether a bridge update draft differs from what was loaded.
 *
 * `UpdateBridgeRequest` is an all-pointer patch and the daemon rejects a patch
 * with nothing in it, so the editor must not offer a save that cannot succeed.
 * Provider config is compared as parsed JSON rather than as text: reformatting
 * the same object is not a change the contract can carry.
 */
export function isBridgeUpdateDraftDirty(
  draft: BridgeUpdateDraft,
  pristine: BridgeUpdateDraft
): boolean {
  if (draft.displayName.trim() !== pristine.displayName.trim()) return true;
  if (draft.dmPolicy !== pristine.dmPolicy) return true;
  if (stableStringify(draft.routingPolicy) !== stableStringify(pristine.routingPolicy)) return true;
  if (
    stableStringify(compactBridgeDeliveryDefaults(draft.deliveryDefaults)) !==
    stableStringify(compactBridgeDeliveryDefaults(pristine.deliveryDefaults))
  ) {
    return true;
  }

  const draftConfig = parseBridgeProviderConfig(draft.providerConfigText);
  const pristineConfig = parseBridgeProviderConfig(pristine.providerConfigText);
  // An unparseable draft is blocked by its own validation, not by this gate.
  if (draftConfig.error) return true;
  return stableStringify(draftConfig.value) !== stableStringify(pristineConfig.value);
}
