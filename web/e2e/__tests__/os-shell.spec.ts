import { execFile } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import type { Browser, Page, WebSocketRoute } from "@playwright/test";

import type { BrowserRuntime, WorkspacePayload } from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const execFileAsync = promisify(execFile);
const browserLifecycleAgent = "os-shell-agent";
const browserLifecycleFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_lifecycle_fixture.json"
);

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          agentName: browserLifecycleAgent,
          fixtureAgent: "browser-lifecycle-agent",
          fixturePath: browserLifecycleFixture,
        },
      ],
    },
  },
});

test("E2E-001: fresh boot renders the empty desktop without opening a window", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);

  await expect(appPage.getByRole("navigation", { name: "Dock" })).toBeVisible();
  await expect(appPage.getByRole("banner", { name: "System bar" })).toBeVisible();
  await expect(appPage.getByTestId("os-desk-hint")).toContainText("⌘K");
  await expect(appPage.locator('[data-testid^="os-window-"]')).toHaveCount(0);
});

test("E2E-004: minimize exposes the dock state and restore remounts content", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");

  await tasks.getByRole("button", { name: "Minimize window" }).click();
  await expect(tasks).toBeHidden();
  const dockItem = appPage.getByRole("button", { name: "Tasks" });
  await expect(dockItem).toHaveAttribute("data-state", "minimized");

  await dockItem.click();
  await expect(tasks).toBeVisible();
  await expect(tasks.getByTestId("os-pending-app")).toContainText("Tasks");
});

test("E2E-008: palette stays global while RuntimeSelector owns scoped ⌘J", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const session = await createNamedSession(runtime, workspace.id, "Palette target");

  await appPage.keyboard.press("ControlOrMeta+K");
  const palette = appPage.getByTestId("os-command-palette");
  await expect(palette).toBeVisible();
  const search = palette.getByPlaceholder("Search apps, sessions, actions…");
  await search.fill("tasks");
  await search.press("Enter");
  await expect(appPage.getByTestId("os-window-app:tasks")).toBeVisible();

  await appPage.keyboard.press("ControlOrMeta+K");
  await search.fill("Palette target");
  await expect(palette.getByTestId(`os-palette-session-${session.id}`)).toBeVisible();
  await search.press("Enter");
  const sessionWindow = appPage.getByTestId(`os-window-session:${session.id}`);
  await expect(sessionWindow).toBeVisible();
  await expect(palette).toHaveCount(0);
  const composer = sessionWindow.getByTestId("composer-textarea");
  await expect(composer).toBeVisible();
  await composer.focus();
  await appPage.keyboard.press("ControlOrMeta+K");
  await expect(palette).toBeVisible();
  await appPage.keyboard.press("Escape");
  await expect(palette).toHaveCount(0);

  await openMenu(appPage, "Session");
  await appPage.getByTestId("os-menu-new-session").click();
  const createDialog = appPage.getByTestId("session-create-dialog");
  await expect(createDialog).toBeVisible();
  const runtimeTrigger = createDialog.getByTestId("session-create-runtime-select");
  await expect(runtimeTrigger).toContainText("⌘J");
  await runtimeTrigger.locator('button[data-focus="model"]').first().focus();
  await appPage.keyboard.press("ControlOrMeta+J");
  await expect(appPage.getByTestId("runtime-selector-popup")).toBeVisible();
});

test("E2E-010 and E2E-018: two contexts converge after one and simultaneous drags", async ({
  appPage,
  browser,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const second = await openPeerPage(browser, runtime);
  try {
    const firstWindow = await openDockApp(appPage, "Tasks", "tasks");
    const secondWindow = second.getByTestId("os-window-app:tasks");
    await expect(secondWindow).toBeVisible();

    await dragWindowBy(appPage, firstWindow, 84, 46);
    await expect
      .poll(() => windowPosition(second, secondWindow))
      .toEqual(await windowPosition(appPage, firstWindow));

    await Promise.all([
      dragWindowBy(appPage, firstWindow, 58, 24),
      dragWindowBy(second, secondWindow, -42, 64),
    ]);
    await expect
      .poll(async () => {
        const [first, peer] = await Promise.all([
          windowPosition(appPage, firstWindow),
          windowPosition(second, secondWindow),
        ]);
        const authoritative = await desktopWindowPosition(runtime, workspace.id, "tasks");
        return (
          positionsMatch(first, peer) &&
          positionsMatch(first, authoritative) &&
          positionsMatch(peer, authoritative)
        );
      })
      .toBe(true);
  } finally {
    await second.context().close();
  }
});

test("E2E-012: blocked desktop stream degrades without blocking work and recovers", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const degradedPage = await appPage.context().newPage();
  const stream = await routeDesktopStream(degradedPage, true);
  await degradedPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });

  const degradedStatus = degradedPage.getByRole("status", { name: /Desktop sync paused/ });
  await expect(degradedStatus).toBeVisible();
  const tasks = await openDockApp(degradedPage, "Tasks", "tasks");
  const before = await windowPosition(degradedPage, tasks);
  await dragWindowBy(degradedPage, tasks, 76, 38);
  expect(await windowPosition(degradedPage, tasks)).not.toEqual(before);

  stream.unblock();
  await expect(degradedStatus).toHaveCount(0);
  const recovered = await windowPosition(degradedPage, tasks);
  await degradedPage.reload({ waitUntil: "domcontentloaded" });
  await expect(degradedPage.getByTestId("os-window-app:tasks")).toBeVisible();
  await expect
    .poll(() => windowPosition(degradedPage, degradedPage.getByTestId("os-window-app:tasks")))
    .toEqual(recovered);
});

test("E2E-014: CLI desktop-state mutation moves an open web window live", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  const target = { x: 512, y: 136, w: 610, h: 430 };

  await setWindowFromCLI(runtime, workspace.id, "tasks", target);

  await expect.poll(() => windowPosition(appPage, tasks)).toEqual({ x: target.x, y: target.y });
});

test("E2E-017: palette unwinds above the bell one overlay at a time", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  await appPage.getByRole("button", { name: "Approvals" }).click();
  await expect(appPage.getByTestId("os-bell-popover")).toBeVisible();

  await appPage.keyboard.press("ControlOrMeta+K");
  await expect(appPage.getByTestId("os-command-palette")).toBeVisible();
  await expect(appPage.getByTestId("os-bell-popover")).toHaveCount(0);

  await appPage.keyboard.press("Escape");
  await expect(appPage.getByTestId("os-command-palette")).toHaveCount(0);
  await appPage.keyboard.press("Escape");
  await expect
    .poll(() =>
      appPage.getByTestId("os-desktop").evaluate(node => node.contains(document.activeElement))
    )
    .toBe(true);
});

test("E2E-019: degraded recovery preserves touched keys and adopts daemon truth", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  await putWindow(runtime, workspace.id, "dashboard", { x: 48, y: 44, w: 640, h: 500 });

  const degradedPage = await appPage.context().newPage();
  const stream = await routeDesktopStream(degradedPage, true);
  await degradedPage.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  const degradedStatus = degradedPage.getByRole("status", { name: /Desktop sync paused/ });
  await expect(degradedStatus).toBeVisible();

  // One page owns one pointer/focus stream; open sequentially, then arrange.
  const arranged = [
    await openDockApp(degradedPage, "Tasks", "tasks"),
    await openDockApp(degradedPage, "Agents", "agents"),
    await openDockApp(degradedPage, "Loops", "loops"),
  ];
  await dragWindowBy(degradedPage, arranged[0], 60, 18);
  await dragWindowBy(degradedPage, arranged[1], -32, 52);
  await dragWindowBy(degradedPage, arranged[2], 44, -26);
  const expected = await Promise.all(arranged.map(win => windowPosition(degradedPage, win)));

  stream.unblock();
  await expect(degradedStatus).toHaveCount(0);
  await expect(degradedPage.getByTestId("os-window-app:dashboard")).toBeVisible();
  await degradedPage.reload({ waitUntil: "domcontentloaded" });
  await expect(degradedPage.getByTestId("os-window-app:dashboard")).toBeVisible();
  for (const [index, app] of ["tasks", "agents", "loops"].entries()) {
    await expect
      .poll(() => windowPosition(degradedPage, degradedPage.getByTestId(`os-window-app:${app}`)))
      .toEqual(expected[index]);
  }
});

test("E2E-022: menubar operates workspaces, sessions, Spaces, help, logo, and settings", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const secondWorkspace = await addSecondWorkspace(runtime);
  await appPage.reload({ waitUntil: "domcontentloaded" });

  await appPage.locator('[data-slot="os-menubar-workspace"]').click();
  await expect(appPage.getByTestId(`os-workspace-option-${workspace.id}`)).toBeVisible();
  await appPage.getByTestId(`os-workspace-option-${secondWorkspace.id}`).click();
  await expect(appPage.locator('[data-slot="os-menubar-workspace"]')).toContainText(
    secondWorkspace.name
  );

  await openMenu(appPage, "Session");
  await appPage.getByTestId("os-menu-new-session").click();
  await expect(appPage.getByTestId("session-create-dialog")).toBeVisible();
  await appPage.keyboard.press("Escape");

  await openMenu(appPage, "View");
  await appPage.getByTestId("os-menu-spaces-overview").click();
  const spaces = appPage.getByTestId("os-spaces-overview");
  await expect(spaces).toBeVisible();
  await expect(
    spaces.getByRole("button", { name: new RegExp(`Current workspace ${secondWorkspace.name}`) })
  ).toBeVisible();
  await expect(
    spaces.getByRole("button", { name: new RegExp(`Switch to ${workspace.name}`) })
  ).toBeVisible();
  await appPage.keyboard.press("Escape");
  await appPage.keyboard.press("ControlOrMeta+Shift+S");
  await expect(spaces).toBeVisible();
  await appPage.keyboard.press("Escape");

  await openMenu(appPage, "Help");
  await expect(appPage.getByTestId("os-help-shortcuts")).toContainText("⇧⌘S");
  await appPage.keyboard.press("Escape");

  await appPage.getByRole("button", { name: "AGH" }).click();
  await expect(appPage.getByTestId("os-window-app:dashboard")).toBeVisible();
  await appPage.getByRole("button", { name: "Settings" }).click();
  await expect(appPage.getByTestId("os-window-app:settings")).toBeVisible();
});

async function prepareShell(page: Page, runtime: BrowserRuntime): Promise<WorkspacePayload> {
  await useGlobalWorkspaceIfPrompted(page);
  await expect(page.getByTestId("os-desktop")).toBeVisible();
  const payload = await runtime.requestJSON<{ workspaces: WorkspacePayload[] }>("/api/workspaces");
  const workspace = payload.workspaces[0];
  if (!workspace) throw new Error("OS shell E2E requires one resolved workspace");
  return workspace;
}

async function openPeerPage(browser: Browser, runtime: BrowserRuntime): Promise<Page> {
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  const page = await context.newPage();
  await page.goto(runtime.url("/"), { waitUntil: "domcontentloaded" });
  await useGlobalWorkspaceIfPrompted(page);
  return page;
}

async function openDockApp(page: Page, name: string, app: string) {
  await page.getByRole("button", { name }).click();
  const win = page.getByTestId(`os-window-app:${app}`);
  await expect(win).toBeVisible();
  return win;
}

async function dragWindowBy(page: Page, win: ReturnType<Page["locator"]>, dx: number, dy: number) {
  const head = win.locator('[data-slot="os-window-head"]');
  const box = await head.boundingBox();
  if (!box) throw new Error("window head must have a visible bounding box before dragging");
  const x = box.x + box.width / 2;
  const y = box.y + box.height / 2;
  await page.mouse.move(x, y);
  await page.mouse.down();
  await page.mouse.move(x + dx, y + dy, { steps: 6 });
  await page.mouse.up();
}

async function windowPosition(page: Page, win: ReturnType<Page["locator"]>) {
  const [windowBox, layerBox] = await Promise.all([
    win.boundingBox(),
    page.locator('[data-slot="os-win-layer"]').boundingBox(),
  ]);
  if (!windowBox || !layerBox) throw new Error("window and win-layer must be visible");
  return { x: Math.round(windowBox.x - layerBox.x), y: Math.round(windowBox.y - layerBox.y) };
}

function positionsMatch(
  first: { x: number; y: number },
  second: { x: number; y: number }
): boolean {
  return Math.abs(first.x - second.x) <= 1 && Math.abs(first.y - second.y) <= 1;
}

async function createNamedSession(runtime: BrowserRuntime, workspaceId: string, name: string) {
  const payload = await runtime.requestJSON<{
    session: { id: string; agent_name: string };
  }>("/api/sessions", {
    method: "POST",
    body: JSON.stringify({ agent_name: browserLifecycleAgent, name, workspace: workspaceId }),
  });
  return payload.session;
}

async function routeDesktopStream(page: Page, initiallyBlocked: boolean) {
  let blocked = initiallyBlocked;
  await page.routeWebSocket("**/desktop-state/stream", async (socket: WebSocketRoute) => {
    if (blocked) {
      await socket.close({ code: 1013, reason: "E2E stream blocked" });
      return;
    }
    socket.connectToServer();
  });
  return { unblock: () => (blocked = false) };
}

async function setWindowFromCLI(
  runtime: BrowserRuntime,
  workspaceId: string,
  app: string,
  rect: { x: number; y: number; w: number; h: number }
): Promise<void> {
  if (!runtime.paths) throw new Error("E2E-014 requires launch-mode runtime paths");
  const value = JSON.stringify(windowPayload(app, rect));
  await execFileAsync(
    runtime.paths.cliShim,
    [
      "desktop-state",
      "set",
      "--workspace",
      workspaceId,
      "--key",
      `win:app:${app}`,
      "--value",
      value,
      "-o",
      "json",
    ],
    {
      env: { ...process.env, AGH_HOME: runtime.paths.homeDir, HOME: runtime.paths.homeDir },
      maxBuffer: 10 * 1024 * 1024,
    }
  );
}

async function putWindow(
  runtime: BrowserRuntime,
  workspaceId: string,
  app: string,
  rect: { x: number; y: number; w: number; h: number }
): Promise<void> {
  await runtime.requestJSON(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/desktop-state/${encodeURIComponent(`win:app:${app}`)}`,
    { method: "PUT", body: JSON.stringify({ value: windowPayload(app, rect) }) }
  );
}

async function desktopWindowPosition(
  runtime: BrowserRuntime,
  workspaceId: string,
  app: string
): Promise<{ x: number; y: number }> {
  const entry = await runtime.requestJSON<{ value: { rect: { x: number; y: number } } }>(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/desktop-state/${encodeURIComponent(`win:app:${app}`)}`
  );
  return { x: entry.value.rect.x, y: entry.value.rect.y };
}

function windowPayload(app: string, rect: { x: number; y: number; w: number; h: number }) {
  return {
    v: 1,
    app,
    instanceKey: null,
    location: { pathname: app === "dashboard" ? "/" : `/${app}`, search: {} },
    rect,
    prevRect: null,
    z: 1,
    minimized: false,
    maximized: false,
  };
}

async function addSecondWorkspace(runtime: BrowserRuntime): Promise<WorkspacePayload> {
  if (!runtime.paths) throw new Error("workspace switch E2E requires launch-mode runtime paths");
  const rootDir = path.join(runtime.paths.homeDir, "os-shell-second-workspace");
  const { mkdir } = await import("node:fs/promises");
  await mkdir(rootDir, { recursive: true });
  return await runtime.resolveWorkspace(rootDir);
}

async function openMenu(page: Page, name: "Session" | "View" | "Help"): Promise<void> {
  await page.getByRole("button", { name, exact: true }).click();
  await expect(page.getByTestId(`os-menu-${name.toLowerCase()}`)).toBeVisible();
}
