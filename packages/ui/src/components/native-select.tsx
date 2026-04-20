import * as React from "react";
import { ChevronDownIcon } from "lucide-react";

import { cn } from "../lib/utils";

type NativeSelectProps = Omit<React.ComponentProps<"select">, "size"> & {
  size?: "sm" | "default";
};

function NativeSelect({ className, size = "default", ...props }: NativeSelectProps) {
  return (
    <div
      className={cn(
        "group/native-select relative w-fit has-[select:disabled]:opacity-50",
        className
      )}
      data-slot="native-select-wrapper"
      data-size={size}
    >
      <select
        data-slot="native-select"
        data-size={size}
        className="h-9 w-full min-w-0 appearance-none rounded-lg border border-input bg-[color:var(--color-surface-elevated)] py-0 pr-9 pl-3 text-sm text-[color:var(--color-text-primary)] transition-colors outline-none select-none selection:bg-[color:var(--color-accent-tint-strong)] selection:text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)] focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/20 disabled:pointer-events-none disabled:cursor-not-allowed disabled:border-[color:var(--color-surface-elevated)] disabled:bg-[color:var(--color-surface)] disabled:text-[color:var(--color-disabled)] disabled:opacity-100 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 data-[size=sm]:h-8 data-[size=sm]:rounded-[min(var(--radius-md),10px)] data-[size=sm]:pr-8 data-[size=sm]:pl-2.5"
        {...props}
      />
      <ChevronDownIcon
        className="pointer-events-none absolute top-1/2 right-2.5 size-4 -translate-y-1/2 text-muted-foreground select-none"
        aria-hidden="true"
        data-slot="native-select-icon"
      />
    </div>
  );
}

function NativeSelectOption({ ...props }: React.ComponentProps<"option">) {
  return <option data-slot="native-select-option" {...props} />;
}

function NativeSelectOptGroup({ className, ...props }: React.ComponentProps<"optgroup">) {
  return <optgroup data-slot="native-select-optgroup" className={cn(className)} {...props} />;
}

export { NativeSelect, NativeSelectOptGroup, NativeSelectOption };
