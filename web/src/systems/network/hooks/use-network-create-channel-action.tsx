import { useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { useAgents } from "@/systems/agent";
import { useActiveWorkspace } from "@/systems/workspace";
import { Button, toast } from "@agh/ui";

import { NetworkCreateChannelDialog } from "../components/network-create-channel-dialog";
import { createNetworkChannelDraft, sortAgentsForNetwork } from "../lib/network-formatters";
import { useCreateNetworkChannel } from "./use-network-actions";

export function useNetworkCreateChannelAction({ enabled }: { enabled: boolean }) {
  const navigate = useNavigate();
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();
  const agentsQuery = useAgents(activeWorkspaceId);
  const createChannel = useCreateNetworkChannel();
  const [createOpen, setCreateOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState(createNetworkChannelDraft);
  const sortedAgents = useMemo(
    () => sortAgentsForNetwork(agentsQuery.data ?? []),
    [agentsQuery.data]
  );
  const canCreateChannel =
    enabled &&
    activeWorkspaceId != null &&
    createDraft.channelName.trim() !== "" &&
    createDraft.purpose.trim() !== "" &&
    createDraft.selectedAgentNames.length > 0;

  const openNetworkSettings = () => {
    void navigate({ to: "/settings/network" });
  };

  const handleCreateChannel = async () => {
    if (!enabled || !activeWorkspaceId || !canCreateChannel) {
      return;
    }

    try {
      const response = await createChannel.mutateAsync({
        agent_names: createDraft.selectedAgentNames,
        channel: createDraft.channelName.trim(),
        purpose: createDraft.purpose.trim(),
        workspace_id: activeWorkspaceId,
      });
      const channel = response.channel.channel;
      setCreateDraft(createNetworkChannelDraft());
      setCreateOpen(false);
      void navigate({
        params: { workspaceId: activeWorkspaceId, channel },
        to: "/network/$workspaceId/$channel/threads",
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to create network channel";
      toast.error(message);
    }
  };

  const action = enabled ? (
    <Button
      data-testid="network-open-create-dialog"
      disabled={agentsQuery.isLoading || activeWorkspaceId == null}
      onClick={() => setCreateOpen(true)}
      size="sm"
      type="button"
      variant="outline"
    >
      <Plus aria-hidden="true" className="size-3" />
      Channel
    </Button>
  ) : null;
  const dialog = (
    <NetworkCreateChannelDialog
      agents={sortedAgents}
      canSubmit={canCreateChannel}
      draft={createDraft}
      isSubmitting={createChannel.isPending}
      onAgentSelectionChange={selectedAgentNames =>
        setCreateDraft(current => ({ ...current, selectedAgentNames }))
      }
      onChannelNameChange={channelName => setCreateDraft(current => ({ ...current, channelName }))}
      onOpenChange={setCreateOpen}
      onPurposeChange={purpose => setCreateDraft(current => ({ ...current, purpose }))}
      onSubmit={handleCreateChannel}
      open={createOpen}
      workspaceName={activeWorkspace?.name}
    />
  );

  return { action, dialog, openNetworkSettings };
}
