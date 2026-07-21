import { describe, expect, it, vi } from "vitest";

import { createDesktopPersistence } from "../../lib/desktop-persistence";
import { deriveSnapRect } from "../../lib/os-snap-geometry";
import { OS_SNAP_ZONES } from "../../lib/os-snap-zones";
import type { OsStateClient } from "../../lib/os-state-client";
import { encodeDesktopPayload, encodeWindowPayload } from "../../lib/os-state-payloads";
import { OS_DESKTOP_KEY, osWindowKey, type OsStateEntry, type OsWindow } from "../../lib/os-types";
import { createDesktopStore } from "../desktop-store";

function windowEntry(win: OsWindow, seq: number, rev = 1): OsStateEntry {
  return {
    key: osWindowKey(win.id),
    value: encodeWindowPayload(win),
    rev,
    seq,
    deleted: false,
    updated_at: "2026-07-20T00:00:00Z",
  };
}

function makeWindow(overrides: Partial<OsWindow> & Pick<OsWindow, "id" | "app">): OsWindow {
  return {
    instanceKey: null,
    location: { pathname: "/", search: {} },
    rect: { x: 10, y: 10, w: 400, h: 300 },
    prevRect: null,
    z: 1,
    minimized: false,
    maximized: false,
    snap: null,
    ...overrides,
  };
}

describe("desktop store", () => {
  it("Should create a window with registry defaults, focused at max z (UT-040)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });

    expect(id).toBe("app:tasks");
    const win = store.getState().windows[id];
    expect(win).toBeDefined();
    expect(win.rect).toEqual({ x: 200, y: 88, w: 660, h: 480 });
    expect(win.location.pathname).toBe("/tasks");
    expect(store.getState().focusedId).toBe(id);
    const zValues = Object.values(store.getState().windows).map(w => w.z);
    expect(win.z).toBe(Math.max(...zValues));
  });

  it("Should focus the existing window instead of duplicating (UT-041, invariant 14)", () => {
    const store = createDesktopStore();
    store.getState().openOrFocus({ app: "tasks" });
    store.getState().openOrFocus({ app: "settings" });

    const again = store.getState().openOrFocus({ app: "tasks" });

    expect(again).toBe("app:tasks");
    expect(Object.keys(store.getState().windows)).toHaveLength(2);
    expect(store.getState().focusedId).toBe("app:tasks");
  });

  it("Should keep session windows multi-instance (UT-042 seam)", () => {
    const store = createDesktopStore();
    store.getState().openOrFocus({
      app: "session",
      instanceKey: "s1",
      location: { pathname: "/agents/a/sessions/s1", search: {} },
    });
    store.getState().openOrFocus({
      app: "session",
      instanceKey: "s2",
      location: { pathname: "/agents/a/sessions/s2", search: {} },
    });

    expect(Object.keys(store.getState().windows).sort()).toEqual(["session:s1", "session:s2"]);
  });

  it("Should keep one idempotent window per session instance key (UT-042)", () => {
    const store = createDesktopStore();
    const first = store.getState().openOrFocus({
      app: "session",
      instanceKey: "s1",
      location: { pathname: "/agents/a/sessions/s1", search: {} },
    });
    const again = store.getState().openOrFocus({
      app: "session",
      instanceKey: "s1",
      location: { pathname: "/agents/a/sessions/s1", search: {} },
    });

    expect(first).toBe("session:s1");
    expect(again).toBe(first);
    expect(Object.keys(store.getState().windows)).toEqual(["session:s1"]);
  });

  it("Should raise a background window to max z on focus (UT-043)", () => {
    const store = createDesktopStore();
    const tasks = store.getState().openOrFocus({ app: "tasks" });
    const settings = store.getState().openOrFocus({ app: "settings" });

    store.getState().focusWindow(tasks);

    const state = store.getState();
    expect(state.focusedId).toBe(tasks);
    expect(state.windows[tasks].z).toBeGreaterThan(state.windows[settings].z);
  });

  it("Should compact hydrated z values to 1..N preserving order (UT-044, invariant 12)", () => {
    const store = createDesktopStore();
    const entries = [
      windowEntry(makeWindow({ id: "app:tasks", app: "tasks", z: 7 }), 1),
      windowEntry(makeWindow({ id: "app:settings", app: "settings", z: 903 }), 2),
      windowEntry(makeWindow({ id: "app:vault", app: "vault", z: 12 }), 3),
    ];

    store.getState().hydrate(entries);

    const windows = store.getState().windows;
    expect(windows["app:tasks"].z).toBe(1);
    expect(windows["app:vault"].z).toBe(2);
    expect(windows["app:settings"].z).toBe(3);
  });

  it("Should survive minimize with rect intact and restore with focus (UT-045)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().commitRect(id, { x: 42, y: 24, w: 500, h: 400 });

    store.getState().minimizeWindow(id);
    let win = store.getState().windows[id];
    expect(win.minimized).toBe(true);
    expect(win.rect).toEqual({ x: 42, y: 24, w: 500, h: 400 });

    store.getState().restoreWindow(id);
    win = store.getState().windows[id];
    expect(win.minimized).toBe(false);
    expect(store.getState().focusedId).toBe(id);
  });

  it("Should zoom storing prevRect and restore it exactly on the second toggle (UT-046)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().commitRect(id, { x: 33, y: 44, w: 555, h: 444 });

    store.getState().toggleZoom(id);
    let win = store.getState().windows[id];
    expect(win.maximized).toBe(true);
    expect(win.prevRect).toEqual({ x: 33, y: 44, w: 555, h: 444 });

    store.getState().toggleZoom(id);
    win = store.getState().windows[id];
    expect(win.maximized).toBe(false);
    expect(win.rect).toEqual({ x: 33, y: 44, w: 555, h: 444 });
    expect(win.prevRect).toBeNull();
  });

  it("Should pass focus to the next-highest z on close and null on last close (UT-047)", () => {
    const store = createDesktopStore();
    const tasks = store.getState().openOrFocus({ app: "tasks" });
    const settings = store.getState().openOrFocus({ app: "settings" });
    const vault = store.getState().openOrFocus({ app: "vault" });

    store.getState().closeWindow(vault);
    expect(store.getState().focusedId).toBe(settings);

    store.getState().closeWindow(settings);
    expect(store.getState().focusedId).toBe(tasks);

    store.getState().closeWindow(tasks);
    expect(store.getState().focusedId).toBeNull();
  });

  it("Should ignore remote events at or below the settled seq and apply higher ones (UT-048, invariant 10)", () => {
    const store = createDesktopStore();
    const win = makeWindow({ id: "app:tasks", app: "tasks" });
    store.getState().hydrate([windowEntry(win, 5)]);

    const stale = windowEntry({ ...win, rect: { x: 1, y: 1, w: 300, h: 200 } }, 5, 2);
    store.getState().applyRemote({ entry: stale, origin: "conn-b" });
    expect(store.getState().windows["app:tasks"].rect).toEqual(win.rect);

    const fresh = windowEntry({ ...win, rect: { x: 99, y: 88, w: 320, h: 240 } }, 6, 2);
    store.getState().applyRemote({ entry: fresh, origin: "conn-b" });
    expect(store.getState().windows["app:tasks"].rect).toEqual({ x: 99, y: 88, w: 320, h: 240 });
  });

  it("Should remove the window on a remote delete event (UT-049)", () => {
    const store = createDesktopStore();
    const win = makeWindow({ id: "app:tasks", app: "tasks" });
    store.getState().hydrate([windowEntry(win, 1)]);

    store.getState().applyRemote({
      entry: {
        key: "win:app:tasks",
        value: null,
        rev: 2,
        seq: 2,
        deleted: true,
        updated_at: "2026-07-20T00:00:01Z",
      },
      origin: "conn-b",
    });

    expect(store.getState().windows["app:tasks"]).toBeUndefined();
  });

  it("Should derive compact presentation and gate rect ops at the breakpoint (UT-061)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    const rectBefore = store.getState().windows[id].rect;

    store.getState().setPresentation("compact");
    store.getState().commitRect(id, { x: 1, y: 2, w: 3, h: 4 });
    store.getState().toggleZoom(id);

    let win = store.getState().windows[id];
    expect(win.rect).toEqual(rectBefore);
    expect(win.maximized).toBe(false);

    store.getState().setPresentation("floating");
    win = store.getState().windows[id];
    expect(win.rect).toEqual(rectBefore);
  });

  it("Should toggle the rail and retain collapsed groups in the desktop document (UT-068, UT-084)", () => {
    const store = createDesktopStore();

    store.getState().toggleRail();
    store.getState().toggleRailGroup("codex");
    expect(store.getState()).toMatchObject({
      railOpen: true,
      railCollapsedAgentIds: ["codex"],
    });

    const restored = createDesktopStore();
    restored.getState().hydrate([
      {
        key: OS_DESKTOP_KEY,
        value: encodeDesktopPayload({
          focusedId: null,
          railOpen: store.getState().railOpen,
          railCollapsedAgentIds: store.getState().railCollapsedAgentIds,
          wallpaper: "ember",
        }),
        rev: 1,
        seq: 1,
        deleted: false,
        updated_at: "2026-07-20T00:00:00Z",
      },
    ]);

    expect(restored.getState()).toMatchObject({
      railOpen: true,
      railCollapsedAgentIds: ["codex"],
    });
    restored.getState().toggleRail();
    restored.getState().toggleRailGroup("codex");
    expect(restored.getState()).toMatchObject({
      railOpen: false,
      railCollapsedAgentIds: [],
    });
  });

  it("Should drop invalid hydration entries individually and restore the valid windows (UT-063)", () => {
    const store = createDesktopStore();
    const valid1 = windowEntry(makeWindow({ id: "app:tasks", app: "tasks", z: 1 }), 1);
    const valid2 = windowEntry(makeWindow({ id: "app:settings", app: "settings", z: 2 }), 2);
    const unknownApp: OsStateEntry = {
      key: "win:app:zzz",
      value: {
        v: 1,
        app: "zzz",
        instanceKey: null,
        location: { pathname: "/zzz", search: {} },
        rect: { x: 0, y: 0, w: 100, h: 100 },
        prevRect: null,
        z: 3,
        minimized: false,
        maximized: false,
      },
      rev: 1,
      seq: 3,
      deleted: false,
      updated_at: "2026-07-20T00:00:00Z",
    };
    const unreadable: OsStateEntry = {
      key: "win:app:vault",
      value: { v: 1, garbage: true },
      rev: 1,
      seq: 4,
      deleted: false,
      updated_at: "2026-07-20T00:00:00Z",
    };

    store.getState().hydrate([unknownApp, valid1, unreadable, valid2]);

    expect(Object.keys(store.getState().windows).sort()).toEqual(["app:settings", "app:tasks"]);
  });

  it("Should emit the soft-cap guidance exactly once at the 13th window (UT-064)", () => {
    const store = createDesktopStore();
    const apps = [
      "dashboard",
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
    ] as const;
    for (const app of apps) store.getState().openOrFocus({ app });
    expect(store.getState().softCapNotice).toBe(false);

    store.getState().openOrFocus({
      app: "session",
      instanceKey: "s13",
      location: { pathname: "/agents/a/sessions/s13", search: {} },
    });
    expect(store.getState().softCapNotice).toBe(true);

    store.getState().clearSoftCapNotice();
    store.getState().openOrFocus({
      app: "session",
      instanceKey: "s14",
      location: { pathname: "/agents/a/sessions/s14", search: {} },
    });
    expect(store.getState().softCapNotice).toBe(false);
  });

  it("Should re-clamp out-of-bounds windows on shrink and not move them on grow (UT-065)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().commitRect(id, { x: 900, y: 500, w: 400, h: 300 });

    store.getState().clampToViewport({ width: 800, height: 600 });
    const clamped = store.getState().windows[id].rect;
    expect(clamped.x + clamped.w).toBeLessThanOrEqual(800 - 8);
    expect(clamped.y).toBeLessThanOrEqual(600 - 120);

    store.getState().clampToViewport({ width: 1600, height: 1000 });
    expect(store.getState().windows[id].rect).toEqual(clamped);
  });

  it("Should keep desktop-doc remote focus honest against present windows", () => {
    const store = createDesktopStore();
    const win = makeWindow({ id: "app:tasks", app: "tasks" });
    store.getState().hydrate([windowEntry(win, 1)]);

    store.getState().applyRemote({
      entry: {
        key: OS_DESKTOP_KEY,
        value: encodeDesktopPayload({ focusedId: "app:tasks", railOpen: true, wallpaper: "mesh" }),
        rev: 1,
        seq: 2,
        deleted: false,
        updated_at: "2026-07-20T00:00:01Z",
      },
      origin: "conn-b",
    });

    const state = store.getState();
    expect(state.focusedId).toBe("app:tasks");
    expect(state.railOpen).toBe(true);
    expect(state.wallpaper).toBe("mesh");
  });

  it("Should clear workspace-scoped desktop state before the next workspace hydrates", () => {
    const store = createDesktopStore();
    const win = makeWindow({ id: "app:tasks", app: "tasks" });
    store.getState().hydrate([windowEntry(win, 1)]);
    store.getState().applyRemote({
      entry: {
        key: OS_DESKTOP_KEY,
        value: encodeDesktopPayload({ focusedId: "app:tasks", railOpen: true, wallpaper: "mesh" }),
        rev: 1,
        seq: 2,
        deleted: false,
        updated_at: "2026-07-20T00:00:01Z",
      },
      origin: "conn-b",
    });

    store.getState().resetForWorkspace();

    expect(store.getState()).toMatchObject({
      windows: {},
      focusedId: null,
      railOpen: false,
      railCollapsedAgentIds: [],
      wallpaper: "ember",
      hydration: "pending",
      softCapNotice: false,
    });
  });

  it("Should snap storing prevRect and commit this client's derivation as rect (UT-095, invariant 19)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().commitRect(id, { x: 40, y: 30, w: 500, h: 380 });
    store.getState().clampToViewport({ width: 1440, height: 900 });

    store.getState().snapWindow(id, { fx: 0, fy: 0, fw: 0.5, fh: 1 });

    const win = store.getState().windows[id];
    expect(win.snap).toEqual({ fx: 0, fy: 0, fw: 0.5, fh: 1 });
    expect(win.prevRect).toEqual({ x: 40, y: 30, w: 500, h: 380 });
    expect(win.maximized).toBe(false);
    // Derived-geometry selector: work area (insets 10/8/10/78) × fractions.
    const derived = deriveSnapRect({ fx: 0, fy: 0, fw: 0.5, fh: 1 }, { width: 1440, height: 900 });
    expect(derived).toEqual({ x: 10, y: 8, w: 710, h: 814 });
    expect(win.rect).toEqual(derived);
  });

  it("Should keep snap and maximized exclusive and restore prevRect exactly on every path (UT-096, invariant 19)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().clampToViewport({ width: 1440, height: 900 });
    store.getState().commitRect(id, { x: 33, y: 44, w: 555, h: 444 });

    store.getState().snapWindow(id, OS_SNAP_ZONES.left);
    store.getState().toggleZoom(id);
    let win = store.getState().windows[id];
    expect(win.maximized).toBe(true);
    expect(win.snap).toBeNull();
    expect(win.prevRect).toEqual({ x: 33, y: 44, w: 555, h: 444 });

    store.getState().snapWindow(id, OS_SNAP_ZONES.right);
    win = store.getState().windows[id];
    expect(win.maximized).toBe(false);
    expect(win.snap).toEqual(OS_SNAP_ZONES.right);
    expect(win.prevRect).toEqual({ x: 33, y: 44, w: 555, h: 444 });

    store.getState().snapWindow(id, null);
    win = store.getState().windows[id];
    expect(win.snap).toBeNull();
    expect(win.rect).toEqual({ x: 33, y: 44, w: 555, h: 444 });
    expect(win.prevRect).toBeNull();

    store.getState().snapWindow(id, OS_SNAP_ZONES.left);
    store.getState().toggleZoom(id);
    store.getState().toggleZoom(id);
    win = store.getState().windows[id];
    expect(win.maximized).toBe(false);
    expect(win.rect).toEqual({ x: 33, y: 44, w: 555, h: 444 });
  });

  it("Should clear snap on any commitRect and land drag-away geometry in the same mutation (UT-097, invariant 19)", () => {
    const store = createDesktopStore();
    const id = store.getState().openOrFocus({ app: "tasks" });
    store.getState().clampToViewport({ width: 1440, height: 900 });
    store.getState().commitRect(id, { x: 100, y: 90, w: 600, h: 400 });
    store.getState().snapWindow(id, OS_SNAP_ZONES.left);

    // Drag-away: the gesture restores prevRect dimensions under the pointer
    // and commits them; snap and prevRect clear atomically.
    store.getState().commitRect(id, { x: 250, y: 40, w: 600, h: 400 });

    const win = store.getState().windows[id];
    expect(win.snap).toBeNull();
    expect(win.prevRect).toBeNull();
    expect(win.rect).toEqual({ x: 250, y: 40, w: 600, h: 400 });
  });

  it("Should re-derive snapped geometry on viewport resize with zero snapped-window writes (UT-098, invariant 19)", () => {
    const store = createDesktopStore();
    const persistence = createDesktopPersistence(store);
    const snappedId = store.getState().openOrFocus({ app: "tasks" });
    const floatingId = store.getState().openOrFocus({ app: "settings" });
    store.getState().clampToViewport({ width: 1440, height: 900 });
    store.getState().snapWindow(snappedId, OS_SNAP_ZONES.right);
    store.getState().commitRect(floatingId, { x: 900, y: 500, w: 400, h: 300 });
    const snappedRectAtCommit = store.getState().windows[snappedId].rect;

    const schedulePut = vi.fn();
    const applyNow = vi.fn();
    const unbind = persistence.bind({ schedulePut, applyNow } as unknown as OsStateClient);
    store.getState().clampToViewport({ width: 1000, height: 700 });
    unbind();

    // The snapped window's stored rect never rewrites on resize; its rendered
    // rect derives from the new work area locally.
    expect(store.getState().windows[snappedId].rect).toEqual(snappedRectAtCommit);
    expect(applyNow).not.toHaveBeenCalled();
    expect(schedulePut).not.toHaveBeenCalledWith(osWindowKey(snappedId));
    // The floating window only re-clamps (UT-065) — a rect-only debounce.
    expect(schedulePut).toHaveBeenCalledWith(osWindowKey(floatingId));
    const derived = deriveSnapRect(OS_SNAP_ZONES.right, { width: 1000, height: 700 });
    expect(derived).toEqual({ x: 500, y: 8, w: 490, h: 614 });

    // A quarter deriving below the window minimum clamps to 280×180.
    const clamped = deriveSnapRect(OS_SNAP_ZONES["bottom-right"], { width: 500, height: 400 });
    expect(clamped.w).toBe(280);
    expect(clamped.h).toBe(180);
  });
});
