"use client";

import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { cva, type VariantProps } from "class-variance-authority";
import { useReducedMotionConfig } from "motion/react";
import * as React from "react";

import { cn } from "../../lib/utils";

export type PillTone = "neutral" | "accent" | "success" | "warning" | "danger" | "info";
export type PillSize = "xs" | "sm" | "md";

const TONE_DOT_COLOR: Record<PillTone, string> = {
  neutral: "var(--color-text-tertiary)",
  accent: "var(--color-accent)",
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  info: "var(--color-info)",
};

type PillContextValue = {
  size: PillSize;
  mono: boolean;
  tone: PillTone;
};

const PillContext = React.createContext<PillContextValue | null>(null);

const pillVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1.5 whitespace-nowrap border border-transparent transition-colors duration-150 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--color-accent) focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50 [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      tone: {
        neutral: "bg-(--color-neutral-tint) text-(--color-text-secondary)",
        accent: "bg-(--color-accent-tint) text-(--color-accent)",
        success: "bg-(--color-success-tint) text-(--color-success)",
        warning: "bg-(--color-warning-tint) text-(--color-warning)",
        danger: "bg-(--color-danger-tint) text-(--color-danger)",
        info: "bg-(--color-info-tint) text-(--color-info)",
      },
      size: {
        xs: "h-auto rounded-(--radius-chip) px-1.5 py-px text-[10px] leading-[14px]",
        sm: "h-[22px] rounded-(--radius-mono-badge) px-2 py-0.5 text-[11px] leading-[14px]",
        md: "h-8 rounded-[var(--radius-xl,20px)] px-3.5 text-[11px] leading-none",
      },
      mono: {
        true: "font-mono",
        false: "font-sans",
      },
      uppercase: {
        true: "uppercase",
        false: "normal-case",
      },
      solid: { true: "", false: "" },
      active: { true: "", false: "" },
    },
    compoundVariants: [
      { mono: true, size: "xs", className: "font-medium tracking-[0.04em]" },
      { mono: true, size: "sm", className: "font-medium tracking-[0.06em]" },
      { mono: true, size: "md", className: "font-semibold tracking-[0.12em]" },
      { mono: false, size: "xs", className: "font-medium" },
      { mono: false, size: "sm", className: "font-medium" },
      { mono: false, size: "md", className: "font-semibold tracking-wide" },
      {
        solid: true,
        tone: "neutral",
        className: "bg-(--color-text-secondary) text-(--color-canvas)",
      },
      { solid: true, tone: "accent", className: "bg-(--color-accent) text-(--color-accent-ink)" },
      { solid: true, tone: "success", className: "bg-(--color-success) text-(--color-canvas)" },
      { solid: true, tone: "warning", className: "bg-(--color-warning) text-(--color-canvas)" },
      { solid: true, tone: "danger", className: "bg-(--color-danger) text-(--color-canvas)" },
      { solid: true, tone: "info", className: "bg-(--color-info) text-(--color-canvas)" },
      {
        active: true,
        className:
          "border-(--color-text-tertiary) bg-(--color-surface-elevated) text-(--color-text-primary)",
      },
      {
        active: false,
        tone: "neutral",
        solid: false,
        className:
          "border-(--color-divider) bg-(--color-surface) text-(--color-text-secondary) hover:border-(--color-text-tertiary) hover:text-(--color-text-primary)",
      },
    ],
    defaultVariants: {
      tone: "neutral",
      size: "sm",
      mono: false,
      solid: false,
      uppercase: false,
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
  /** Override automatic uppercase. Defaults: `xs` → false, `sm`/`md` mono → true, otherwise false. */
  uppercase?: boolean;
}

function Pill({
  tone: toneProp,
  size: sizeProp,
  mono: monoProp,
  solid: solidProp,
  active,
  uppercase,
  className,
  render,
  ...props
}: PillProps) {
  const tone: PillTone = toneProp ?? "neutral";
  const size: PillSize = sizeProp ?? "sm";
  const mono = Boolean(monoProp);
  const solid = Boolean(solidProp);
  const computedUppercase = uppercase ?? (size === "md" ? true : size === "xs" ? false : mono);
  const ctx = React.useMemo<PillContextValue>(() => ({ size, mono, tone }), [size, mono, tone]);
  const element = useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(
          pillVariants({ tone, size, mono, solid, active, uppercase: computedUppercase }),
          className
        ),
      } as Record<string, unknown>,
      {
        "data-slot": "pill",
        "data-tone": tone,
        "data-size": size,
        "data-mono": mono ? "true" : undefined,
        "data-solid": solid ? "true" : undefined,
        "data-active": active === true ? "true" : active === false ? "false" : undefined,
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
  pulse = false,
  size: explicitSize,
  className,
  style,
  ...props
}: PillDotProps) {
  const ctx = React.useContext(PillContext);
  const reduced = useReducedMotionConfig();
  const shouldAnimate = pulse && !reduced;
  const effectiveSize: "sm" | "md" =
    explicitSize ?? (ctx ? (ctx.size === "md" ? "md" : "sm") : "md");
  const effectiveTone: PillTone = tone ?? ctx?.tone ?? "neutral";
  const background = color ?? TONE_DOT_COLOR[effectiveTone];
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
        shouldAnimate && "animate-pulse",
        className
      )}
      style={{ backgroundColor: background, ...style }}
      {...props}
    />
  );
}

const PillRoot = Pill as typeof Pill & { Dot: typeof PillDot };
PillRoot.Dot = PillDot;

export { PillRoot as Pill, PillDot, pillVariants };
