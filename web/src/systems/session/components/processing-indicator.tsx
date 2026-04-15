import { useEffect, useState } from "react";

export interface ProcessingIndicatorProps {
  className?: string;
}

function formatElapsed(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return s > 0 ? `${m}m ${s}s` : `${m}m`;
}

function WorkingTimer() {
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
        <span className="inline-flex items-center gap-[3px]">
          <span className="size-1 rounded-full bg-[color:var(--color-text-tertiary)]/30 animate-pulse" />
          <span className="size-1 rounded-full bg-[color:var(--color-text-tertiary)]/30 animate-pulse [animation-delay:200ms]" />
          <span className="size-1 rounded-full bg-[color:var(--color-text-tertiary)]/30 animate-pulse [animation-delay:400ms]" />
        </span>
        <span className="text-[11px] text-[color:var(--color-text-tertiary)]/70">
          Working for <WorkingTimer />
        </span>
      </div>
    </div>
  );
}
