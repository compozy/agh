import { memo, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { Button } from "@agh/ui";

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

  if (tools.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "mx-4 my-1 rounded-[var(--radius-lg)] border px-2 py-1.5",
        "border-[color:var(--color-divider)]/60 bg-[color:var(--color-surface)]/40"
      )}
      data-testid="tool-group"
    >
      {(hasOverflow || tools.length > 1) && (
        <div className="mb-1.5 flex items-center justify-between gap-2 px-0.5">
          <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
            Tool calls · {tools.length}
          </span>
          {hasOverflow && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setIsExpanded(v => !v)}
              className="h-6 gap-1 px-1.5 text-[11px] text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-secondary)]"
              data-testid="tool-group-toggle"
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
            </Button>
          )}
        </div>
      )}
      <div className="space-y-1">
        {visibleTools.map(tool => (
          <ToolCallCard key={tool.id} message={tool} />
        ))}
      </div>
    </div>
  );
});
