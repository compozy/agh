import { Users } from "lucide-react";
import type { FormEvent } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Empty,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  Pill,
  Section,
  Spinner,
  Textarea,
} from "@agh/ui";
import { AgentCommandMultiSelect, type AgentPayload } from "@/systems/agent";

import type { NetworkCreateChannelDraft } from "../types";

interface NetworkCreateChannelDialogProps {
  agents: AgentPayload[];
  canSubmit: boolean;
  draft: NetworkCreateChannelDraft;
  isSubmitting: boolean;
  onChannelNameChange: (value: string) => void;
  onOpenChange: (open: boolean) => void;
  onPurposeChange: (value: string) => void;
  onAgentSelectionChange: (agentNames: string[]) => void;
  onSubmit: () => void;
  open: boolean;
  workspaceName?: string | null;
}

export function NetworkCreateChannelDialog({
  agents,
  canSubmit,
  draft,
  isSubmitting,
  onChannelNameChange,
  onOpenChange,
  onPurposeChange,
  onAgentSelectionChange,
  onSubmit,
  open,
  workspaceName,
}: NetworkCreateChannelDialogProps) {
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canSubmit || isSubmitting) return;
    onSubmit();
  };

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 text-fg sm:max-w-120"
        data-testid="network-create-channel-dialog"
      >
        <DialogHeader className="border-b border-line px-5 py-4">
          <DialogTitle>Create channel</DialogTitle>
          <DialogDescription>
            {workspaceName
              ? `Spawn one new session per selected agent inside ${workspaceName}.`
              : "Create a materialized network channel by spawning one new session per selected agent."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-5 p-5">
            <Field>
              <FieldLabel htmlFor="network-channel-name">Channel name</FieldLabel>
              <FieldDescription>
                Use lowercase letters, numbers, underscores, or hyphens; e.g. coord_core.
              </FieldDescription>
              <Input
                className="h-10 font-mono"
                data-testid="network-channel-name-input"
                id="network-channel-name"
                onChange={event => onChannelNameChange(event.target.value)}
                placeholder="e.g. website_copy"
                value={draft.channelName}
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="network-channel-purpose">Purpose</FieldLabel>
              <FieldDescription>
                Required. This becomes the room intro and the right-rail about summary.
              </FieldDescription>
              <Textarea
                aria-required="true"
                className="min-h-24 border-line bg-canvas-soft p-3 text-small-body leading-6"
                data-testid="network-channel-purpose-input"
                id="network-channel-purpose"
                onChange={event => onPurposeChange(event.target.value)}
                placeholder="e.g. Coordinate release handoffs across coder and reviewer agents."
                required
                value={draft.purpose}
              />
            </Field>

            <Section
              label="Add agents"
              right={
                <Pill mono data-testid="network-selected-agents-count">
                  {draft.selectedAgentNames.length} selected
                </Pill>
              }
            >
              {agents.length === 0 ? (
                <Empty
                  icon={Users}
                  title="No agents available"
                  description="This workspace has no local agents to invite."
                />
              ) : (
                <AgentCommandMultiSelect
                  agents={agents}
                  value={draft.selectedAgentNames}
                  onToggle={onAgentSelectionChange}
                  triggerTestId="network-create-channel-agent-trigger"
                  placeholder="Select agents"
                  itemTestId={agent => `network-agent-option-${agent.name}`}
                />
              )}

              {!workspaceName ? (
                <p className="mt-2 text-xs leading-relaxed text-warning">
                  Select an active workspace before creating a channel.
                </p>
              ) : null}
            </Section>
          </div>

          <DialogFooter className="border-t border-line bg-canvas-soft px-5 py-3">
            <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="network-create-channel-submit"
              disabled={!canSubmit || isSubmitting}
              type="submit"
            >
              {isSubmitting ? <Spinner aria-hidden="true" className="size-4" /> : null}
              Create channel
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
