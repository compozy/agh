import type * as React from "react";
import { Input as InputPrimitive } from "@base-ui/react/input";

import { cn } from "../lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        "h-8 w-full min-w-0 rounded-md border border-(--line) bg-(--canvas-soft) px-3 py-0 text-sm text-(--fg) transition-colors outline-none selection:bg-(--accent-tint-strong) selection:text-(--fg) file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-(--fg) placeholder:text-(--subtle) focus-visible:border-(--line-strong) disabled:pointer-events-none disabled:cursor-not-allowed disabled:border-(--canvas-soft) disabled:bg-(--canvas-soft) disabled:text-(--disabled) disabled:opacity-100 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20",
        className
      )}
      {...props}
    />
  );
}

export { Input };
