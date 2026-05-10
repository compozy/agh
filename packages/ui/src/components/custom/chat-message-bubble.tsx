"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export type ChatMessageRole = "user" | "agent" | "system" | "tool" | "diff";

export type ChatMessageAlign = "left" | "right";

export interface ChatMessageBubbleProps extends Omit<React.ComponentProps<"div">, "role"> {
  role: ChatMessageRole;
  meta?: React.ReactNode;
  children: React.ReactNode;
  align?: ChatMessageAlign;
}

/**
 * Presentational chat message shell per DESIGN.md §4 "Chat Components". Role
 * drives alignment + wrapper style: `user` is right-aligned with a surface-
 * elevated bubble, `agent` is left-aligned with no bubble, `system` is a full
 * width hairline row, and `tool`/`diff` are pass-through blocks so callers can
 * drop a `ToolCallCard` or diff card inside.
 */
function ChatMessageBubble({
  role,
  meta,
  children,
  align,
  className,
  ...props
}: ChatMessageBubbleProps) {
  const resolvedAlign: ChatMessageAlign = align ?? (role === "user" ? "right" : "left");
  const isRightAligned = resolvedAlign === "right";
  const nonUserAlignClass = isRightAligned ? "items-end text-right" : "text-left";
  const nonUserMetaAlignClass = isRightAligned ? "justify-end text-right" : "justify-start";

  if (role === "system") {
    return (
      <div
        data-slot="chat-message"
        data-role="system"
        data-align={resolvedAlign}
        className={cn("flex w-full items-center gap-3 py-2 text-(--subtle)", className)}
        {...props}
      >
        <span aria-hidden="true" className="h-px flex-1 bg-(--line)" />
        <div
          data-slot="chat-message-body"
          className="font-mono text-[11px] leading-[16px] tracking-[0.06em]"
        >
          {children}
        </div>
        <span aria-hidden="true" className="h-px flex-1 bg-(--line)" />
      </div>
    );
  }

  if (role === "user") {
    return (
      <div
        data-slot="chat-message"
        data-role="user"
        data-align={resolvedAlign}
        className={cn(
          "flex w-full",
          resolvedAlign === "right" ? "justify-end" : "justify-start",
          className
        )}
        {...props}
      >
        <div
          data-slot="chat-message-inner"
          className="flex max-w-[min(640px,84%)] flex-col gap-1.5"
        >
          {meta ? (
            <div
              data-slot="chat-message-meta"
              className={cn(
                "font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-(--subtle)",
                resolvedAlign === "right" ? "text-right" : "text-left"
              )}
            >
              {meta}
            </div>
          ) : null}
          <div
            data-slot="chat-message-body"
            className="rounded-lg bg-(--elevated) px-5 py-4 text-[14px] leading-[1.6] text-(--fg)"
          >
            {children}
          </div>
        </div>
      </div>
    );
  }

  if (role === "agent") {
    return (
      <div
        data-slot="chat-message"
        data-role="agent"
        data-align={resolvedAlign}
        className={cn("flex w-full flex-col gap-1.5", nonUserAlignClass, className)}
        {...props}
      >
        {meta ? (
          <div
            data-slot="chat-message-meta"
            className={cn(
              "flex items-center gap-2 font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-(--muted)",
              nonUserMetaAlignClass
            )}
          >
            {meta}
          </div>
        ) : null}
        <div data-slot="chat-message-body" className="text-[14px] leading-[1.6] text-(--muted)">
          {children}
        </div>
      </div>
    );
  }

  return (
    <div
      data-slot="chat-message"
      data-role={role}
      data-align={resolvedAlign}
      className={cn("flex w-full flex-col gap-1.5", nonUserAlignClass, className)}
      {...props}
    >
      {meta ? (
        <div
          data-slot="chat-message-meta"
          className={cn(
            "flex items-center gap-2 font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-(--muted)",
            nonUserMetaAlignClass
          )}
        >
          {meta}
        </div>
      ) : null}
      <div data-slot="chat-message-body">{children}</div>
    </div>
  );
}

export { ChatMessageBubble };
