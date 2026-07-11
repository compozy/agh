import { useEffect, useRef } from "react";

import { registerCommandK, triggerVisible } from "./command-k-registry";

export interface UseCommandKOwnershipArgs {
  id: string;
  disabled: boolean;
  open: boolean;
  triggerRef: { current: HTMLElement | null };
  onOpen: () => void;
  onClose: () => void;
}

/**
 * Register this selector with the global `⌘K` owner registry for its lifetime.
 * The entry closures read the latest props through a ref, so a single mount-time
 * registration always reflects current open/disabled state without re-subscribing.
 */
export function useCommandKOwnership(args: UseCommandKOwnershipArgs): void {
  const latest = useRef(args);
  latest.current = args;
  useEffect(() => {
    return registerCommandK({
      id: latest.current.id,
      isEligible: () => !latest.current.disabled,
      isVisible: () => triggerVisible(latest.current.triggerRef.current),
      isOpen: () => latest.current.open,
      open: () => latest.current.onOpen(),
      close: () => latest.current.onClose(),
    });
    // Registered once for the component's lifetime; live state is read via `latest`.
  }, []);
}
