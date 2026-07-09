"use client";

import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import * as React from "react";

import { cn } from "../../lib/utils";

interface ListingRowProps extends React.ComponentProps<"div"> {
  selected?: boolean;
  /**
   * Row-hover glaze. Defaults on for navigable / actionable rows. Set `false`
   * for rows with no click target (e.g. marketplace rows) so hover does not
   * imply a false affordance.
   */
  interactive?: boolean;
}

type ListingRowLinkProps = useRender.ComponentProps<"a">;

type ListingRowIconProps = React.ComponentProps<"span">;
type ListingRowMainProps = React.ComponentProps<"div">;
type ListingRowDescriptionProps = React.ComponentProps<"p">;
type ListingRowMetaProps = React.ComponentProps<"div">;
type ListingRowTrailProps = React.ComponentProps<"div">;
type ListingRowStatProps = React.ComponentProps<"div">;
type ListingRowSlugProps = React.ComponentProps<"span">;

interface ListingRowNameProps extends React.ComponentProps<"div"> {
  mono?: boolean;
}

interface ListingRowTitleProps extends React.ComponentProps<"b"> {
  mono?: boolean;
}

type ListingRowNameContextValue = {
  mono: boolean;
};

const ListingRowNameContext = React.createContext<ListingRowNameContextValue>({ mono: false });

function ListingRow({
  selected = false,
  interactive = true,
  className,
  ...props
}: ListingRowProps) {
  return (
    <div
      data-slot="listing-row"
      data-selected={selected ? "true" : undefined}
      data-interactive={interactive ? "true" : undefined}
      className={cn(
        "grid grid-cols-[34px_minmax(0,1fr)_auto] items-center gap-3.5 border-b border-line-soft px-4 py-3 text-fg transition-colors duration-base ease-out last:border-b-0",
        interactive && "hover:bg-row-hover",
        selected && "bg-row-selected",
        className
      )}
      {...props}
    />
  );
}

function ListingRowLink({ className, render, ...props }: ListingRowLinkProps) {
  return useRender({
    defaultTagName: "a",
    props: mergeProps<"a">(
      {
        className: cn(
          "col-span-2 grid min-w-0 grid-cols-[34px_minmax(0,1fr)] items-center gap-3.5 rounded-sm outline-none focus-visible:shadow-focus-ring-inset",
          className
        ),
      } as Record<string, unknown>,
      {
        "data-slot": "listing-row-link",
      } as Record<string, unknown>,
      props
    ),
    render,
    state: { slot: "listing-row-link" },
  });
}

function ListingRowIcon({ className, ...props }: ListingRowIconProps) {
  return (
    <span
      aria-hidden="true"
      data-slot="listing-row-icon"
      className={cn(
        "grid size-[34px] shrink-0 place-items-center rounded-md bg-elevated text-muted",
        className
      )}
      {...props}
    />
  );
}

function ListingRowMain({ className, ...props }: ListingRowMainProps) {
  return <div data-slot="listing-row-main" className={cn("min-w-0", className)} {...props} />;
}

function ListingRowName({ mono = false, className, ...props }: ListingRowNameProps) {
  const ctx = React.useMemo<ListingRowNameContextValue>(() => ({ mono }), [mono]);
  return (
    <ListingRowNameContext.Provider value={ctx}>
      <div
        data-slot="listing-row-name"
        data-mono={mono ? "true" : undefined}
        className={cn("flex min-w-0 items-center gap-2", className)}
        {...props}
      />
    </ListingRowNameContext.Provider>
  );
}

function ListingRowTitle({ mono, className, ...props }: ListingRowTitleProps) {
  const nameCtx = React.use(ListingRowNameContext);
  const useMono = mono ?? nameCtx.mono;
  return (
    <b
      data-slot="listing-row-title"
      data-mono={useMono ? "true" : undefined}
      className={cn(
        "min-w-0 truncate font-medium text-fg-strong",
        useMono ? "font-mono text-xs tracking-normal" : "font-sans text-sm tracking-[-0.01em]",
        className
      )}
      {...props}
    />
  );
}

function ListingRowSlug({ className, ...props }: ListingRowSlugProps) {
  return (
    <span
      data-slot="listing-row-slug"
      className={cn("shrink-0 whitespace-nowrap font-mono text-eyebrow text-faint", className)}
      {...props}
    />
  );
}

function ListingRowDescription({ className, ...props }: ListingRowDescriptionProps) {
  return (
    <p
      data-slot="listing-row-description"
      className={cn("mt-1 truncate text-[12.5px] leading-[1.45] text-muted", className)}
      {...props}
    />
  );
}

function ListingRowMeta({ className, ...props }: ListingRowMetaProps) {
  return (
    <div
      data-slot="listing-row-meta"
      className={cn("mt-1.5 flex flex-wrap items-center gap-2 text-eyebrow text-faint", className)}
      {...props}
    />
  );
}

/** Neutral dot separator between `ListingRow.Meta` facts. */
function ListingRowMetaDot({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      data-slot="listing-row-meta-dot"
      className={cn("size-0.5 rounded-full bg-faint", className)}
      {...props}
    />
  );
}

function ListingRowTrail({ className, ...props }: ListingRowTrailProps) {
  return (
    <div
      data-slot="listing-row-trail"
      className={cn("flex shrink-0 items-center gap-3", className)}
      {...props}
    />
  );
}

function ListingRowStat({ className, children, ...props }: ListingRowStatProps) {
  return (
    <div
      data-slot="listing-row-stat"
      className={cn("flex flex-col items-end gap-0.5", className)}
      {...props}
    >
      {children}
    </div>
  );
}

function ListingRowStatValue({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="listing-row-stat-value"
      className={cn("font-mono text-xs font-semibold tabular-nums text-fg", className)}
      {...props}
    />
  );
}

function ListingRowStatLabel({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="listing-row-stat-label"
      className={cn("text-badge text-faint", className)}
      {...props}
    />
  );
}

const ListingRowStatCompound = Object.assign(ListingRowStat, {
  Value: ListingRowStatValue,
  Label: ListingRowStatLabel,
});

const ListingRowCompound = Object.assign(ListingRow, {
  Link: ListingRowLink,
  Icon: ListingRowIcon,
  Main: ListingRowMain,
  Name: ListingRowName,
  Title: ListingRowTitle,
  Slug: ListingRowSlug,
  Description: ListingRowDescription,
  Meta: ListingRowMeta,
  MetaDot: ListingRowMetaDot,
  Trail: ListingRowTrail,
  Stat: ListingRowStatCompound,
});

export { ListingRowCompound as ListingRow };
export type {
  ListingRowDescriptionProps,
  ListingRowIconProps,
  ListingRowLinkProps,
  ListingRowMainProps,
  ListingRowMetaProps,
  ListingRowNameProps,
  ListingRowProps,
  ListingRowSlugProps,
  ListingRowStatProps,
  ListingRowTitleProps,
  ListingRowTrailProps,
};
