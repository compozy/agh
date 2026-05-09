"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type CatalogCardTone = "accent" | "neutral" | "success" | "warning" | "danger" | "info";

interface CatalogCardProps extends React.ComponentProps<"article"> {
  selected?: boolean;
  actionable?: boolean;
}

interface CatalogCardLogoProps extends React.ComponentProps<"span"> {
  tone?: CatalogCardTone;
}

type CatalogCardTitleProps = React.ComponentProps<"h3">;
type CatalogCardDescriptionProps = React.ComponentProps<"p">;
type CatalogCardMetaProps = React.ComponentProps<"div">;
type CatalogCardActionsProps = React.ComponentProps<"div">;

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
        "flex min-w-0 flex-col gap-3 rounded-[var(--radius-diagram)] border border-(--color-divider) bg-(--color-surface) p-3 text-(--color-text-primary) transition-colors",
        actionable &&
          "hover:border-(--color-accent) hover:bg-(--color-hover) focus-within:border-(--color-accent)",
        selected && "border-(--color-accent) bg-(--color-surface-panel)",
        className
      )}
      {...props}
    />
  );
}

function CatalogCardLogo({ tone = "accent", className, ...props }: CatalogCardLogoProps) {
  return (
    <span
      aria-hidden="true"
      data-slot="catalog-card-logo"
      data-tone={tone}
      className={cn(
        "inline-flex size-9 shrink-0 items-center justify-center rounded-[var(--radius-diagram)] bg-(--color-surface-elevated)",
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
        "min-w-0 truncate text-small-body font-medium text-(--color-text-primary)",
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
      className={cn("text-small-body leading-6 text-(--color-text-secondary)", className)}
      {...props}
    />
  );
}

function CatalogCardMeta({ className, ...props }: CatalogCardMetaProps) {
  return (
    <div
      data-slot="catalog-card-meta"
      className={cn(
        "flex flex-wrap items-center gap-2 font-mono text-badge uppercase tracking-badge text-(--color-text-tertiary)",
        className
      )}
      {...props}
    />
  );
}

function CatalogCardActions({ className, ...props }: CatalogCardActionsProps) {
  return (
    <div
      data-slot="catalog-card-actions"
      className={cn(
        "mt-auto flex flex-wrap items-center gap-2 border-t border-(--color-divider) pt-3",
        className
      )}
      {...props}
    />
  );
}

function catalogCardLogoToneClass(tone: CatalogCardTone): string {
  switch (tone) {
    case "success":
      return "text-(--color-success)";
    case "warning":
      return "text-(--color-warning)";
    case "danger":
      return "text-(--color-danger)";
    case "info":
      return "text-(--color-info)";
    case "neutral":
      return "text-(--color-text-secondary)";
    case "accent":
      return "text-(--color-accent)";
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
  CatalogCardMetaProps,
  CatalogCardProps,
  CatalogCardTitleProps,
  CatalogCardTone,
};
