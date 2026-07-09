import { Play } from "lucide-react";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

interface LoopRunButtonProps extends ComponentProps<"button"> {
  loopName: string;
  onRun: () => void;
}

export function LoopRunButton({ loopName, onRun, className, ...props }: LoopRunButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex h-button-default shrink-0 items-center gap-1.5 rounded-md border border-line bg-btn-default-fill px-2.5 text-xs font-medium text-fg outline-none transition-colors hover:border-transparent hover:bg-accent hover:text-accent-ink focus-visible:shadow-focus-ring",
        className
      )}
      data-testid={`loop-catalog-run-${loopName}`}
      onClick={onRun}
      type="button"
      {...props}
    >
      <Play aria-hidden="true" className="size-3" />
      Run
    </button>
  );
}
