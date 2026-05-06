import { Loader2, Users } from "lucide-react";
import type { FormEvent } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Empty,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  Pill,
  Section,
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
        className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-[30rem]"
        data-testid="network-create-channel-dialog"
      >
        <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Create channel</DialogTitle>
          <DialogDescription>
            {workspaceName
              ? `Spawn one new session per selected agent inside ${workspaceName}.`
              : "Create a materialized network channel by spawning one new session per selected agent."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-5 px-5 py-5">
            <Field>
              <FieldLabel htmlFor="network-channel-name">Channel name</FieldLabel>
              <FieldDescription>
                Dot-notation encouraged — e.g. coord.core, ops.alerts.
              </FieldDescription>
              <Input
                className="h-10 font-mono"
                data-testid="network-channel-name-input"
                id="network-channel-name"
                onChange={event => onChannelNameChange(event.target.value)}
                placeholder="e.g. deployments"
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
                className="min-h-24 border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-3 py-3 text-[13px] leading-6"
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
                <p className="mt-2 text-[12px] leading-relaxed text-[color:var(--color-warning)]">
                  Select an active workspace before creating a channel.
                </p>
              ) : null}
            </Section>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-3">
            <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="network-create-channel-submit"
              disabled={!canSubmit || isSubmitting}
              type="submit"
            >
              {isSubmitting ? <Loader2 aria-hidden="true" className="size-4 animate-spin" /> : null}
              Create channel
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
