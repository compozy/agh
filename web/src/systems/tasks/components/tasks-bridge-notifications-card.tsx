import { useState } from "react";
import { AlertCircle, Loader2, Plus, Radio, Trash2, TriangleAlert } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Empty,
  Input,
  Label,
  Pill,
  Section,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

import { formatRelativeTime } from "../lib/task-formatters";
import type {
  TaskBridgeNotificationDeliveryMode,
  TaskBridgeNotificationSubscription,
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskBridgeNotificationSubscriptionScope,
} from "../types";

export interface TasksBridgeNotificationsCardProps {
  subscriptions: TaskBridgeNotificationSubscription[];
  isLoading?: boolean;
  errorMessage?: string | null;
  isCreatePending?: boolean;
  isDeletePending?: boolean;
  onCreate: (data: TaskBridgeNotificationSubscriptionCreateRequest) => Promise<void>;
  onDelete: (subscriptionId: string) => Promise<void>;
}

interface CreateFormState {
  bridgeInstanceId: string;
  scope: TaskBridgeNotificationSubscriptionScope;
  deliveryMode: TaskBridgeNotificationDeliveryMode;
  workspaceId: string;
  peerId: string;
  groupId: string;
  threadId: string;
  subscriptionId: string;
}

const EMPTY_FORM: CreateFormState = {
  bridgeInstanceId: "",
  scope: "workspace",
  deliveryMode: "direct-send",
  workspaceId: "",
  peerId: "",
  groupId: "",
  threadId: "",
  subscriptionId: "",
};

function formatLastSequence(value: number | undefined | null): string {
  if (typeof value !== "number") {
    return "0";
  }
  return String(value);
}

function trimOrUndefined(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function TasksBridgeNotificationsCard({
  subscriptions,
  isLoading = false,
  errorMessage = null,
  isCreatePending = false,
  isDeletePending = false,
  onCreate,
  onDelete,
}: TasksBridgeNotificationsCardProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<CreateFormState>(EMPTY_FORM);
  const [createError, setCreateError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<TaskBridgeNotificationSubscription | null>(null);

  const handleCreateOpenChange = (open: boolean) => {
    setCreateOpen(open);
    if (!open) {
      setForm(EMPTY_FORM);
      setCreateError(null);
    }
  };

  const handleCreateSubmit = async () => {
    setCreateError(null);
    if (form.bridgeInstanceId.trim() === "") {
      setCreateError("Bridge instance id is required.");
      return;
    }
    if (form.scope === "workspace" && form.workspaceId.trim() === "") {
      setCreateError("Workspace id is required for workspace-scoped subscriptions.");
      return;
    }
    const payload: TaskBridgeNotificationSubscriptionCreateRequest = {
      bridge_instance_id: form.bridgeInstanceId.trim(),
      scope: form.scope,
      delivery_mode: form.deliveryMode,
      workspace_id: trimOrUndefined(form.workspaceId),
      peer_id: trimOrUndefined(form.peerId),
      group_id: trimOrUndefined(form.groupId),
      thread_id: trimOrUndefined(form.threadId),
      subscription_id: trimOrUndefined(form.subscriptionId),
    } as TaskBridgeNotificationSubscriptionCreateRequest;
    try {
      await onCreate(payload);
      handleCreateOpenChange(false);
    } catch {
      // toast surfaced by route hook; keep dialog open so the operator can retry.
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) {
      return;
    }
    try {
      await onDelete(deleteTarget.subscription_id);
      setDeleteTarget(null);
    } catch {
      // toast surfaced by route hook; keep dialog open so the operator can retry.
    }
  };

  return (
    <Section
      aria-label="Bridge notification subscriptions"
      className="w-full gap-4"
      data-testid="tasks-bridge-notifications-card"
      label="Bridge notifications"
      right={
        <Button
          data-testid="tasks-bridge-notifications-create-trigger"
          disabled={isCreatePending}
          onClick={() => handleCreateOpenChange(true)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Plus className="size-3.5" />
          New subscription
        </Button>
      }
    >
      <p
        className="text-[12px] text-[color:var(--color-text-tertiary)]"
        data-testid="tasks-bridge-notifications-disclaimer"
      >
        Cursor diagnostics are read-only delivery progress. Zero-state cursors render as no sequence
        and no delivery id; advancement happens only after a confirmed bridge delivery.
      </p>
      {isLoading && subscriptions.length === 0 ? (
        <div
          className="flex min-h-[120px] items-center justify-center"
          data-testid="tasks-bridge-notifications-loading"
        >
          <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
        </div>
      ) : null}
      {errorMessage && subscriptions.length === 0 ? (
        <Empty
          data-testid="tasks-bridge-notifications-error"
          description={errorMessage}
          icon={AlertCircle}
          title="Unable to load subscriptions"
        />
      ) : null}
      {!isLoading && !errorMessage && subscriptions.length === 0 ? (
        <Empty
          data-testid="tasks-bridge-notifications-empty"
          description="No bridge subscriptions deliver this task's terminal events. Create one to opt a bridge target into the task notification stream."
          icon={Radio}
          title="No subscriptions yet"
        />
      ) : null}
      {subscriptions.length > 0 ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Subscription</TableHead>
              <TableHead>Bridge target</TableHead>
              <TableHead>Cursor</TableHead>
              <TableHead>Last delivery</TableHead>
              <TableHead>Last error</TableHead>
              <TableHead className="w-8" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {subscriptions.map(sub => {
              const cursor = sub.cursor;
              const isZeroCursor =
                (cursor?.last_sequence ?? 0) === 0 &&
                (cursor?.last_delivery_id ?? "") === "" &&
                !cursor?.last_delivered_at;
              return (
                <TableRow
                  data-testid={`tasks-bridge-notifications-row-${sub.subscription_id}`}
                  key={sub.subscription_id}
                >
                  <TableCell className="max-w-[280px]">
                    <div className="flex min-w-0 flex-col gap-1">
                      <Pill mono>{sub.subscription_id}</Pill>
                      <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
                        <Pill tone="info">{sub.scope}</Pill>
                        <Pill tone="neutral">{sub.delivery_mode}</Pill>
                      </div>
                      <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
                        Created {formatRelativeTime(sub.created_at)}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1 text-[12px]">
                      <span className="font-mono text-[12px] text-[color:var(--color-text-primary)]">
                        bridge {sub.bridge_instance_id}
                      </span>
                      {sub.workspace_id ? (
                        <span className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                          workspace {sub.workspace_id}
                        </span>
                      ) : null}
                      {sub.peer_id ? (
                        <span className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                          peer {sub.peer_id}
                        </span>
                      ) : null}
                      {sub.group_id ? (
                        <span className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                          group {sub.group_id}
                        </span>
                      ) : null}
                      {sub.thread_id ? (
                        <span className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                          thread {sub.thread_id}
                        </span>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    {isZeroCursor ? (
                      <span
                        className="inline-flex items-center gap-1 font-mono text-[11px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]"
                        data-testid={`tasks-bridge-notifications-row-${sub.subscription_id}-cursor-zero`}
                      >
                        <Pill tone="neutral">zero state</Pill>
                        seq 0
                      </span>
                    ) : (
                      <div className="flex flex-col gap-0.5 font-mono text-[11px] text-[color:var(--color-text-primary)]">
                        <span
                          data-testid={`tasks-bridge-notifications-row-${sub.subscription_id}-cursor-seq`}
                        >
                          seq {formatLastSequence(cursor?.last_sequence)}
                        </span>
                        {cursor?.last_delivery_id ? (
                          <span className="text-[10px] text-[color:var(--color-text-tertiary)]">
                            delivery {cursor.last_delivery_id}
                          </span>
                        ) : null}
                      </div>
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                    {cursor?.last_delivered_at ? formatRelativeTime(cursor.last_delivered_at) : "—"}
                    {cursor?.updated_at ? (
                      <span className="block text-[10px]">
                        updated {formatRelativeTime(cursor.updated_at)}
                      </span>
                    ) : null}
                  </TableCell>
                  <TableCell className="max-w-[220px] text-[12px]">
                    {cursor?.last_error ? (
                      <span
                        className="inline-flex items-center gap-1 text-[color:var(--color-warning)]"
                        data-testid={`tasks-bridge-notifications-row-${sub.subscription_id}-cursor-error`}
                      >
                        <TriangleAlert className="size-3.5" />
                        {cursor.last_error}
                      </span>
                    ) : (
                      <span className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                        —
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="w-8 pr-4">
                    <Button
                      aria-label={`Delete subscription ${sub.subscription_id}`}
                      data-testid={`tasks-bridge-notifications-row-${sub.subscription_id}-delete`}
                      disabled={isDeletePending}
                      onClick={() => setDeleteTarget(sub)}
                      size="sm"
                      type="button"
                      variant="ghost"
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      ) : null}

      <Dialog open={createOpen} onOpenChange={handleCreateOpenChange}>
        <DialogContent
          className="max-w-lg"
          data-testid="tasks-bridge-notifications-create-dialog"
          showCloseButton={!isCreatePending}
        >
          <DialogHeader>
            <DialogTitle>New bridge notification subscription</DialogTitle>
            <DialogDescription>
              Subscriptions deliver this task's terminal events to a bridge target. Cursor
              diagnostics advance only after the bridge confirms delivery.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label htmlFor="bridge-instance-id">Bridge instance id</Label>
              <Input
                data-testid="tasks-bridge-notifications-create-bridge-instance-id"
                disabled={isCreatePending}
                id="bridge-instance-id"
                onChange={event => setForm({ ...form, bridgeInstanceId: event.target.value })}
                placeholder="bridge_instance_alpha"
                value={form.bridgeInstanceId}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="grid gap-1.5">
                <Label htmlFor="scope-select">Scope</Label>
                <Select
                  onValueChange={value =>
                    setForm({
                      ...form,
                      scope: value as TaskBridgeNotificationSubscriptionScope,
                    })
                  }
                  value={form.scope}
                >
                  <SelectTrigger
                    data-testid="tasks-bridge-notifications-create-scope"
                    id="scope-select"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="workspace">workspace</SelectItem>
                    <SelectItem value="global">global</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="delivery-mode-select">Delivery mode</Label>
                <Select
                  onValueChange={value =>
                    setForm({
                      ...form,
                      deliveryMode: value as TaskBridgeNotificationDeliveryMode,
                    })
                  }
                  value={form.deliveryMode}
                >
                  <SelectTrigger
                    data-testid="tasks-bridge-notifications-create-delivery-mode"
                    id="delivery-mode-select"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="direct-send">direct-send</SelectItem>
                    <SelectItem value="reply">reply</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="grid gap-1.5">
                <Label htmlFor="workspace-id">Workspace id</Label>
                <Input
                  data-testid="tasks-bridge-notifications-create-workspace-id"
                  disabled={isCreatePending || form.scope !== "workspace"}
                  id="workspace-id"
                  onChange={event => setForm({ ...form, workspaceId: event.target.value })}
                  placeholder="ws_default"
                  value={form.workspaceId}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="peer-id">Peer id (optional)</Label>
                <Input
                  data-testid="tasks-bridge-notifications-create-peer-id"
                  disabled={isCreatePending}
                  id="peer-id"
                  onChange={event => setForm({ ...form, peerId: event.target.value })}
                  placeholder="peer_observer"
                  value={form.peerId}
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="grid gap-1.5">
                <Label htmlFor="group-id">Group id (optional)</Label>
                <Input
                  data-testid="tasks-bridge-notifications-create-group-id"
                  disabled={isCreatePending}
                  id="group-id"
                  onChange={event => setForm({ ...form, groupId: event.target.value })}
                  placeholder="group_observers"
                  value={form.groupId}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="thread-id">Thread id (optional)</Label>
                <Input
                  data-testid="tasks-bridge-notifications-create-thread-id"
                  disabled={isCreatePending}
                  id="thread-id"
                  onChange={event => setForm({ ...form, threadId: event.target.value })}
                  placeholder="thread_launch"
                  value={form.threadId}
                />
              </div>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="subscription-id">Subscription id (optional)</Label>
              <Input
                data-testid="tasks-bridge-notifications-create-subscription-id"
                disabled={isCreatePending}
                id="subscription-id"
                onChange={event => setForm({ ...form, subscriptionId: event.target.value })}
                placeholder="bsub_001"
                value={form.subscriptionId}
              />
            </div>
            {createError ? (
              <p
                className="text-[12px] text-[color:var(--color-danger)]"
                data-testid="tasks-bridge-notifications-create-error"
              >
                {createError}
              </p>
            ) : null}
          </div>
          <DialogFooter className="gap-2">
            <Button
              data-testid="tasks-bridge-notifications-create-cancel"
              disabled={isCreatePending}
              onClick={() => handleCreateOpenChange(false)}
              type="button"
              variant="ghost"
            >
              Cancel
            </Button>
            <Button
              data-testid="tasks-bridge-notifications-create-submit"
              disabled={isCreatePending}
              onClick={handleCreateSubmit}
              type="button"
              variant="default"
            >
              {isCreatePending ? <Loader2 className="size-3.5 animate-spin" /> : null}
              Create subscription
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={open => {
          if (!open) {
            setDeleteTarget(null);
          }
        }}
      >
        <DialogContent
          className="max-w-md"
          data-testid="tasks-bridge-notifications-delete-dialog"
          showCloseButton={!isDeletePending}
        >
          <DialogHeader>
            <DialogTitle>Delete bridge subscription?</DialogTitle>
            <DialogDescription>
              This stops the bridge target from receiving this task's terminal notifications. Stale
              cursor diagnostics remain inspectable by cursor key for replay.
            </DialogDescription>
          </DialogHeader>
          {deleteTarget ? (
            <p className="font-mono text-[12px] text-[color:var(--color-text-secondary)]">
              {deleteTarget.subscription_id}
            </p>
          ) : null}
          <DialogFooter className="gap-2">
            <Button
              data-testid="tasks-bridge-notifications-delete-cancel"
              disabled={isDeletePending}
              onClick={() => setDeleteTarget(null)}
              type="button"
              variant="ghost"
            >
              Cancel
            </Button>
            <Button
              data-testid="tasks-bridge-notifications-delete-confirm"
              disabled={isDeletePending}
              onClick={handleDeleteConfirm}
              type="button"
              variant="destructive"
            >
              {isDeletePending ? <Loader2 className="size-3.5 animate-spin" /> : null}
              Delete subscription
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Section>
  );
}
