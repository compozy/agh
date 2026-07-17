import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import * as React from "react";

import { cn } from "../../lib/utils";

export type ListingRowLinkProps = useRender.ComponentProps<"a">;
export type ListingRowIconProps = React.ComponentProps<"span">;
export type ListingRowMainProps = React.ComponentProps<"div">;
export type ListingRowDescriptionProps = React.ComponentProps<"p">;
export type ListingRowMetaProps = React.ComponentProps<"div">;
export type ListingRowTrailProps = React.ComponentProps<"div">;
export type ListingRowStatProps = React.ComponentProps<"div">;
export type ListingRowSlugProps = React.ComponentProps<"span">;

export interface ListingRowNameProps extends React.ComponentProps<"div"> {
  mono?: boolean;
}

export interface ListingRowTitleProps extends React.ComponentProps<"b"> {
  mono?: boolean;
}

type ListingRowNameContextValue = { mono: boolean };
const ListingRowNameContext = React.createContext<ListingRowNameContextValue>({ mono: false });

export function ListingRowLink({ className, render, ...props }: ListingRowLinkProps) {
  const element = useRender({
    defaultTagName: "a",
    props: mergeProps<"a">(
      {
        className: cn(
          "col-span-2 grid min-w-0 grid-cols-[34px_minmax(0,1fr)] items-center gap-3.5 rounded-sm outline-none focus-visible:shadow-focus-inset",
          className
        ),
      } as Record<string, unknown>,
      { "data-slot": "listing-row-link" } as Record<string, unknown>,
      props
    ),
    render,
    state: { slot: "listing-row-link" },
  });
  return <>{element}</>;
}

export function ListingRowIcon({ className, ...props }: ListingRowIconProps) {
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

export function ListingRowMain({ className, ...props }: ListingRowMainProps) {
  return <div data-slot="listing-row-main" className={cn("min-w-0", className)} {...props} />;
}

export function ListingRowName({ mono = false, className, ...props }: ListingRowNameProps) {
  const contextValue: ListingRowNameContextValue = { mono };
  return (
    <ListingRowNameContext.Provider value={contextValue}>
      <div
        data-slot="listing-row-name"
        data-mono={mono ? "true" : undefined}
        className={cn("flex min-w-0 items-center gap-2", className)}
        {...props}
      />
    </ListingRowNameContext.Provider>
  );
}

export function ListingRowTitle({ mono, className, ...props }: ListingRowTitleProps) {
  const nameContext = React.use(ListingRowNameContext);
  const useMono = mono ?? nameContext.mono;
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

export function ListingRowSlug({ className, ...props }: ListingRowSlugProps) {
  return (
    <span
      data-slot="listing-row-slug"
      className={cn("shrink-0 whitespace-nowrap font-mono text-eyebrow text-faint", className)}
      {...props}
    />
  );
}

export function ListingRowDescription({ className, ...props }: ListingRowDescriptionProps) {
  return (
    <p
      data-slot="listing-row-description"
      className={cn("mt-1 truncate text-[12.5px] leading-[1.45] text-muted", className)}
      {...props}
    />
  );
}

export function ListingRowMeta({ className, ...props }: ListingRowMetaProps) {
  return (
    <div
      data-slot="listing-row-meta"
      className={cn("mt-1.5 flex flex-wrap items-center gap-2 text-eyebrow text-faint", className)}
      {...props}
    />
  );
}

export function ListingRowMetaDot({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      data-slot="listing-row-meta-dot"
      className={cn("size-0.5 rounded-full bg-faint", className)}
      {...props}
    />
  );
}

export function ListingRowTrail({ className, ...props }: ListingRowTrailProps) {
  return (
    <div
      data-slot="listing-row-trail"
      className={cn("flex shrink-0 items-center gap-3", className)}
      {...props}
    />
  );
}

export function ListingRowStat({ className, children, ...props }: ListingRowStatProps) {
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

export function ListingRowStatValue({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="listing-row-stat-value"
      className={cn("font-mono text-xs font-semibold tabular-nums text-fg", className)}
      {...props}
    />
  );
}

export function ListingRowStatLabel({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="listing-row-stat-label"
      className={cn("text-badge text-faint", className)}
      {...props}
    />
  );
}
