"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface FieldRowProps extends React.ComponentProps<"div"> {
  label: React.ReactNode;
  description?: React.ReactNode;
  /** Optional explicit ID linking the label and the rendered control. */
  htmlFor?: string;
  /** Layout: stacked label-on-top (default) or two-column with label gutter. */
  layout?: "stacked" | "two-column";
  control: React.ReactNode;
}

function FieldRow({
  label,
  description,
  htmlFor,
  layout = "stacked",
  control,
  className,
  ...props
}: FieldRowProps) {
  return (
    <div
      data-slot="field-row"
      data-layout={layout}
      className={cn(
        layout === "two-column"
          ? "grid grid-cols-[minmax(0,180px)_1fr] items-start gap-x-4 gap-y-1"
          : "flex min-w-0 flex-col gap-1.5",
        className
      )}
      {...props}
    >
      <label
        data-slot="field-row-label"
        htmlFor={htmlFor}
        className="text-[12px] font-medium tracking-[-0.005em] text-(--fg)"
      >
        {label}
      </label>
      <div
        data-slot="field-row-control"
        className={cn("min-w-0", layout === "two-column" && "row-span-2 col-start-2 row-start-1")}
      >
        {control}
      </div>
      {description ? (
        <p
          data-slot="field-row-description"
          className={cn(
            "text-[12px] text-(--muted)",
            layout === "two-column" && "col-start-1 row-start-2"
          )}
        >
          {description}
        </p>
      ) : null}
    </div>
  );
}

export { FieldRow };
