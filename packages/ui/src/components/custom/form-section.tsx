"use client";

import type { LucideIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";

export type FormSectionSize = "comfortable" | "compact";

export interface FormSectionProps extends Omit<React.ComponentProps<"section">, "title"> {
  /** Section title (rendered at 13/510/-0.008em via the section-head token). */
  title: React.ReactNode;
  /** Optional leading icon, painted `--subtle` at 14 px with stroke 1.75. */
  icon?: LucideIcon;
  /** Optional right-aligned eyebrow at 11 px (counts, status, hints). */
  rightLabel?: React.ReactNode;
  /** Optional description rendered between the head and the body. */
  description?: React.ReactNode;
  /** Padding/density `compact` shrinks the padding for dialog hosts. */
  size?: FormSectionSize;
  children: React.ReactNode;
}

const SIZE_CLASSNAME: Record<FormSectionSize, string> = {
  comfortable: "px-5 py-[18px]",
  compact: "px-4 py-[14px]",
};

function FormSection({
  title,
  icon: Icon,
  rightLabel,
  description,
  size = "comfortable",
  className,
  children,
  ...props
}: FormSectionProps) {
  return (
    <section
      data-slot="form-section"
      data-size={size}
      className={cn(
        "flex flex-col rounded-(--radius-lg) bg-(--canvas-soft)",
        SIZE_CLASSNAME[size],
        className
      )}
      {...props}
    >
      <header data-slot="form-section-head" className="mb-4 flex min-w-0 items-center gap-2">
        {Icon ? (
          <span
            aria-hidden="true"
            data-slot="form-section-icon"
            className="inline-flex size-[14px] shrink-0 items-center justify-center text-(--subtle)"
          >
            <Icon width={14} height={14} strokeWidth={1.75} />
          </span>
        ) : null}
        <h3
          data-slot="form-section-title"
          className="min-w-0 truncate text-[length:var(--text-section-head)] tracking-(--tracking-section-head) text-(--fg-strong)"
          style={{ fontWeight: 510 }}
        >
          {title}
        </h3>
        {rightLabel ? (
          <Eyebrow
            data-slot="form-section-right-label"
            className="ml-auto min-w-0 truncate text-(--muted)"
          >
            {rightLabel}
          </Eyebrow>
        ) : null}
      </header>
      {description ? (
        <p data-slot="form-section-description" className="mb-3 text-[12px] text-(--muted)">
          {description}
        </p>
      ) : null}
      <div data-slot="form-section-body" className="flex min-w-0 flex-col gap-[14px] [&>*+*]:mt-0">
        {children}
      </div>
    </section>
  );
}

export { FormSection };
