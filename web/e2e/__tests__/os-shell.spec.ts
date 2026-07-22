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
  const sessionsModal = appPage.getByTestId("os-sessions-modal");
  await expect(sessionsModal).toBeVisible();
  await sessionsModal.getByTestId(`os-sessions-modal-session-${primary.id}`).first().click();
  await expect(sessionsModal).toHaveCount(0);
  await appPage.getByRole("button", { name: "Sessions" }).click();
  await expect(sessionsModal).toBeVisible();
  await sessionsModal.getByTestId(`os-sessions-modal-session-${secondary.id}`).first().click();
  await expect(sessionsModal).toHaveCount(0);

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
  const restoredModal = appPage.getByTestId("os-sessions-modal");
  await expect(restoredModal).toBeVisible();
  await restoredModal.getByTestId(`os-sessions-modal-session-${primary.id}`).first().click();
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
  const sessionsModal = appPage.getByTestId("os-sessions-modal");
  await expect(sessionsModal).toBeVisible();
  await sessionsModal.getByTestId(`os-sessions-modal-session-${session.id}`).first().click();
  await expect(sessionsModal).toHaveCount(0);
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

test("E2E-025: drag-snap previews the zone, persists through reload, and reflows on resize", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  const overlay = appPage.getByTestId("os-snap-overlay");

  const grip = await windowGrip(tasks);
  const layerBox = await appPage.locator('[data-slot="os-win-layer"]').boundingBox();
  if (!layerBox) throw new Error("win-layer must be visible");
  await appPage.mouse.move(grip.x, grip.y);
  await appPage.mouse.down();
  // Mid-desk: dragging, but no zone captured — the overlay must not render yet.
  await appPage.mouse.move(grip.x + 40, grip.y + 30, { steps: 4 });
  await expect(overlay).toHaveCount(0);
  // Inside the 32px edge band of the right edge: the preview appears.
  await appPage.mouse.move(layerBox.x + layerBox.width - 12, grip.y + 30, { steps: 8 });
  await expect(overlay).toBeVisible();
  await appPage.mouse.up();

  await expect.poll(() => snappedToHalf(appPage, tasks, "right")).toBe(true);
  const committed = await desktopWindowPosition(runtime, workspace.id, "tasks");

  await appPage.reload({ waitUntil: "domcontentloaded" });
  const restored = appPage.getByTestId("os-window-app:tasks");
  await expect(restored).toBeVisible();
  await expect.poll(() => snappedToHalf(appPage, restored, "right")).toBe(true);

  // Derived reflow: the window keeps filling the right half at the new
  // viewport while the persisted rect stays the commit-time derivation.
  await appPage.setViewportSize({ width: 1040, height: 720 });
  await expect.poll(() => snappedToHalf(appPage, restored, "right")).toBe(true);
  expect(await desktopWindowPosition(runtime, workspace.id, "tasks")).toEqual(committed);
});

test("E2E-026: snap fractions converge across viewports and an agent snap arranges live", async ({
  appPage,
  browser,
  runtime,
}) => {
  await appPage.setViewportSize({ width: 1440, height: 900 });
  const workspace = await prepareShell(appPage, runtime);
  const second = await openPeerPage(browser, runtime);
  try {
    const firstWindow = await openDockApp(appPage, "Tasks", "tasks");
    const secondWindow = second.getByTestId("os-window-app:tasks");
    await expect(secondWindow).toBeVisible();

    await appPage.keyboard.press("Control+Alt+ArrowLeft");
    // Fractions converge: each client renders the LEFT HALF of its own
    // viewport; px intentionally differ between the two viewports.
    await expect.poll(() => snappedToHalf(appPage, firstWindow, "left")).toBe(true);
    await expect.poll(() => snappedToHalf(second, secondWindow, "left")).toBe(true);
    const firstRect = await windowRect(appPage, firstWindow);
    const secondRect = await windowRect(second, secondWindow);
    expect(firstRect.w).not.toBe(secondRect.w);

    // Agent-arranged snap (SD-011): a CLI write of snap fractions moves the
    // window to the right half in both clients, live.
    await setSnappedWindowFromCLI(runtime, workspace.id, "tasks", {
      fx: 0.5,
      fy: 0,
      fw: 0.5,
      fh: 1,
    });
    await expect.poll(() => snappedToHalf(appPage, firstWindow, "right")).toBe(true);
    await expect.poll(() => snappedToHalf(second, secondWindow, "right")).toBe(true);
  } finally {
    await second.context().close();
  }
});

test("E2E-027: palette snap, drag-away restore, and reduced-motion overlay", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  const preSnap = await windowRect(appPage, tasks);

  await appPage.keyboard.press("ControlOrMeta+K");
  const palette = appPage.getByTestId("os-command-palette");
  await expect(palette).toBeVisible();
  const search = palette.getByPlaceholder("Search apps, sessions, actions…");
  await search.fill("Snap left half");
  await search.press("Enter");
  await expect.poll(() => snappedToHalf(appPage, tasks, "left")).toBe(true);
  // The palette scrim blocks pointers while its exit animation runs; wait for
  // it to unmount before starting the head drag.
  await expect(appPage.locator('[data-slot="dialog-overlay"]')).toHaveCount(0);

  // Drag away from the zone: the window detaches and its pre-snap size
  // restores under the cursor.
  const layerBox = await appPage.locator('[data-slot="os-win-layer"]').boundingBox();
  if (!layerBox) throw new Error("win-layer must be visible");
  const grip = await windowGrip(tasks);
  await appPage.mouse.move(grip.x, grip.y);
  await appPage.mouse.down();
  await appPage.mouse.move(layerBox.x + layerBox.width / 2, layerBox.y + 260, { steps: 10 });
  await appPage.mouse.up();
  await expect
    .poll(async () => {
      try {
        const rect = await windowRect(appPage, tasks);
        return rect.w === preSnap.w && rect.h === preSnap.h;
      } catch {
        return false; // frame mid-remount; poll again
      }
    })
    .toBe(true);

  // Reduced motion: the overlay renders without fade/morph (attribute-gated
  // collapse) while snapping still works.
  await appPage.emulateMedia({ reducedMotion: "reduce" });
  const gripAgain = await windowGrip(tasks);
  await appPage.mouse.move(gripAgain.x, gripAgain.y);
  await appPage.mouse.down();
  await appPage.mouse.move(layerBox.x + 12, layerBox.y + layerBox.height / 2, { steps: 8 });
  const overlay = appPage.getByTestId("os-snap-overlay");
  await expect(overlay).toBeVisible();
  await expect(overlay).toHaveAttribute("data-reduced-motion", "");
  await appPage.mouse.up();
  await expect.poll(() => snappedToHalf(appPage, tasks, "left")).toBe(true);
});

test("E2E-028: resize-in-place keeps a window snapped and the linked seam resizes the pair", async ({
  appPage,
  runtime,
}) => {
  await appPage.setViewportSize({ width: 1440, height: 900 });
  await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  const settings = await openDockApp(appPage, "Settings", "settings");

  await focusWindow(appPage, tasks);
  await appPage.keyboard.press("Control+Alt+ArrowLeft");
  await expect.poll(() => snappedToHalf(appPage, tasks, "left")).toBe(true);
  await focusWindow(appPage, settings);
  await appPage.keyboard.press("Control+Alt+ArrowRight");
  await expect.poll(() => snappedToHalf(appPage, settings, "right")).toBe(true);

  // Resize-in-place: dragging the snapped window's own handle narrows it,
  // and it STAYS snapped (fractions rewrite — the neighbor is untouched).
  const before = await windowRect(appPage, tasks);
  await resizeWindowBy(appPage, tasks, -120, 0);
  await expect(tasks).toHaveAttribute("data-snapped", "");
  await expect.poll(async () => (await windowRect(appPage, tasks)).w).toBeLessThan(before.w - 100);
  await expect.poll(() => snappedToHalf(appPage, settings, "right")).toBe(true);

  // Re-snap to restore fraction adjacency, then drag the shared seam: BOTH
  // windows resize together (linked JointResize posture) and stay snapped.
  await focusWindow(appPage, tasks);
  await appPage.keyboard.press("Control+Alt+ArrowLeft");
  await expect.poll(() => snappedToHalf(appPage, tasks, "left")).toBe(true);
  const seam = appPage.locator('[data-slot="os-snap-seam"]');
  await expect(seam).toHaveCount(1);
  const seamBox = await seam.boundingBox();
  if (!seamBox) throw new Error("seam must be visible between snapped halves");
  await appPage.mouse.move(seamBox.x + seamBox.width / 2, seamBox.y + seamBox.height / 2);
  await appPage.mouse.down();
  await appPage.mouse.move(seamBox.x + seamBox.width / 2 + 150, seamBox.y + seamBox.height / 2, {
    steps: 8,
  });
  await appPage.mouse.up();
  const tasksAfter = await windowRect(appPage, tasks);
  const settingsAfter = await windowRect(appPage, settings);
  expect(tasksAfter.w).toBeGreaterThan(before.w + 100);
  expect(settingsAfter.w).toBeLessThan(before.w - 100);
  await expect(tasks).toHaveAttribute("data-snapped", "");
  await expect(settings).toHaveAttribute("data-snapped", "");
  // The pair still meets across the 8px gutter at the new boundary.
  expect(settingsAfter.x - (tasksAfter.x + tasksAfter.w)).toBeLessThanOrEqual(10);
  expect(settingsAfter.x - (tasksAfter.x + tasksAfter.w)).toBeGreaterThanOrEqual(6);

  // Fractions persist: the pair reloads at the seam-set ratio.
  await appPage.reload({ waitUntil: "domcontentloaded" });
  const tasksBack = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksBack).toBeVisible();
  await expect
    .poll(async () => {
      try {
        return Math.abs((await windowRect(appPage, tasksBack)).w - tasksAfter.w) <= 2;
      } catch {
        return false;
      }
    })
    .toBe(true);
});

test("E2E-029: dropping onto a snapped window splits its space and the zoom menu arranges presets", async ({
  appPage,
  runtime,
}) => {
  await appPage.setViewportSize({ width: 1440, height: 900 });
  await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  const settings = await openDockApp(appPage, "Settings", "settings");

  await focusWindow(appPage, tasks);
  await appPage.keyboard.press("Control+Alt+ArrowRight");
  await expect.poll(() => snappedToHalf(appPage, tasks, "right")).toBe(true);

  // Drag Settings over the snapped half's bottom third: the split preview
  // appears (window-relative zone), and the drop stacks both as quarters
  // separated by the gutter.
  const tasksRect = await tasks.boundingBox();
  if (!tasksRect) throw new Error("snapped tasks window must be visible");
  const grip = await windowGrip(settings);
  await appPage.mouse.move(grip.x, grip.y);
  await appPage.mouse.down();
  await appPage.mouse.move(
    tasksRect.x + tasksRect.width / 2,
    tasksRect.y + tasksRect.height * 0.85,
    { steps: 10 }
  );
  await expect(appPage.getByTestId("os-snap-overlay")).toBeVisible();
  await appPage.mouse.up();
  await expect(settings).toHaveAttribute("data-snapped", "");
  await expect(tasks).toHaveAttribute("data-snapped", "");
  const topRect = await windowRect(appPage, tasks);
  const bottomRect = await windowRect(appPage, settings);
  expect(bottomRect.y - (topRect.y + topRect.h)).toBeLessThanOrEqual(10);
  expect(bottomRect.y - (topRect.y + topRect.h)).toBeGreaterThanOrEqual(6);

  // Zoom-menu preset: hovering the zoom control opens Move & Resize / Fill &
  // Arrange; "Arrange left & right" pairs this window with the most recent.
  const zoomButton = tasks.locator('button[data-action="zoom"]');
  await zoomButton.hover();
  const menu = appPage.getByTestId("os-zoom-menu");
  await expect(menu).toBeVisible();
  await menu.getByTestId("os-zoom-menu-two-up").click();
  await expect.poll(() => snappedToHalf(appPage, tasks, "left")).toBe(true);
  await expect.poll(() => snappedToHalf(appPage, settings, "right")).toBe(true);
});

test("E2E-009: workspace spaces stay independent and the overview restores arrangements", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const secondWorkspace = await addSecondWorkspace(runtime);

  // Arrange workspace A: two windows, one dragged to a distinctive spot.
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  await dragWindowBy(appPage, tasks, 140, 90);
  const arrangedTasks = await windowPosition(appPage, tasks);
  await openDockApp(appPage, "Agents", "agents");

  // Switch to B via the menubar: a different, empty space appears.
  await appPage.locator('[data-slot="os-menubar-workspace"]').click();
  await appPage.getByTestId(`os-workspace-option-${secondWorkspace.id}`).click();
  await expect(appPage.getByTestId("os-desk-hint")).toBeVisible();
  await expect(appPage.getByTestId("os-window-app:tasks")).toHaveCount(0);

  // One window in B, then the ⇧⌘S overview shows both spaces with thumbnails.
  await openDockApp(appPage, "Vault", "vault");
  await appPage.keyboard.press("ControlOrMeta+Shift+S");
  const spaces = appPage.getByTestId("os-spaces-overview");
  await expect(spaces).toBeVisible();
  const cardA = spaces.locator(`[data-workspace-id="${workspace.id}"]`);
  const cardB = spaces.locator(`[data-workspace-id="${secondWorkspace.id}"]`);
  await expect(cardB).toHaveAttribute("data-current", "true");
  await expect(cardB.locator('[data-slot="os-space-mini-win"]')).toHaveCount(1);
  // A's persisted arrangement thumbnails load over HTTP while the overview is open.
  await expect(cardA.locator('[data-slot="os-space-mini-win"]')).toHaveCount(2);

  // Choosing A restores its exact arrangement.
  await cardA.click();
  const tasksBack = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksBack).toBeVisible();
  await expect(appPage.getByTestId("os-window-app:agents")).toBeVisible();
  await expect(appPage.getByTestId("os-window-app:vault")).toHaveCount(0);
  await expect
    .poll(async () => positionsMatch(await windowPosition(appPage, tasksBack), arrangedTasks))
    .toBe(true);
});

test("E2E-011: the compact stack round-trips with floating rects preserved", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  await dragWindowBy(appPage, tasks, 120, 80);
  const floatingRect = await windowRect(appPage, tasks);

  // Below the breakpoint: stacked fullscreen presentation with the tab bar.
  await appPage.setViewportSize({ width: 390, height: 844 });
  await expect(tasks).toHaveAttribute("data-presentation", "compact");
  await expect(appPage.locator('[data-slot="os-dock-tabbar"]')).toBeVisible();
  await expect(tasks.locator(".os-window-resize-handle")).toHaveCount(0);
  await expect(tasks.getByRole("button", { name: "Zoom window" })).toHaveCount(0);
  const closeTarget = await tasks.getByRole("button", { name: "Close window" }).boundingBox();
  const minimizeTarget = await tasks.getByRole("button", { name: "Minimize window" }).boundingBox();
  if (!closeTarget || !minimizeTarget) {
    throw new Error("compact window controls must expose measurable touch targets");
  }
  expect(closeTarget.width).toBeGreaterThanOrEqual(44);
  expect(closeTarget.height).toBeGreaterThanOrEqual(44);
  expect(minimizeTarget.width).toBeGreaterThanOrEqual(44);
  expect(minimizeTarget.height).toBeGreaterThanOrEqual(44);
  expect(closeTarget.x + closeTarget.width).toBeLessThanOrEqual(minimizeTarget.x);
  const stackBox = await tasks.boundingBox();
  const viewport = appPage.viewportSize();
  if (!stackBox || !viewport) throw new Error("compact stack window must be measurable");
  expect(Math.round(stackBox.width)).toBe(viewport.width);

  // Back above the breakpoint: the floating rect returns exactly.
  await appPage.setViewportSize({ width: 1280, height: 720 });
  await expect(tasks).not.toHaveAttribute("data-presentation", "compact");
  await expect
    .poll(async () => rectsClose(await windowRect(appPage, tasks), floatingRect))
    .toBe(true);
});

test("E2E-013: wallpaper persists per space and reduce-motion makes minimize instant", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  await openDockApp(appPage, "Tasks", "tasks");

  // Appearance pane via View → Appearance…: pick the carbon wallpaper.
  await openMenu(appPage, "View");
  await appPage.getByTestId("os-menu-appearance").click();
  await expect(appPage.getByTestId("os-appearance-pane")).toBeVisible();
  await appPage.getByTestId("os-wallpaper-option-carbon").click();
  const wallpaper = appPage.locator('[data-slot="os-wallpaper"]');
  await expect(wallpaper).toHaveAttribute("data-wallpaper", "carbon");

  // The choice persists with the space across a reload.
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(appPage.locator('[data-slot="os-wallpaper"]')).toHaveAttribute(
    "data-wallpaper",
    "carbon"
  );

  // Full motion first: the genie fold class appears during minimize.
  const tasksWindow = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksWindow).toBeVisible();
  await appPage.evaluate(() => {
    const flag = { saw: false };
    Reflect.set(window, "__osGenie", flag);
    const observer = new MutationObserver(() => {
      if (document.querySelector(".os-window-minimizing")) flag.saw = true;
    });
    observer.observe(document.body, {
      subtree: true,
      attributes: true,
      attributeFilter: ["class"],
    });
  });
  await tasksWindow.getByRole("button", { name: "Minimize window" }).click();
  await expect(tasksWindow).toBeHidden();
  expect(await appPage.evaluate(() => Reflect.get(window, "__osGenie").saw)).toBe(true);

  // Restore, enable the in-product reduce-motion toggle: minimize is instant.
  await appPage.getByRole("button", { name: "Tasks", exact: true }).click();
  await expect(tasksWindow).toBeVisible();
  // The menubar cog refocuses the settings window above the restored tasks.
  await appPage.getByRole("button", { name: "Settings" }).click();
  await expect(appPage.getByTestId("os-appearance-pane")).toBeVisible();
  await appPage.getByTestId("os-appearance-reduce-motion").click();
  await appPage.evaluate(() => {
    Reflect.set(window, "__osGenie", { saw: false });
    const flag = Reflect.get(window, "__osGenie") as { saw: boolean };
    const observer = new MutationObserver(() => {
      if (document.querySelector(".os-window-minimizing")) flag.saw = true;
    });
    observer.observe(document.body, {
      subtree: true,
      attributes: true,
      attributeFilter: ["class"],
    });
  });
  // Dock click refocuses tasks above settings without a pointer-position race.
  await appPage.getByRole("button", { name: "Tasks", exact: true }).click();
  await tasksWindow.getByRole("button", { name: "Minimize window" }).click();
  await expect(tasksWindow).toBeHidden();
  expect(await appPage.evaluate(() => Reflect.get(window, "__osGenie").saw)).toBe(false);
});

test("E2E-016: a cross-workspace session deep link switches spaces and leaves both intact", async ({
  appPage,
  runtime,
}) => {
  const workspace = await prepareShell(appPage, runtime);
  const secondWorkspace = await addSecondWorkspace(runtime);
  const session = await createNamedSession(runtime, secondWorkspace.id, "cross-space-session");

  // Arrange A so its integrity is checkable after the round trip.
  const tasks = await openDockApp(appPage, "Tasks", "tasks");
  await dragWindowBy(appPage, tasks, 130, 70);
  const arrangedTasks = await windowPosition(appPage, tasks);

  // Follow the session link owned by B: the shell switches to B's space and
  // opens that session focused there — never cross-workspace in place.
  await appPage.goto(
    runtime.url(
      `/agents/${encodeURIComponent(session.agent_name)}/sessions/${encodeURIComponent(session.id)}`
    ),
    { waitUntil: "domcontentloaded" }
  );
  await expect(appPage.locator('[data-slot="os-menubar-workspace"]')).toContainText(
    secondWorkspace.name
  );
  const sessionWindow = appPage.getByTestId(`os-window-session:${session.id}`);
  await expect(sessionWindow).toBeVisible();
  await expect(appPage.getByTestId("os-window-app:tasks")).toHaveCount(0);

  // Switching back shows A untouched: same windows, same position, no leak.
  await appPage.locator('[data-slot="os-menubar-workspace"]').click();
  await appPage.getByTestId(`os-workspace-option-${workspace.id}`).click();
  const tasksBack = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksBack).toBeVisible();
  await expect(appPage.getByTestId(`os-window-session:${session.id}`)).toHaveCount(0);
  await expect
    .poll(async () => positionsMatch(await windowPosition(appPage, tasksBack), arrangedTasks))
    .toBe(true);
});

test("E2E-020: compact keeps deep links, truthful badges, and the rail overlay working", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  await createApprovalTask(runtime, "Compact parity approval");
  const detailTask = await createTask(runtime, "Compact deep link target");

  await appPage.setViewportSize({ width: 390, height: 844 });
  await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(detailTask.id)}`), {
    waitUntil: "domcontentloaded",
  });
  await useGlobalWorkspaceIfPrompted(appPage);

  // Deep link lands focused in the stack.
  const tasksWindow = appPage.getByTestId("os-window-app:tasks");
  await expect(tasksWindow).toBeVisible();
  await expect(tasksWindow).toHaveAttribute("data-presentation", "compact");
  await expect(tasksWindow).toContainText(detailTask.title);

  // Badges render truthfully in the tab bar (awaiting-approval projection).
  const tabbar = appPage.locator('[data-slot="os-dock-tabbar"]');
  await expect(tabbar).toBeVisible();
  await expect(tabbar.locator('[data-app="tasks"] [data-slot="os-dock-badge"]')).toHaveText("1");

  // Tab-bar semantics: tapping the focused app switches to it — never minimizes.
  await tabbar.locator('[data-app="tasks"]').click();
  await expect(tasksWindow).toBeVisible();

  // The sessions catalog presents as a global modal; dismissing returns intact.
  await tabbar.locator('[data-app="session"]').click();
  const sessionsModal = appPage.getByTestId("os-sessions-modal");
  await expect(sessionsModal).toBeVisible();
  await appPage.keyboard.press("Escape");
  await expect(sessionsModal).toHaveCount(0);
  await expect(tasksWindow).toBeVisible();
  await expect(tasksWindow).toContainText(detailTask.title);
});

test("E2E-021: the system reduced-motion preference wins over the in-product toggle", async ({
  appPage,
  runtime,
}) => {
  await prepareShell(appPage, runtime);
  await appPage.emulateMedia({ reducedMotion: "reduce" });

  // In-product motion stays "full" (toggle off — the default), system says
  // reduce: dock magnification must stay static (US-015.EC-1).
  await openDockApp(appPage, "Tasks", "tasks");
  const dockItem = appPage.locator('[data-slot="os-dock"] [data-app="tasks"]');
  const box = await dockItem.boundingBox();
  if (!box) throw new Error("dock item must be visible");
  await appPage.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await appPage.mouse.move(box.x + box.width / 2 + 4, box.y + box.height / 2, { steps: 3 });
  await expect
    .poll(() => dockItem.evaluate(element => (element as HTMLElement).style.transform))
    .toBe("");

  // And the genie minimize collapses to instant despite the toggle being off.
  await appPage.evaluate(() => {
    const flag = { saw: false };
    Reflect.set(window, "__osGenie", flag);
    const observer = new MutationObserver(() => {
      if (document.querySelector(".os-window-minimizing")) flag.saw = true;
    });
    observer.observe(document.body, {
      subtree: true,
      attributes: true,
      attributeFilter: ["class"],
    });
  });
  const tasksWindow = appPage.getByTestId("os-window-app:tasks");
  await tasksWindow.getByRole("button", { name: "Minimize window" }).click();
  await expect(tasksWindow).toBeHidden();
  expect(await appPage.evaluate(() => Reflect.get(window, "__osGenie").saw)).toBe(false);
});

const PERF_APPS = [
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

test("E2E-023: the 12-window envelope holds for drag frames, restore, and convergence", async ({
  appPage,
  browser,
  runtime,
}, testInfo) => {
  const workspace = await prepareShell(appPage, runtime);

  // Seed 12 windows through the public desktop-state surface.
  for (const [index, app] of PERF_APPS.entries()) {
    await putAppWindow(
      runtime,
      workspace.id,
      app,
      { pathname: app === "dashboard" ? "/" : `/${app}`, search: {} },
      { x: 16 + index * 24, y: 12 + (index % 4) * 30, w: 480, h: 360 },
      index + 1
    );
  }

  // Restore instrumentation: first desktop-state stream frame → 12th window.
  await appPage.addInitScript(() => {
    const perf = { wsFirstFrame: null as number | null, windowsPlaced: null as number | null };
    Reflect.set(window, "__osPerf", perf);
    const placed = () => {
      if (perf.windowsPlaced !== null) return;
      const count = document.querySelectorAll('[data-testid^="os-window-app:"]').length;
      if (count >= 12) perf.windowsPlaced = performance.now();
    };
    // Init scripts run at document start — observe `document` itself so the
    // hook works before <html>/<body> exist.
    new MutationObserver(placed).observe(document, { childList: true, subtree: true });
    const NativeWebSocket = window.WebSocket;
    class MeasuredWebSocket extends NativeWebSocket {
      constructor(url: string | URL, protocols?: string | string[]) {
        super(url, protocols);
        if (String(url).includes("/desktop-state/stream")) {
          this.addEventListener(
            "message",
            () => {
              if (perf.wsFirstFrame === null) perf.wsFirstFrame = performance.now();
            },
            { once: true }
          );
        }
      }
    }
    window.WebSocket = MeasuredWebSocket as typeof WebSocket;
  });
  await appPage.reload({ waitUntil: "domcontentloaded" });
  for (const app of PERF_APPS) {
    await expect(appPage.getByTestId(`os-window-app:${app}`)).toBeAttached();
  }
  const restore = await appPage.evaluate(() => {
    const perf = Reflect.get(window, "__osPerf") as {
      wsFirstFrame: number | null;
      windowsPlaced: number | null;
    };
    return perf.wsFirstFrame !== null && perf.windowsPlaced !== null
      ? perf.windowsPlaced - perf.wsFirstFrame
      : null;
  });
  expect(restore).not.toBeNull();
  expect(restore ?? Number.POSITIVE_INFINITY).toBeLessThan(500);

  // The envelope measures steady-state pointer fluidity: wait until the main
  // thread has been long-task quiet for 600ms so the 12 window bodies' initial
  // content burst can't masquerade as drag jank. (networkidle never settles
  // here — the shell keeps WebSocket/SSE connections open by design.)
  await appPage.evaluate(() => {
    const settle = { last: performance.now() };
    Reflect.set(window, "__osSettle", settle);
    new PerformanceObserver(list => {
      for (const entry of list.getEntries()) {
        settle.last = Math.max(settle.last, entry.startTime + entry.duration);
      }
    }).observe({ type: "longtask", buffered: true });
  });
  await appPage.waitForFunction(() => {
    const settle = Reflect.get(window, "__osSettle") as { last: number };
    return performance.now() - settle.last > 600;
  });

  // Long-task probe during a 3s continuous drag of one window.
  await appPage.evaluate(() => {
    const tasks: number[] = [];
    Reflect.set(window, "__osLongTasks", tasks);
    new PerformanceObserver(list => {
      for (const entry of list.getEntries()) tasks.push(entry.duration);
    }).observe({ type: "longtask", buffered: false });
  });
  const dragged = appPage.getByTestId("os-window-app:vault");
  await focusWindow(appPage, dragged);
  const grip = await windowGrip(dragged);
  await appPage.mouse.move(grip.x, grip.y);
  await appPage.mouse.down();
  const start = Date.now();
  let step = 0;
  while (Date.now() - start < 3000) {
    const angle = (step / 20) * Math.PI * 2;
    await appPage.mouse.move(
      grip.x + 120 + Math.cos(angle) * 90,
      grip.y + 100 + Math.sin(angle) * 60,
      { steps: 2 }
    );
    step += 1;
  }
  await appPage.mouse.up();
  const longTasks = await appPage.evaluate(() => Reflect.get(window, "__osLongTasks") as number[]);
  const worstFrame = longTasks.length > 0 ? Math.max(...longTasks) : 0;

  // Three-client convergence: two peers adopt a CLI move without long tasks.
  const peerA = await openPeerPage(browser, runtime);
  const peerB = await openPeerPage(browser, runtime);
  for (const peer of [peerA, peerB]) {
    await expect(peer.getByTestId("os-window-app:sandbox")).toBeAttached();
    await peer.evaluate(() => {
      const tasks: number[] = [];
      Reflect.set(window, "__osLongTasks", tasks);
      new PerformanceObserver(list => {
        for (const entry of list.getEntries()) tasks.push(entry.duration);
      }).observe({ type: "longtask", buffered: false });
    });
  }
  await setWindowFromCLI(runtime, workspace.id, "sandbox", { x: 420, y: 240, w: 500, h: 380 });
  for (const peer of [peerA, peerB]) {
    const sandbox = peer.getByTestId("os-window-app:sandbox");
    await expect
      .poll(async () => {
        try {
          const [windowBox, layerBox] = await Promise.all([
            sandbox.boundingBox(),
            peer.locator('[data-slot="os-win-layer"]').boundingBox(),
          ]);
          if (!windowBox || !layerBox) return false;
          return (
            Math.abs(windowBox.x - layerBox.x - 420) <= 2 &&
            Math.abs(windowBox.y - layerBox.y - 240) <= 2
          );
        } catch {
          return false;
        }
      })
      .toBe(true);
  }
  const peerLongTasks = await Promise.all(
    [peerA, peerB].map(peer =>
      peer.evaluate(() => Reflect.get(window, "__osLongTasks") as number[])
    )
  );
  const worstPeerTask = Math.max(0, ...peerLongTasks.flat());
  await peerA.context().close();
  await peerB.context().close();

  const envelope = {
    restoreMsFromFirstStreamFrame: restore,
    dragLongTasksOver50ms: longTasks,
    worstDragFrameMs: worstFrame,
    worstPeerConvergenceTaskMs: worstPeerTask,
  };
  // Surfaced on stdout so completion notes can record the measured numbers.
  console.log(`[perf-envelope] ${JSON.stringify(envelope)}`);
  await testInfo.attach("perf-envelope", {
    body: JSON.stringify(envelope, null, 2),
    contentType: "application/json",
  });

  // Envelope: no shell frame beyond 50ms during the drag, no peer thrash.
  expect(worstFrame).toBeLessThanOrEqual(50);
  expect(worstPeerTask).toBeLessThanOrEqual(50);
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

/**
 * Poll body: is the window rendering as the derived half? Branch swaps
 * (Rnd ↔ absolute path) briefly detach the frame while snap toggles, so
 * measurement failures report false for the next poll instead of aborting.
 */
async function snappedToHalf(
  page: Page,
  win: ReturnType<Page["locator"]>,
  side: "left" | "right"
): Promise<boolean> {
  try {
    return rectsClose(await windowRect(page, win), await snapHalfRect(page, side));
  } catch {
    return false;
  }
}

/**
 * A guaranteed drag surface on the window head: the identity (glyph + title)
 * area is never inside the drag-cancel selectors, unlike the head center,
 * which can land on the mode tabs (`topbar-nav`) once an app publishes them.
 */
async function windowGrip(win: ReturnType<Page["locator"]>): Promise<{ x: number; y: number }> {
  const title = win.locator('[data-slot="topbar-title"]');
  const box = await title.boundingBox();
  if (!box) throw new Error("window title must be visible to start a head drag");
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

/**
 * Expected derived rect for a half zone at the CURRENT layer size — the same
 * work-area math clients run (insets 10/8/10/78, fractions with the inner
 * seam edge inset by half the 8px gutter).
 */
async function snapHalfRect(page: Page, side: "left" | "right") {
  const layerBox = await page.locator('[data-slot="os-win-layer"]').boundingBox();
  if (!layerBox) throw new Error("win-layer must be visible to derive snap rects");
  const area = {
    x: 10,
    y: 8,
    w: Math.max(1, layerBox.width - 20),
    h: Math.max(1, layerBox.height - 86),
  };
  const halfGutter = 4;
  const mid = area.x + Math.round(area.w * 0.5);
  const right = area.x + Math.round(area.w);
  const h = Math.round(area.h);
  return side === "left"
    ? { x: area.x, y: area.y, w: mid - halfGutter - area.x, h }
    : { x: mid + halfGutter, y: area.y, w: right - mid - halfGutter, h };
}

function rectsClose(
  first: { x: number; y: number; w: number; h: number },
  second: { x: number; y: number; w: number; h: number },
  tolerance = 2
): boolean {
  return (
    Math.abs(first.x - second.x) <= tolerance &&
    Math.abs(first.y - second.y) <= tolerance &&
    Math.abs(first.w - second.w) <= tolerance &&
    Math.abs(first.h - second.h) <= tolerance
  );
}

async function setSnappedWindowFromCLI(
  runtime: BrowserRuntime,
  workspaceId: string,
  app: string,
  snap: { fx: number; fy: number; fw: number; fh: number }
): Promise<void> {
  if (!runtime.paths) throw new Error("E2E-026 requires launch-mode runtime paths");
  const value = JSON.stringify({
    ...windowPayload(app, { x: 10, y: 8, w: 640, h: 480 }),
    snap,
  });
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
