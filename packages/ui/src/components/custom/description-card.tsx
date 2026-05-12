"use client";

import * as React from "react";
import { Streamdown, defaultUrlTransform } from "streamdown";

import { cn } from "../../lib/utils";

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
 * Exported as `STREAMDOWN_SAFE_CONFIG` so future markdown consumers (chat-thread bodies,
 * knowledge notes, tool params) reuse the same contract without redefining it.
 */
const SAFE_DISALLOWED_ELEMENTS = [
  "script",
  "iframe",
  "object",
  "embed",
  "form",
  "input",
  "button",
  "style",
  "link",
  "meta",
  "base",
  "svg",
  "math",
] as const;

function isExternalUrl(value: string): boolean {
  if (!value) return false;
  // Protocol-relative URLs are external.
  if (value.startsWith("//")) return true;
  // Absolute URLs with any scheme are external; the URL transform handles the
  // disallowed schemes — anything reaching here is already on the allowlist
  // (https / http / mailto / tel). For images we still want to block remote loads.
  return /^[a-z][a-z0-9+.-]*:/i.test(value);
}

function SafeImage({
  src,
  alt,
  width,
  height,
  title,
  className,
}: React.ImgHTMLAttributes<HTMLImageElement>) {
  const url = typeof src === "string" ? src : "";
  const altText = typeof alt === "string" && alt.length > 0 ? alt : "image";
  if (isExternalUrl(url)) {
    return (
      <span
        data-slot="description-card-image-fallback"
        className="text-muted italic"
      >{`[image: ${altText}]`}</span>
    );
  }
  return (
    <img
      data-slot="description-card-image"
      src={url}
      alt={altText}
      width={width}
      height={height}
      title={title}
      className={cn("max-w-full rounded", className)}
    />
  );
}

/**
 * Component overrides that force streamdown to emit the canonical HTML elements named in
 * the safe-mode allowlist (`strong`, `em`, `code`, `kbd`, `s`, `del`, `ins`, `mark`,
 * `blockquote`, `a`) instead of streamdown's `<span data-streamdown="...">` wrappers.
 * Keeps prose styling addressable through `[&_<tag>]:` Tailwind selectors and ensures
 * the rendered DOM matches the TechSpec allowlist contract.
 */
const SAFE_COMPONENT_OVERRIDES = {
  strong: "strong",
  em: "em",
  code: "code",
  kbd: "kbd",
  s: "s",
  del: "del",
  ins: "ins",
  mark: "mark",
  blockquote: "blockquote",
  img: SafeImage,
} as const;

export const STREAMDOWN_SAFE_CONFIG = {
  /** Strip raw HTML markup from the input source. */
  skipHtml: true as const,
  /** Block security-sensitive elements at the output stage as defense-in-depth. */
  disallowedElements: SAFE_DISALLOWED_ELEMENTS,
  /** Allow only https/http/mailto/tel/internal-hash schemes; rewrite the rest. */
  urlTransform: defaultUrlTransform,
  /** Remove the in-rendered table copy/download dropdowns + line-numbers shipped by default. */
  controls: false as const,
  lineNumbers: false as const,
  /** Force canonical HTML elements (per TechSpec allowlist) + safe `<img>` override. */
  components: SAFE_COMPONENT_OVERRIDES,
};

export interface DescriptionCardProps extends Omit<React.ComponentProps<"section">, "children"> {
  /** Markdown source — operator-authored or model-streamed. */
  children: string;
  /** Render the card chrome — set `bare` to consume only the prose styles inline. */
  bare?: boolean;
}

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
      <div
        data-slot="description-card-prose"
        className="max-w-[72ch] text-card-title leading-prose text-fg [&_a]:text-accent [&_a]:underline [&_a]:underline-offset-2 [&_blockquote]:my-3 [&_blockquote]:border-l [&_blockquote]:border-line-strong [&_blockquote]:pl-3 [&_blockquote]:text-muted [&_code]:rounded-xs [&_code]:bg-surface-glaze [&_code]:px-1 [&_code]:py-px [&_code]:text-fg-strong [&_code]:font-mono [&_code]:text-form-input [&_h1]:mt-5 [&_h1]:mb-2 [&_h1]:text-(length:--text-section-head) [&_h1]:font-medium [&_h1]:tracking-section-head [&_h1]:text-fg-strong [&_h2]:mt-5 [&_h2]:mb-2 [&_h2]:text-(length:--text-section-head) [&_h2]:font-medium [&_h2]:tracking-section-head [&_h2]:text-fg-strong [&_h3]:mt-4 [&_h3]:mb-1.5 [&_h3]:text-small-body [&_h3]:font-medium [&_h3]:text-fg-strong [&_h4]:mt-3 [&_h4]:mb-1 [&_h4]:text-form-input [&_h4]:font-medium [&_h4]:text-fg [&_h5]:mt-3 [&_h5]:mb-1 [&_h5]:text-form-label [&_h5]:font-medium [&_h5]:text-fg [&_h6]:mt-3 [&_h6]:mb-1 [&_h6]:text-form-label [&_h6]:font-medium [&_h6]:text-muted [&_hr]:my-4 [&_hr]:border-line [&_li]:my-0.5 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:my-2 [&_pre]:my-3 [&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:bg-canvas [&_pre]:p-3 [&_pre]:text-form-input [&_pre]:font-mono [&_pre_code]:bg-transparent [&_pre_code]:px-0 [&_strong]:text-fg-strong [&_table]:my-3 [&_table]:w-full [&_table]:border-collapse [&_th]:border-b [&_th]:border-line-strong [&_th]:px-2 [&_th]:py-1.5 [&_th]:text-left [&_th]:text-form-label [&_th]:font-medium [&_th]:text-fg-strong [&_td]:border-b [&_td]:border-line [&_td]:px-2 [&_td]:py-1.5 [&_td]:text-form-input [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5"
      >
        <Streamdown {...STREAMDOWN_SAFE_CONFIG}>{children}</Streamdown>
      </div>
    </section>
  );
}

export { DescriptionCard };
