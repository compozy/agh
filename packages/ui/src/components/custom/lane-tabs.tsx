"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface LaneTabsItem<T extends string> {
  value: T;
  label: React.ReactNode;
  count?: number | string;
}

export interface LaneTabsProps<T extends string> extends Omit<
  React.ComponentProps<"div">,
  "onChange"
> {
  items: ReadonlyArray<LaneTabsItem<T>>;
  value: T;
  onChange: (next: T) => void;
  ariaLabel?: string;
}

function LaneTabs<T extends string>({
  items,
  value,
  onChange,
  ariaLabel,
  className,
  ...props
}: LaneTabsProps<T>) {
  const refs = React.useRef<Map<T, HTMLButtonElement | null>>(new Map());

  const focusItem = React.useCallback((next: T) => {
    refs.current.get(next)?.focus();
  }, []);

  const onKeyDown = React.useCallback(
    (event: React.KeyboardEvent<HTMLButtonElement>) => {
      const index = items.findIndex(item => item.value === value);
      if (index < 0) return;
      let nextIndex: number | null = null;
      switch (event.key) {
        case "ArrowLeft":
          nextIndex = (index - 1 + items.length) % items.length;
          break;
        case "ArrowRight":
          nextIndex = (index + 1) % items.length;
          break;
        case "Home":
          nextIndex = 0;
          break;
        case "End":
          nextIndex = items.length - 1;
          break;
        default:
          return;
      }
      event.preventDefault();
      const nextItem = items[nextIndex];
      onChange(nextItem.value);
      requestAnimationFrame(() => focusItem(nextItem.value));
    },
    [focusItem, items, onChange, value]
  );

  return (
    <div
      data-slot="lane-tabs"
      role="tablist"
      aria-label={ariaLabel}
      className={cn("inline-flex items-center gap-1 border-b border-(--line)", className)}
      {...props}
    >
      {items.map(item => {
        const isActive = item.value === value;
        const hasCount = item.count !== undefined && item.count !== null && item.count !== "";
        return (
          <button
            key={item.value}
            ref={node => {
              refs.current.set(item.value, node);
            }}
            type="button"
            role="tab"
            aria-selected={isActive}
            aria-current={isActive ? "page" : undefined}
            data-slot="lane-tabs-item"
            data-value={item.value}
            data-active={isActive ? "true" : undefined}
            tabIndex={isActive ? 0 : -1}
            onClick={() => onChange(item.value)}
            onKeyDown={onKeyDown}
            className={cn(
              "relative inline-flex h-9 items-center gap-1.5 px-2 text-[12px] font-medium tracking-[-0.005em] transition-colors duration-(--dur) ease-(--ease) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--line-strong) focus-visible:ring-offset-0",
              isActive
                ? "text-(--fg-strong) after:absolute after:right-0 after:bottom-[-1px] after:left-0 after:h-[1.5px] after:bg-(--accent)"
                : "text-(--muted) hover:text-(--fg)"
            )}
          >
            <span>{item.label}</span>
            {hasCount ? (
              <span
                data-slot="lane-tabs-count"
                className="inline-flex h-[17px] min-w-[17px] items-center justify-center rounded-(--radius-mono-badge) bg-(--canvas-soft) px-1 font-mono text-[10px] font-medium tabular-nums text-(--muted)"
              >
                {item.count}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}

export { LaneTabs };
