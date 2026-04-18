"use client";

import { SearchIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../lib/utils";

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
 * Search field — matches `SearchInput` in `docs/design/web-inspiration/src/primitives.jsx`.
 * Standard 36px row, search glyph on the left, optional kbd hint on the right.
 */
function SearchInput({
  value,
  onChange,
  placeholder = "Search…",
  kbd,
  className,
  containerClassName,
  disabled,
  ...props
}: SearchInputProps) {
  return (
    <div
      data-slot="search-input"
      data-disabled={disabled ? "true" : undefined}
      className={cn(
        "flex h-9 min-w-0 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 text-[13px] text-[color:var(--color-text-primary)] transition-colors focus-within:border-[color:var(--color-accent)] focus-within:ring-1 focus-within:ring-[color:var(--color-accent)]",
        "data-[disabled=true]:cursor-not-allowed data-[disabled=true]:opacity-60",
        containerClassName
      )}
    >
      <SearchIcon
        aria-hidden="true"
        className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
      />
      <input
        type="search"
        data-slot="search-input-control"
        placeholder={placeholder}
        value={value ?? ""}
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
          className="hidden items-center gap-1 font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)] sm:inline-flex"
        >
          {kbd}
        </span>
      ) : null}
    </div>
  );
}

export { SearchInput };
