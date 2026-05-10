"use client";

import { CheckIcon, CopyIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Button } from "../button";
import { Eyebrow } from "./eyebrow";

export interface CodeBlockProps extends Omit<React.ComponentProps<"div">, "children"> {
  code: string;
  language?: string;
  showPrompt?: boolean;
  copyable?: boolean;
  copyLabel?: string;
  copiedLabel?: string;
  tone?: CodeBlockTone;
  truncateLines?: number;
}

export type CodeBlockTone = "default" | "warning" | "danger" | "success" | "info" | "accent";

export interface CopyIconButtonProps extends Omit<React.ComponentProps<typeof Button>, "children"> {
  copiedLabel?: string;
  copyLabel?: string;
  value: string;
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
  tone = "default",
  truncateLines,
  className,
  ...props
}: CodeBlockProps) {
  const lines = React.useMemo(() => code.split("\n"), [code]);
  const displayLines = React.useMemo(() => {
    const seen = new Map<string, number>();
    return lines.map(line => {
      const count = seen.get(line) ?? 0;
      seen.set(line, count + 1);
      return { id: `${line || "blank"}-${count}`, line };
    });
  }, [lines]);
  const clampedLines =
    typeof truncateLines === "number" && Number.isFinite(truncateLines) && truncateLines > 0
      ? Math.floor(truncateLines)
      : undefined;

  return (
    <div
      data-slot="code-block"
      data-tone={tone}
      className={cn("relative rounded-lg bg-(--canvas)", codeBlockToneClass(tone), className)}
      {...props}
    >
      {language ? (
        <Eyebrow
          data-slot="code-block-language"
          case="upper"
          tone="subtle"
          className="absolute top-3 left-5"
        >
          {language}
        </Eyebrow>
      ) : null}
      {copyable ? (
        <CopyIconButton
          value={code}
          copyLabel={copyLabel}
          copiedLabel={copiedLabel}
          className="absolute top-2 right-2 text-(--subtle) hover:text-(--accent) data-[copied=true]:text-(--success)"
        />
      ) : null}
      <pre
        data-slot="code-block-pre"
        style={
          clampedLines ? ({ "--code-block-lines": clampedLines } as React.CSSProperties) : undefined
        }
        className={cn(
          "overflow-x-auto px-5 py-4 font-mono text-[14px] leading-[1.6] text-(--fg)",
          codeBlockToneTextClass(tone),
          language ? "pt-9" : null,
          copyable ? "pr-12" : null,
          clampedLines ? "max-h-[calc(var(--code-block-lines)*1.6em+2rem)] overflow-y-auto" : null
        )}
      >
        <code data-slot="code-block-code">
          {displayLines.map(({ id, line }) => {
            const withPrompt = showPrompt && shouldRenderPrompt(line);
            return (
              <span key={id} data-slot="code-block-line" className="block">
                {withPrompt ? (
                  <span
                    data-slot="code-block-prompt"
                    aria-hidden="true"
                    className="text-(--accent) select-none"
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

function CopyIconButton({
  className,
  copiedLabel = "Copied",
  copyLabel = "Copy to clipboard",
  value,
  variant = "ghost",
  size = "icon-xs",
  type = "button",
  ...props
}: CopyIconButtonProps) {
  const [copied, setCopied] = React.useState(false);
  const copyFeedbackTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  React.useEffect(
    () => () => {
      if (copyFeedbackTimerRef.current) {
        clearTimeout(copyFeedbackTimerRef.current);
      }
    },
    []
  );

  const handleCopy = React.useCallback(async () => {
    if (typeof navigator === "undefined" || !navigator.clipboard?.writeText) return;
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      if (copyFeedbackTimerRef.current) {
        clearTimeout(copyFeedbackTimerRef.current);
      }
      copyFeedbackTimerRef.current = setTimeout(() => setCopied(false), COPY_FEEDBACK_MS);
    } catch {
      // Some browsers block clipboard access in insecure contexts.
    }
  }, [value]);

  return (
    <Button
      type={type}
      variant={variant}
      size={size}
      aria-label={copied ? copiedLabel : copyLabel}
      data-slot="code-block-copy"
      data-copied={copied ? "true" : undefined}
      onClick={event => {
        props.onClick?.(event);
        if (!event.defaultPrevented) void handleCopy();
      }}
      className={className}
      {...props}
    >
      {copied ? <CheckIcon className="size-3" /> : <CopyIcon className="size-3" />}
    </Button>
  );
}

function codeBlockToneClass(tone: CodeBlockTone): string {
  switch (tone) {
    case "warning":
      return "ring-1 ring-(--warning)/35 bg-(--warning-tint)";
    case "danger":
      return "ring-1 ring-(--danger)/35 bg-(--danger-tint)";
    case "success":
      return "ring-1 ring-(--success)/35 bg-(--success-tint)";
    case "info":
      return "ring-1 ring-(--info)/35 bg-(--info-tint)";
    case "accent":
      return "ring-1 ring-(--accent)/35 bg-(--accent-tint)";
    default:
      return "";
  }
}

function codeBlockToneTextClass(tone: CodeBlockTone): string {
  switch (tone) {
    case "warning":
      return "text-(--warning)";
    case "danger":
      return "text-(--danger)";
    case "success":
      return "text-(--success)";
    case "info":
      return "text-(--info)";
    case "accent":
      return "text-(--accent)";
    default:
      return "";
  }
}

function shouldRenderPrompt(line: string): boolean {
  if (line.length === 0) return false;
  if (/^\s/.test(line)) return false;
  if (line.startsWith("#")) return false;
  return true;
}

export { CodeBlock, CopyIconButton };
