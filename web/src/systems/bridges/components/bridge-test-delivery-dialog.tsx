import {
  ActionResultBanner,
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
  MetadataList,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Section,
  Spinner,
  Textarea,
  type PillTone,
} from "@agh/ui";

import { describeBridgeTestTarget } from "../lib/bridge-formatters";
import type {
  BridgeTestDeliveryDraft,
  SendBridgeTestResponse,
  TestBridgeDeliveryResponse,
} from "../types";

interface CommonDialogProps {
  bridgeName?: string;
  draft: BridgeTestDeliveryDraft;
  isPending: boolean;
  onDraftChange: (draft: BridgeTestDeliveryDraft) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  open: boolean;
}

type BridgeTestDeliveryDialogProps = CommonDialogProps &
  (
    | {
        intent: "dry-run";
        result: TestBridgeDeliveryResponse | null;
      }
    | {
        intent: "send-test";
        result: SendBridgeTestResponse | null;
      }
  );

function resultTone(status: string): PillTone {
  switch (status) {
    case "resolved":
    case "ready":
    case "delivered":
      return "success";
    case "error":
    case "failed":
      return "danger";
    case "pending":
    case "committed_result_unavailable":
      return "warning";
    default:
      return "neutral";
  }
}

function updateTargetField(
  draft: BridgeTestDeliveryDraft,
  field: "group_id" | "peer_id" | "thread_id",
  value: string
): BridgeTestDeliveryDraft {
  return { ...draft, target: { ...draft.target, [field]: value } };
}

function DeliveryTargetFields({
  draft,
  onDraftChange,
}: Pick<CommonDialogProps, "draft" | "onDraftChange">) {
  return (
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
      {(
        [
          ["peer_id", "Peer ID", "peer_123", "test-delivery-peer-input"],
          ["thread_id", "Thread ID", "thread_456", "test-delivery-thread-input"],
          ["group_id", "Group ID", "group_789", "test-delivery-group-input"],
        ] as const
      ).map(([field, label, placeholder, testId]) => (
        <Field key={field}>
          <FieldContent>
            <FieldTitle>{label}</FieldTitle>
          </FieldContent>
          <Input
            data-testid={testId}
            onChange={event => onDraftChange(updateTargetField(draft, field, event.target.value))}
            placeholder={placeholder}
            value={draft.target[field] ?? ""}
          />
        </Field>
      ))}
    </div>
  );
}

function DryRunResult({ result }: { result: TestBridgeDeliveryResponse }) {
  return (
    <Section
      data-testid="bridge-test-delivery-result"
      label="Resolved target"
      right={
        <Pill mono tone={resultTone(result.status)}>
          {result.status}
        </Pill>
      }
    >
      <p className="text-small-body text-fg">{describeBridgeTestTarget(result.delivery_target)}</p>
      {result.message ? (
        <p className="mt-2 text-small-body leading-relaxed text-muted">Message: {result.message}</p>
      ) : null}
    </Section>
  );
}

function SendTestResult({ result }: { result: SendBridgeTestResponse }) {
  const committedResultUnavailable = result.status === "committed_result_unavailable";
  const committedResultMessage =
    result.error?.message ??
    "The provider accepted the mutation but did not return a verifiable result.";
  const committedResultSentence = /[.!?]$/.test(committedResultMessage)
    ? committedResultMessage
    : `${committedResultMessage}.`;

  return (
    <Section
      data-testid="bridge-send-test-result"
      label="Delivery result"
      right={
        <Pill mono tone={resultTone(result.status)}>
          {result.status}
        </Pill>
      }
    >
      <MetadataList className="grid gap-3 sm:grid-cols-2">
        <MetadataList.Row label="Delivery ID">
          <span className="break-all font-mono">{result.delivery_id}</span>
        </MetadataList.Row>
        <MetadataList.Row label="Target">
          {describeBridgeTestTarget(result.delivery_target)}
        </MetadataList.Row>
        {result.remote_message_id ? (
          <MetadataList.Row label="Remote message ID">
            <span className="break-all font-mono">{result.remote_message_id}</span>
          </MetadataList.Row>
        ) : null}
      </MetadataList>
      {committedResultUnavailable ? (
        <ActionResultBanner
          className="mt-4"
          description={`${committedResultSentence} Inspect the provider before sending this test again.`}
          title="Delivery result is indeterminate"
          tone="warning"
        />
      ) : null}
    </Section>
  );
}

export function BridgeTestDeliveryDialog(props: BridgeTestDeliveryDialogProps) {
  const { bridgeName, draft, intent, isPending, onDraftChange, onOpenChange, onSubmit, open } =
    props;
  const sendsMessage = intent === "send-test";
  const missingMessage = sendsMessage && draft.message.trim() === "";

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="gap-0 p-0 text-fg sm:max-w-2xl" showCloseButton={false} unframed>
        <div
          className="flex max-h-[min(80vh,var(--height-modal-md))] flex-col"
          data-intent={intent}
          data-testid={sendsMessage ? "bridge-send-test-dialog" : "bridge-test-delivery-dialog"}
        >
          <DialogHeader variant="ruled">
            <DialogTitle>
              {sendsMessage ? "Send test message" : "Check delivery target"}
            </DialogTitle>
            <DialogDescription>
              {sendsMessage
                ? `Send one real provider message through ${bridgeName ?? "the selected bridge"}.`
                : `Resolve the outbound target for ${bridgeName ?? "the selected bridge"} without sending a provider message.`}
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto p-5">
            <FieldGroup className="gap-4">
              <Field>
                <FieldContent>
                  <FieldTitle>Message</FieldTitle>
                  <FieldDescription>
                    {sendsMessage
                      ? "Required. This content is sent through the provider."
                      : "Optional preview echoed with the resolved target; no message is sent."}
                  </FieldDescription>
                </FieldContent>
                <Textarea
                  aria-required={sendsMessage}
                  data-testid="test-delivery-message"
                  onChange={event => onDraftChange({ ...draft, message: event.target.value })}
                  placeholder="Deliver a short operator ping."
                  required={sendsMessage}
                  value={draft.message}
                />
              </Field>

              <DeliveryTargetFields draft={draft} onDraftChange={onDraftChange} />

              {intent === "dry-run" && props.result ? <DryRunResult result={props.result} /> : null}
              {intent === "send-test" && props.result ? (
                <SendTestResult result={props.result} />
              ) : null}
            </FieldGroup>
          </div>

          <DialogFooter variant="ruled">
            <Button onClick={() => onOpenChange(false)} size="sm" type="button" variant="outline">
              Close
            </Button>
            <Button
              data-testid={sendsMessage ? "submit-send-test" : "submit-test-delivery"}
              disabled={isPending || missingMessage}
              onClick={onSubmit}
              size="sm"
              type="button"
            >
              {isPending ? <Spinner aria-hidden="true" className="size-3" /> : null}
              {isPending
                ? sendsMessage
                  ? "Sending…"
                  : "Checking…"
                : sendsMessage
                  ? "Send message"
                  : "Check target"}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
