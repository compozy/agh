import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import {
  BlockLoading,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldGroup,
  FieldLabel,
  Input,
  MonoId,
  NativeSelect,
  NativeSelectOption,
  Time,
} from "@agh/ui";

import type {
  TaskBridgeNotificationSubscription,
  TaskBridgeNotificationSubscriptionCreateRequest,
} from "../types";

export interface TaskBridgeSubscriptionsPaneProps {
  subscriptions: TaskBridgeNotificationSubscription[];
  isLoading?: boolean;
  errorMessage?: string | null;
  isCreatePending?: boolean;
  isDeletePending?: boolean;
  onCreate: (request: TaskBridgeNotificationSubscriptionCreateRequest) => Promise<void> | void;
  onDelete: (subscriptionId: string) => Promise<void> | void;
}

const EMPTY_FORM = {
  bridge_instance_id: "",
  delivery_mode: "direct-send" as const,
  scope: "workspace" as const,
  workspace_id: "",
  peer_id: "",
  group_id: "",
  thread_id: "",
  subscription_id: "",
};

type FormState = typeof EMPTY_FORM;

function toRequest(form: FormState): TaskBridgeNotificationSubscriptionCreateRequest {
  return {
    bridge_instance_id: form.bridge_instance_id.trim(),
    delivery_mode: form.delivery_mode,
    scope: form.scope,
    workspace_id: form.workspace_id.trim() || undefined,
    peer_id: form.peer_id.trim() || undefined,
    group_id: form.group_id.trim() || undefined,
    thread_id: form.thread_id.trim() || undefined,
    subscription_id: form.subscription_id.trim() || undefined,
  };
}

function SubscriptionRow({
  subscription,
  isDeletePending,
  onDelete,
}: {
  subscription: TaskBridgeNotificationSubscription;
  isDeletePending: boolean;
  onDelete: (subscriptionId: string) => Promise<void> | void;
}) {
  return (
    <div
      className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3 rounded-md border border-line-soft bg-input-fill px-3.5 py-3"
      data-testid={`tasks-bridges-row-${subscription.subscription_id}`}
    >
      <div className="min-w-0">
        <MonoId value={subscription.subscription_id} />
        <p className="mt-1 text-form-label leading-relaxed text-muted">
          Delivers task events to bridge{" "}
          <span className="font-mono text-eyebrow text-fg">{subscription.bridge_instance_id}</span>{" "}
          · scope {subscription.scope} · mode {subscription.delivery_mode}
        </p>
        <p className="mt-1 font-mono text-micro text-subtle">
          cursor seq {subscription.cursor.last_sequence}
          {subscription.cursor.last_delivered_at ? (
            <>
              {" "}
              · delivered <Time iso={subscription.cursor.last_delivered_at} mode="relative" />
            </>
          ) : null}
          {subscription.cursor.last_error
            ? ` · last error: ${subscription.cursor.last_error}`
            : null}
        </p>
      </div>
      <Button
        aria-label={`Remove subscription ${subscription.subscription_id}`}
        data-testid={`tasks-bridges-delete-${subscription.subscription_id}`}
        disabled={isDeletePending}
        onClick={() => void onDelete(subscription.subscription_id)}
        size="icon-sm"
        type="button"
        variant="ghost"
      >
        <Trash2 aria-hidden="true" className="size-3.5" />
      </Button>
    </div>
  );
}

/**
 * Inspect drawer › Bridges pane (§4.8): operator-facing bridge notification
 * subscriptions. All eight create fields stay — this is operator turf.
 */
export function TaskBridgeSubscriptionsPane({
  subscriptions,
  isLoading = false,
  errorMessage = null,
  isCreatePending = false,
  isDeletePending = false,
  onCreate,
  onDelete,
}: TaskBridgeSubscriptionsPaneProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const canSubmit = form.bridge_instance_id.trim() !== "" && !isCreatePending;

  const submit = async () => {
    if (!canSubmit) return;
    await onCreate(toRequest(form));
    setForm(EMPTY_FORM);
    setCreateOpen(false);
  };

  if (isLoading && subscriptions.length === 0) {
    return <BlockLoading label="Loading subscriptions" size="sm" surface="bare" />;
  }

  return (
    <div className="flex flex-col gap-3" data-testid="tasks-bridges-pane">
      {errorMessage && subscriptions.length === 0 ? (
        <p className="text-small-body text-danger">{errorMessage}</p>
      ) : null}
      {subscriptions.length === 0 && !errorMessage ? (
        <p className="text-small-body text-muted">
          No bridge subscriptions. Task events stay inside the runtime until a bridge subscribes.
        </p>
      ) : (
        subscriptions.map(subscription => (
          <SubscriptionRow
            isDeletePending={isDeletePending}
            key={subscription.subscription_id}
            onDelete={onDelete}
            subscription={subscription}
          />
        ))
      )}

      <div>
        <Button
          data-testid="tasks-bridges-add"
          onClick={() => setCreateOpen(true)}
          size="sm"
          type="button"
          variant="neutral"
        >
          <Plus aria-hidden="true" className="size-3" />
          Add subscription
        </Button>
      </div>

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-md" data-testid="tasks-bridges-create-dialog">
          <DialogHeader>
            <DialogTitle>Add bridge subscription</DialogTitle>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="tasks-bridges-instance">Bridge instance id</FieldLabel>
              <Input
                data-testid="tasks-bridges-instance"
                id="tasks-bridges-instance"
                onChange={event =>
                  setForm(prev => ({ ...prev, bridge_instance_id: event.target.value }))
                }
                placeholder="bridge_9f2c1e"
                value={form.bridge_instance_id}
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="tasks-bridges-mode">Delivery mode</FieldLabel>
                <NativeSelect
                  id="tasks-bridges-mode"
                  onChange={event =>
                    setForm(prev => ({
                      ...prev,
                      delivery_mode: event.target.value as FormState["delivery_mode"],
                    }))
                  }
                  value={form.delivery_mode}
                >
                  <NativeSelectOption value="direct-send">direct-send</NativeSelectOption>
                  <NativeSelectOption value="reply">reply</NativeSelectOption>
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="tasks-bridges-scope">Scope</FieldLabel>
                <NativeSelect
                  id="tasks-bridges-scope"
                  onChange={event =>
                    setForm(prev => ({ ...prev, scope: event.target.value as FormState["scope"] }))
                  }
                  value={form.scope}
                >
                  <NativeSelectOption value="workspace">workspace</NativeSelectOption>
                  <NativeSelectOption value="global">global</NativeSelectOption>
                </NativeSelect>
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="tasks-bridges-workspace">Workspace id (optional)</FieldLabel>
                <Input
                  id="tasks-bridges-workspace"
                  onChange={event =>
                    setForm(prev => ({ ...prev, workspace_id: event.target.value }))
                  }
                  value={form.workspace_id}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="tasks-bridges-peer">Peer id (optional)</FieldLabel>
                <Input
                  id="tasks-bridges-peer"
                  onChange={event => setForm(prev => ({ ...prev, peer_id: event.target.value }))}
                  value={form.peer_id}
                />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="tasks-bridges-group">Group id (optional)</FieldLabel>
                <Input
                  id="tasks-bridges-group"
                  onChange={event => setForm(prev => ({ ...prev, group_id: event.target.value }))}
                  value={form.group_id}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="tasks-bridges-thread">Thread id (optional)</FieldLabel>
                <Input
                  id="tasks-bridges-thread"
                  onChange={event => setForm(prev => ({ ...prev, thread_id: event.target.value }))}
                  value={form.thread_id}
                />
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="tasks-bridges-subscription">
                Subscription id (optional)
              </FieldLabel>
              <Input
                id="tasks-bridges-subscription"
                onChange={event =>
                  setForm(prev => ({ ...prev, subscription_id: event.target.value }))
                }
                placeholder="Generated when empty"
                value={form.subscription_id}
              />
            </Field>
          </FieldGroup>
          <DialogFooter className="gap-2">
            <Button
              disabled={isCreatePending}
              onClick={() => setCreateOpen(false)}
              size="sm"
              type="button"
              variant="neutral"
            >
              Cancel
            </Button>
            <Button
              data-testid="tasks-bridges-create-submit"
              disabled={!canSubmit}
              onClick={() => void submit()}
              size="sm"
              type="button"
            >
              Add subscription
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
