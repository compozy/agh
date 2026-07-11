import { useCallback, useMemo, useRef, useState } from "react";

import { runtimeModelKey } from "./model-key";
import { useRuntimeFavorites } from "./use-runtime-favorites";
import {
  resolveReasoningState,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
} from "./types";

export type RailFilter = "all" | "fav" | (string & {});

const HARNESS_BADGES: Record<string, string> = { acp: "cli", pi_acp: "api key" };

export interface GroupAvailability {
  tone: "success" | "warning" | "danger";
  label: string;
}

export interface RuntimeGroupModel {
  model: RuntimeModelOption;
  /** Compound `(provider, model)` identity — the DOM/list/favorite/recent key. */
  key: string;
  /** Stable identity for this rendered pinned/group occurrence. */
  cursor: string;
  /** Position among all selectable (model) rows in render order — keyboard index. */
  rowIndex: number;
  selected: boolean;
  favorite: boolean;
}

export interface RuntimeListGroup {
  key: string;
  name: string;
  /** Icon key for the group glyph (provider `runtime_provider` or id). */
  iconKind: string;
  harnessBadge?: string;
  availability: GroupAvailability;
  models: RuntimeGroupModel[];
}

export interface RuntimeListModel {
  searching: boolean;
  pinned: RuntimeGroupModel[];
  pinnedHeading: string;
  groups: RuntimeListGroup[];
  customLabel: string;
  customCommit: string;
  /** Rendered model rows in keyboard order, including pinned/group occurrences. */
  flatRows: RuntimeGroupModel[];
}

export interface UseRuntimeSelectorArgs {
  value: RuntimeSelectorValue;
  onChange: (next: RuntimeSelectorValue) => void;
  providers: RuntimeProviderOption[];
  models: RuntimeModelOption[];
  /**
   * The catalog query has RESOLVED (not merely "not loading"). Drives the strict
   * favorites/recents purge so a legitimately loaded-empty catalog wipes stale
   * persisted entries, while a still-loading/disabled query never does.
   */
  catalogLoaded?: boolean;
}

function providerBadge(harness: string | undefined): string | undefined {
  const key = (harness ?? "").trim().toLowerCase();
  return HARNESS_BADGES[key];
}

function providerIconKind(provider: RuntimeProviderOption | undefined, providerId: string): string {
  return provider?.runtime_provider ?? provider?.id ?? providerId;
}

function groupAvailability(
  provider: RuntimeProviderOption | undefined,
  models: RuntimeModelOption[]
): GroupAvailability {
  if (provider?.needs_auth) return { tone: "danger", label: "Sign in" };
  if (models.some(model => model.availability === "stale")) {
    return { tone: "warning", label: "Stale" };
  }
  if (models.length > 0 && models.every(model => model.availability === "unavailable")) {
    return { tone: "danger", label: "Unavailable" };
  }
  return { tone: "success", label: "Live" };
}

function matchesQuery(model: RuntimeModelOption, providerName: string, tokens: string[]): boolean {
  if (tokens.length === 0) return true;
  const haystack = `${model.name} ${model.id} ${model.provider} ${providerName}`.toLowerCase();
  return tokens.every(token => haystack.includes(token));
}

/** Ranking within a visible set: favorites first, then featured, then daemon order (stable). */
function rankModels(
  models: RuntimeModelOption[],
  isFavorite: (model: RuntimeModelOption) => boolean
): RuntimeModelOption[] {
  return models
    .map((model, index) => ({ model, index }))
    .sort((a, b) => {
      const favDelta = Number(isFavorite(b.model)) - Number(isFavorite(a.model));
      if (favDelta !== 0) return favDelta;
      const featuredDelta = Number(Boolean(b.model.featured)) - Number(Boolean(a.model.featured));
      if (featuredDelta !== 0) return featuredDelta;
      return a.index - b.index;
    })
    .map(entry => entry.model);
}

export function useRuntimeSelector({
  value,
  onChange,
  providers,
  models,
  catalogLoaded = false,
}: UseRuntimeSelectorArgs) {
  const [open, setOpen] = useState(false);
  const [railFilter, setRailFilter] = useState<RailFilter>("all");
  const [query, setQuery] = useState("");
  // The active (keyboard/pointer) row is tracked by its compound (provider,model)
  // KEY, never a numeric list index. Favorite-driven reordering rebuilds the list
  // but keeps the same active target — the derived index below just follows it.
  const [activeRow, setActiveRow] = useState<{ cursor: string; key: string } | null>(null);
  // Polite announcement text for the favorite toggle (Alt+F while focus stays in
  // search): "Favorited …" / "Unfavorited …" spoken via an aria-live region.
  const [favoriteAnnouncement, setFavoriteAnnouncement] = useState("");
  const focusIntentRef = useRef<"provider" | "model" | "reasoning">("model");

  const providerById = useMemo(
    () => new Map(providers.map(provider => [provider.id, provider])),
    [providers]
  );
  const modelByKey = useMemo(
    () => new Map(models.map(model => [runtimeModelKey(model.provider, model.id), model])),
    [models]
  );
  // Favorites/recents are validated against the current compound-tuple keys so
  // stale/foreign entries are never shown or persisted (see use-runtime-favorites).
  const validKeys = useMemo(() => new Set(modelByKey.keys()), [modelByKey]);
  const favorites = useRuntimeFavorites(validKeys, catalogLoaded);
  const selectedModel = value.model
    ? modelByKey.get(runtimeModelKey(value.provider, value.model))
    : undefined;
  const activeProvider = providerById.get(value.provider);
  const reasoningState = useMemo(() => resolveReasoningState(selectedModel), [selectedModel]);

  const isFavoriteModel = useCallback(
    (model: RuntimeModelOption) => favorites.isFavorite(runtimeModelKey(model.provider, model.id)),
    [favorites]
  );

  // A custom ID targets exactly one EXPLICIT, KNOWN provider — the rail-filtered
  // provider when one is active, otherwise the current selection's provider ONLY
  // when it is itself a real provider. There is no default substitution: when
  // neither is a known provider this is "", which disables the custom commit
  // entirely (a custom ID with no provider target must never be emitted).
  const activeCustomProvider = useMemo(() => {
    if (railFilter !== "all" && railFilter !== "fav" && providerById.has(railFilter)) {
      return railFilter;
    }
    return providerById.has(value.provider) ? value.provider : "";
  }, [railFilter, providerById, value.provider]);

  const listModel = useMemo<RuntimeListModel>(() => {
    const trimmedQuery = query.trim().toLowerCase();
    const tokens = trimmedQuery.length > 0 ? trimmedQuery.split(/\s+/) : [];
    const searching = tokens.length > 0;
    // The rail is a local list filter only; a non-empty search always spans
    // every provider (design HTML: search ignores the rail selection).
    const effectiveRail: RailFilter = searching ? "all" : railFilter;
    const providerDisplayName = (id: string): string => providerById.get(id)?.name ?? id;
    const flatRows: RuntimeGroupModel[] = [];

    const toRow = (model: RuntimeModelOption, occurrence: "pinned" | "group") => {
      const key = runtimeModelKey(model.provider, model.id);
      const row: RuntimeGroupModel = {
        model,
        key,
        cursor: `${occurrence}:${key}`,
        rowIndex: flatRows.length,
        selected: model.provider === value.provider && model.id === value.model,
        favorite: isFavoriteModel(model),
      };
      flatRows.push(row);
      return row;
    };

    // Pinned recents + favorites (browse only, "all"/"fav" rail). Both span
    // providers: favorites and recents are keyed by the compound identity.
    let pinned: RuntimeGroupModel[] = [];
    let pinnedHeading = "";
    if (!searching && (railFilter === "all" || railFilter === "fav")) {
      const favoriteModels = models.filter(isFavoriteModel);
      if (railFilter === "fav") {
        pinned = favoriteModels.map(model => toRow(model, "pinned"));
        pinnedHeading = "Favorites";
      } else {
        const recentModels = favorites.recents
          .map(key => modelByKey.get(key))
          .filter((model): model is RuntimeModelOption => model !== undefined);
        const seen = new Set<string>();
        const merged: RuntimeModelOption[] = [];
        for (const model of [...recentModels, ...favoriteModels]) {
          const key = runtimeModelKey(model.provider, model.id);
          if (seen.has(key)) continue;
          seen.add(key);
          merged.push(model);
        }
        pinned = merged.map(model => toRow(model, "pinned"));
        pinnedHeading = "Recent & favorites";
      }
    }

    // Grouped models across providers. Browse shows the curated subset; search
    // spans the full `view=all` set. A provider-scoped rail is a local filter
    // over the loaded cross-provider set; "fav" hides the grouped section
    // (favorites already rendered as the pinned block).
    const groups: RuntimeListGroup[] = [];
    if (effectiveRail !== "fav") {
      const providerFilter = effectiveRail === "all" ? null : effectiveRail;
      const grouped = new Map<string, RuntimeModelOption[]>();
      for (const model of models) {
        if (providerFilter && model.provider !== providerFilter) continue;
        if (!searching && !model.curated) continue;
        if (searching && !matchesQuery(model, providerDisplayName(model.provider), tokens))
          continue;
        const bucket = grouped.get(model.provider) ?? [];
        bucket.push(model);
        grouped.set(model.provider, bucket);
      }
      for (const [providerId, bucket] of grouped) {
        const provider = providerById.get(providerId);
        groups.push({
          key: providerId,
          name: provider?.name ?? providerId,
          iconKind: providerIconKind(provider, providerId),
          harnessBadge: providerBadge(provider?.harness),
          availability: groupAvailability(provider, bucket),
          models: rankModels(bucket, isFavoriteModel).map(model => toRow(model, "group")),
        });
      }
    }

    // The custom affordance requires an explicit provider target; without one it
    // is neither shown nor committable. Known-match is scoped to the EXACT
    // (activeProvider, id) tuple — a model published under another provider must
    // not block the same id as a custom target for the active provider. The raw
    // (case-preserving) query is the custom id: model ids are case-sensitive.
    const customId = query.trim();
    const knownForActiveProvider =
      activeCustomProvider.length > 0 &&
      modelByKey.has(runtimeModelKey(activeCustomProvider, customId));
    const canCommitCustom = searching && activeCustomProvider.length > 0 && !knownForActiveProvider;
    const customCommit = canCommitCustom ? customId : "";
    const customLabel = customCommit ? `Use "${customCommit}"` : "Use an exact custom model ID…";
    const showCustom = effectiveRail !== "fav" && activeCustomProvider.length > 0;
    return {
      searching,
      pinned,
      pinnedHeading,
      groups,
      customLabel: showCustom ? customLabel : "",
      customCommit,
      flatRows,
    };
  }, [
    query,
    railFilter,
    models,
    modelByKey,
    providerById,
    value.model,
    value.provider,
    activeCustomProvider,
    favorites.recents,
    isFavoriteModel,
  ]);

  // Numeric position of the active row, DERIVED from its stable compound key
  // against the current render order. When favoriting reorders the list the key
  // is unchanged, so this index (and the live `aria-activedescendant`) tracks the
  // same (provider, model) target to its new position instead of pointing at
  // whatever model happens to land on the old index.
  const highlightIndex = useMemo(() => {
    if (!activeRow) return -1;
    const exact = listModel.flatRows.findIndex(row => row.cursor === activeRow.cursor);
    return exact >= 0 ? exact : listModel.flatRows.findIndex(row => row.key === activeRow.key);
  }, [activeRow, listModel.flatRows]);

  const openWith = useCallback((intent: "provider" | "model" | "reasoning") => {
    focusIntentRef.current = intent;
    setRailFilter("all");
    setQuery("");
    setActiveRow(null);
    setFavoriteAnnouncement("");
    setOpen(true);
  }, []);

  const close = useCallback(() => setOpen(false), []);

  const changeRail = useCallback((target: RailFilter) => {
    // The rail is a local list filter only — `all`, `fav`, and provider IDs
    // never mutate the controlled value or clear the current model/effort.
    // Selecting a model row is what adopts that model's provider.
    setActiveRow(null);
    setRailFilter(target);
  }, []);

  const changeQuery = useCallback((next: string) => {
    setQuery(next);
    setActiveRow(null);
  }, []);

  const emitSelection = useCallback(
    (provider: string, id: string, model: RuntimeModelOption | undefined) => {
      const rz = resolveReasoningState(model);
      const keepsLevel =
        rz.mode === "levels" &&
        (value.reasoning_effort === "" || rz.levels.includes(value.reasoning_effort));
      onChange({
        provider,
        model: id,
        reasoning_effort: keepsLevel ? value.reasoning_effort : "",
      });
      favorites.pushRecent(runtimeModelKey(provider, id));
    },
    [onChange, value.reasoning_effort, favorites]
  );

  const pickModel = useCallback(
    (provider: string, modelId: string) => {
      const id = modelId.trim();
      if (id.length === 0) return;
      const model = modelByKey.get(runtimeModelKey(provider, id));
      if (model?.disabled) return;
      emitSelection(provider, id, model);
    },
    [modelByKey, emitSelection]
  );

  const pickCustom = useCallback(
    (modelId: string) => {
      const id = modelId.trim();
      if (id.length === 0) return;
      const provider = activeCustomProvider;
      // Fail closed on a missing/unknown provider — a custom ID must never be
      // emitted with an empty or guessed provider (no default substitution).
      if (provider.length === 0 || !providerById.has(provider)) return;
      // A custom ID may coincide with a known row for the active provider; reuse
      // its reasoning profile, otherwise commit it as a provisional exact ID.
      emitSelection(provider, id, modelByKey.get(runtimeModelKey(provider, id)));
    },
    [activeCustomProvider, providerById, modelByKey, emitSelection]
  );

  const setReasoning = useCallback(
    (effort: RuntimeSelectorValue["reasoning_effort"]) => {
      onChange({ ...value, reasoning_effort: effort });
    },
    [onChange, value]
  );

  const toggleFavorite = useCallback(
    (provider: string, id: string) => favorites.toggleFavorite(runtimeModelKey(provider, id)),
    [favorites]
  );

  // Keyboard moves resolve to a target row index, then commit the row's compound
  // KEY as the active target (never a raw index) so the highlight survives any
  // later reorder of the same list.
  const moveHighlight = useCallback(
    (direction: 1 | -1) => {
      const rows = listModel.flatRows;
      const enabled = rows
        .map((row, index) => ({ row, index }))
        .filter(entry => !entry.row.model.disabled)
        .map(entry => entry.index);
      if (enabled.length === 0) return;
      let target: number;
      if (highlightIndex < 0) {
        target = direction === 1 ? enabled[0] : enabled[enabled.length - 1];
      } else {
        const position = enabled.indexOf(highlightIndex);
        target =
          position < 0
            ? enabled[0]
            : enabled[(position + direction + enabled.length) % enabled.length];
      }
      const row = rows[target];
      setActiveRow({ cursor: row.cursor, key: row.key });
    },
    [listModel.flatRows, highlightIndex]
  );

  const moveHighlightEdge = useCallback(
    (edge: "first" | "last") => {
      const rows = listModel.flatRows;
      const enabled = rows
        .map((row, index) => ({ row, index }))
        .filter(entry => !entry.row.model.disabled)
        .map(entry => entry.index);
      if (enabled.length === 0) return;
      const target = edge === "first" ? enabled[0] : enabled[enabled.length - 1];
      const row = rows[target];
      setActiveRow({ cursor: row.cursor, key: row.key });
    },
    [listModel.flatRows]
  );

  const commitHighlight = useCallback(() => {
    const row = highlightIndex >= 0 ? listModel.flatRows[highlightIndex]?.model : undefined;
    if (row && !row.disabled) {
      pickModel(row.provider, row.id);
      return true;
    }
    if (listModel.customCommit.length > 0) {
      pickCustom(listModel.customCommit);
      return true;
    }
    return false;
  }, [highlightIndex, listModel.flatRows, listModel.customCommit, pickModel, pickCustom]);

  const toggleHighlightedFavorite = useCallback(() => {
    const row = highlightIndex >= 0 ? listModel.flatRows[highlightIndex]?.model : undefined;
    if (!row || row.disabled) return false;
    const wasFavorite = isFavoriteModel(row);
    toggleFavorite(row.provider, row.id);
    // Focus stays in the search field during Alt+F, so the button's aria-pressed
    // flip is not announced; a polite status carries the result instead.
    const providerName = providerById.get(row.provider)?.name ?? row.provider;
    setFavoriteAnnouncement(
      `${wasFavorite ? "Unfavorited" : "Favorited"} ${row.name} from ${providerName}`
    );
    return true;
  }, [highlightIndex, listModel.flatRows, isFavoriteModel, providerById, toggleFavorite]);

  // Pointer hover activates a (selectable) row by its compound key so the
  // external favorite action always targets the row under the cursor — no
  // interactive control nests in a listbox option.
  const highlightRow = useCallback(
    (rowIndex: number) => {
      const row = listModel.flatRows[rowIndex];
      if (!row || row.model.disabled) return;
      setActiveRow({ cursor: row.cursor, key: row.key });
    },
    [listModel.flatRows]
  );

  // The currently-highlighted model + its favorite state, for the external
  // favorite toggle button's label/pressed state (undefined when none is active).
  const highlightedRow = useMemo(() => {
    const model = highlightIndex >= 0 ? listModel.flatRows[highlightIndex]?.model : undefined;
    if (!model || model.disabled) return undefined;
    return { model, favorite: isFavoriteModel(model) };
  }, [highlightIndex, listModel.flatRows, isFavoriteModel]);

  return {
    favorites,
    open,
    openWith,
    close,
    focusIntent: focusIntentRef,
    railFilter,
    changeRail,
    query,
    changeQuery,
    highlightIndex,
    highlightRow,
    highlightedRow,
    favoriteAnnouncement,
    moveHighlight,
    moveHighlightEdge,
    commitHighlight,
    toggleHighlightedFavorite,
    listModel,
    selectedModel,
    activeProvider,
    reasoningState,
    pickModel,
    pickCustom,
    setReasoning,
  };
}

export type RuntimeSelectorController = ReturnType<typeof useRuntimeSelector>;
