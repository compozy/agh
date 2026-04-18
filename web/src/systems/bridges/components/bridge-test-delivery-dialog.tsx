import { Loader2 } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Input,
  Pill,
} from "@agh/ui";

import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldTitle,
} from "@/components/ui/field";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { Textarea } from "@/components/ui/textarea";
import {
  bridgeProviderStateTone,
  describeBridgeTestTarget,
} from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeTestDeliveryDraft, TestBridgeDeliveryResponse } from "@/systems/bridges/types";

import { pillVariantFromTone } from "@/lib/pill-variant";
interface BridgeTestDeliveryDialogProps {
  bridgeName?: string;
  draft: BridgeTestDeliveryDraft;
  isPending: boolean;
  onDraftChange: (draft: BridgeTestDeliveryDraft) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  open: boolean;
  result: TestBridgeDeliveryResponse | null;
}

export function BridgeTestDeliveryDialog({
  bridgeName,
  draft,
  isPending,
  onDraftChange,
  onOpenChange,
  onSubmit,
  open,
  result,
}: BridgeTestDeliveryDialogProps) {
  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="max-w-[calc(100%-2rem)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-0 text-[color:var(--color-text-primary)] sm:max-w-2xl"
        showCloseButton={false}
      >
        <form
          className="flex max-h-[min(80vh,760px)] flex-col"
          data-testid="bridge-test-delivery-dialog"
          onSubmit={event => {
            event.preventDefault();
            if (isPending) {
              return;
            }
            onSubmit();
          }}
        >
          <DialogHeader className="space-y-2 px-6 pt-6">
            <DialogTitle>Test Delivery</DialogTitle>
            <DialogDescription className="text-[color:var(--color-text-secondary)]">
              Resolve the outbound target for {bridgeName ?? "the selected bridge"} using the saved
              defaults plus any explicit overrides below.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto px-6 py-6">
            <FieldGroup className="gap-4">
              <Field>
                <FieldContent>
                  <FieldTitle>Message</FieldTitle>
                  <FieldDescription>
                    Optional message preview echoed back with the resolved delivery target.
                  </FieldDescription>
                </FieldContent>
                <Textarea
                  data-testid="test-delivery-message"
                  onChange={event =>
                    onDraftChange({
                      ...draft,
                      message: event.target.value,
                    })
                  }
                  placeholder="Deliver a short operator ping."
                  value={draft.message}
                />
              </Field>

              <div className="grid gap-4 lg:grid-cols-2">
                <Field>
                  <FieldContent>
                    <FieldTitle>Mode</FieldTitle>
                  </FieldContent>
                  <NativeSelect
                    data-testid="test-delivery-mode-select"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        target: {
                          ...draft.target,
                          mode:
                            event.target.value === ""
                              ? undefined
                              : (event.target.value as NonNullable<typeof draft.target.mode>),
                        },
                      })
                    }
                    value={draft.target.mode ?? ""}
                  >
                    <NativeSelectOption value="">Use bridge default</NativeSelectOption>
                    <NativeSelectOption value="reply">Reply</NativeSelectOption>
                    <NativeSelectOption value="direct-send">Direct send</NativeSelectOption>
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldContent>
                    <FieldTitle>Peer ID</FieldTitle>
                  </FieldContent>
                  <Input
                    data-testid="test-delivery-peer-input"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        target: {
                          ...draft.target,
                          peer_id: event.target.value,
                        },
                      })
                    }
                    placeholder="peer_123"
                    value={draft.target.peer_id ?? ""}
                  />
                </Field>
                <Field>
                  <FieldContent>
                    <FieldTitle>Thread ID</FieldTitle>
                  </FieldContent>
                  <Input
                    data-testid="test-delivery-thread-input"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        target: {
                          ...draft.target,
                          thread_id: event.target.value,
                        },
                      })
                    }
                    placeholder="thread_456"
                    value={draft.target.thread_id ?? ""}
                  />
                </Field>
                <Field>
                  <FieldContent>
                    <FieldTitle>Group ID</FieldTitle>
                  </FieldContent>
                  <Input
                    data-testid="test-delivery-group-input"
                    onChange={event =>
                      onDraftChange({
                        ...draft,
                        target: {
                          ...draft.target,
                          group_id: event.target.value,
                        },
                      })
                    }
                    placeholder="group_789"
                    value={draft.target.group_id ?? ""}
                  />
                </Field>
              </div>

              {result ? (
                <section
                  className="space-y-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4"
                  data-testid="bridge-test-delivery-result"
                >
                  <div className="flex items-center justify-between gap-3">
                    <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                      Resolved target
                    </p>
                    <Pill variant={pillVariantFromTone(bridgeProviderStateTone(result.status))}>
                      {result.status}
                    </Pill>
                  </div>
                  <p className="text-sm text-[color:var(--color-text-primary)]">
                    {describeBridgeTestTarget(result.delivery_target)}
                  </p>
                  {result.message ? (
                    <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                      Message: {result.message}
                    </p>
                  ) : null}
                </section>
              ) : null}
            </FieldGroup>
          </div>

          <div className="flex items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-6 py-4">
            <Button
              className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
              onClick={() => onOpenChange(false)}
              size="lg"
              type="button"
              variant="outline"
            >
              Close
            </Button>
            <Button data-testid="submit-test-delivery" disabled={isPending} size="lg" type="submit">
              {isPending ? (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Resolving…
                </>
              ) : (
                "Resolve Target"
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
