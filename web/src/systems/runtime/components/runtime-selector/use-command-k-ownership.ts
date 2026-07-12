import { useEffect, useEffectEvent } from "react";

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
 * The entry closures read the latest props through Effect Events, so one registration
 * registration always reflects current open/disabled state without re-subscribing.
 */
export function useCommandKOwnership(args: UseCommandKOwnershipArgs): void {
  const isEligible = useEffectEvent(() => !args.disabled);
  const isVisible = useEffectEvent(() => triggerVisible(args.triggerRef.current));
  const isOpen = useEffectEvent(() => args.open);
  const open = useEffectEvent(() => args.onOpen());
  const close = useEffectEvent(() => args.onClose());
  useEffect(() => {
    return registerCommandK({
      id: args.id,
      isEligible,
      isVisible,
      isOpen,
      open,
      close,
    });
  }, [args.id]);
}
