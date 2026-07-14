import { useState } from "react";
import { toast } from "sonner";

import {
  compactBridgeDeliveryDefaults,
  createBridgeTestDeliveryDraft,
  useSendBridgeTest,
  useTestBridgeDelivery,
  type BridgeSummary,
  type BridgeTestDeliveryDraft,
  type SendBridgeTestResponse,
  type TestBridgeDeliveryResponse,
} from "@/systems/bridges";
function optionalMessage(value: string): string | undefined {
  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

export function useBridgeDeliveryTests(bridge: BridgeSummary | undefined) {
  const [isDryRunOpen, setDryRunOpen] = useState(false);
  const [isSendTestOpen, setSendTestOpen] = useState(false);
  const [dryRunDraft, setDryRunDraft] = useState<BridgeTestDeliveryDraft>(() =>
    createBridgeTestDeliveryDraft()
  );
  const [sendTestDraft, setSendTestDraft] = useState<BridgeTestDeliveryDraft>(() =>
    createBridgeTestDeliveryDraft()
  );
  const [dryRunResult, setDryRunResult] = useState<TestBridgeDeliveryResponse | null>(null);
  const [sendTestResult, setSendTestResult] = useState<SendBridgeTestResponse | null>(null);

  const dryRunMutation = useTestBridgeDelivery();
  const sendTestMutation = useSendBridgeTest();

  const openDryRun = () => {
    setDryRunDraft(createBridgeTestDeliveryDraft(bridge));
    setDryRunResult(null);
    setDryRunOpen(true);
  };

  const openSendTest = () => {
    setSendTestDraft(createBridgeTestDeliveryDraft(bridge));
    setSendTestResult(null);
    setSendTestOpen(true);
  };

  const submitDryRun = async () => {
    if (!bridge) return;

    try {
      const result = await dryRunMutation.mutateAsync({
        id: bridge.id,
        data: {
          message: optionalMessage(dryRunDraft.message),
          target: {
            bridge_instance_id: bridge.id,
            ...compactBridgeDeliveryDefaults(dryRunDraft.target),
          },
        },
      });
      setDryRunResult(result);
      toast.success(`Resolved delivery target for ${bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to resolve bridge target");
    }
  };

  const submitSendTest = async () => {
    if (!bridge) return;
    if (!bridge.enabled) {
      toast.error("Enable the bridge before sending a real test message.");
      return;
    }

    const message = sendTestDraft.message.trim();
    if (message === "") {
      toast.error("Enter a message before sending a real bridge test.");
      return;
    }

    try {
      const result = await sendTestMutation.mutateAsync({
        id: bridge.id,
        data: {
          message,
          target: {
            bridge_instance_id: bridge.id,
            ...compactBridgeDeliveryDefaults(sendTestDraft.target),
          },
        },
      });
      setSendTestResult(result);
      toast.success(`Sent a test message through ${bridge.display_name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to send bridge test message");
    }
  };

  return {
    dryRunDialogProps: {
      bridgeName: bridge?.display_name,
      draft: dryRunDraft,
      intent: "dry-run" as const,
      isPending: dryRunMutation.isPending,
      onDraftChange: setDryRunDraft,
      onOpenChange: setDryRunOpen,
      onSubmit: submitDryRun,
      open: isDryRunOpen,
      result: dryRunResult,
    },
    openDryRun,
    openSendTest,
    sendTestDialogProps: {
      bridgeName: bridge?.display_name,
      draft: sendTestDraft,
      intent: "send-test" as const,
      isPending: sendTestMutation.isPending,
      onDraftChange: setSendTestDraft,
      onOpenChange: setSendTestOpen,
      onSubmit: submitSendTest,
      open: isSendTestOpen,
      result: sendTestResult,
    },
  };
}
