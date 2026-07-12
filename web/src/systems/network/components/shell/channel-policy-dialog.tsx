import { useState, type FormEvent } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
} from "@agh/ui";

import type { ChannelMember } from "../../hooks/use-channel-members";
import type {
  NetworkChannel,
  NetworkChannelSummary,
  NetworkChannelUpdateRequest,
  NetworkFanoutPolicy,
} from "../../types";

const NO_COORDINATOR_VALUE = "__none__";

const FANOUT_POLICIES: ReadonlyArray<{
  value: NetworkFanoutPolicy;
  label: string;
  description: string;
}> = [
  {
    value: "capability_match",
    label: "Capability match",
    description: "Activate the smallest matching peer set for unaddressed thread traffic.",
  },
  {
    value: "coordinator",
    label: "Coordinator",
    description: "Route unaddressed thread traffic to one named coordinator peer.",
  },
  {
    value: "all_members",
    label: "All members",
    description: "Deliver unaddressed thread traffic to every eligible member.",
  },
];

interface ChannelPolicyDialogProps {
  channel: NetworkChannelSummary;
  detail: NetworkChannel | null;
  members: ReadonlyArray<ChannelMember>;
  isSubmitting: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: NetworkChannelUpdateRequest) => Promise<void>;
  open: boolean;
}

interface ChannelPolicyDialogFormState {
  coordinatorPeerId: string;
  policy: NetworkFanoutPolicy;
  purpose: string;
  submitError: string | null;
}

function normalizeFanoutPolicy(value?: string | null): NetworkFanoutPolicy {
  if (value === "coordinator" || value === "all_members") {
    return value;
  }
  return "capability_match";
}

function currentCoordinator(channel: NetworkChannelSummary, detail: NetworkChannel | null): string {
  return detail?.coordinator_peer_id ?? channel.coordinator_peer_id ?? "";
}

function currentPurpose(channel: NetworkChannelSummary, detail: NetworkChannel | null): string {
  return detail?.purpose ?? channel.purpose ?? "";
}

function initialFormState(
  channel: NetworkChannelSummary,
  detail: NetworkChannel | null
): ChannelPolicyDialogFormState {
  return {
    coordinatorPeerId: currentCoordinator(channel, detail),
    policy: normalizeFanoutPolicy(detail?.fanout_policy ?? channel.fanout_policy),
    purpose: currentPurpose(channel, detail),
    submitError: null,
  };
}

function channelPolicyFormKey(
  channel: NetworkChannelSummary,
  detail: NetworkChannel | null,
  open: boolean
): string {
  return JSON.stringify([
    open ? "open" : "closed",
    channel.channel,
    currentPurpose(channel, detail),
    normalizeFanoutPolicy(detail?.fanout_policy ?? channel.fanout_policy),
    currentCoordinator(channel, detail),
  ]);
}

function ChannelPolicyDialogForm({
  channel,
  detail,
  members,
  isSubmitting,
  onOpenChange,
  onSubmit,
}: Omit<ChannelPolicyDialogProps, "open">) {
  const [formState, setFormState] = useState(() => initialFormState(channel, detail));
  const selectedPolicy =
    FANOUT_POLICIES.find(candidate => candidate.value === formState.policy) ?? FANOUT_POLICIES[0];

  const setPurpose = (purpose: string) => {
    setFormState(current => ({ ...current, purpose }));
  };
  const setPolicy = (policy: NetworkFanoutPolicy) => {
    setFormState(current => ({
      ...current,
      coordinatorPeerId: policy === "coordinator" ? current.coordinatorPeerId : "",
      policy,
    }));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormState(current => ({ ...current, submitError: null }));
    const payload: NetworkChannelUpdateRequest = {
      purpose: formState.purpose.trim(),
      fanout_policy: formState.policy,
      coordinator_peer_id: formState.coordinatorPeerId.trim(),
    };
    try {
      await onSubmit(payload);
      onOpenChange(false);
    } catch (error) {
      const submitError = error instanceof Error ? error.message : "Couldn't save delivery policy.";
      setFormState(current => ({ ...current, submitError }));
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="space-y-5 p-5">
        <Field>
          <FieldLabel htmlFor="network-channel-policy-purpose">Purpose</FieldLabel>
          <FieldDescription>
            Used by agents to decide whether this room is relevant.
          </FieldDescription>
          <Input
            className="font-mono"
            data-testid="channel-policy-purpose"
            id="network-channel-policy-purpose"
            onChange={event => setPurpose(event.target.value)}
            value={formState.purpose}
          />
        </Field>

        <Field>
          <FieldLabel>Fanout policy</FieldLabel>
          <FieldDescription>{selectedPolicy?.description}</FieldDescription>
          <Select
            onValueChange={value => {
              const nextPolicy = normalizeFanoutPolicy(value);
              setPolicy(nextPolicy);
            }}
            value={formState.policy}
          >
            <SelectTrigger data-testid="channel-policy-fanout">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {FANOUT_POLICIES.map(option => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>

        <Field>
          <FieldLabel>Coordinator peer</FieldLabel>
          <FieldDescription>
            Required for coordinator policy; leave empty for other policies.
          </FieldDescription>
          <Select
            onValueChange={value =>
              setFormState(current => ({
                ...current,
                coordinatorPeerId: value === NO_COORDINATOR_VALUE ? "" : (value ?? ""),
              }))
            }
            value={formState.coordinatorPeerId || NO_COORDINATOR_VALUE}
          >
            <SelectTrigger data-testid="channel-policy-coordinator">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={NO_COORDINATOR_VALUE}>No coordinator</SelectItem>
              {members.map(member => (
                <SelectItem key={member.peerId} value={member.peerId}>
                  {member.displayName || member.peerId}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        {formState.submitError ? (
          <p className="text-xs text-danger" role="alert">
            {formState.submitError}
          </p>
        ) : null}
      </div>

      <DialogFooter className="border-t border-line bg-canvas-soft px-5 py-3">
        <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
          Cancel
        </Button>
        <Button data-testid="channel-policy-submit" disabled={isSubmitting} type="submit">
          {isSubmitting ? <Spinner aria-hidden="true" className="size-4" /> : null}
          Save policy
        </Button>
      </DialogFooter>
    </form>
  );
}

export function ChannelPolicyDialog({
  channel,
  detail,
  members,
  isSubmitting,
  onOpenChange,
  onSubmit,
  open,
}: ChannelPolicyDialogProps) {
  const formKey = channelPolicyFormKey(channel, detail, open);

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="gap-0 p-0 text-fg sm:max-w-120" data-testid="channel-policy-dialog">
        <DialogHeader className="border-b border-line px-5 py-4">
          <DialogTitle>Delivery policy</DialogTitle>
          <DialogDescription>
            Control how unaddressed thread traffic fans out in #{channel.channel}.
          </DialogDescription>
        </DialogHeader>

        <ChannelPolicyDialogForm
          channel={channel}
          detail={detail}
          isSubmitting={isSubmitting}
          key={formKey}
          members={members}
          onOpenChange={onOpenChange}
          onSubmit={onSubmit}
        />
      </DialogContent>
    </Dialog>
  );
}
