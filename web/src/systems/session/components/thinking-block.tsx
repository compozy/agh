import { memo, useState } from "react";
import { ChevronRight, Brain } from "lucide-react";

import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

export interface ThinkingBlockProps {
  thinking: string;
  thinkingComplete?: boolean;
}

export const ThinkingBlock = memo(
  function ThinkingBlock({ thinking, thinkingComplete }: ThinkingBlockProps) {
    const [open, setOpen] = useState(false);

    return (
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger
          className={cn(
            "flex w-full items-center gap-1.5 px-4 py-1.5 text-xs",
            "text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-secondary)]",
            "cursor-pointer transition-colors"
          )}
          data-testid="thinking-trigger"
        >
          <Brain className="size-3" />
          <span className="italic">{thinkingComplete ? "Thought process" : "Thinking..."}</span>
          <ChevronRight className={cn("size-3 transition-transform", open && "rotate-90")} />
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div
            className={cn(
              "mx-4 mb-2 rounded-lg border px-3 py-2",
              "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
              "text-xs leading-relaxed text-[color:var(--color-text-secondary)]",
              "max-h-60 overflow-y-auto whitespace-pre-wrap"
            )}
            data-testid="thinking-content"
          >
            {thinking}
          </div>
        </CollapsibleContent>
      </Collapsible>
    );
  },
  (prev, next) => prev.thinking === next.thinking && prev.thinkingComplete === next.thinkingComplete
);
