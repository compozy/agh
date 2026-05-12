import { describeBridgeTestTarget } from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeTestDeliveryDraft, TestBridgeDeliveryResponse } from "@/systems/bridges/types";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldTitle,
  Input,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Section,
  Spinner,
  Textarea,
  type PillTone,
} from "@agh/ui";

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

function resultTone(status: string): PillTone {
  switch (status) {
    case "resolved":
    case "ready":
      return "success";
    case "error":
    case "failed":
      return "danger";
    case "pending":
      return "warning";
    default:
      return "neutral";
  }
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
      <DialogContent className="gap-0 p-0 text-fg sm:max-w-2xl" showCloseButton={false} unframed>
        <div
          className="flex max-h-[min(80vh,var(--height-modal-md))] flex-col"
          data-testid="bridge-test-delivery-dialog"
        >
          <DialogHeader variant="ruled">
            <DialogTitle>Test Delivery</DialogTitle>
            <DialogDescription>
              Resolve the outbound target for {bridgeName ?? "the selected bridge"} using the saved
              defaults plus any explicit overrides below.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto p-5">
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
                <Section
                  data-testid="bridge-test-delivery-result"
                  label="Resolved target"
                  right={
                    <Pill mono tone={resultTone(result.status)}>
                      {result.status}
                    </Pill>
                  }
                >
                  <p className="text-small-body text-fg">
                    {describeBridgeTestTarget(result.delivery_target)}
                  </p>
                  {result.message ? (
                    <p className="mt-2 text-small-body leading-relaxed text-muted">
                      Message: {result.message}
                    </p>
                  ) : null}
                </Section>
              ) : null}
            </FieldGroup>
          </div>

          <DialogFooter variant="ruled">
            <Button onClick={() => onOpenChange(false)} size="sm" type="button" variant="outline">
              Close
            </Button>
            <Button
              data-testid="submit-test-delivery"
              disabled={isPending}
              onClick={onSubmit}
              size="sm"
              type="button"
            >
              {isPending ? (
                <>
                  <Spinner className="size-3" />
                  Resolving…
                </>
              ) : (
                "Resolve Target"
              )}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
