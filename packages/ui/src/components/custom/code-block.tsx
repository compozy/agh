"use client";

import { CheckIcon, CopyIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Button } from "../button";

export interface CodeBlockProps extends Omit<React.ComponentProps<"div">, "children"> {
  code: string;
  language?: string;
  showPrompt?: boolean;
  copyable?: boolean;
  copyLabel?: string;
  copiedLabel?: string;
}

const COPY_FEEDBACK_MS = 1500;

/**
 * Terminal-style code block per DESIGN.md §4. Canvas-deep container, JetBrains
 * Mono body at 14px/1.6, optional accent `$ ` prompt, optional language eyebrow,
 * and a ghost copy button that flips to a success checkmark for 1.5s on copy.
 */
function CodeBlock({
  code,
  language,
  showPrompt = true,
  copyable = true,
  copyLabel = "Copy to clipboard",
  copiedLabel = "Copied",
  className,
  ...props
}: CodeBlockProps) {
  const [copied, setCopied] = React.useState(false);
  const [copyFeedbackKey, setCopyFeedbackKey] = React.useState(0);

  React.useEffect(() => {
    if (copyFeedbackKey === 0) return;
    const timer = setTimeout(() => setCopied(false), COPY_FEEDBACK_MS);
    return () => clearTimeout(timer);
  }, [copyFeedbackKey]);

  const handleCopy = React.useCallback(async () => {
    if (typeof navigator === "undefined" || !navigator.clipboard?.writeText) return;
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setCopyFeedbackKey(value => value + 1);
    } catch {
      // Silently ignore — some browsers block clipboard access in insecure contexts.
    }
  }, [code]);

  const lines = React.useMemo(() => code.split("\n"), [code]);

  return (
    <div
      data-slot="code-block"
      className={cn(
        "relative rounded-[var(--radius-diagram)] bg-[color:var(--color-canvas-deep)]",
        className
      )}
      {...props}
    >
      {language ? (
        <span
          data-slot="code-block-language"
          className="absolute top-3 left-5 font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
        >
          {language}
        </span>
      ) : null}
      {copyable ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={copied ? copiedLabel : copyLabel}
          data-slot="code-block-copy"
          data-copied={copied ? "true" : undefined}
          onClick={handleCopy}
          className="absolute top-2 right-2 text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-accent)] data-[copied=true]:text-[color:var(--color-success)]"
        >
          {copied ? <CheckIcon className="size-3" /> : <CopyIcon className="size-3" />}
        </Button>
      ) : null}
      <pre
        data-slot="code-block-pre"
        className={cn(
          "overflow-x-auto px-5 py-4 font-mono text-[14px] leading-[1.6] text-[color:var(--color-text-primary)]",
          language ? "pt-9" : null,
          copyable ? "pr-12" : null
        )}
      >
        <code data-slot="code-block-code">
          {lines.map((line, index) => {
            const withPrompt = showPrompt && shouldRenderPrompt(line);
            return (
              <span key={`${index}-${line}`} data-slot="code-block-line" className="block">
                {withPrompt ? (
                  <span
                    data-slot="code-block-prompt"
                    aria-hidden="true"
                    className="text-[color:var(--color-accent)] select-none"
                  >
                    {"$ "}
                  </span>
                ) : null}
                {line.length > 0 ? line : "\u00A0"}
              </span>
            );
          })}
        </code>
      </pre>
    </div>
  );
}

function shouldRenderPrompt(line: string): boolean {
  if (line.length === 0) return false;
  if (/^\s/.test(line)) return false;
  if (line.startsWith("#")) return false;
  return true;
}

export { CodeBlock };
