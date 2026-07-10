import { useCallback, useEffect, useMemo, useState } from "react";

import {
  dedupeFavorites,
  FAVORITES_STORAGE_KEY,
  readFavoritesList,
  RECENTS_LIMIT,
  RECENTS_STORAGE_KEY,
  writeFavoritesList,
  type RuntimeFavoritesStore,
} from "./favorites";

/**
 * Browser-local favorites + recents for the runtime selector, keyed by the exact
 * compound `(provider, model)` identity.
 *
 * Storage is strict-greenfield: an entry is honored only when it is an EXACT
 * current compound-tuple key (`validKeys`, derived from the loaded catalog). Bare
 * model ids, foreign strings, legacy formats, malformed keys, and models no
 * longer in the catalog are neither shown nor persisted — once the catalog is
 * loaded they are purged and re-written so they can never reappear. There is no
 * compatibility decoder and no migration alias; a stored key either matches a
 * current tuple exactly or it is dropped. The same model id under two providers
 * stays distinct because each is a different `validKeys` member.
 *
 * @param validKeys the set of `runtimeModelKey(provider, model)` for every model
 *   currently in the catalog.
 * @param catalogLoaded whether the catalog query has RESOLVED. The reconcile pass
 *   runs once the catalog is loaded — even when it loaded EMPTY (every persisted
 *   entry is then invalid and purged). It is skipped only while the catalog is
 *   still loading, so a valid favorite is never wiped pre-load. Any present model
 *   (`validKeys` non-empty) implies loaded, so a loading state is never conflated
 *   with a legitimately loaded-empty set.
 */
export function useRuntimeFavorites(
  validKeys: ReadonlySet<string>,
  catalogLoaded: boolean
): RuntimeFavoritesStore {
  const loaded = catalogLoaded || validKeys.size > 0;
  const [rawFavorites, setRawFavorites] = useState<string[]>(() =>
    dedupeFavorites(readFavoritesList(FAVORITES_STORAGE_KEY))
  );
  const [rawRecents, setRawRecents] = useState<string[]>(() =>
    dedupeFavorites(readFavoritesList(RECENTS_STORAGE_KEY)).slice(0, RECENTS_LIMIT)
  );

  // Once the catalog is loaded, drop every stored entry that is not an exact
  // current compound tuple and re-persist the pruned list so ghosts never return.
  // A loaded-empty catalog purges everything; a still-loading catalog purges
  // nothing.
  useEffect(() => {
    if (!loaded) return;
    setRawFavorites(current => {
      const kept = dedupeFavorites(current.filter(key => validKeys.has(key)));
      writeFavoritesList(FAVORITES_STORAGE_KEY, kept);
      return kept.length === current.length && kept.every((key, index) => key === current[index])
        ? current
        : kept;
    });
    setRawRecents(current => {
      const kept = dedupeFavorites(current.filter(key => validKeys.has(key))).slice(
        0,
        RECENTS_LIMIT
      );
      writeFavoritesList(RECENTS_STORAGE_KEY, kept);
      return kept.length === current.length && kept.every((key, index) => key === current[index])
        ? current
        : kept;
    });
  }, [validKeys, loaded]);

  // Effective sets exclude anything outside the current allow-list, so a stale
  // entry never renders even in the window before the reconcile pass persists.
  const favorites = useMemo(
    () => new Set(rawFavorites.filter(key => validKeys.has(key))),
    [rawFavorites, validKeys]
  );
  const recents = useMemo(
    () => rawRecents.filter(key => validKeys.has(key)),
    [rawRecents, validKeys]
  );

  const toggleFavorite = useCallback(
    (id: string) => {
      const trimmed = id.trim();
      if (!validKeys.has(trimmed)) return;
      setRawFavorites(current => {
        const next = current.includes(trimmed)
          ? current.filter(key => key !== trimmed)
          : [...current, trimmed];
        writeFavoritesList(FAVORITES_STORAGE_KEY, next);
        return next;
      });
    },
    [validKeys]
  );

  const pushRecent = useCallback(
    (id: string) => {
      const trimmed = id.trim();
      if (!validKeys.has(trimmed)) return;
      setRawRecents(current => {
        const next = dedupeFavorites([trimmed, ...current]).slice(0, RECENTS_LIMIT);
        writeFavoritesList(RECENTS_STORAGE_KEY, next);
        return next;
      });
    },
    [validKeys]
  );

  const isFavorite = useCallback((id: string) => favorites.has(id), [favorites]);

  return { favorites, recents, isFavorite, toggleFavorite, pushRecent };
}
