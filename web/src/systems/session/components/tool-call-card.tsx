import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AlertCircle, ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";
import type { UIMessage } from "../types";
import { getToolIcon, getToolLabel, getToolCompactSummary } from "../lib/tool-labels";
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
    const Icon = getToolIcon(message.toolName ?? "");
    const summary = getToolCompactSummary(message.toolName ?? "", message.toolInput);

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

    return (
      <div className="min-w-0" data-testid="tool-call-card">
        <button
          type="button"
          onClick={handleToggle}
          className={cn(
            "group relative flex w-full items-center gap-2 py-1",
            "text-[13px] hover:text-[color:var(--color-text-primary)] transition-colors cursor-pointer overflow-hidden"
          )}
          aria-expanded={expanded}
          data-testid="tool-card-trigger"
        >
          <div className="relative flex items-center gap-2 min-w-0">
            {isError ? (
              <AlertCircle className="size-3.5 shrink-0 text-red-400/70" />
            ) : (
              <Icon className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]/35" />
            )}
            {isRunning ? (
              <span
                className="shrink-0 whitespace-nowrap font-medium animate-shimmer bg-clip-text text-transparent bg-[length:200%_100%] bg-gradient-to-r from-[color:var(--color-text-tertiary)] via-[color:var(--color-text-primary)] to-[color:var(--color-text-tertiary)]"
                data-testid="tool-card-executing"
              >
                {getToolLabel(message.toolName ?? "", "active")}
              </span>
            ) : (
              <span
                className={cn(
                  "shrink-0 whitespace-nowrap font-medium",
                  isError ? "text-red-400/70" : "text-[color:var(--color-text-secondary)]"
                )}
                data-testid={isError ? "tool-card-error" : "tool-card-success"}
              >
                {isError
                  ? `Failed to ${getToolLabel(message.toolName ?? "", "failure")}`
                  : getToolLabel(message.toolName ?? "", "past")}
              </span>
            )}
            {summary && (
              <span className="truncate text-[color:var(--color-text-tertiary)]/40">{summary}</span>
            )}
          </div>

          {hasResult && (
            <ChevronRight
              className={cn(
                "ms-auto size-3 shrink-0 text-[color:var(--color-text-tertiary)]/30",
                "opacity-0 group-hover:opacity-100 transition-all duration-200",
                expanded && "rotate-90"
              )}
            />
          )}
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
