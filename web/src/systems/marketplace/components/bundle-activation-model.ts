import type { BundlePreviewRequest, MarketplaceEntryResponse } from "../types";

export function buildBundleRequest(
  data: MarketplaceEntryResponse,
  profile: string,
  scope: "global" | "workspace",
  workspaceId: string | null | undefined,
  bindPrimaryChannel: boolean
): BundlePreviewRequest {
  if (!data.bundle) throw new Error("Bundle detail is required");
  return {
    bind_primary_channel_as_default: bindPrimaryChannel,
    bundle_name: data.entry.name,
    extension_name: data.bundle.extension_name,
    profile_name: profile,
    scope,
    workspace: scope === "workspace" ? (workspaceId ?? undefined) : undefined,
  };
}
