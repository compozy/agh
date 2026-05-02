"use client";

import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Check, Copy } from "lucide-react";
import { Button } from "@agh/ui";
import { cn } from "@agh/ui/utils";

interface CodeBlockProps {
  code: string;
  language?: string;
  copyable?: boolean;
  caption?: string;
  className?: string;
  /** Prefix each line with a shell prompt. */
  shell?: boolean;
}

export function CodeBlock({
  code,
  language,
  copyable = true,
  caption,
  className,
  shell = false,
}: CodeBlockProps) {
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current);
      }
    };
  }, []);

  function scheduleReset() {
    if (resetTimerRef.current) {
      clearTimeout(resetTimerRef.current);
    }
    resetTimerRef.current = setTimeout(() => setCopyState("idle"), 1500);
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopyState("copied");
      scheduleReset();
    } catch {
      setCopyState("failed");
      scheduleReset();
    }
  }

  const lines = code.split("\n");

  return (
    <div
      className={cn(
        "min-w-0 max-w-full overflow-hidden rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-canvas-deep)",
        className
      )}
    >
      {(caption || language || copyable) && (
        <div className="flex min-w-0 items-start justify-between gap-3 border-b border-(--color-divider) px-4 py-2.5">
          <span className="min-w-0 font-mono text-[10px] leading-relaxed font-medium uppercase tracking-(--tracking-mono) text-(--color-text-tertiary) [overflow-wrap:anywhere]">
            {caption ?? language ?? "shell"}
          </span>
          {copyable ? (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={handleCopy}
              aria-label={
                copyState === "copied"
                  ? "Copied"
                  : copyState === "failed"
                    ? "Copy failed"
                    : "Copy to clipboard"
              }
              aria-live="polite"
              className={cn(
                "text-(--color-text-tertiary) hover:text-(--color-accent)",
                copyState === "failed" && "text-(--color-danger) hover:text-(--color-danger)"
              )}
            >
              {copyState === "copied" ? (
                <Check aria-hidden className="size-3" />
              ) : copyState === "failed" ? (
                <AlertTriangle aria-hidden className="size-3" />
              ) : (
                <Copy aria-hidden className="size-3" />
              )}
            </Button>
          ) : null}
        </div>
      )}
      <pre className="overflow-x-auto px-4 py-4 font-mono text-[13px] leading-[1.7] text-(--color-text-primary)">
        <code>
          {lines.map((line, i) => (
            <div key={i} className={line === "" ? "h-[1.1em]" : undefined}>
              {shell && line !== "" && !line.startsWith("#") ? (
                <span className="select-none text-(--color-accent)">$ </span>
              ) : null}
              {line.startsWith("#") ? (
                <span className="text-(--color-text-tertiary)">{line}</span>
              ) : (
                line
              )}
            </div>
          ))}
        </code>
      </pre>
    </div>
  );
}
