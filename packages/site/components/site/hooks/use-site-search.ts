"use client";

import { use } from "react";
import { SiteSearchContext } from "../site-search-context";

export function useSiteSearch() {
  const context = use(SiteSearchContext);
  if (!context) {
    throw new Error("Missing <SiteSearchProvider />");
  }
  return context;
}
