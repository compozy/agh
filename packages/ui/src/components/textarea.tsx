import * as React from "react";

import { cn } from "../lib/utils";

export type TextareaVariant = "default" | "mono";

export interface TextareaProps extends React.ComponentProps<"textarea"> {
  /** `mono` switches to `font-mono` + 12 px wave-2 / analysis §4. */
  variant?: TextareaVariant;
}

const VARIANT_CLASSNAME: Record<TextareaVariant, string> = {
  default: "text-[13px]",
  mono: "font-mono text-[12px]",
};

function Textarea({ className, variant = "default", ...props }: TextareaProps) {
  return (
    <textarea
      data-slot="textarea"
      data-variant={variant}
      className={cn(
        "flex field-sizing-content min-h-16 w-full rounded-md border border-line bg-elevated px-2.5 py-2 text-fg transition-colors outline-none placeholder:text-subtle focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] focus-visible:border-line-strong disabled:cursor-not-allowed disabled:bg-canvas disabled:border-line-soft disabled:opacity-50 aria-invalid:border-danger",
        VARIANT_CLASSNAME[variant],
        className
      )}
      {...props}
    />
  );
}

export { Textarea };
