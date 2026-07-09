import { Play } from "lucide-react";

import { cn } from "@/lib/utils";

interface LoopRunButtonProps {
  loopName: string;
  onRun: () => void;
  className?: string;
}

/**
 * Inline "Run" launch shared by the catalog row and card. Neutral resting fill
 * that flips to accent on hover, with the design-system focus ring so keyboard
 * users get a visible target. Test id is suffixed by loop name to stay unique
 * across the grid/list.
 */
export function LoopRunButton({ loopName, onRun, className }: LoopRunButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex h-button-default shrink-0 items-center gap-1.5 rounded-md border border-line bg-btn-default-fill px-2.5 text-xs font-medium text-fg outline-none transition-colors hover:border-transparent hover:bg-accent hover:text-accent-ink focus-visible:shadow-focus-ring",
        className
      )}
      data-testid={`loop-catalog-run-${loopName}`}
      onClick={onRun}
      type="button"
    >
      <Play aria-hidden="true" className="size-3" />
      Run
    </button>
  );
}
