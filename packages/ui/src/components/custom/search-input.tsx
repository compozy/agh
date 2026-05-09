"use client";

import { SearchIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export interface SearchInputProps extends Omit<
  React.ComponentProps<"input">,
  "onChange" | "value" | "size"
> {
  value?: string;
  onChange?: (next: string) => void;
  placeholder?: string;
  kbd?: React.ReactNode;
  containerClassName?: string;
}

/**
 * Search field , mirrors `.search-input` in
 * `docs/design/web-inspiration/styles/app.css`. Compact 28px row, panel-tone
 * surface, soft tertiary focus border (no accent ring), bordered kbd hint.
 */
function SearchInput({
  value,
  onChange,
  placeholder = "Search...",
  kbd,
  className,
  containerClassName,
  disabled,
  ...props
}: SearchInputProps) {
  const isControlled = value !== undefined;

  return (
    <div
      data-slot="search-input"
      data-disabled={disabled ? "true" : undefined}
      className={cn(
        "flex h-[28px] min-w-0 items-center gap-2 rounded-[7px] border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-2 text-[13px] text-[color:var(--color-text-primary)] transition-colors focus-within:border-[color:var(--color-text-tertiary)]",
        "data-[disabled=true]:cursor-not-allowed data-[disabled=true]:opacity-60",
        containerClassName
      )}
    >
      <SearchIcon
        aria-hidden="true"
        className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]"
      />
      <input
        type="search"
        data-slot="search-input-control"
        placeholder={placeholder}
        {...(isControlled ? { value } : {})}
        onChange={event => onChange?.(event.target.value)}
        disabled={disabled}
        className={cn(
          "min-w-0 flex-1 bg-transparent text-[13px] text-[color:var(--color-text-primary)] outline-none placeholder:text-[color:var(--color-text-tertiary)] disabled:cursor-not-allowed",
          className
        )}
        {...props}
      />
      {kbd ? (
        <span
          data-slot="search-input-kbd"
          aria-hidden="true"
          className="hidden items-center rounded-[4px] border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-1 py-px font-mono text-[9px] uppercase text-[color:var(--color-text-tertiary)] sm:inline-flex"
        >
          {kbd}
        </span>
      ) : null}
    </div>
  );
}

export { SearchInput };
