"use client";

import { TerminalIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../lib/utils";
import { MonoBadge, type MonoBadgeTone } from "./mono-badge";

export type ToolCallStatus = "running" | "done" | "error";

type ToolCallIconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface ToolCallCardProps extends React.ComponentProps<"div"> {
  toolName: React.ReactNode;
  filePath?: React.ReactNode;
  status: ToolCallStatus;
  icon?: ToolCallIconComponent | React.ReactNode;
  children?: React.ReactNode;
}

function isIconComponent(value: unknown): value is ToolCallIconComponent {
  if (typeof value === "function") return true;
  if (typeof value === "object" && value !== null && "render" in value) return true;
  return false;
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
  icon,
  children,
  className,
  ...props
}: ToolCallCardProps) {
  let iconContent: React.ReactNode;
  const iconClass = "size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]";
  if (icon === undefined) {
    iconContent = (
      <TerminalIcon aria-hidden="true" data-slot="tool-call-card-icon" className={iconClass} />
    );
  } else if (isIconComponent(icon)) {
    const IconComp = icon;
    iconContent = (
      <IconComp aria-hidden="true" data-slot="tool-call-card-icon" className={iconClass} />
    );
  } else {
    iconContent = icon;
  }

  return (
    <div
      data-slot="tool-call-card"
      data-status={status}
      className={cn(
        "overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
        "data-[status=error]:border-[color:var(--color-danger)]/40",
        className
      )}
      {...props}
    >
      <div
        data-slot="tool-call-card-header"
        className="flex min-w-0 items-center gap-3 px-4 py-2.5"
      >
        {iconContent}
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
