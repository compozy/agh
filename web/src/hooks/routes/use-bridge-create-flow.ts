import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import {
  buildBridgeCreateRequest,
  createBridgeCreateDraft,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
  useCreateBridge,
  useSlackBridgeManifest,
  type BridgeCreateDraft,
  type BridgeProvider,
} from "@/systems/bridges";
import { getBridgeSetupProfile } from "@/systems/bridges/lib/bridge-setup";

interface BridgeCreateFlowInput {
  activeWorkspaceId?: string | null;
  activeWorkspaceName?: string | null;
  providers: BridgeProvider[];
}

interface PersistedBridge {
  displayName: string;
  id: string;
}

interface CommittedBridge extends PersistedBridge {
  manifest: "slack";
}

export function useBridgeCreateFlow({
  activeWorkspaceId,
  activeWorkspaceName,
  providers,
}: BridgeCreateFlowInput) {
  const navigate = useNavigate({ from: "/bridges" });
  const [isOpen, setOpen] = useState(false);
  const [draft, setDraft] = useState<BridgeCreateDraft>(() =>
    createBridgeCreateDraft([], activeWorkspaceId)
  );
  const [committedBridge, setCommittedBridge] = useState<CommittedBridge | null>(null);
  const createMutation = useCreateBridge();

  const selectedProvider = findBridgeProviderByKey(providers, draft.selectedProviderKey);
  const supportsManifest =
    committedBridge?.manifest === "slack" ||
    (selectedProvider !== undefined &&
      getBridgeSetupProfile(selectedProvider.platform)?.manifest === "slack");
  const manifestQuery = useSlackBridgeManifest(committedBridge?.id ?? "", {
    enabled: isOpen && committedBridge !== null,
  });

  const navigateToBridge = (bridge: PersistedBridge) => {
    setOpen(false);
    setCommittedBridge(null);
    void navigate({ params: { id: bridge.id }, to: "/bridges/$id" });
  };

  const openCreateDialog = () => {
    setCommittedBridge(null);
    setDraft(createBridgeCreateDraft(providers, activeWorkspaceId));
    setOpen(true);
  };

  const handleOpenChange = (open: boolean) => {
    if (!open && committedBridge) {
      navigateToBridge(committedBridge);
      return;
    }
    setOpen(open);
  };

  const submit = async () => {
    const provider = findBridgeProviderByKey(providers, draft.selectedProviderKey);
    if (!provider || !isBridgeProviderSelectable(provider)) {
      toast.error("Select an available bridge provider before creating the bridge.");
      return;
    }

    if (draft.scope === "workspace" && !activeWorkspaceId) {
      toast.error("Select an active workspace before creating a workspace-scoped bridge.");
      return;
    }

    const requestResult = buildBridgeCreateRequest(draft, provider, activeWorkspaceId);
    if (!requestResult.ok) {
      toast.error(requestResult.error);
      return;
    }

    try {
      const result = await createMutation.mutateAsync(requestResult.data);
      const committed = {
        displayName: result.bridge.display_name,
        id: result.bridge.id,
      };
      if (getBridgeSetupProfile(provider.platform)?.manifest === "slack") {
        setCommittedBridge({ ...committed, manifest: "slack" });
        toast.success(`Created bridge ${committed.displayName}. Continue with its Slack manifest.`);
        return;
      }

      toast.success(`Created bridge ${committed.displayName}.`);
      navigateToBridge(committed);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create bridge");
    }
  };

  return {
    canCreateBridge: providers.some(isBridgeProviderSelectable),
    createDialogProps: {
      activeWorkspaceId,
      activeWorkspaceName,
      draft,
      isPending: createMutation.isPending,
      manifestState: committedBridge
        ? {
            bridgeId: committedBridge.id,
            error: manifestQuery.error,
            isLoading: manifestQuery.isLoading,
            manifestJSON: manifestQuery.data
              ? JSON.stringify(manifestQuery.data.manifest, null, 2)
              : undefined,
            onOpenBridge: () => navigateToBridge(committedBridge),
            onRetry: () => void manifestQuery.refetch(),
          }
        : null,
      onDraftChange: setDraft,
      onOpenChange: handleOpenChange,
      onSubmit: submit,
      open: isOpen,
      providers,
      supportsManifest,
    },
    openCreateDialog,
  };
}
