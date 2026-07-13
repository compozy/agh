import { useReducedMotionConfig } from "motion/react";

const DIALOG_MOTION_EASE = [0.2, 0, 0, 1] as const;

/**
 * Portaled popup/overlay nodes are not direct AnimatePresence children, so enter
 * `initial` values can stick forever. Skip enter motion (`initial={false}`) and
 * only animate exit; honor reduced-motion by zeroing exit duration.
 */
export function useDialogMotionTransition() {
  const reduced = useReducedMotionConfig();
  return { duration: reduced ? 0 : 0.2, ease: DIALOG_MOTION_EASE };
}
