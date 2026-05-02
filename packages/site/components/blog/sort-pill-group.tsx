"use client";

import { cn } from "@agh/ui";
import { useState } from "react";

export type SortKey = "newest" | "oldest";

export interface SortPillGroupProps {
  initial?: SortKey;
  onChange?: (key: SortKey) => void;
  className?: string;
}

const labels: Record<SortKey, string> = { newest: "NEWEST", oldest: "OLDEST" };

export function SortPillGroup({ initial = "newest", onChange, className }: SortPillGroupProps) {
  const [active, setActive] = useState<SortKey>(initial);
  return (
    <div
      className={cn(
        "inline-flex items-center gap-0.5 rounded-full border border-(--color-divider) bg-(--color-surface-elevated) p-0.5",
        className
      )}
    >
      {(Object.keys(labels) as SortKey[]).map(key => (
        <button
          key={key}
          type="button"
          aria-pressed={active === key}
          onClick={() => {
            setActive(key);
            onChange?.(key);
          }}
          className={cn(
            "h-5 rounded-full px-2 font-mono text-[10px] font-semibold tracking-[0.08em] transition-colors",
            active === key
              ? "bg-(--color-accent-tint) text-(--color-accent)"
              : "text-(--color-text-tertiary) hover:text-(--color-text-primary)"
          )}
        >
          {labels[key]}
        </button>
      ))}
    </div>
  );
}
