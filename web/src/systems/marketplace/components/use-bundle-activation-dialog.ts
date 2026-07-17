import { useEffect, useState } from "react";
import { toast } from "sonner";

import {
  useActivateMarketplaceBundle,
  usePreviewMarketplaceBundle,
} from "../hooks/use-marketplace-actions";
import type { MarketplaceEntryResponse } from "../types";
import { buildBundleRequest } from "./bundle-activation-model";

interface UseBundleActivationDialogOptions {
  data: MarketplaceEntryResponse;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  workspaceId?: string | null;
}

export function useBundleActivationDialog({
  data,
  onOpenChange,
  open,
  workspaceId,
}: UseBundleActivationDialogOptions) {
  const bundle = data.bundle;
  const defaultProfile = bundle?.profiles[0]?.name ?? "default";
  const [profile, setProfile] = useState(defaultProfile);
  const [scope, setScope] = useState<"global" | "workspace">(workspaceId ? "workspace" : "global");
  const [bindPrimaryChannel, setBindPrimaryChannel] = useState(false);
  const preview = usePreviewMarketplaceBundle();
  const activate = useActivateMarketplaceBundle();
  useEffect(() => {
    if (!open || !bundle) return;
    preview.mutate(buildBundleRequest(data, profile, scope, workspaceId, bindPrimaryChannel));
  }, [bindPrimaryChannel, bundle, data, open, preview.mutate, profile, scope, workspaceId]);

  const rerunPreview = () => {
    activate.reset();
    preview.mutate(buildBundleRequest(data, profile, scope, workspaceId, bindPrimaryChannel));
  };

  const activateBundle = async () => {
    try {
      await activate.mutateAsync(
        buildBundleRequest(data, profile, scope, workspaceId, bindPrimaryChannel)
      );
      toast.success(`${data.entry.name} activated`, {
        action: {
          label: "Manage →",
          onClick: () => {
            window.location.href = "/extensions?tab=bundles";
          },
        },
      });
      onOpenChange(false);
    } catch {
      // The mutation error stays visible in the dialog.
    }
  };

  return {
    activate,
    activateBundle,
    bindPrimaryChannel,
    error: preview.error ?? activate.error,
    preview,
    profile,
    rerunPreview,
    scope,
    setBindPrimaryChannel,
    setProfile,
    setScope,
  };
}

export type { UseBundleActivationDialogOptions };
