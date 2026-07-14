import { SearchCheck, SendHorizontal } from "lucide-react";

import { Button, Eyebrow } from "@agh/ui";

export function BridgeDeliveryActions({
  bridgeDisabled,
  onOpenDryRun,
  onOpenSendTest,
}: {
  bridgeDisabled: boolean;
  onOpenDryRun: () => void;
  onOpenSendTest: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-md border border-line bg-canvas-soft px-5 py-4">
      <div className="min-w-0 flex-1 space-y-1">
        <Eyebrow className="block text-muted">Delivery checks</Eyebrow>
        <p className="text-small-body text-muted">
          Resolve a target without provider side effects, or send one real message through the
          enabled bridge.
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button
          data-testid="open-test-delivery-btn"
          onClick={onOpenDryRun}
          size="sm"
          type="button"
          variant="outline"
        >
          <SearchCheck className="size-3" />
          Check target (dry run)
        </Button>
        <Button
          data-testid="open-send-test-btn"
          disabled={bridgeDisabled}
          onClick={onOpenSendTest}
          size="sm"
          type="button"
        >
          <SendHorizontal className="size-3" />
          Send test message
        </Button>
      </div>
    </div>
  );
}
