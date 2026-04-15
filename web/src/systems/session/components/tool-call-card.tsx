import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AlertCircle, ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { UIMessage } from "../types";
import {
  getToolIcon,
  getToolLabel,
  getToolCompactSummary,
  getToolTone,
  toolToneClass,
} from "../lib/tool-labels";
import { ExpandedToolContent } from "./tool-renderers/expanded-tool-content";

// ── localStorage persistence ──

function usePersistedToolState(
  messageId: string,
  defaultValue: boolean
): [expanded: boolean, setExpanded: (v: boolean) => void, hasStored: boolean] {
  const storageKey = `tool:${messageId}`;

  const [hasStored, initial] = useMemo(() => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored !== null) return [true, stored === "true"];
    } catch {
      // localStorage unavailable
    }
    return [false, defaultValue];
  }, [storageKey, defaultValue]);

  const [value, setValueRaw] = useState(initial);

  const setValue = useCallback(
    (next: boolean) => {
      setValueRaw(next);
      try {
        localStorage.setItem(storageKey, String(next));
      } catch {
        // localStorage unavailable
      }
    },
    [storageKey]
  );

  return [value, setValue, hasStored];
}

// ── ToolCallCard ──

export interface ToolCallCardProps {
  message: UIMessage;
}

export const ToolCallCard = memo(
  function ToolCallCard({ message }: ToolCallCardProps) {
    const isEditLike = message.toolName === "Edit" || message.toolName === "Write";
    const [expanded, setExpanded, hasStoredExpanded] = usePersistedToolState(
      message.id,
      isEditLike
    );
    const hasResult = !!message.toolResult;
    const isRunning = !hasResult;
    const isError = !!message.toolError;
    const tone = getToolTone(message);
    const Icon = getToolIcon(message.toolName ?? "", message.toolInput);
    const summary = getToolCompactSummary(message.toolName ?? "", message.toolInput);

    // For Bash tools, show full raw command in tooltip when it's truncated
    const rawCommand = message.toolName === "Bash" ? String(message.toolInput?.command ?? "") : "";
    const showCommandTooltip = rawCommand.length > 80;

    // Track whether toolResult was present at mount (history → skip auto-expand)
    const initialHadResult = useRef(hasResult);
    // Track whether user manually toggled (cancel auto-collapse)
    const userToggled = useRef(false);
    const autoCollapseTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

    // Auto-expand on result arrival, then auto-collapse after 2s
    useEffect(() => {
      if (!hasResult || initialHadResult.current || isEditLike || hasStoredExpanded) return;
      if (userToggled.current) return;
      setExpanded(true);
      autoCollapseTimer.current = setTimeout(() => {
        if (!userToggled.current) setExpanded(false);
      }, 2000);
      return () => clearTimeout(autoCollapseTimer.current);
    }, [hasResult, hasStoredExpanded, isEditLike, setExpanded]);

    const handleToggle = useCallback(() => {
      userToggled.current = true;
      clearTimeout(autoCollapseTimer.current);
      setExpanded(!expanded);
    }, [expanded, setExpanded]);

    const statusBadge = isRunning ? (
      <span
        className="shrink-0 rounded-full bg-[color:var(--color-accent-tint)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-[color:var(--color-accent)]"
        data-testid="tool-status-badge-running"
      >
        Running
      </span>
    ) : isError ? (
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
          onClick={handleToggle}
          className={cn(
            "group flex w-full items-center gap-2.5 rounded-lg border px-3 py-2",
            "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
            "text-[13px] transition-colors cursor-pointer overflow-hidden",
            "hover:border-[color:var(--color-hover)]"
          )}
          aria-expanded={expanded}
          data-testid="tool-card-trigger"
        >
          {isError ? (
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
              isError
                ? "text-[color:var(--color-danger)]"
                : "text-[color:var(--color-text-primary)]"
            )}
            data-testid={
              isRunning ? "tool-card-executing" : isError ? "tool-card-error" : "tool-card-success"
            }
          >
            {isRunning
              ? getToolLabel(message.toolName ?? "", "active")
              : isError
                ? `Failed to ${getToolLabel(message.toolName ?? "", "failure")}`
                : getToolLabel(message.toolName ?? "", "past")}
          </span>

          {summary && showCommandTooltip ? (
            <Tooltip>
              <TooltipTrigger
                className={cn("min-w-0 truncate cursor-default", toolToneClass(tone))}
              >
                {summary}
              </TooltipTrigger>
              <TooltipContent
                side="bottom"
                className="max-w-[min(56rem,calc(100vw-2rem))] px-0 py-0"
              >
                <div className="overflow-x-auto px-2 py-1.5 font-mono text-[11px] whitespace-nowrap">
                  {rawCommand}
                </div>
              </TooltipContent>
            </Tooltip>
          ) : summary ? (
            <span className={cn("min-w-0 truncate", toolToneClass(tone))}>{summary}</span>
          ) : null}

          <div className="ml-auto flex items-center gap-2">
            {statusBadge}
            {hasResult && (
              <ChevronRight
                className={cn(
                  "size-3 shrink-0 text-[color:var(--color-text-tertiary)]",
                  "opacity-0 group-hover:opacity-100 transition-all duration-200",
                  expanded && "rotate-90"
                )}
              />
            )}
          </div>
        </button>

        {expanded && hasResult && (
          <div className="mt-1 mb-2" data-testid="tool-card-expanded">
            <ExpandedToolContent message={message} />
          </div>
        )}
      </div>
    );
  },
  (prev, next) =>
    prev.message.toolInput === next.message.toolInput &&
    prev.message.toolResult === next.message.toolResult &&
    prev.message.toolError === next.message.toolError
);
