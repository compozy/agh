import { execFile } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import type { Browser, Page, WebSocketRoute } from "@playwright/test";

import type { BrowserRuntime, WorkspacePayload } from "../fixtures/runtime";
import { tasksOperatorSelectors } from "../fixtures/selectors";
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
  "os_shell_multi_session_fixture.json"
);

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          agentName: browserLifecycleAgent,
          fixtureAgent: "os-shell-multi-session-agent",
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

test("E2E-002: Tasks drag persists through reload", async ({ appPage, runtime }) => {
  const workspace = await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  await expect(appPage).toHaveURL(/\/tasks$/);
  const opened = await windowRect(appPage, tasks);
  await expect
    .poll(() => desktopWindowPosition(runtime, workspace.id, "tasks"))
    .toEqual({ x: opened.x, y: opened.y });

  await dragWindowBy(appPage, tasks, 92, 48);
  const dragged = await windowRect(appPage, tasks);
  await expect
    .poll(() => desktopWindowPosition(runtime, workspace.id, "tasks"))
    .toEqual({ x: dragged.x, y: dragged.y });

  await appPage.reload({ waitUntil: "domcontentloaded" });
  const restored = appPage.getByTestId("os-window-app:tasks");
  await expect(restored).toBeVisible();
  await expect.poll(() => windowRect(appPage, restored)).toEqual(dragged);
});

test("E2E-003: Tasks resize persists and zoom restores its exact rect", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  await resizeWindowBy(appPage, tasks, 74, 46);
  const resized = await windowRect(appPage, tasks);

  await appPage.reload({ waitUntil: "domcontentloaded" });
  const restored = appPage.getByTestId("os-window-app:tasks");
  await expect(restored).toBeVisible();
  await expect.poll(() => windowRect(appPage, restored)).toEqual(resized);

  await restored.getByRole("button", { name: "Zoom window" }).click();
  await expect.poll(() => windowRect(appPage, restored)).not.toEqual(resized);
  await restored.getByRole("button", { name: "Zoom window" }).click();
  await expect.poll(() => windowRect(appPage, restored)).toEqual(resized);
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
  await expect(tasks.getByTestId("tasks-shell")).toBeVisible();
});

test("E2E-005: a direct task detail deep link returns to the catalog with Back", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const task = await createTask(runtime, "Deep-link task");
  await appPage.goto(runtime.url("/tasks"), { waitUntil: "domcontentloaded" });
  await expect(appPage.getByTestId("os-window-app:tasks")).toBeVisible();

  await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(task.id)}`), {
    waitUntil: "domcontentloaded",
  });
  const tasksUI = tasksOperatorSelectors(appPage);
  const tasksWindow = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksUI.detailContent).toBeVisible();
  await expect(tasksUI.detailBreadcrumbTasks).toBeVisible();
  await expect(tasksWindow.getByTestId("tasks-detail-title")).toContainText(task.title);
  await expect(tasksWindow.locator('[data-slot="topbar-title"]')).toContainText(task.title);
  await expect(tasksWindow.locator('[data-slot="topbar-crumbs"]')).toBeVisible();
  await expect(tasksWindow.locator('[data-slot="topbar-crumbs"]')).not.toContainText(/^agh\b/);

  await appPage.goBack({ waitUntil: "domcontentloaded" });
  await expect(appPage).toHaveURL(/\/tasks$/);
  await expect(tasksWindow.getByTestId("tasks-shell")).toBeVisible();
});

test("E2E-007: browser history refocuses Tasks and Agents without closing either", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  const agents = await openDockApp(appPage, "Agents", "agents");
  const tasksOpened = await windowPosition(appPage, tasks);
  await expect
    .poll(() => desktopWindowPosition(runtime, workspace.id, "tasks"))
    .toEqual(tasksOpened);
  const agentsOpened = await windowPosition(appPage, agents);
  await expect
    .poll(() => desktopWindowPosition(runtime, workspace.id, "agents"))
    .toEqual(agentsOpened);

  await putAppWindow(
    runtime,
    workspace.id,
    "tasks",
    { pathname: "/tasks", search: {} },
    { x: 24, y: 20, w: 560, h: 520 },
    1
  );
  await expect.poll(() => windowPosition(appPage, tasks)).toEqual({ x: 24, y: 20 });
  await putAppWindow(
    runtime,
    workspace.id,
    "agents",
    { pathname: "/agents", search: {} },
    { x: 660, y: 20, w: 520, h: 520 },
    2
  );
  await expect.poll(() => windowPosition(appPage, agents)).toEqual({ x: 660, y: 20 });

  await focusWindow(appPage, tasks);
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/tasks");
  await focusWindow(appPage, agents);
  await expect(appPage).toHaveURL(/\/agents$/);
  await focusWindow(appPage, tasks);
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/tasks");

  await appPage.goBack({ waitUntil: "domcontentloaded" });
  await expect(appPage).toHaveURL(/\/agents$/);
  await expect(agents).toHaveAttribute("data-focused", "");
  await expect(tasks).toBeVisible();
  await expect(agents).toBeVisible();

  await appPage.goForward({ waitUntil: "domcontentloaded" });
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/tasks");
  await expect(tasks).toHaveAttribute("data-focused", "");
  await expect(agents).toBeVisible();
});

test("E2E-006: two session windows stream independently through minimize and restore", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const primary = await createNamedSession(runtime, workspace.id, "Primary observer");
  const secondary = await createNamedSession(runtime, workspace.id, "Secondary responder");

  await appPage.getByRole("button", { name: "Sessions" }).click();
  const rail = appPage.getByTestId("os-sessions-rail");
  await expect(rail).toBeVisible();
  await rail.getByTestId(`os-rail-session-${primary.id}`).first().click();
  await rail.getByTestId(`os-rail-session-${secondary.id}`).first().click();
  await rail.getByRole("button", { name: "Close sessions" }).click();

  const primaryWindow = appPage.getByTestId(`os-window-session:${primary.id}`);
  const secondaryWindow = appPage.getByTestId(`os-window-session:${secondary.id}`);
  await expect(primaryWindow).toBeVisible();
  await expect(secondaryWindow).toBeVisible();

  await Promise.all([
    putSessionWindow(runtime, workspace.id, primary, { x: 8, y: 16, w: 610, h: 560 }, 1),
    putSessionWindow(runtime, workspace.id, secondary, { x: 626, y: 16, w: 610, h: 560 }, 2),
  ]);
  await expect.poll(() => windowPosition(appPage, primaryWindow)).toEqual({ x: 8, y: 16 });
  await expect.poll(() => windowPosition(appPage, secondaryWindow)).toEqual({ x: 626, y: 16 });
  const [primaryBox, secondaryBox] = await Promise.all([
    primaryWindow.boundingBox(),
    secondaryWindow.boundingBox(),
  ]);
  if (!primaryBox || !secondaryBox) throw new Error("session windows must have visible bounds");
  expect(primaryBox.x + primaryBox.width).toBeLessThanOrEqual(secondaryBox.x);

  const primaryComposer = primaryWindow.getByTestId("composer-textarea");
  const primaryTranscript = primaryWindow.getByTestId("chat-view");
  await primaryComposer.fill("observe primary stream");
  await primaryComposer.press("Enter");
  await expect(primaryTranscript).toContainText("Primary stream is warming up.");
  const parkedScrollTop = await primaryTranscript.evaluate(element => {
    const viewport = element as HTMLElement;
    if (viewport.scrollHeight <= viewport.clientHeight) return -1;
    viewport.scrollTop = Math.max(
      1,
      Math.floor((viewport.scrollHeight - viewport.clientHeight) / 3)
    );
    viewport.dispatchEvent(new Event("scroll"));
    return viewport.scrollTop;
  });
  expect(parkedScrollTop).toBeGreaterThan(0);

  const secondaryComposer = secondaryWindow.getByTestId("composer-textarea");
  const secondaryTranscript = secondaryWindow.getByTestId("chat-view");
  await secondaryComposer.fill("reply in secondary window");
  await secondaryComposer.press("Enter");
  await expect(secondaryTranscript).toContainText("Secondary stream started.");
  await expect(secondaryTranscript).toContainText("Secondary stream completed independently.");
  await expect(secondaryWindow).toHaveAttribute("data-focused", "");
  await expect(primaryWindow).not.toHaveAttribute("data-focused", "");
  await expect
    .poll(() => primaryTranscript.evaluate(element => element.scrollTop))
    .toBe(parkedScrollTop);

  await primaryWindow.getByRole("button", { name: "Minimize window" }).click();
  await expect(primaryWindow).toHaveCount(0);
  await expect
    .poll(() => sessionHistoryContains(runtime, workspace.id, primary.id, "arrived while"))
    .toBe(true);

  await appPage.getByRole("button", { name: "Sessions" }).click();
  const restoredRail = appPage.getByTestId("os-sessions-rail");
  await expect(restoredRail).toBeVisible();
  await restoredRail.getByTestId(`os-rail-session-${primary.id}`).first().click();
  const restoredPrimary = appPage.getByTestId(`os-window-session:${primary.id}`);
  await expect(restoredPrimary).toBeVisible();
  await expect(restoredPrimary.getByTestId("chat-view")).toContainText(
    "Primary stream event arrived while the window was minimized."
  );
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

test("E2E-015: bell approval stays live and a CLI-resolved item reports truthful conflict", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const tasksUI = tasksOperatorSelectors(appPage);
  const first = await createApprovalTask(runtime, "Primary approval");
  const bell = appPage.getByRole("button", { name: "Approvals" });

  await expect(bell).toHaveText("1");
  await bell.click();
  const firstAttentionRow = appPage.getByTestId(`os-attention-task-${first.id}`);
  await expect(firstAttentionRow).toContainText(first.title);
  await firstAttentionRow.click();
  const tasksWindow = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksWindow).toBeVisible();
  await expect(tasksUI.detailContent).toBeVisible();
  await expect(tasksUI.detailLifecycle).toHaveText(/awaiting approval/i);

  await tasksUI.detailBreadcrumbTasks.click();
  await tasksUI.modeInbox.click();
  await expect(tasksUI.inboxItem(first.id)).toBeVisible();
  const approveResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(first.id)}/approve`)
    );
  });
  await tasksUI.inboxApprove(first.id).click();
  const approveResponse = await approveResponsePromise;
  expect(approveResponse.ok()).toBe(true);
  await expect.poll(() => taskApprovalState(runtime, first.id)).toBe("approved");
  await expect(tasksUI.inboxItem(first.id)).toHaveCount(0);
  await expect(bell).toHaveText("");

  const second = await createApprovalTask(runtime, "CLI race approval");
  await expect(bell).toHaveText("1");
  await bell.click();
  await appPage.getByTestId(`os-attention-task-${second.id}`).click();
  await expect(tasksUI.detailLifecycle).toHaveText(/awaiting approval/i);
  await tasksUI.detailBreadcrumbTasks.click();
  await tasksUI.modeInbox.click();
  await expect(tasksUI.inboxItem(second.id)).toBeVisible();

  await approveTaskFromCLI(runtime, second.id);
  const rejectResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(second.id)}/reject`)
    );
  });
  await tasksUI.inboxReject(second.id).click();
  const rejectResponse = await rejectResponsePromise;
  expect(rejectResponse.status()).toBe(409);
  await expect(appPage.locator("[data-sonner-toast]:last-of-type")).toContainText(
    'cannot transition approval from "approved" to "rejected"'
  );
  await expect.poll(() => taskApprovalState(runtime, second.id)).toBe("approved");
  await expect(bell).toHaveText("");
});

test("E2E-024: a Tasks confirm stays scoped while a session remains interactive", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const session = await createNamedSession(runtime, workspace.id, "Scoped modal session");
  const task = await createTask(runtime, "Window-scoped deletion");

  await appPage.getByRole("button", { name: "Sessions" }).click();
  const rail = appPage.getByTestId("os-sessions-rail");
  await rail.getByTestId(`os-rail-session-${session.id}`).first().click();
  await rail.getByRole("button", { name: "Close sessions" }).click();
  const sessionWindow = appPage.getByTestId(`os-window-session:${session.id}`);
  const composer = sessionWindow.getByTestId("composer-textarea");
  await composer.fill("observe primary stream");
  await composer.press("Enter");
  await expect(sessionWindow.getByTestId("chat-view")).toContainText(
    "Primary stream is warming up."
  );

  await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(task.id)}`), {
    waitUntil: "domcontentloaded",
  });
  const tasksWindow = appPage.getByTestId("os-window-app:tasks");
  const tasksUI = tasksOperatorSelectors(appPage);
  await expect(tasksUI.detailContent).toBeVisible();
  await Promise.all([
    putSessionWindow(runtime, workspace.id, session, { x: 700, y: 20, w: 520, h: 560 }, 1),
    putAppWindow(
      runtime,
      workspace.id,
      "tasks",
      { pathname: `/tasks/${encodeURIComponent(task.id)}`, search: {} },
      { x: 24, y: 20, w: 640, h: 540 },
      2
    ),
  ]);
  await expect.poll(() => windowPosition(appPage, tasksWindow)).toEqual({ x: 24, y: 20 });
  await expect.poll(() => windowPosition(appPage, sessionWindow)).toEqual({ x: 700, y: 20 });

  await tasksUI.detailOverflow.click();
  await tasksUI.detailDelete.click();
  const dialog = tasksUI.detailDeleteDialog;
  await expect(dialog).toBeVisible();
  await expect(
    tasksWindow
      .locator('[data-slot="os-window-overlays"]')
      .getByTestId("tasks-detail-delete-dialog")
  ).toBeVisible();
  await expect(
    sessionWindow
      .locator('[data-slot="os-window-overlays"]')
      .getByTestId("tasks-detail-delete-dialog")
  ).toHaveCount(0);

  await composer.fill("session remains interactive while Tasks confirms");
  await expect(composer).toHaveValue("session remains interactive while Tasks confirms");

  const [windowBefore, dialogBefore] = await Promise.all([
    windowPosition(appPage, tasksWindow),
    dialog.boundingBox(),
  ]);
  if (!dialogBefore) throw new Error("task delete dialog must have visible bounds");
  await dragWindowBy(appPage, tasksWindow, 68, 34);
  const [windowAfter, dialogAfter] = await Promise.all([
    windowPosition(appPage, tasksWindow),
    dialog.boundingBox(),
  ]);
  if (!dialogAfter) throw new Error("task delete dialog must remain visible after dragging");
  expect({
    x: Math.round(dialogAfter.x - dialogBefore.x),
    y: Math.round(dialogAfter.y - dialogBefore.y),
  }).toEqual({ x: windowAfter.x - windowBefore.x, y: windowAfter.y - windowBefore.y });

  const deleteResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "DELETE" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(task.id)}`)
    );
  });
  await tasksUI.detailDeleteConfirm.click();
  expect((await deleteResponsePromise).ok()).toBe(true);
  await expect(dialog).toHaveCount(0);
  await expect(sessionWindow).toBeVisible();
  await expect(composer).toHaveValue("session remains interactive while Tasks confirms");
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

async function resizeWindowBy(
  page: Page,
  win: ReturnType<Page["locator"]>,
  dx: number,
  dy: number
) {
  const handle = win.locator("..").locator(".os-window-resize-handle");
  const box = await handle.boundingBox();
  if (!box) throw new Error("window resize handle must have a visible bounding box");
  const x = box.x + box.width / 2;
  const y = box.y + box.height / 2;
  await page.mouse.move(x, y);
  await page.mouse.down();
  await page.mouse.move(x + dx, y + dy, { steps: 6 });
  await page.mouse.up();
}

async function focusWindow(page: Page, win: ReturnType<Page["locator"]>) {
  const head = win.locator('[data-slot="os-window-head"]');
  const box = await head.boundingBox();
  if (!box) throw new Error("window head must have a visible bounding box before focusing");
  await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
}

async function windowPosition(page: Page, win: ReturnType<Page["locator"]>) {
  const [windowBox, layerBox] = await Promise.all([
    win.boundingBox(),
    page.locator('[data-slot="os-win-layer"]').boundingBox(),
  ]);
  if (!windowBox || !layerBox) throw new Error("window and win-layer must be visible");
  return { x: Math.round(windowBox.x - layerBox.x), y: Math.round(windowBox.y - layerBox.y) };
}

async function windowRect(page: Page, win: ReturnType<Page["locator"]>) {
  const [windowBox, layerBox] = await Promise.all([
    win.boundingBox(),
    page.locator('[data-slot="os-win-layer"]').boundingBox(),
  ]);
  if (!windowBox || !layerBox) throw new Error("window and win-layer must be visible");
  return {
    x: Math.round(windowBox.x - layerBox.x),
    y: Math.round(windowBox.y - layerBox.y),
    w: Math.round(windowBox.width),
    h: Math.round(windowBox.height),
  };
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

async function createApprovalTask(
  runtime: BrowserRuntime,
  title: string
): Promise<{ id: string; title: string }> {
  const payload = await runtime.requestJSON<{ task: { id: string; title: string } }>("/api/tasks", {
    method: "POST",
    body: JSON.stringify({
      approval_policy: "manual",
      description: "OS shell attention E2E approval fixture.",
      owner: { kind: "human", ref: "os-shell-operator" },
      priority: "high",
      scope: "global",
      title,
    }),
  });
  return payload.task;
}

async function createTask(
  runtime: BrowserRuntime,
  title: string
): Promise<{ id: string; title: string }> {
  const payload = await runtime.requestJSON<{ task: { id: string; title: string } }>("/api/tasks", {
    method: "POST",
    body: JSON.stringify({
      description: "OS shell window-controller E2E fixture.",
      owner: { kind: "human", ref: "os-shell-operator" },
      priority: "medium",
      scope: "global",
      title,
    }),
  });
  return payload.task;
}

async function taskApprovalState(runtime: BrowserRuntime, taskId: string): Promise<string> {
  const payload = await runtime.requestJSON<{
    task: {
      summary?: { approval_state?: string | null };
      task?: { approval_state?: string | null };
    };
  }>(`/api/tasks/${encodeURIComponent(taskId)}`);
  return payload.task.summary?.approval_state ?? payload.task.task?.approval_state ?? "";
}

async function approveTaskFromCLI(runtime: BrowserRuntime, taskId: string): Promise<void> {
  if (!runtime.paths) throw new Error("E2E-015 requires launch-mode runtime paths");
  await execFileAsync(
    runtime.paths.cliShim,
    ["task", "approve", taskId, "--idempotency-key", `os-shell-cli-${taskId}`, "-o", "json"],
    {
      env: { ...process.env, AGH_HOME: runtime.paths.homeDir, HOME: runtime.paths.homeDir },
      maxBuffer: 10 * 1024 * 1024,
    }
  );
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

async function putSessionWindow(
  runtime: BrowserRuntime,
  workspaceId: string,
  session: { id: string; agent_name: string },
  rect: { x: number; y: number; w: number; h: number },
  z: number
): Promise<void> {
  await runtime.requestJSON(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/desktop-state/${encodeURIComponent(`win:session:${session.id}`)}`,
    {
      method: "PUT",
      body: JSON.stringify({
        value: {
          v: 1,
          app: "session",
          instanceKey: session.id,
          location: {
            pathname: `/agents/${encodeURIComponent(session.agent_name)}/sessions/${encodeURIComponent(session.id)}`,
            search: {},
          },
          rect,
          prevRect: null,
          z,
          minimized: false,
          maximized: false,
        },
      }),
    }
  );
}

async function putAppWindow(
  runtime: BrowserRuntime,
  workspaceId: string,
  app: string,
  location: { pathname: string; search: Record<string, unknown> },
  rect: { x: number; y: number; w: number; h: number },
  z: number
): Promise<void> {
  await runtime.requestJSON(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/desktop-state/${encodeURIComponent(`win:app:${app}`)}`,
    {
      method: "PUT",
      body: JSON.stringify({
        value: {
          v: 1,
          app,
          instanceKey: null,
          location,
          rect,
          prevRect: null,
          z,
          minimized: false,
          maximized: false,
        },
      }),
    }
  );
}

async function sessionHistoryContains(
  runtime: BrowserRuntime,
  workspaceId: string,
  sessionId: string,
  expected: string
): Promise<boolean> {
  const history = await runtime.requestJSON<unknown>(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/sessions/${encodeURIComponent(sessionId)}/history`
  );
  return JSON.stringify(history).includes(expected);
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
