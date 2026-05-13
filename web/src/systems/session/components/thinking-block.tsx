import { memo, useState } from "react";
import { ChevronRight, Brain } from "lucide-react";

import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@agh/ui";

export interface ThinkingBlockProps {
  thinking: string;
  thinkingComplete?: boolean;
}

export const ThinkingBlock = memo(
  function ThinkingBlock({ thinking, thinkingComplete }: ThinkingBlockProps) {
    const [userOpen, setUserOpen] = useState<boolean | null>(null);
    const open = userOpen ?? !thinkingComplete;
    const label = thinkingComplete ? "Thought process" : "Thinking";

    return (
      <Collapsible open={open} onOpenChange={setUserOpen}>
        <CollapsibleTrigger
          className={cn(
            "flex w-full items-center gap-1.5 rounded-md px-4 py-1.5 text-xs",
            "text-subtle hover:text-muted",
            "cursor-pointer transition-colors",
            "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-line-strong"
          )}
          data-testid="thinking-trigger"
        >
          <Brain className="size-3" />
          <span className="italic">{label}</span>
          {!thinkingComplete ? (
            <span className="h-1.5 w-1.5 rounded-full bg-info" aria-hidden="true" />
          ) : null}
          <ChevronRight className={cn("size-3 transition-transform", open && "rotate-90")} />
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div
            className={cn(
              "mx-4 mb-2 rounded-lg border px-3 py-2",
              "border-line bg-canvas-soft",
              "text-xs leading-relaxed text-muted",
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
