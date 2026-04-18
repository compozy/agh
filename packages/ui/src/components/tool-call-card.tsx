"use client";

import { TerminalIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../lib/utils";
import { MonoBadge, type MonoBadgeTone } from "./mono-badge";

export type ToolCallStatus = "running" | "done" | "error";

export interface ToolCallCardProps extends React.ComponentProps<"div"> {
  toolName: string;
  filePath?: string;
  status: ToolCallStatus;
  children?: React.ReactNode;
}

const STATUS_TONE: Record<ToolCallStatus, MonoBadgeTone> = {
  running: "accent",
  done: "success",
  error: "danger",
};

const STATUS_LABEL: Record<ToolCallStatus, string> = {
  running: "RUNNING",
  done: "DONE",
  error: "ERROR",
};

/**
 * Inline tool-execution card per DESIGN.md §4 "Tool Call Card". Surface bg with
 * a 1px divider border, terminal icon + tool name + optional file path, status
 * badge pinned to the right. Optional `children` renders below the header in a
 * bordered body slot for diffs, stdout, or other compositional content.
 */
function ToolCallCard({
  toolName,
  filePath,
  status,
  children,
  className,
  ...props
}: ToolCallCardProps) {
  return (
    <div
      data-slot="tool-call-card"
      data-status={status}
      className={cn(
        "overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
        className
      )}
      {...props}
    >
      <div
        data-slot="tool-call-card-header"
        className="flex min-w-0 items-center gap-3 px-4 py-2.5"
      >
        <TerminalIcon
          aria-hidden="true"
          data-slot="tool-call-card-icon"
          className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
        />
        <span
          data-slot="tool-call-card-tool"
          className="text-[14px] font-medium text-[color:var(--color-text-primary)]"
        >
          {toolName}
        </span>
        {filePath ? (
          <span
            data-slot="tool-call-card-path"
            className="min-w-0 truncate text-[13px] text-[color:var(--color-text-tertiary)]"
          >
            {filePath}
          </span>
        ) : null}
        <MonoBadge
          tone={STATUS_TONE[status]}
          className="ml-auto shrink-0"
          data-slot="tool-call-card-status"
        >
          {STATUS_LABEL[status]}
        </MonoBadge>
      </div>
      {children ? (
        <div
          data-slot="tool-call-card-body"
          className="border-t border-[color:var(--color-divider)] px-4 py-3"
        >
          {children}
        </div>
      ) : null}
    </div>
  );
}

export { ToolCallCard };
