"use client";

import { cn } from "@agh/ui";
import { Check, Copy } from "lucide-react";
import { useCallback, useRef, useState, type ComponentProps } from "react";

export interface CodeBlockProps extends ComponentProps<"pre"> {
  "data-language"?: string;
}

export function CodeBlock({ className, children, ...props }: CodeBlockProps) {
  const ref = useRef<HTMLPreElement>(null);
  const [copied, setCopied] = useState(false);

  const onCopy = useCallback(() => {
    const text = ref.current?.textContent ?? "";
    if (!text) return;
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      })
      .catch(() => {});
  }, []);

  return (
    <div className="group relative mt-7 overflow-hidden rounded-xl border border-(--color-divider) bg-(--color-canvas-deep)">
      <button
        type="button"
        onClick={onCopy}
        aria-label={copied ? "Copied" : "Copy code"}
        className="absolute right-2 top-2 inline-flex h-7 w-7 items-center justify-center rounded-md text-(--color-text-tertiary) opacity-0 transition-opacity hover:text-(--color-text-primary) focus-visible:opacity-100 group-hover:opacity-100"
      >
        {copied ? <Check size={13} aria-hidden /> : <Copy size={13} aria-hidden />}
      </button>
      <pre
        ref={ref}
        {...props}
        className={cn(
          "overflow-x-auto px-5 py-4 font-mono text-[13px] leading-[1.7] text-(--color-text-primary)",
          className
        )}
      >
        {children}
      </pre>
    </div>
  );
}
