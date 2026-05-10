"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface MetadataTileProps extends React.ComponentProps<"div"> {
  label: React.ReactNode;
  value: React.ReactNode;
  detail?: React.ReactNode;
  icon?: IconComponent;
  /** Casing for the label; mono UPPERCASE by default. */
  labelCase?: "sentence" | "upper";
}

function MetadataTile({
  label,
  value,
  detail,
  icon: Icon,
  labelCase = "upper",
  className,
  ...props
}: MetadataTileProps) {
  return (
    <div
      data-slot="metadata-tile"
      className={cn(
        "flex min-w-0 flex-col gap-1 rounded-(--radius) border border-(--line) bg-(--canvas-soft) px-3 py-2.5",
        className
      )}
      {...props}
    >
      <div data-slot="metadata-tile-head" className="flex min-w-0 items-center gap-1.5">
        {Icon ? <Icon aria-hidden="true" className="size-3 shrink-0 text-(--muted)" /> : null}
        <span
          data-slot="metadata-tile-label"
          data-case={labelCase}
          className={cn(
            "min-w-0 truncate",
            labelCase === "upper"
              ? "font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
              : "text-[12px] font-medium text-(--muted)"
          )}
        >
          {label}
        </span>
      </div>
      <div
        data-slot="metadata-tile-value"
        className="truncate text-[13px] font-medium text-(--fg) tabular-nums"
      >
        {value}
      </div>
      {detail ? (
        <p data-slot="metadata-tile-detail" className="truncate text-[11.5px] text-(--subtle)">
          {detail}
        </p>
      ) : null}
    </div>
  );
}

export { MetadataTile };
