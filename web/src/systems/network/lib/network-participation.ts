/** Public create/edit participation request shape (UT-060). */
export type NetworkParticipationMode = "local" | "live";

export interface NetworkParticipationDraft {
  mode: NetworkParticipationMode;
  channelId: string;
  channelStrategy: string;
}

export interface NetworkParticipationPayload {
  mode: NetworkParticipationMode;
  channel_id?: string;
  channel_strategy?: string;
}

export const DEFAULT_NETWORK_PARTICIPATION_DRAFT: NetworkParticipationDraft = {
  mode: "local",
  channelId: "",
  channelStrategy: "",
};

/**
 * Serialize draft to exact `network_participation` contract.
 * Local default omits channel fields; never emits legacy participation keys.
 */
export function serializeNetworkParticipation(
  draft: NetworkParticipationDraft
): NetworkParticipationPayload {
  if (draft.mode === "local") {
    return { mode: "local" };
  }
  const channelId = draft.channelId.trim();
  const channelStrategy = draft.channelStrategy.trim();
  return {
    mode: "live",
    ...(channelId ? { channel_id: channelId } : {}),
    ...(channelStrategy ? { channel_strategy: channelStrategy } : {}),
  };
}
