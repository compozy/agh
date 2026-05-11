"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export type RouteStateMode = "loading" | "empty" | "error";

export interface RouteStateProps extends Omit<React.ComponentProps<"div">, "title"> {
  mode: RouteStateMode;
  title?: React.ReactNode;
  message?: React.ReactNode;
  icon?: IconComponent;
  action?: React.ReactNode;
  /** Optional cause node rendered for `mode="error"`. Stack traces are not auto-rendered. */
  cause?: React.ReactNode;
}

function RouteState({
  mode,
  title,
  message,
  icon: Icon,
  action,
  cause,
  className,
  children,
  ...props
}: RouteStateProps) {
  const isLoading = mode === "loading";
  const ariaProps = isLoading ? { role: "status", "aria-live": "polite" as const } : {};
  const loadingLabel = title ?? "Loading";
  return (
    <div
      data-slot="route-state"
      data-mode={mode}
      className={cn(
        "flex min-h-[160px] flex-col items-center justify-center gap-3 rounded-lg border border-line bg-canvas-soft px-6 py-8 text-center",
        className
      )}
      {...ariaProps}
      {...props}
    >
      {Icon ? (
        <span
          aria-hidden="true"
          data-slot="route-state-icon"
          className="inline-flex size-9 items-center justify-center rounded-icon-well bg-canvas text-muted"
        >
          <Icon className="size-4" />
        </span>
      ) : null}
      {isLoading ? (
        <p data-slot="route-state-loading-label" className="text-[13px] text-muted">
          {loadingLabel}
        </p>
      ) : (
        <>
          {title ? (
            <h2
              data-slot="route-state-title"
              className="text-[18px] font-medium tracking-empty-h1 text-fg-strong"
            >
              {title}
            </h2>
          ) : null}
          {message ? (
            <p data-slot="route-state-message" className="max-w-md text-[13px] text-muted">
              {message}
            </p>
          ) : null}
          {cause ? (
            <div
              data-slot="route-state-cause"
              className="max-w-md rounded border border-line bg-canvas px-3 py-2 font-mono text-[11px] text-subtle"
            >
              {cause}
            </div>
          ) : null}
          {action ? <div data-slot="route-state-action">{action}</div> : null}
          {children}
        </>
      )}
    </div>
  );
}

export { RouteState };
