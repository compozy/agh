import { useTopbarSlot } from "@agh/ui";
import { BridgeDetailPanel, BridgeEditDialog, BridgeTestDeliveryDialog } from "@/systems/bridges";
import { useBridgeDetailPage } from "./use-bridge-detail-page";

export function BridgeDetailLocation({ id }: { id: string }) {
  const page = useBridgeDetailPage(id);
  useTopbarSlot({ crumb: `Bridges / ${id}` });

  return (
    <>
      <BridgeDetailPanel {...page.detailPanelProps} />
      <BridgeEditDialog {...page.editDialogProps} />
      <BridgeTestDeliveryDialog {...page.testDeliveryDialogProps} />
      <BridgeTestDeliveryDialog {...page.sendTestDialogProps} />
    </>
  );
}
