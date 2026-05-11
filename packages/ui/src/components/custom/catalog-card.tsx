"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type CatalogCardTone = "accent" | "neutral" | "success" | "warning" | "danger" | "info";

/**
 * Logo well size,
 * `default` — 24 × 24 (`--size-catalog-logo`); browse/marketplace surfaces.
 * `lg` — 40 × 40 (`--size-provider-logo-well`); configured/connected provider surfaces.
 */
type CatalogCardLogoSize = "default" | "lg";

interface CatalogCardProps extends React.ComponentProps<"article"> {
  selected?: boolean;
  actionable?: boolean;
}

interface CatalogCardLogoProps extends React.ComponentProps<"span"> {
  tone?: CatalogCardTone;
  size?: CatalogCardLogoSize;
}

type CatalogCardTitleProps = React.ComponentProps<"h3">;
type CatalogCardDescriptionProps = React.ComponentProps<"p">;
type CatalogCardMetaProps = React.ComponentProps<"div">;
type CatalogCardActionsProps = React.ComponentProps<"div">;

const LOGO_SIZE_CLASS: Record<CatalogCardLogoSize, string> = {
  default: "size-(--size-catalog-logo)",
  lg: "size-(--size-provider-logo-well)",
};

function CatalogCard({
  selected = false,
  actionable = false,
  className,
  ...props
}: CatalogCardProps) {
  return (
    <article
      data-slot="catalog-card"
      data-selected={selected ? "true" : undefined}
      data-actionable={actionable ? "true" : undefined}
      className={cn(
        "flex min-w-0 flex-col gap-3 rounded-lg bg-canvas-soft p-4 text-fg transition-colors duration-base ease-out",
        actionable && "hover:bg-elevated",
        selected && "bg-surface-glaze shadow-[inset_0_0_0_1px_var(--line-strong)]",
        className
      )}
      {...props}
    />
  );
}

function CatalogCardLogo({
  tone = "accent",
  size = "default",
  className,
  ...props
}: CatalogCardLogoProps) {
  return (
    <span
      aria-hidden="true"
      data-slot="catalog-card-logo"
      data-tone={tone}
      data-size={size}
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded bg-surface-glaze",
        LOGO_SIZE_CLASS[size],
        catalogCardLogoToneClass(tone),
        className
      )}
      {...props}
    />
  );
}

function CatalogCardTitle({ className, ...props }: CatalogCardTitleProps) {
  return (
    <div
      role="heading"
      aria-level={3}
      data-slot="catalog-card-title"
      className={cn(
        "min-w-0 truncate text-[13px] font-[510] tracking-modal-title text-fg-strong",
        className
      )}
      {...props}
    />
  );
}

function CatalogCardDescription({ className, ...props }: CatalogCardDescriptionProps) {
  return (
    <p
      data-slot="catalog-card-description"
      className={cn("text-small-body leading-6 text-muted", className)}
      {...props}
    />
  );
}

function CatalogCardMeta({ className, ...props }: CatalogCardMetaProps) {
  return (
    <div
      data-slot="catalog-card-meta"
      className={cn("eyebrow flex flex-wrap items-center gap-2 text-subtle", className)}
      {...props}
    />
  );
}

function CatalogCardActions({ className, ...props }: CatalogCardActionsProps) {
  return (
    <div
      data-slot="catalog-card-actions"
      className={cn(
        "mt-auto flex flex-wrap items-center gap-2 border-t border-line pt-3",
        className
      )}
      {...props}
    />
  );
}

function catalogCardLogoToneClass(tone: CatalogCardTone): string {
  switch (tone) {
    case "success":
      return "text-success";
    case "warning":
      return "text-warning";
    case "danger":
      return "text-danger";
    case "info":
      return "text-info";
    case "neutral":
      return "text-muted";
    case "accent":
      return "text-accent";
  }
}

const CatalogCardCompound = Object.assign(CatalogCard, {
  Logo: CatalogCardLogo,
  Title: CatalogCardTitle,
  Description: CatalogCardDescription,
  Meta: CatalogCardMeta,
  Actions: CatalogCardActions,
});

export { CatalogCardCompound as CatalogCard };
export type {
  CatalogCardActionsProps,
  CatalogCardDescriptionProps,
  CatalogCardLogoProps,
  CatalogCardLogoSize,
  CatalogCardMetaProps,
  CatalogCardProps,
  CatalogCardTitleProps,
  CatalogCardTone,
};
