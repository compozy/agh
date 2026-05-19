"use client";

import { useCallback, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { SiteSearchContext } from "@/components/site/site-search-context";
import type { SearchSeed } from "@/components/site/site-search-context";

export function SiteSearchProvider({ children }: { children: ReactNode }) {
  const [query, setQuery] = useState("");
  const [seed, setSeed] = useState<SearchSeed>({ query: "", version: 0 });

  const openWithQuery = useCallback((nextQuery: string) => {
    setQuery(nextQuery);
    setSeed(current => ({ query: nextQuery, version: current.version + 1 }));
  }, []);

  const value = useMemo(
    () => ({ query, seed, openWithQuery, setQuery }),
    [openWithQuery, query, seed]
  );

  return <SiteSearchContext value={value}>{children}</SiteSearchContext>;
}
