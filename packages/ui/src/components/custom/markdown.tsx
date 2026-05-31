"use client";

import * as React from "react";
import { Streamdown, type Components } from "streamdown";

import { cn } from "../../lib/utils";
import { STREAMDOWN_SAFE_CONFIG } from "./markdown-config";

/**
 * Markdown Safe-Mode Contract.
 *
 * Operator-authored markdown is user input rendered to other operators — an XSS surface.
 * This is the explicit allowlist the runtime ships:
 *
 * - Raw HTML markup is stripped at the parser via `skipHtml: true`.
 * - Output-side security-sensitive elements are blocked via `disallowedElements`.
 * - URL schemes are constrained to https/http/mailto/tel/internal hashes via streamdown's
 *   `defaultUrlTransform` (rehype-harden). `javascript:`, `data:`, `vbscript:`, `file:`,
 *   `about:`, and other schemes are rewritten to a `[blocked]` span.
 * - External images are rewritten to a textual `[image: alt text]` fallback. Relative
 *   URLs (`./`, `../`, `/`) keep rendering through a styled `<img>`.
 *
 * Owned by `<Markdown />` — every markdown surface in the runtime UI (description
 * cards, chat messages, tool-call panels) consumes the same contract.
 */
/**
 * Canonical markdown primitive for the AGH runtime UI. Wraps `streamdown` with
 * the `STREAMDOWN_SAFE_CONFIG` security contract and a single set of prose
 * Tailwind selectors driven by AGH design tokens, so every markdown surface —
 * description cards, chat messages, tool call inputs/outputs — renders against
 * the same grammar.
 *
 * Use `compact` for dense surfaces (tool call panels, inline previews); use
 * `streaming` to opt into streamdown's incremental parser for in-flight model
 * output. Extra `components` are merged on top of the safe-mode defaults.
 */
export interface MarkdownProps extends Omit<React.ComponentProps<"div">, "children"> {
  /** Markdown source — operator-authored or model-streamed. */
  children: string;
  /** Reduce vertical margins for dense surfaces (tool inputs/outputs, inline previews). */
  compact?: boolean;
  /** Enable streamdown's incremental parser for partial markdown. */
  streaming?: boolean;
  /** Merge extra component overrides on top of the safe-mode defaults. */
  components?: Partial<Components>;
}

const PROSE_BASE = [
  "text-card-title leading-prose text-fg",
  "[&>*:first-child]:mt-0 [&>*:last-child]:mb-0",
  "[&_a]:text-accent [&_a]:underline [&_a]:underline-offset-2",
  "[&_strong]:text-fg-strong",
  "[&_blockquote]:border-l [&_blockquote]:border-line-strong [&_blockquote]:pl-3 [&_blockquote]:text-muted",
  "[&_code]:rounded-xs [&_code]:bg-surface-glaze [&_code]:px-1 [&_code]:py-px [&_code]:font-mono [&_code]:text-form-input [&_code]:text-fg-strong",
  "[&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:bg-canvas [&_pre]:p-3 [&_pre]:text-form-input [&_pre]:font-mono",
  "[&_pre_code]:bg-transparent [&_pre_code]:px-0",
  "[&_hr]:border-line",
  "[&_ol]:list-decimal [&_ol]:pl-5",
  "[&_ul]:list-disc [&_ul]:pl-5",
  "[&_table]:w-full [&_table]:border-collapse",
  "[&_th]:border-b [&_th]:border-line-strong [&_th]:px-2 [&_th]:py-1.5 [&_th]:text-left [&_th]:text-form-label [&_th]:font-medium [&_th]:text-fg-strong",
  "[&_td]:border-b [&_td]:border-line [&_td]:px-2 [&_td]:py-1.5 [&_td]:text-form-input",
  "[&_h1]:text-(length:--text-section-head) [&_h1]:font-medium [&_h1]:tracking-section-head [&_h1]:text-fg-strong",
  "[&_h2]:text-(length:--text-section-head) [&_h2]:font-medium [&_h2]:tracking-section-head [&_h2]:text-fg-strong",
  "[&_h3]:text-small-body [&_h3]:font-medium [&_h3]:text-fg-strong",
  "[&_h4]:text-form-input [&_h4]:font-medium [&_h4]:text-fg",
  "[&_h5]:text-form-label [&_h5]:font-medium [&_h5]:text-fg",
  "[&_h6]:text-form-label [&_h6]:font-medium [&_h6]:text-muted",
].join(" ");

const PROSE_NORMAL = [
  "[&_p]:my-2",
  "[&_blockquote]:my-3 [&_pre]:my-3 [&_table]:my-3 [&_hr]:my-4",
  "[&_ol]:my-2 [&_ul]:my-2 [&_li]:my-0.5",
  "[&_h1]:mt-5 [&_h1]:mb-2",
  "[&_h2]:mt-5 [&_h2]:mb-2",
  "[&_h3]:mt-4 [&_h3]:mb-1.5",
  "[&_h4]:mt-3 [&_h4]:mb-1",
  "[&_h5]:mt-3 [&_h5]:mb-1",
  "[&_h6]:mt-3 [&_h6]:mb-1",
].join(" ");

const PROSE_COMPACT = [
  "[&_p]:my-1",
  "[&_blockquote]:my-2 [&_pre]:my-2 [&_table]:my-2 [&_hr]:my-2",
  "[&_ol]:my-1 [&_ul]:my-1 [&_li]:my-0",
  "[&_h1]:mt-3 [&_h1]:mb-1",
  "[&_h2]:mt-3 [&_h2]:mb-1",
  "[&_h3]:mt-2.5 [&_h3]:mb-1",
  "[&_h4]:mt-2 [&_h4]:mb-0.5",
  "[&_h5]:mt-2 [&_h5]:mb-0.5",
  "[&_h6]:mt-2 [&_h6]:mb-0.5",
].join(" ");

function MarkdownInner({
  children,
  compact = false,
  streaming = false,
  components,
  className,
  ...props
}: MarkdownProps) {
  const mergedComponents = React.useMemo<Partial<Components>>(
    () => ({
      ...(STREAMDOWN_SAFE_CONFIG.components as Partial<Components>),
      ...components,
    }),
    [components]
  );
  const streamingProps = streaming
    ? ({ mode: "streaming" as const, parseIncompleteMarkdown: true } as const)
    : undefined;
  return (
    <div
      data-slot="markdown"
      data-compact={compact ? "true" : undefined}
      className={cn(PROSE_BASE, compact ? PROSE_COMPACT : PROSE_NORMAL, className)}
      {...props}
    >
      <Streamdown {...STREAMDOWN_SAFE_CONFIG} {...streamingProps} components={mergedComponents}>
        {children}
      </Streamdown>
    </div>
  );
}

const Markdown = React.memo(MarkdownInner);
Markdown.displayName = "Markdown";

export { Markdown };
export { STREAMDOWN_SAFE_CONFIG };
