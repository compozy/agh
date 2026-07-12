import { Settings2, Waypoints } from "lucide-react";

import {
  Field,
  FieldContent,
  FieldDescription,
  FieldTitle,
  FormSection,
  Input,
  NativeSelect,
  NativeSelectOption,
  Switch,
} from "@agh/ui";

import { describeBridgeRoutingPolicy } from "../lib/bridge-formatters";
import type { BridgeCreateDraft } from "../types";

interface DeliveryStepProps {
  draft: BridgeCreateDraft;
  onDraftChange: (draft: BridgeCreateDraft) => void;
}

export function DeliveryStep({ draft, onDraftChange }: DeliveryStepProps) {
  return (
    <>
      <FormSection
        data-testid="bridge-wizard-section-routing"
        description={describeBridgeRoutingPolicy(draft.routingPolicy)}
        icon={Waypoints}
        title="Routing policy"
      >
        <Field orientation="horizontal">
          <Switch
            checked={draft.routingPolicy.include_peer}
            data-testid="bridge-routing-include-peer"
            onCheckedChange={checked =>
              onDraftChange({
                ...draft,
                routingPolicy: { ...draft.routingPolicy, include_peer: checked },
              })
            }
          />
          <FieldContent>
            <FieldTitle>Include peer</FieldTitle>
            <FieldDescription>Differentiate direct targets by peer identifier.</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Switch
            checked={draft.routingPolicy.include_group}
            data-testid="bridge-routing-include-group"
            onCheckedChange={checked =>
              onDraftChange({
                ...draft,
                routingPolicy: { ...draft.routingPolicy, include_group: checked },
              })
            }
          />
          <FieldContent>
            <FieldTitle>Include group</FieldTitle>
            <FieldDescription>
              Keep routes isolated per group or channel when the platform supports it.
            </FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Switch
            checked={draft.routingPolicy.include_thread}
            data-testid="bridge-routing-include-thread"
            onCheckedChange={checked =>
              onDraftChange({
                ...draft,
                routingPolicy: { ...draft.routingPolicy, include_thread: checked },
              })
            }
          />
          <FieldContent>
            <FieldTitle>Include thread</FieldTitle>
            <FieldDescription>
              Use thread identity as an additional routing dimension.
            </FieldDescription>
          </FieldContent>
        </Field>
      </FormSection>

      <FormSection
        data-testid="bridge-wizard-section-delivery"
        description="These defaults are applied when resolving outbound delivery targets."
        icon={Settings2}
        title="Delivery defaults"
      >
        <div className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldContent>
              <FieldTitle>Mode</FieldTitle>
            </FieldContent>
            <NativeSelect
              data-testid="bridge-delivery-mode-select"
              onChange={event =>
                onDraftChange({
                  ...draft,
                  deliveryDefaults: {
                    ...draft.deliveryDefaults,
                    mode:
                      event.target.value === ""
                        ? undefined
                        : (event.target.value as NonNullable<
                            BridgeCreateDraft["deliveryDefaults"]["mode"]
                          >),
                  },
                })
              }
              value={draft.deliveryDefaults.mode ?? ""}
            >
              <NativeSelectOption value="">Use runtime default</NativeSelectOption>
              <NativeSelectOption value="reply">Reply</NativeSelectOption>
              <NativeSelectOption value="direct-send">Direct send</NativeSelectOption>
            </NativeSelect>
          </Field>
          <Field>
            <FieldContent>
              <FieldTitle>Peer ID</FieldTitle>
            </FieldContent>
            <Input
              data-testid="bridge-delivery-peer-input"
              onChange={event =>
                onDraftChange({
                  ...draft,
                  deliveryDefaults: { ...draft.deliveryDefaults, peer_id: event.target.value },
                })
              }
              placeholder="peer_123"
              value={draft.deliveryDefaults.peer_id ?? ""}
            />
          </Field>
          <Field>
            <FieldContent>
              <FieldTitle>Thread ID</FieldTitle>
            </FieldContent>
            <Input
              data-testid="bridge-delivery-thread-input"
              onChange={event =>
                onDraftChange({
                  ...draft,
                  deliveryDefaults: { ...draft.deliveryDefaults, thread_id: event.target.value },
                })
              }
              placeholder="thread_456"
              value={draft.deliveryDefaults.thread_id ?? ""}
            />
          </Field>
          <Field>
            <FieldContent>
              <FieldTitle>Group ID</FieldTitle>
            </FieldContent>
            <Input
              data-testid="bridge-delivery-group-input"
              onChange={event =>
                onDraftChange({
                  ...draft,
                  deliveryDefaults: { ...draft.deliveryDefaults, group_id: event.target.value },
                })
              }
              placeholder="group_789"
              value={draft.deliveryDefaults.group_id ?? ""}
            />
          </Field>
        </div>
      </FormSection>
    </>
  );
}
