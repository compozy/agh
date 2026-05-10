import { memo } from "react";
import { AlertCircle, ChevronRight } from "lucide-react";

import {
  ToolCallCard as PrimitiveToolCallCard,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@agh/ui";

import { cn } from "@/lib/utils";
import { useToolCallCard } from "../hooks/use-tool-call-card";
import { toolToneClass } from "../lib/tool-labels";
import type { UIMessage } from "../types";
import { ExpandedToolContent } from "./tool-renderers/expanded-tool-content";

export interface ToolCallCardProps {
  message: UIMessage;
}

export const ToolCallCard = memo(
  function ToolCallCard({ message }: ToolCallCardProps) {
    const card = useToolCallCard(message);
    const Icon = card.Icon;

    const iconNode = card.isError ? (
      <AlertCircle
        aria-hidden="true"
        data-slot="tool-call-card-icon"
        data-testid="tool-call-icon"
        className="size-3.5 shrink-0 text-(--danger)"
      />
    ) : (
      <Icon
        aria-hidden="true"
        data-slot="tool-call-card-icon"
        data-testid="tool-call-icon"
        className={cn("size-3.5 shrink-0", card.isRunning ? "text-(--accent)" : "text-(--subtle)")}
      />
    );

    const toneClass = toolToneClass(card.tone);

    const pathNode = card.summary ? (
      card.showSummaryTooltip ? (
        <Tooltip>
          <TooltipTrigger className={cn("min-w-0 cursor-default truncate", toneClass)}>
            {card.summary}
          </TooltipTrigger>
          <TooltipContent side="bottom" className="max-w-[min(56rem,calc(100vw-2rem))] p-0">
            <div className="overflow-x-auto px-2 py-1.5 font-mono text-eyebrow whitespace-nowrap">
              {card.fullSummary}
            </div>
          </TooltipContent>
        </Tooltip>
      ) : (
        <span className={cn("min-w-0 truncate", toneClass)}>{card.summary}</span>
      )
    ) : null;

    return (
      <div className="group min-w-0" data-testid="tool-call-card">
        <button
          type="button"
          onClick={card.handleToggle}
          aria-expanded={card.expanded}
          data-testid="tool-card-trigger"
          className={cn(
            "block w-full cursor-pointer text-left outline-none",
            "focus-visible:ring-2 focus-visible:ring-(--accent)/40 focus-visible:ring-offset-0",
            "rounded-md"
          )}
        >
          <PrimitiveToolCallCard
            toolName={
              <span
                data-testid={card.labelTestId}
                className={cn(card.isError ? "text-(--danger)" : "text-(--fg)")}
              >
                {card.label}
              </span>
            }
            filePath={pathNode ?? undefined}
            status={card.status}
            icon={iconNode}
            className={cn(
              "rounded-md transition-colors hover:border-(--hover)",
              "data-[status=error]:border-(--danger)/40"
            )}
          />
          {card.hasResult ? (
            <ChevronRight
              aria-hidden="true"
              data-slot="tool-call-card-chevron"
              className={cn("sr-only", card.expanded && "rotate-90")}
            />
          ) : null}
        </button>

        {card.expanded && card.hasResult ? (
          <div
            className="mt-1 mb-2 rounded-md border border-(--line) bg-(--canvas) p-3"
            data-testid="tool-card-expanded"
          >
            <ExpandedToolContent message={message} />
          </div>
        ) : null}
      </div>
    );
  },
  (previous, next) =>
    previous.message.toolInput === next.message.toolInput &&
    previous.message.toolResult === next.message.toolResult &&
    previous.message.toolError === next.message.toolError
);
