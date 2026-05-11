"use client";

import { Progress as ProgressPrimitive } from "@base-ui/react/progress";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

const progressIndicatorVariants = cva("h-full transition-all", {
  variants: {
    tone: {
      accent: "bg-accent",
      success: "bg-success",
      warning: "bg-warning",
      danger: "bg-danger",
      info: "bg-info",
      neutral: "bg-neutral",
    },
  },
  defaultVariants: {
    tone: "accent",
  },
});

interface ProgressProps extends ProgressPrimitive.Root.Props {
  tone?: VariantProps<typeof progressIndicatorVariants>["tone"];
}

function Progress({ className, children, value, tone = "accent", ...props }: ProgressProps) {
  return (
    <ProgressPrimitive.Root
      value={value}
      data-slot="progress"
      data-tone={tone}
      className={cn("flex flex-wrap gap-2", className)}
      {...props}
    >
      {children}
      <ProgressTrack>
        <ProgressIndicator className={progressIndicatorVariants({ tone })} />
      </ProgressTrack>
    </ProgressPrimitive.Root>
  );
}

function ProgressTrack({ className, ...props }: ProgressPrimitive.Track.Props) {
  return (
    <ProgressPrimitive.Track
      className={cn(
        "relative flex h-1 w-full items-center overflow-x-hidden rounded-full bg-canvas-tint",
        className
      )}
      data-slot="progress-track"
      {...props}
    />
  );
}

function ProgressIndicator({ className, ...props }: ProgressPrimitive.Indicator.Props) {
  return (
    <ProgressPrimitive.Indicator
      data-slot="progress-indicator"
      className={cn("h-full bg-accent transition-all", className)}
      {...props}
    />
  );
}

function ProgressLabel({ className, ...props }: ProgressPrimitive.Label.Props) {
  return (
    <ProgressPrimitive.Label
      className={cn("text-[13px] font-medium text-fg", className)}
      data-slot="progress-label"
      {...props}
    />
  );
}

function ProgressValue({ className, ...props }: ProgressPrimitive.Value.Props) {
  return (
    <ProgressPrimitive.Value
      className={cn("ml-auto text-[13px] text-muted tabular-nums", className)}
      data-slot="progress-value"
      {...props}
    />
  );
}

export {
  Progress,
  ProgressIndicator,
  progressIndicatorVariants,
  ProgressLabel,
  ProgressTrack,
  ProgressValue,
};
