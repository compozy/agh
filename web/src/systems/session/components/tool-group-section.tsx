import { memo, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { cn } from "@/lib/utils";
import type { UIMessage } from "../types";
import { ToolCallCard } from "./tool-call-card";

const MAX_VISIBLE_ENTRIES = 6;

export interface ToolGroupSectionProps {
  tools: UIMessage[];
}

export const ToolGroupSection = memo(function ToolGroupSection({ tools }: ToolGroupSectionProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const hasOverflow = tools.length > MAX_VISIBLE_ENTRIES;
  const visibleTools = hasOverflow && !isExpanded ? tools.slice(-MAX_VISIBLE_ENTRIES) : tools;
  const hiddenCount = tools.length - visibleTools.length;

  return (
    <div
      className={cn(
        "mx-4 my-1 rounded-xl border px-2 py-1.5",
        "border-[color:var(--color-divider)]/45 bg-[color:var(--color-surface)]/25"
      )}
      data-testid="tool-group"
    >
      {(hasOverflow || tools.length > 1) && (
        <div className="mb-1.5 flex items-center justify-between gap-2 px-0.5">
          <p className="text-[9px] uppercase tracking-[0.16em] text-[color:var(--color-text-tertiary)]/55">
            Tool calls ({tools.length})
          </p>
          {hasOverflow && (
            <button
              type="button"
              onClick={() => setIsExpanded(v => !v)}
              className={cn(
                "flex items-center gap-1 rounded-md px-1.5 py-0.5",
                "text-[10px] text-[color:var(--color-text-tertiary)]",
                "hover:bg-[color:var(--color-surface-elevated)] transition-colors"
              )}
            >
              {isExpanded ? (
                <>
                  <ChevronUp className="size-3" />
                  Show less
                </>
              ) : (
                <>
                  <ChevronDown className="size-3" />
                  Show {hiddenCount} more
                </>
              )}
            </button>
          )}
        </div>
      )}
      <div className="space-y-0.5">
        {visibleTools.map(tool => (
          <ToolCallCard key={tool.id} message={tool} />
        ))}
      </div>
    </div>
  );
});
