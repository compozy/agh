"use client";

import { BoxIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;
type EmptyTitleTag = "div" | "h1" | "h2" | "h3" | "h4" | "h5" | "h6" | "p" | "span";

export interface EmptyProps extends Omit<React.ComponentProps<"div">, "title"> {
  icon?: IconComponent | React.ReactNode;
  title: React.ReactNode;
  titleAs?: EmptyTitleTag;
  description?: React.ReactNode;
  action?: React.ReactNode;
  fill?: boolean;
}

function isComponentType(value: unknown): value is IconComponent {
  if (typeof value === "function") return true;
  if (typeof value === "object" && value !== null && "render" in value) {
    return true;
  }
  return false;
}

function resolveTitleTag(title: React.ReactNode): EmptyTitleTag {
  return typeof title === "string" || typeof title === "number" ? "h3" : "div";
}

function Empty({
  icon,
  title,
  titleAs,
  description,
  action,
  fill = true,
  className,
  ...props
}: EmptyProps) {
  let iconContent: React.ReactNode;
  if (icon === undefined) {
    iconContent = <BoxIcon className="size-4" />;
  } else if (isComponentType(icon)) {
    const IconComp = icon;
    iconContent = <IconComp className="size-4" />;
  } else {
    iconContent = icon;
  }

  const TitleTag = titleAs ?? resolveTitleTag(title);

  return (
    <div
      data-slot="empty"
      data-fill={fill ? "true" : "false"}
      className={cn(
        "flex w-full flex-col items-center justify-center gap-3 rounded-lg text-center",
        fill && "h-full min-h-0 flex-1",
        className
      )}
      {...props}
    >
      <span
        aria-hidden="true"
        data-slot="empty-icon"
        className="inline-flex size-empty-icon items-center justify-center rounded-lg bg-canvas-soft text-subtle"
      >
        {iconContent}
      </span>
      <TitleTag
        data-slot="empty-title"
        className="text-lg font-medium leading-snug tracking-empty-h1 text-fg-strong"
      >
        {title}
      </TitleTag>
      {description ? (
        <p
          data-slot="empty-description"
          className="max-w-md text-small-body leading-relaxed text-muted"
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
