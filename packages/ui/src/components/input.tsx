import type * as React from "react";
import { Input as InputPrimitive } from "@base-ui/react/input";

import { cn } from "../lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-md border border-(--line) bg-(--elevated) px-3 py-0 text-[13px] text-(--fg) transition-colors outline-none selection:bg-(--accent-tint-strong) selection:text-(--fg) file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-[13px] file:font-[510] file:text-(--fg) placeholder:text-(--subtle) focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] focus-visible:border-(--line-strong) disabled:pointer-events-none disabled:cursor-not-allowed disabled:border-(--line-soft) disabled:bg-(--canvas) disabled:text-(--disabled) disabled:opacity-100 aria-invalid:border-(--danger) aria-invalid:shadow-none",
        className
      )}
      {...props}
    />
  );
}

export { Input };
