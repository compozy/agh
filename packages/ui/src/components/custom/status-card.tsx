"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Pill, type PillDotProps, type PillTone } from "./pill";

type StatusCardTone = PillTone;
type DataAttributes = {
  [key: `data-${string}`]: string | number | boolean | undefined;
};

interface StatusCardProps extends React.ComponentProps<"section"> {
  tone?: StatusCardTone;
}

interface StatusCardHeaderProps extends React.ComponentProps<"div"> {
  label?: React.ReactNode;
  dotProps?: Omit<PillDotProps, "tone"> & DataAttributes;
  labelProps?: React.ComponentProps<"span"> & DataAttributes;
}

type StatusCardBodyProps = React.ComponentProps<"div">;
type StatusCardFooterProps = React.ComponentProps<"div">;
type StatusCardActionProps = React.ComponentProps<"div">;

const StatusCardContext = React.createContext<{ tone: StatusCardTone } | null>(null);

function useStatusCardTone(tone?: StatusCardTone): StatusCardTone {
  const context = React.use(StatusCardContext);
  return tone ?? context?.tone ?? "neutral";
}

function StatusCard({ tone = "neutral", className, children, ...props }: StatusCardProps) {
  return (
    <StatusCardContext.Provider value={{ tone }}>
      <section
        className={cn("flex min-w-0 flex-col gap-3 rounded-lg bg-canvas-soft px-5 py-4", className)}
        data-slot="status-card"
        data-tone={tone}
        {...props}
      >
        {children}
      </section>
    </StatusCardContext.Provider>
  );
}

function StatusCardHeader({
  label,
  dotProps,
  labelProps,
  className,
  children,
  ...props
}: StatusCardHeaderProps) {
  const tone = useStatusCardTone();
  const { className: dotClassName, ...restDotProps } = dotProps ?? {};
  const { className: labelClassName, ...restLabelProps } = labelProps ?? {};

  return (
    <div
      className={cn("flex min-w-0 items-center gap-3", className)}
      data-slot="status-card-header"
      {...props}
    >
      <Pill.Dot
        aria-hidden="true"
        className={dotClassName}
        data-slot="status-card-dot"
        size="md"
        tone={tone}
        {...restDotProps}
      />
      {label ? (
        <span
          className={cn(
            "min-w-0 truncate text-item-title font-medium text-fg-strong",
            labelClassName
          )}
          data-slot="status-card-label"
          {...restLabelProps}
        >
          {label}
        </span>
      ) : null}
      {children}
    </div>
  );
}

function StatusCardBody({ className, ...props }: StatusCardBodyProps) {
  return (
    <div
      className={cn("text-small-body leading-5 text-muted", className)}
      data-slot="status-card-body"
      {...props}
    />
  );
}

function StatusCardFooter({ className, ...props }: StatusCardFooterProps) {
  return (
    <div
      className={cn("flex flex-wrap items-center gap-2", className)}
      data-slot="status-card-footer"
      {...props}
    />
  );
}

function StatusCardAction({ className, ...props }: StatusCardActionProps) {
  return (
    <div
      className={cn("flex flex-wrap items-center gap-2", className)}
      data-slot="status-card-action"
      {...props}
    />
  );
}

const StatusCardCompound = Object.assign(StatusCard, {
  Header: StatusCardHeader,
  Body: StatusCardBody,
  Footer: StatusCardFooter,
  Action: StatusCardAction,
});

export { StatusCardCompound as StatusCard };
export type {
  StatusCardActionProps,
  StatusCardBodyProps,
  StatusCardFooterProps,
  StatusCardHeaderProps,
  StatusCardProps,
  StatusCardTone,
};
