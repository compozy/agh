import { useCallback, useEffect, useRef } from "react";

import type { ToolCallStatus } from "@agh/ui";

import {
  getToolCompactSummary,
  getToolFullSummary,
  getToolIcon,
  getToolLabel,
  getToolTone,
} from "../lib/tool-labels";
import type { UIMessage } from "../types";
import { usePersistedToolState } from "./use-persisted-tool-state";

export function useToolCallCard(message: UIMessage) {
  const isEditLike = message.toolName === "Edit" || message.toolName === "Write";
  const [expanded, setExpanded, hasStoredExpanded] = usePersistedToolState(message.id, isEditLike);
  const hasResult = Boolean(message.toolResult);
  const isError = Boolean(message.toolError);
  const isRunning = !hasResult;
  const tone = getToolTone(message);
  const Icon = getToolIcon(message.toolName ?? "", message.toolInput);
  const fullSummary = getToolFullSummary(message.toolName ?? "", message.toolInput);
  const summary = getToolCompactSummary(message.toolName ?? "", message.toolInput);
  const showSummaryTooltip = Boolean(summary && fullSummary && fullSummary !== summary);

  const initialHadResultRef = useRef(hasResult);
  const userToggledRef = useRef(false);
  const autoCollapseTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (!hasResult || initialHadResultRef.current || isEditLike || hasStoredExpanded) {
      return;
    }

    if (userToggledRef.current) {
      return;
    }

    setExpanded(true);
    autoCollapseTimerRef.current = setTimeout(() => {
      if (!userToggledRef.current) {
        setExpanded(false);
      }
    }, 2000);

    return () => clearTimeout(autoCollapseTimerRef.current);
  }, [hasResult, hasStoredExpanded, isEditLike, setExpanded]);

  const handleToggle = useCallback(() => {
    userToggledRef.current = true;
    clearTimeout(autoCollapseTimerRef.current);
    setExpanded(!expanded);
  }, [expanded, setExpanded]);

  const label = isRunning
    ? getToolLabel(message.toolName ?? "", "active")
    : isError
      ? `Failed to ${getToolLabel(message.toolName ?? "", "failure")}`
      : getToolLabel(message.toolName ?? "", "past");

  const labelTestId = isRunning
    ? "tool-card-executing"
    : isError
      ? "tool-card-error"
      : "tool-card-success";

  const status: ToolCallStatus = isRunning ? "running" : isError ? "error" : "done";

  return {
    expanded,
    fullSummary,
    handleToggle,
    hasResult,
    Icon,
    isError,
    isRunning,
    label,
    labelTestId,
    showSummaryTooltip,
    status,
    summary,
    tone,
  };
}
