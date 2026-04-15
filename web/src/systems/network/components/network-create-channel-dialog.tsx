import { Check, Loader2 } from "lucide-react";
import type { FormEvent } from "react";

import { Button, Input } from "@agh/ui";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { AgentIcon, type AgentPayload } from "@/systems/agent";

import type { NetworkCreateChannelDraft } from "../types";

interface NetworkCreateChannelDialogProps {
  agents: AgentPayload[];
  canSubmit: boolean;
  draft: NetworkCreateChannelDraft;
  isSubmitting: boolean;
  onChannelNameChange: (value: string) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  onToggleAgent: (agentName: string) => void;
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
  onSubmit,
  onToggleAgent,
  open,
  workspaceName,
}: NetworkCreateChannelDialogProps) {
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSubmit();
  };

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="max-w-[28rem] gap-0 border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-0 text-[color:var(--color-text-primary)] ring-0"
        data-testid="network-create-channel-dialog"
      >
        <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Create Channel</DialogTitle>
          <DialogDescription className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
            {workspaceName
              ? `Spawn one new session per selected agent inside ${workspaceName}.`
              : "Create a materialized network channel by spawning one new session per selected agent."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-5 px-5 py-4">
            <div className="space-y-2">
              <label
                className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
                htmlFor="network-channel-name"
              >
                Channel name
              </label>
              <Input
                className="h-9 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                data-testid="network-channel-name-input"
                id="network-channel-name"
                onChange={event => onChannelNameChange(event.target.value)}
                placeholder="e.g., deployments"
                value={draft.channelName}
              />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <p className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                  Add agents
                </p>
                <span className="text-xs text-[color:var(--color-text-tertiary)]">
                  {draft.selectedAgentNames.length} selected
                </span>
              </div>

              <div className="overflow-hidden rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]">
                {agents.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm text-[color:var(--color-text-secondary)]">
                    No local agents available in this workspace.
                  </div>
                ) : (
                  agents.map(agent => {
                    const isSelected = draft.selectedAgentNames.includes(agent.name);

                    return (
                      <button
                        aria-pressed={isSelected}
                        className={cn(
                          "flex w-full items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors last:border-b-0",
                          "hover:bg-[color:var(--color-surface)]",
                          isSelected && "bg-[color:var(--color-surface)]"
                        )}
                        data-testid={`network-agent-option-${agent.name}`}
                        key={agent.name}
                        onClick={() => onToggleAgent(agent.name)}
                        type="button"
                      >
                        <span
                          className={cn(
                            "flex size-4 shrink-0 items-center justify-center rounded border",
                            isSelected
                              ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]"
                              : "border-[color:var(--color-divider)] bg-transparent text-transparent"
                          )}
                        >
                          <Check className="size-3" />
                        </span>
                        <AgentIcon
                          className="size-4 text-[color:var(--color-text-tertiary)]"
                          provider={agent.provider}
                        />
                        <span className="min-w-0 flex-1 truncate text-sm text-[color:var(--color-text-primary)]">
                          {agent.name}
                        </span>
                        <span className="shrink-0 font-mono text-[0.64rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
                          {agent.provider}
                        </span>
                      </button>
                    );
                  })
                )}
              </div>

              {!workspaceName ? (
                <p className="text-xs leading-relaxed text-[color:var(--color-warning)]">
                  Select an active workspace before creating a channel.
                </p>
              ) : null}
            </div>
          </div>

          <DialogFooter className="border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]">
            <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="network-create-channel-submit"
              disabled={!canSubmit || isSubmitting}
              type="submit"
            >
              {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
              Create Channel
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
