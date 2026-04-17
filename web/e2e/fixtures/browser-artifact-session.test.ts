// @vitest-environment jsdom

import { describe, expect, it } from "vitest";

import { captureRouteState } from "./browser-artifact-session";

describe("captureRouteState", () => {
  it("captures network route context that explains a failed operator flow", async () => {
    window.history.replaceState({}, "", "/network");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="network-tab-pills">
        <button data-testid="network-tab-channels" aria-pressed="false"></button>
        <button data-testid="network-tab-peers" aria-pressed="true"></button>
      </div>
      <div data-testid="network-channels-list-panel">
        <button data-testid="network-channel-item-builders"></button>
      </div>
      <section data-testid="network-channel-detail-panel">
        <h2>builders</h2>
      </section>
      <div data-testid="network-peers-list-panel">
        <button data-testid="network-peer-item-peer_ops"></button>
        <button data-testid="network-peer-item-peer_patch"></button>
      </div>
      <section data-testid="network-peer-detail-panel">
        <h2>mock-patch-worker</h2>
      </section>
      <article data-testid="network-channel-message-browser_msg_say_01"></article>
      <article data-testid="network-channel-message-browser_msg_direct_01"></article>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/network",
      title: "AGH",
      chat_view_visible: false,
      message_count: 0,
      network_view_visible: true,
      network_active_tab: "peers",
      network_channel_count: 1,
      network_peer_count: 2,
      network_message_count: 2,
      network_selected_channel: "builders",
      network_selected_peer: "mock-patch-worker",
    });
  });

  it("captures automation route context, selected item, and session-link state", async () => {
    window.history.replaceState({}, "", "/automation");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="automation-kind-tabs">
        <button data-testid="automation-kind-jobs" aria-pressed="true"></button>
        <button data-testid="automation-kind-triggers" aria-pressed="false"></button>
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
      pathname: "/automation",
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
