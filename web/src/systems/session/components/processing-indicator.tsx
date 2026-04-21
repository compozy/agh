import { useEffect, useState } from "react";

import { StatusDot } from "@agh/ui";

export interface ProcessingIndicatorProps {
  className?: string;
}

function formatElapsed(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
}

function ThinkingTimer() {
  const [startMs] = useState(() => Date.now());
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    const id = setInterval(() => {
      setElapsed(Math.floor((Date.now() - startMs) / 1000));
    }, 1000);

    return () => clearInterval(id);
  }, [startMs]);

  return <>{formatElapsed(elapsed)}</>;
}

export function ProcessingIndicator({ className }: ProcessingIndicatorProps) {
  return (
    <div className={className} data-testid="processing-indicator">
      <div className="flex items-center gap-2 py-2 pl-5">
        <span className="inline-flex items-center gap-1">
          <StatusDot size="sm" tone="accent" pulse className="opacity-80" />
          <StatusDot size="sm" tone="accent" pulse className="opacity-60 [animation-delay:200ms]" />
          <StatusDot size="sm" tone="accent" pulse className="opacity-40 [animation-delay:400ms]" />
        </span>
        <span className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
          Thinking...{" "}
          <span className="text-[color:var(--color-text-label)]">
            · <ThinkingTimer />
          </span>
        </span>
      </div>
    </div>
  );
}
