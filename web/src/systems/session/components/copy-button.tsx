import { useCallback, useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";

import { cn } from "@/lib/utils";

const COPY_RESET_MS = 1200;

export interface CopyButtonProps {
  ariaLabel: string;
  className?: string;
  text: string;
}

export function CopyButton({ ariaLabel, className, text }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => clearTimeout(timerRef.current);
  }, []);

  const handleCopy = useCallback(async () => {
    clearTimeout(timerRef.current);

    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      timerRef.current = setTimeout(() => setCopied(false), COPY_RESET_MS);
    } catch (error) {
      setCopied(false);
      console.error("Failed to copy text to clipboard", error);
    }
  }, [text]);

  return (
    <button
      type="button"
      onClick={() => void handleCopy()}
      className={cn(className)}
      aria-label={ariaLabel}
      data-state={copied ? "copied" : "idle"}
    >
      {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
    </button>
  );
}
