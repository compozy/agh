"use client";

import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { cva, type VariantProps } from "class-variance-authority";
import { useReducedMotionConfig } from "motion/react";
import * as React from "react";

import { cn } from "../../lib/utils";

export type PillTone = "neutral" | "accent" | "success" | "warning" | "danger" | "info";
export type PillSize = "xs" | "sm" | "md";

const TONE_DOT_BG_CLASS: Record<PillTone, string> = {
  neutral: "bg-subtle",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

type PillContextValue = {
  size: PillSize;
  mono: boolean;
  tone: PillTone;
  pulse: boolean;
};

const PillContext = React.createContext<PillContextValue | null>(null);

const pillVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-xs transition-colors duration-base ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-line-strong focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50 [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      tone: {
        neutral: "bg-neutral-tint text-muted",
        accent: "bg-accent-tint text-accent",
        success: "bg-success-tint text-success",
        warning: "bg-warning-tint text-warning",
        danger: "bg-danger-tint text-danger",
        info: "bg-info-tint text-info",
      },
      size: {
        xs: "h-pill-xs px-1.5 leading-none",
        sm: "h-pill-sm px-2 leading-none",
        md: "h-pill-md px-2.5 leading-none",
      },
      mono: {
        true: "font-mono",
        false: "font-sans",
      },
      solid: { true: "", false: "" },
      active: { true: "", false: "" },
    },
    compoundVariants: [
      { mono: true, size: "xs", className: "text-mono-id font-semibold tracking-mono-id" },
      { mono: true, size: "sm", className: "text-mono-id font-semibold tracking-mono-id" },
      { mono: true, size: "md", className: "text-mono-id font-semibold tracking-mono-id" },
      { mono: false, size: "xs", className: "text-eyebrow font-medium tracking-eyebrow" },
      { mono: false, size: "sm", className: "text-eyebrow font-medium tracking-eyebrow" },
      { mono: false, size: "md", className: "text-eyebrow font-medium tracking-eyebrow" },
      { solid: true, tone: "neutral", className: "bg-muted text-canvas" },
      { solid: true, tone: "accent", className: "bg-accent text-accent-ink" },
      { solid: true, tone: "success", className: "bg-success text-canvas" },
      { solid: true, tone: "warning", className: "bg-warning text-canvas" },
      { solid: true, tone: "danger", className: "bg-danger text-canvas" },
      { solid: true, tone: "info", className: "bg-info text-canvas" },
      {
        active: true,
        className: "bg-elevated text-fg-strong",
      },
    ],
    defaultVariants: {
      tone: "neutral",
      size: "sm",
      mono: false,
      solid: false,
    },
  }
);

type PillVariantOptions = VariantProps<typeof pillVariants>;

export interface PillProps
  extends
    Omit<useRender.ComponentProps<"span">, "color">,
    Pick<PillVariantOptions, "tone" | "size" | "mono" | "solid"> {
  /** Toggle state. Only meaningful when the Pill is rendered as a control (e.g. `render={<button />}`). */
  active?: boolean;
  /** Propagate a pulsing state to the inner `Pill.Dot` when the dot does not set `pulse` itself. */
  pulse?: boolean;
}

export type PillLinkProps = Omit<PillProps, "active" | "render" | "solid"> &
  Omit<React.ComponentProps<"a">, keyof PillProps | "color"> & {
    render?: PillProps["render"];
    solid?: PillProps["solid"];
  };

function Pill({
  tone: toneProp,
  size: sizeProp,
  mono: monoProp,
  solid: solidProp,
  active,
  pulse,
  className,
  render,
  ...props
}: PillProps) {
  const tone: PillTone = toneProp ?? "neutral";
  const size: PillSize = sizeProp ?? "sm";
  const mono = Boolean(monoProp);
  const solid = Boolean(solidProp);
  const ctx = React.useMemo<PillContextValue>(
    () => ({ size, mono, tone, pulse: Boolean(pulse) }),
    [size, mono, tone, pulse]
  );
  const element = useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(pillVariants({ tone, size, mono, solid, active }), className),
      } as Record<string, unknown>,
      {
        "data-slot": "pill",
        "data-tone": tone,
        "data-size": size,
        "data-mono": mono ? "true" : undefined,
        "data-solid": solid ? "true" : undefined,
        "data-active": active === true ? "true" : active === false ? "false" : undefined,
        "data-pulse": pulse ? "true" : undefined,
        "aria-pressed": active === true || active === false ? active : undefined,
      } as Record<string, unknown>,
      props
    ),
    render,
    state: { slot: "pill", tone, size, mono, solid, active },
  });
  return <PillContext.Provider value={ctx}>{element}</PillContext.Provider>;
}

export interface PillDotProps extends Omit<React.ComponentProps<"span">, "color"> {
  tone?: PillTone;
  /** CSS color or `var(...)` reference. Overrides `tone`-derived color. */
  color?: string;
  pulse?: boolean;
  size?: "sm" | "md";
}

function PillDot({
  tone,
  color,
  pulse,
  size: explicitSize,
  className,
  style,
  ...props
}: PillDotProps) {
  const ctx = React.use(PillContext);
  const reduced = useReducedMotionConfig();
  const effectivePulse = pulse ?? ctx?.pulse ?? false;
  const shouldAnimate = effectivePulse && !reduced;
  const effectiveSize: "sm" | "md" =
    explicitSize ?? (ctx ? (ctx.size === "md" ? "md" : "sm") : "md");
  const effectiveTone: PillTone = tone ?? ctx?.tone ?? "neutral";
  return (
    <span
      aria-hidden="true"
      data-slot="pill-dot"
      data-tone={effectiveTone}
      data-size={effectiveSize}
      data-pulse={shouldAnimate ? "true" : undefined}
      className={cn(
        "inline-block shrink-0 rounded-full",
        effectiveSize === "sm" ? "size-1.5" : "size-2",
        color === undefined && TONE_DOT_BG_CLASS[effectiveTone],
        shouldAnimate && "animate-pulse",
        className
      )}
      style={color === undefined ? style : { backgroundColor: color, ...style }}
      {...props}
    />
  );
}

function PillLink({
  tone = "accent",
  size = "sm",
  mono = true,
  className,
  href,
  render,
  children,
  ...props
}: PillLinkProps) {
  return (
    <Pill
      tone={tone}
      size={size}
      mono={mono}
      className={cn("hover:border-accent hover:text-accent", className)}
      render={render ?? <a href={href ?? "#"} />}
      {...props}
    >
      {children}
    </Pill>
  );
}

const PillRoot = Pill as typeof Pill & { Dot: typeof PillDot; Link: typeof PillLink };
PillRoot.Dot = PillDot;
PillRoot.Link = PillLink;

export { PillRoot as Pill, PillDot, PillLink, pillVariants };
