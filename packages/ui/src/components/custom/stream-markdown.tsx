"use client";

import * as React from "react";
import { Streamdown, type Components } from "streamdown";

import { normalizeAghCodeLanguage } from "../../lib/code-theme";
import { cn } from "../../lib/utils";
import { CodeBlock } from "./code-block";
import { STREAMDOWN_SAFE_CONFIG } from "./description-card";

type StreamMarkdownCodeProps = React.ComponentProps<"code"> & {
  node?: unknown;
  inline?: boolean;
  metastring?: string;
  "data-block"?: unknown;
};

function extractCodeLanguage(className?: string): string {
  const match = /language-([-\w]+)/.exec(className ?? "");
  return match?.[1]?.toLowerCase() ?? "";
}

function toCodeString(children: React.ReactNode): string {
  return React.Children.toArray(children).join("").replace(/\n$/, "");
}

function StreamMarkdownCode({
  children,
  className,
  node: _node,
  inline: _inline,
  metastring: _metastring,
  "data-block": dataBlock,
  ...props
}: StreamMarkdownCodeProps) {
  const code = toCodeString(children);
  const rawLanguage = extractCodeLanguage(className);
  const isBlock = dataBlock !== undefined || rawLanguage !== "" || code.includes("\n");

  if (isBlock) {
    const normalizedLanguage = normalizeAghCodeLanguage(rawLanguage);
    return (
      <CodeBlock
        code={code}
        language={rawLanguage || undefined}
        caption={rawLanguage ? (normalizedLanguage ?? rawLanguage) : undefined}
        showPrompt={false}
        copyable
        className="my-2"
      />
    );
  }

  return (
    <code
      className={cn(
        "rounded-md bg-elevated px-1.5 py-0.5 font-mono",
        "text-small-body text-fg",
        className
      )}
      {...props}
    >
      {children}
    </code>
  );
}

const STREAM_MARKDOWN_COMPONENTS: Components = {
  ...(STREAMDOWN_SAFE_CONFIG.components as Components),
  code: StreamMarkdownCode,
  inlineCode: StreamMarkdownCode,
};

export interface StreamMarkdownProps extends Omit<React.ComponentProps<"div">, "children"> {
  children: string;
  streaming?: boolean;
}

function StreamMarkdown({ children, streaming = false, className, ...props }: StreamMarkdownProps) {
  return (
    <div
      data-slot="stream-markdown"
      className={cn(
        "max-w-[72ch] text-card-title leading-prose text-fg",
        "[&_a]:text-accent [&_a]:underline [&_a]:underline-offset-2",
        "[&_blockquote]:my-3 [&_blockquote]:border-l [&_blockquote]:border-line-strong",
        "[&_blockquote]:pl-3 [&_blockquote]:text-muted",
        "[&_h1]:mt-5 [&_h1]:mb-2 [&_h1]:text-(length:--text-section-head)",
        "[&_h1]:font-medium [&_h1]:tracking-section-head [&_h1]:text-fg-strong",
        "[&_h2]:mt-5 [&_h2]:mb-2 [&_h2]:text-(length:--text-section-head)",
        "[&_h2]:font-medium [&_h2]:tracking-section-head [&_h2]:text-fg-strong",
        "[&_h3]:mt-4 [&_h3]:mb-1.5 [&_h3]:text-small-body",
        "[&_h3]:font-medium [&_h3]:text-fg-strong",
        "[&_h4]:mt-3 [&_h4]:mb-1 [&_h4]:text-form-input [&_h4]:font-medium [&_h4]:text-fg",
        "[&_h5]:mt-3 [&_h5]:mb-1 [&_h5]:text-form-label [&_h5]:font-medium [&_h5]:text-fg",
        "[&_h6]:mt-3 [&_h6]:mb-1 [&_h6]:text-form-label [&_h6]:font-medium [&_h6]:text-muted",
        "[&_hr]:my-4 [&_hr]:border-line",
        "[&_li]:my-0.5 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5",
        "[&_p]:my-2 [&_pre]:my-3 [&_strong]:text-fg-strong",
        "[&_table]:my-3 [&_table]:w-full [&_table]:border-collapse",
        "[&_th]:border-b [&_th]:border-line-strong [&_th]:px-2 [&_th]:py-1.5",
        "[&_th]:text-left [&_th]:text-form-label [&_th]:font-medium [&_th]:text-fg-strong",
        "[&_td]:border-b [&_td]:border-line [&_td]:px-2 [&_td]:py-1.5 [&_td]:text-form-input",
        "[&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5",
        className
      )}
      {...props}
    >
      <Streamdown
        {...STREAMDOWN_SAFE_CONFIG}
        components={STREAM_MARKDOWN_COMPONENTS}
        mode={streaming ? "streaming" : "static"}
        parseIncompleteMarkdown
        lineNumbers={false}
      >
        {children}
      </Streamdown>
    </div>
  );
}

export { StreamMarkdown };
