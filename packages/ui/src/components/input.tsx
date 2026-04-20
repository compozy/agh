import type * as React from "react";
import { Input as InputPrimitive } from "@base-ui/react/input";

import { cn } from "../lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-lg border border-input bg-[color:var(--color-surface-elevated)] px-3 py-0 text-sm text-[color:var(--color-text-primary)] transition-colors outline-none selection:bg-[color:var(--color-accent-tint-strong)] selection:text-[color:var(--color-text-primary)] file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-[color:var(--color-text-tertiary)] focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/20 disabled:pointer-events-none disabled:cursor-not-allowed disabled:border-[color:var(--color-surface-elevated)] disabled:bg-[color:var(--color-surface)] disabled:text-[color:var(--color-disabled)] disabled:opacity-100 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20",
        className
      )}
      {...props}
    />
  );
}

export { Input };
