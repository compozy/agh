"use client";

import { useDocsSearch } from "fumadocs-core/search/client";
import type { DefaultSearchDialogProps } from "fumadocs-ui/components/dialog/search-default";
import { useI18n } from "fumadocs-ui/contexts/i18n";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSiteSearch } from "./use-site-search";

type SiteSearchDialogStateOptions = Pick<
  DefaultSearchDialogProps,
  "api" | "defaultTag" | "delayMs" | "links" | "type"
>;

export function useSiteSearchDialogState({
  api,
  defaultTag,
  delayMs,
  links = [],
  type = "fetch",
}: SiteSearchDialogStateOptions) {
  const { locale } = useI18n();
  const { seed, setQuery } = useSiteSearch();
  const [tag, setTag] = useState(defaultTag);
  const appliedSeedVersion = useRef(-1);
  const { search, setSearch, query } = useDocsSearch(
    type === "fetch"
      ? {
          type: "fetch",
          api,
          locale,
          tag,
          delayMs,
        }
      : {
          type: "static",
          from: api,
          locale,
          tag,
          delayMs,
        }
  );

  useEffect(() => {
    setTag(defaultTag);
  }, [defaultTag]);

  useEffect(() => {
    if (appliedSeedVersion.current === seed.version) return;
    appliedSeedVersion.current = seed.version;
    setSearch(seed.query);
    setQuery(seed.query);
  }, [seed.query, seed.version, setQuery, setSearch]);

  const handleSearchChange = useCallback(
    (nextQuery: string) => {
      setSearch(nextQuery);
      setQuery(nextQuery);
    },
    [setQuery, setSearch]
  );

  const defaultItems = useMemo(() => {
    if (links.length === 0) return null;
    return links.map(([name, link]) => ({
      type: "page" as const,
      id: name,
      content: name,
      url: link,
    }));
  }, [links]);

  return {
    defaultItems,
    handleSearchChange,
    isLoading: query.isLoading,
    results: query.data,
    search,
    setTag,
    tag,
  };
}
