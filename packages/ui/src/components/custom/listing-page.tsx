"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { ListingPageHead, ListingPageMetaDot } from "./listing-page-parts";

type ListingPageProps = React.ComponentProps<"div"> & {
  /** Optional banner rendered above the scroll area (e.g. cached-data Alert). */
  banner?: React.ReactNode;
  /** Extra classes for the centered content container. */
  bodyClassName?: string;
};

function ListingPage({ banner, bodyClassName, className, children, ...props }: ListingPageProps) {
  return (
    <div
      data-slot="listing-page"
      className={cn("flex min-h-0 flex-1 flex-col overflow-hidden", className)}
      {...props}
    >
      {banner ? <div data-slot="listing-page-banner">{banner}</div> : null}
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        <div
          data-slot="listing-page-container"
          className={cn(
            "mx-auto flex w-full max-w-[1320px] flex-col gap-5 px-9 pt-7 pb-20",
            bodyClassName
          )}
        >
          {children}
        </div>
      </div>
    </div>
  );
}

const ListingPageCompound = Object.assign(ListingPage, {
  Head: ListingPageHead,
  MetaDot: ListingPageMetaDot,
});

export { ListingPageCompound as ListingPage };
export type { ListingPageHeadProps } from "./listing-page-parts";
export type { ListingPageProps };
