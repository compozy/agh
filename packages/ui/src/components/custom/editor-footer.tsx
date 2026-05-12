"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface EditorFooterProps extends React.ComponentProps<"div"> {
  primary?: React.ReactNode;
  secondary?: React.ReactNode;
  meta?: React.ReactNode;
  /** When true, an Escape key press in the rendered tree calls `onEscape`. Default false. */
  escapeHandler?: boolean;
  onEscape?: () => void;
}

function EditorFooter({
  primary,
  secondary,
  meta,
  escapeHandler = false,
  onEscape,
  className,
  children,
  ...props
}: EditorFooterProps) {
  const handleKeyDown = React.useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (!escapeHandler) return;
      if (event.key === "Escape") {
        event.preventDefault();
        onEscape?.();
      }
    },
    [escapeHandler, onEscape]
  );
  return (
    <div
      data-slot="editor-footer"
      data-escape={escapeHandler ? "true" : undefined}
      role="contentinfo"
      onKeyDown={handleKeyDown}
      className={cn(
        "sticky bottom-0 z-10 flex min-h-editor-footer flex-wrap items-center gap-3 border-t border-line bg-canvas px-4 py-2.5",
        className
      )}
      {...props}
    >
      {meta ? (
        <div data-slot="editor-footer-meta" className="text-form-label text-muted">
          {meta}
        </div>
      ) : null}
      <div data-slot="editor-footer-spacer" className="flex-1" />
      {secondary ? (
        <div data-slot="editor-footer-secondary" className="flex items-center gap-2">
          {secondary}
        </div>
      ) : null}
      {primary ? (
        <div data-slot="editor-footer-primary" className="flex items-center gap-2">
          {primary}
        </div>
      ) : null}
      {children}
    </div>
  );
}

export { EditorFooter };
