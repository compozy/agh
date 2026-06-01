"use client";

import type * as React from "react";

import { cn } from "../../lib/utils";
import { Markdown } from "./markdown";

export interface DescriptionCardProps extends Omit<React.ComponentProps<"section">, "children"> {
  /** Markdown source — operator-authored or model-streamed. */
  children: string;
  /** Render the card chrome — set `bare` to consume only the prose styles inline. */
  bare?: boolean;
}

/**
 * Operator-authored markdown card. Rounded `bg-canvas-soft` chrome around a
 * `<Markdown />` body so the same prose grammar is shared with every other
 * markdown surface in the runtime (chat messages, tool-call panels, knowledge
 * notes). Pass `bare` to drop the chrome and consume only the prose styles
 * inline.
 */
function DescriptionCard({ children, bare = false, className, ...props }: DescriptionCardProps) {
  return (
    <section
      data-slot="description-card"
      data-bare={bare ? "true" : undefined}
      className={cn(
        bare ? "flex flex-col" : "flex flex-col rounded-lg bg-canvas-soft px-5 py-4",
        className
      )}
      {...props}
    >
      <Markdown>{children}</Markdown>
    </section>
  );
}

export { DescriptionCard };
