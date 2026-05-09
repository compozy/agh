import type * as React from "react";

import { cn } from "../lib/utils";

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-pulse rounded-md bg-muted", className)}
      {...props}
    />
  );
}

export interface SkeletonRowsProps extends React.ComponentProps<"div"> {
  count?: number;
  rowClassName?: string;
  children?: React.ReactNode;
}

function SkeletonRows({
  count = 3,
  rowClassName,
  className,
  children,
  ...props
}: SkeletonRowsProps) {
  return (
    <div data-slot="skeleton-rows" className={cn("flex flex-col", className)} {...props}>
      {Array.from({ length: count }, (_, index) => (
        <div
          data-slot="skeleton-row"
          className={cn("flex flex-col gap-2", rowClassName)}
          key={index}
        >
          {children ?? (
            <>
              <Skeleton className="h-3.5 w-2/3" />
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-3/4" />
            </>
          )}
        </div>
      ))}
    </div>
  );
}

export { Skeleton, SkeletonRows };
