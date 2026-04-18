"use client";

import { BoxIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface EmptyProps extends Omit<React.ComponentProps<"div">, "title"> {
  icon?: IconComponent | React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
}

function isComponentType(value: unknown): value is IconComponent {
  if (typeof value === "function") return true;
  if (typeof value === "object" && value !== null && "render" in value) {
    // Lucide icons (and other forwardRef memoized components) expose a `render` fn
    // and can be rendered as JSX elements.
    return true;
  }
  return false;
}

/**
 * Empty state primitive — centered icon well + title + description + optional action.
 * Mirrors `Empty` in `docs/design/web-inspiration/src/primitives.jsx` and DESIGN.md §4 "Empty State".
 * `icon` accepts either a Lucide-style component reference or a pre-rendered ReactNode.
 */
function Empty({ icon, title, description, action, className, ...props }: EmptyProps) {
  let iconContent: React.ReactNode;
  if (icon === undefined) {
    iconContent = <BoxIcon className="size-5" />;
  } else if (isComponentType(icon)) {
    const IconComp = icon;
    iconContent = <IconComp className="size-5" />;
  } else {
    iconContent = icon;
  }

  return (
    <div
      data-slot="empty"
      className={cn(
        "flex w-full flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-6 py-10 text-center",
        className
      )}
      {...props}
    >
      <span
        aria-hidden="true"
        data-slot="empty-icon"
        className="inline-flex size-12 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-text-tertiary)]"
      >
        {iconContent}
      </span>
      <h3
        data-slot="empty-title"
        className="text-[15px] font-medium text-[color:var(--color-text-secondary)]"
      >
        {title}
      </h3>
      {description ? (
        <p
          data-slot="empty-description"
          className="max-w-md text-[13px] leading-relaxed text-[color:var(--color-text-tertiary)]"
        >
          {description}
        </p>
      ) : null}
      {action ? (
        <div
          data-slot="empty-action"
          className="mt-1 flex flex-wrap items-center justify-center gap-2"
        >
          {action}
        </div>
      ) : null}
    </div>
  );
}

export { Empty };
