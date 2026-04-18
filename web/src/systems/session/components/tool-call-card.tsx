import { memo } from "react";
import { AlertCircle, ChevronRight } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@agh/ui";
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

    const statusBadge =
      card.status === "running" ? (
        <span
          className="shrink-0 rounded-full bg-[color:var(--color-accent-tint)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-[color:var(--color-accent)]"
          data-testid="tool-status-badge-running"
        >
          Running
        </span>
      ) : card.status === "error" ? (
        <span
          className="shrink-0 rounded-full bg-[color:var(--color-danger-tint)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-[color:var(--color-danger)]"
          data-testid="tool-status-badge-error"
        >
          Error
        </span>
      ) : (
        <span
          className="shrink-0 rounded-full bg-[color:var(--color-success-tint)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-[color:var(--color-success)]"
          data-testid="tool-status-badge-done"
        >
          Done
        </span>
      );

    return (
      <div className="min-w-0" data-testid="tool-call-card">
        <button
          type="button"
          onClick={card.handleToggle}
          className={cn(
            "group flex w-full items-center gap-2.5 rounded-lg border px-3 py-2",
            "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
            "text-[13px] transition-colors cursor-pointer overflow-hidden",
            "hover:border-[color:var(--color-hover)]"
          )}
          aria-expanded={card.expanded}
          data-testid="tool-card-trigger"
        >
          {card.isError ? (
            <AlertCircle
              className="size-3.5 shrink-0 text-[color:var(--color-danger)]"
              data-testid="tool-call-icon"
            />
          ) : (
            <Icon
              className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
              data-testid="tool-call-icon"
            />
          )}

          <span
            className={cn(
              "shrink-0 whitespace-nowrap font-medium",
              card.isError
                ? "text-[color:var(--color-danger)]"
                : "text-[color:var(--color-text-primary)]"
            )}
            data-testid={card.labelTestId}
          >
            {card.label}
          </span>

          {card.summary && card.showSummaryTooltip ? (
            <Tooltip>
              <TooltipTrigger
                className={cn("min-w-0 cursor-default truncate", toolToneClass(card.tone))}
              >
                {card.summary}
              </TooltipTrigger>
              <TooltipContent
                side="bottom"
                className="max-w-[min(56rem,calc(100vw-2rem))] px-0 py-0"
              >
                <div className="overflow-x-auto px-2 py-1.5 font-mono text-[11px] whitespace-nowrap">
                  {card.fullSummary}
                </div>
              </TooltipContent>
            </Tooltip>
          ) : card.summary ? (
            <span className={cn("min-w-0 truncate", toolToneClass(card.tone))}>{card.summary}</span>
          ) : null}

          <div className="ml-auto flex items-center gap-2">
            {statusBadge}
            {card.hasResult && (
              <ChevronRight
                className={cn(
                  "size-3 shrink-0 text-[color:var(--color-text-tertiary)]",
                  "opacity-0 transition-all duration-200 group-hover:opacity-100",
                  card.expanded && "rotate-90"
                )}
              />
            )}
          </div>
        </button>

        {card.expanded && card.hasResult && (
          <div className="mt-1 mb-2" data-testid="tool-card-expanded">
            <ExpandedToolContent message={message} />
          </div>
        )}
      </div>
    );
  },
  (previous, next) =>
    previous.message.toolInput === next.message.toolInput &&
    previous.message.toolResult === next.message.toolResult &&
    previous.message.toolError === next.message.toolError
);
