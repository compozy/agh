import * as React from "react";

import { cn } from "../lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex field-sizing-content min-h-16 w-full rounded-md border border-(--line) bg-(--elevated) px-2.5 py-2 text-[13px] text-(--fg) transition-colors outline-none placeholder:text-(--subtle) focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] focus-visible:border-(--line-strong) disabled:cursor-not-allowed disabled:bg-(--canvas) disabled:border-(--line-soft) disabled:opacity-50 aria-invalid:border-(--danger)",
        className
      )}
      {...props}
    />
  );
}

export { Textarea };
