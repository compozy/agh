"use client";

import { cn } from "@agh/ui";
import { useReducer } from "react";

export type SortKey = "newest" | "oldest";

export interface SortPillGroupProps {
  defaultValue?: SortKey;
  onChange?: (key: SortKey) => void;
  className?: string;
}

const labels: Record<SortKey, string> = { newest: "NEWEST", oldest: "OLDEST" };

function activeSortReducer(_: SortKey, next: SortKey): SortKey {
  return next;
}

export function SortPillGroup({
  defaultValue = "newest",
  onChange,
  className,
}: SortPillGroupProps) {
  const [active, setActive] = useReducer(activeSortReducer, defaultValue);
  return (
    <div
      className={cn(
        "inline-flex items-center gap-0.5 rounded-full border border-(--line) bg-(--elevated) p-0.5",
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
            "h-5 rounded-full px-2 font-mono text-badge font-semibold tracking-badge transition-colors",
            active === key ? "bg-accent text-(--accent-ink)" : "text-(--muted) hover:text-(--fg)"
          )}
        >
          {labels[key]}
        </button>
      ))}
    </div>
  );
}
