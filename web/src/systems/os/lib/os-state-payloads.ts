import { z } from "zod";

import {
  OS_DESKTOP_KEY,
  OS_SNAP_MIN_FRACTION,
  osWindowKey,
  windowIdFromKey,
  type OsStateEntry,
  type OsWindow,
} from "./os-types";

/**
 * Client-owned `os_shell` payload schemas, versioned with `v`. The daemon never
 * parses these; hydration drops entries that fail them individually (US-011.EC).
 */

const osRectSchema = z.object({
  x: z.number().finite(),
  y: z.number().finite(),
  w: z.number().finite().positive(),
  h: z.number().finite().positive(),
});

const osLocationSchema = z.object({
  pathname: z.string().min(1),
  search: z.record(z.string(), z.unknown()).catch({}),
});

/**
 * Snap fractions (ADR-009): each axis inside the unit work area with a
 * minimum span (`OS_SNAP_MIN_FRACTION`). Any failure — out-of-range,
 * negative/sub-minimum sizes, overflow past 1 — salvages the field to `null`
 * without dropping the window (invariant 19; US-001.EC-2 posture).
 */
const osSnapZoneSchema = z
  .object({
    fx: z.number().min(0).max(1),
    fy: z.number().min(0).max(1),
    fw: z.number().min(OS_SNAP_MIN_FRACTION).max(1),
    fh: z.number().min(OS_SNAP_MIN_FRACTION).max(1),
  })
  .refine(zone => zone.fx + zone.fw <= 1 + 1e-6 && zone.fy + zone.fh <= 1 + 1e-6);

const osAppIdSchema = z.enum([
  "dashboard",
  "session",
  "tasks",
  "agents",
  "network",
  "loops",
  "jobs",
  "triggers",
  "marketplace",
  "bridges",
  "knowledge",
  "sandbox",
  "vault",
  "settings",
]);

export const osWindowPayloadSchema = z.object({
  v: z.literal(1),
  app: osAppIdSchema,
  instanceKey: z.string().nullable(),
  location: osLocationSchema,
  rect: osRectSchema,
  prevRect: osRectSchema.nullable(),
  z: z.number().int().nonnegative(),
  minimized: z.boolean(),
  maximized: z.boolean(),
  snap: osSnapZoneSchema.nullable().catch(null),
});

export const osDesktopPayloadSchema = z.object({
  v: z.literal(1),
  focusedId: z.string().nullable(),
  railCollapsedAgentIds: z.array(z.string().min(1)).max(128),
  wallpaper: z.enum(["ember", "mesh", "carbon"]).catch("ember"),
  dockMagnify: z.boolean().catch(true),
  reduceMotion: z.boolean().catch(false),
});

export type OsWindowPayload = z.infer<typeof osWindowPayloadSchema>;
export type OsDesktopPayload = z.infer<typeof osDesktopPayloadSchema>;

export function encodeWindowPayload(win: OsWindow): OsWindowPayload {
  return {
    v: 1,
    app: win.app,
    instanceKey: win.instanceKey,
    location: win.location,
    rect: win.rect,
    prevRect: win.prevRect,
    z: win.z,
    minimized: win.minimized,
    maximized: win.maximized,
    snap: win.snap,
  };
}

export interface OsDesktopDoc {
  focusedId: string | null;
  railCollapsedAgentIds?: readonly string[];
  wallpaper: "ember" | "mesh" | "carbon";
  dockMagnify?: boolean;
  reduceMotion?: boolean;
}

export function encodeDesktopPayload(doc: OsDesktopDoc): OsDesktopPayload {
  return {
    v: 1,
    focusedId: doc.focusedId,
    railCollapsedAgentIds: [...(doc.railCollapsedAgentIds ?? [])],
    wallpaper: doc.wallpaper,
    dockMagnify: doc.dockMagnify ?? true,
    reduceMotion: doc.reduceMotion ?? false,
  };
}

/**
 * Decodes one desktop-state entry into a window. The payload's own identity
 * must agree with its key (`win:<windowId>`), or the entry is dropped.
 */
export function decodeWindowEntry(entry: OsStateEntry): OsWindow | null {
  const windowId = windowIdFromKey(entry.key);
  if (windowId === null || entry.value === null) return null;
  const parsed = osWindowPayloadSchema.safeParse(entry.value);
  if (!parsed.success) return null;
  const payload = parsed.data;
  const expectedKey =
    payload.app === "session" && payload.instanceKey !== null
      ? osWindowKey(`session:${payload.instanceKey}`)
      : osWindowKey(`app:${payload.app}`);
  if (expectedKey !== entry.key) return null;
  return {
    id: windowId,
    app: payload.app,
    instanceKey: payload.instanceKey,
    location: payload.location,
    rect: payload.rect,
    prevRect: payload.prevRect,
    z: payload.z,
    minimized: payload.minimized,
    maximized: payload.maximized,
    // Exclusivity salvage (invariant 19): a doc claiming both derived states
    // keeps `maximized` and drops `snap` — deterministic, window preserved.
    snap: payload.maximized ? null : payload.snap,
  };
}

export function decodeDesktopEntry(entry: OsStateEntry): OsDesktopPayload | null {
  if (entry.key !== OS_DESKTOP_KEY || entry.value === null) return null;
  const parsed = osDesktopPayloadSchema.safeParse(entry.value);
  return parsed.success ? parsed.data : null;
}
