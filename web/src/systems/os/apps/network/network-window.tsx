import { useState } from "react";

import {
  NetworkWindowController,
  type NetworkWindowLocation,
  type NetworkWindowNavigation,
} from "@/systems/network";

import { useDesktop } from "../../hooks/use-desktop";
import { useOsShell } from "../../hooks/use-os-shell";

/** OS adapter: WM location in, routing-coordinator transition out. */
export function NetworkWindow({ windowId }: { windowId: string }) {
  const { coordinator } = useOsShell();
  const active = useDesktop(state => state.focusedId === windowId);
  const location = useDesktop(
    state => state.windows[windowId]?.location ?? { pathname: "/network", search: {} }
  );
  const [navigation] = useState<NetworkWindowNavigation>(() => ({
    push: (next: NetworkWindowLocation) => {
      coordinator.userOpen({ app: "network", location: next });
    },
  }));

  return <NetworkWindowController active={active} location={location} navigation={navigation} />;
}
