"use client";

import * as React from "react";

import { cn } from "../lib/utils";

export interface TypingDotsProps extends React.ComponentProps<"span"> {}

/**
 * Three-dot typing indicator — mirrors `.typing-dots` in
 * `docs/design/web-inspiration/styles/app.css`. Relies on the
 * `typing-bounce` keyframes declared in `packages/ui/src/tokens.css`.
 */
function TypingDots({ className, ...props }: TypingDotsProps) {
  return (
    <span
      {...props}
      data-slot="typing-dots"
      aria-hidden="true"
      className={cn("inline-flex items-center gap-[2px]", className)}
    >
      <span className="size-1 rounded-full bg-[color:var(--color-text-tertiary)] [animation:typing-bounce_1.2s_infinite_ease-in-out]" />
      <span className="size-1 rounded-full bg-[color:var(--color-text-tertiary)] [animation:typing-bounce_1.2s_infinite_ease-in-out_0.15s]" />
      <span className="size-1 rounded-full bg-[color:var(--color-text-tertiary)] [animation:typing-bounce_1.2s_infinite_ease-in-out_0.3s]" />
    </span>
  );
}

export { TypingDots };
