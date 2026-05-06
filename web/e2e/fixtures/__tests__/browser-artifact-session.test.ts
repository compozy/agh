// @vitest-environment jsdom

import { describe, expect, it } from "vitest";

import { captureRouteState } from "../browser-artifact-session";

describe("captureRouteState", () => {
  it("captures network shell context with the channel-pivot information architecture", async () => {
    window.history.replaceState({}, "", "/network/builders/threads");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="network-shell">
        <aside data-testid="network-channel-rail">
          <div data-testid="network-channel-row-builders">
            <a data-testid="network-channel-link-builders" aria-current="page">builders</a>
          </div>
          <div data-testid="network-channel-row-design">
            <a data-testid="network-channel-link-design">design</a>
          </div>
        </aside>
        <main data-testid="network-main-pane">
          <header data-testid="network-channel-header"><h1>#builders</h1></header>
          <section data-testid="network-threads-tab">
            <article data-testid="network-thread-list-row-thread_one"></article>
            <article data-testid="network-thread-list-row-thread_two"></article>
          </section>
        </main>
      </div>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/network/builders/threads",
      title: "AGH",
      chat_view_visible: false,
      message_count: 0,
      network_view_visible: true,
      network_active_tab: "threads",
      network_channel_count: 2,
      network_thread_count: 2,
      network_direct_count: 0,
      network_message_count: 0,
      network_selected_channel: "builders",
    });
    expect(routeState).not.toHaveProperty("network_selected_peer");
    expect(routeState.network_selected_thread).toBeUndefined();
  });

  it("captures the selected thread overlay container id without leaking direct fields", async () => {
    window.history.replaceState({}, "", "/network/builders/threads/thread_launch_command");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="network-shell">
        <main data-testid="network-main-pane">
          <section data-testid="network-threads-tab">
            <article data-testid="network-thread-list-row-thread_launch_command"></article>
          </section>
          <aside data-testid="network-thread-overlay" aria-label="Thread"></aside>
        </main>
      </div>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      network_view_visible: true,
      network_active_tab: "threads",
      network_selected_thread: "thread_launch_command",
    });
    expect(routeState.network_selected_direct).toBeUndefined();
  });

  it("captures the selected direct room container id without leaking thread fields", async () => {
    window.history.replaceState({}, "", "/network/builders/directs/direct_abc123");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="network-shell">
        <main data-testid="network-main-pane">
          <section data-testid="network-direct-detail-slot" aria-label="Direct room direct_abc123 in #builders">
            <article data-testid="network-direct-room" aria-label="Direct room with @peer"></article>
          </section>
        </main>
      </div>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      network_view_visible: true,
      network_active_tab: "directs",
      network_selected_direct: "direct_abc123",
    });
    expect(routeState.network_selected_thread).toBeUndefined();
  });

  it("captures automation route context, selected item, and session-link state", async () => {
    window.history.replaceState({}, "", "/jobs");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="jobs-shell">
        <button data-testid="jobs-scope-all" aria-pressed="true"></button>
      </div>
      <aside data-testid="automation-list-panel">
        <button data-testid="automation-item-job_daily_review"></button>
        <button data-testid="automation-item-job_weekly_triage"></button>
      </aside>
      <section data-testid="automation-detail-panel">
        <h2>deploy-review</h2>
      </section>
      <section data-testid="automation-run-history">
        <article data-testid="automation-run-run_001"></article>
        <article data-testid="automation-run-run_002"></article>
        <a data-testid="automation-run-session-link-run_001"></a>
      </section>
      <form data-testid="automation-job-form"></form>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/jobs",
      title: "AGH",
      automation_view_visible: true,
      automation_active_tab: "jobs",
      automation_editor_kind: "job",
      automation_item_count: 2,
      automation_run_count: 2,
      automation_run_history_visible: true,
      automation_selected_item: "deploy-review",
      automation_session_link_count: 1,
    });
  });

  it("captures bridge route context, selected bridge, and dialog state", async () => {
    window.history.replaceState({}, "", "/bridges");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="bridge-scope-pills">
        <button data-testid="bridge-scope-all" aria-pressed="false"></button>
        <button data-testid="bridge-scope-global" aria-pressed="true"></button>
        <button data-testid="bridge-scope-workspace" aria-pressed="false"></button>
      </div>
      <aside data-testid="bridge-list-panel">
        <button data-testid="bridge-item-brg_ops"></button>
        <button data-testid="bridge-item-brg_support"></button>
      </aside>
      <section data-testid="bridge-detail-panel">
        <h2>Telegram Bridge Ops</h2>
      </section>
      <article data-testid="bridge-secret-binding-bot_token"></article>
      <article data-testid="bridge-route-sess_bridge_01"></article>
      <form data-testid="bridge-edit-dialog"></form>
      <section data-testid="bridge-test-delivery-result"></section>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/bridges",
      title: "AGH",
      bridge_view_visible: true,
      bridge_scope_filter: "global",
      bridge_item_count: 2,
      bridge_selected_item: "Telegram Bridge Ops",
      bridge_secret_binding_count: 1,
      bridge_route_count: 1,
      bridge_edit_dialog_open: true,
      bridge_test_delivery_result_visible: true,
    });
  });
});
