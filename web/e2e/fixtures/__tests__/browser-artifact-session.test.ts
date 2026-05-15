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
          <section data-testid="network-activity-feed">
            <a data-testid="network-activity-entry-thread:thread_one"></a>
          </section>
          <section data-testid="network-work-inspector">
            <li data-testid="network-work-inspector-row-work_one"></li>
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
      network_activity_count: 1,
      network_message_count: 0,
      network_work_count: 1,
      network_selected_channel: "builders",
    });
    expect(routeState).not.toHaveProperty("network_selected_peer");
    expect(routeState.network_selected_thread).toBeUndefined();
  });

  it("captures disabled and no-channel network route states for launch diagnostics", async () => {
    window.history.replaceState({}, "", "/network");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="network-shell">
        <section data-testid="network-no-channels-state"></section>
      </div>
      <section data-testid="network-disabled-state"></section>
      <form data-testid="network-create-channel-dialog"></form>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/network",
      network_create_dialog_open: true,
      network_disabled_visible: true,
      network_no_channels_visible: true,
      network_view_visible: true,
    });
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
        <button data-testid="jobs-scope-global" aria-pressed="false"></button>
        <button data-testid="jobs-scope-workspace" aria-pressed="false"></button>
      </div>
      <aside data-testid="automation-list-panel">
        <button data-testid="automation-item-job_daily_review"></button>
        <button data-testid="automation-item-job_weekly_triage"></button>
      </aside>
      <section data-testid="automation-detail-panel">
        <h2>deploy-review</h2>
        <button data-testid="toggle-automation-btn"></button>
        <button data-testid="trigger-job-btn"></button>
        <button data-testid="delete-automation-btn"></button>
        <div data-testid="automation-job-scheduler"></div>
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
      automation_delete_visible: true,
      automation_enabled_toggle_visible: true,
      automation_editor_kind: "job",
      automation_editor_open: false,
      automation_item_count: 2,
      automation_run_count: 2,
      automation_run_history_visible: true,
      automation_scheduler_visible: true,
      automation_scope_filter: "all",
      automation_selected_item: "deploy-review",
      automation_session_link_count: 1,
      automation_trigger_visible: true,
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

  it("captures task route context, selected run, and graph/review counts", async () => {
    window.history.replaceState({}, "", "/tasks/task_launch/runs/run_launch");
    document.title = "AGH";
    document.body.innerHTML = `
      <div data-testid="tasks-dashboard-view">
        <button data-testid="tasks-mode-dashboard" aria-pressed="false"></button>
        <button data-testid="tasks-mode-inbox" aria-pressed="false"></button>
        <button data-testid="tasks-mode-kanban" aria-pressed="false"></button>
        <button data-testid="tasks-mode-list" aria-pressed="true"></button>
        <article data-testid="task-card-task_launch"></article>
        <article data-testid="task-card-task_review"></article>
      </div>
      <section data-testid="tasks-detail-content">
        <button data-testid="tasks-detail-cancel"></button>
        <table data-testid="tasks-detail-children-panel">
          <tr data-testid="tasks-detail-children-item-task_child"></tr>
        </table>
        <table data-testid="tasks-detail-dependencies-panel">
          <tr data-testid="tasks-detail-dependencies-item-task_dependency"></tr>
        </table>
      </section>
      <section data-testid="tasks-run-detail-content">
        <button data-testid="task-run-detail-cancel"></button>
        <table data-testid="tasks-run-reviews-card">
          <tr data-testid="tasks-run-reviews-row-review_001"></tr>
        </table>
      </section>
      <article data-testid="tasks-inbox-item-task_launch" data-lane="failed_runs"></article>
      <button data-testid="tasks-inbox-item-retry-task_launch"></button>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/tasks/task_launch/runs/run_launch",
      tasks_active_mode: "list",
      tasks_children_count: 1,
      tasks_dependencies_count: 1,
      tasks_detail_cancel_visible: true,
      tasks_detail_visible: true,
      tasks_inbox_count: 1,
      tasks_review_count: 1,
      tasks_run_cancel_visible: true,
      tasks_run_detail_visible: true,
      tasks_selected_run: "run_launch",
      tasks_selected_task: "task_launch",
      tasks_task_count: 2,
      tasks_view_visible: true,
    });
  });

  it("captures knowledge route scope, dialogs, decisions, and selected item", async () => {
    window.history.replaceState({}, "", "/knowledge");
    document.title = "AGH";
    document.body.innerHTML = `
      <main data-testid="knowledge-shell">
        <button data-testid="tab-global" aria-pressed="false"></button>
        <button data-testid="tab-workspace" aria-pressed="true"></button>
        <button data-testid="tab-agent" aria-pressed="false"></button>
        <aside data-testid="knowledge-list-panel">
          <p data-testid="knowledge-search-info">Recall 1 of top-K</p>
          <button data-testid="memory-item-workspace:launch-memory.md" data-state="selected">
            Launch Memory
          </button>
          <button data-testid="memory-item-workspace:other-memory.md">Other Memory</button>
        </aside>
        <section data-testid="knowledge-detail-panel">
          <article data-testid="knowledge-decision-dec_write"></article>
          <button data-testid="revert-memory-decision-dec_write"></button>
        </section>
        <form data-testid="knowledge-create-dialog"></form>
        <form data-testid="knowledge-edit-dialog"></form>
        <form data-testid="knowledge-delete-dialog"></form>
      </main>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/knowledge",
      knowledge_create_dialog_open: true,
      knowledge_decisions_count: 1,
      knowledge_delete_dialog_open: true,
      knowledge_detail_visible: true,
      knowledge_edit_dialog_open: true,
      knowledge_item_count: 2,
      knowledge_revert_button_count: 1,
      knowledge_scope: "workspace",
      knowledge_search_active: true,
      knowledge_selected_item: "Launch Memory",
      knowledge_view_visible: true,
    });
  });

  it("captures Skills route catalog, detail, enabled state, and marketplace context", async () => {
    window.history.replaceState({}, "", "/skills");
    document.title = "AGH";
    document.body.innerHTML = `
      <main data-testid="skills-shell">
        <button data-testid="tab-installed" aria-selected="true"></button>
        <button data-testid="tab-marketplace" aria-selected="false"></button>
        <input data-testid="skill-search-input" value="browser-context" />
        <aside data-testid="skill-list-panel">
          <button data-testid="skill-item-browser-context-skill" data-state="selected">
            Browser Context Skill
          </button>
          <button data-testid="skill-item-browser-other-skill">Other Skill</button>
        </aside>
        <section data-testid="skill-detail-panel">
          <button data-testid="skill-enabled-toggle">Enabled</button>
          <article data-testid="content-body">Skill content</article>
        </section>
      </main>
    `;

    const installedState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(installedState).toMatchObject({
      pathname: "/skills",
      skills_active_tab: "installed",
      skills_content_visible: true,
      skills_detail_visible: true,
      skills_enabled_state: "enabled",
      skills_item_count: 2,
      skills_marketplace_count: 0,
      skills_search_active: true,
      skills_selected_item: "browser-context-skill",
      skills_view_visible: true,
    });

    document.body.innerHTML = `
      <main data-testid="skills-shell">
        <button data-testid="tab-installed" aria-selected="false"></button>
        <button data-testid="tab-marketplace" aria-selected="true"></button>
        <input data-testid="marketplace-search-input" value="browser-marketplace" />
        <section data-testid="marketplace-view">
          <div data-testid="marketplace-grid">
            <article data-testid="marketplace-row-browser-marketplace-skill">installed</article>
          </div>
        </section>
        <section data-testid="skill-detail-panel">
          <button data-testid="skill-enabled-toggle">Disabled</button>
        </section>
      </main>
    `;

    const marketplaceState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(marketplaceState).toMatchObject({
      pathname: "/skills",
      skills_active_tab: "marketplace",
      skills_content_visible: false,
      skills_detail_visible: true,
      skills_enabled_state: "disabled",
      skills_item_count: 0,
      skills_marketplace_count: 1,
      skills_search_active: true,
      skills_view_visible: true,
    });
    expect(marketplaceState.skills_selected_item).toBeUndefined();
  });

  it("captures sandbox route profile counts, dialogs, and restart state", async () => {
    window.history.replaceState({}, "", "/sandbox");
    document.title = "AGH";
    document.body.innerHTML = `
      <main data-testid="sandbox-shell">
        <p data-testid="sandbox-page-total">2 profiles</p>
        <p data-testid="sandbox-page-workspaces">1 workspace reference</p>
        <table data-testid="sandbox-page-list">
          <tbody>
            <tr data-testid="sandbox-page-card-browser-local-sandbox">
              <td data-testid="sandbox-page-card-browser-local-sandbox-profile">local / reuse</td>
              <td data-testid="sandbox-page-card-browser-local-sandbox-source">CONFIG</td>
              <td data-testid="sandbox-page-card-browser-local-sandbox-usage">1 workspace</td>
            </tr>
            <tr data-testid="sandbox-page-card-browser-blocked-sandbox"></tr>
          </tbody>
        </table>
        <form data-testid="settings-sandbox-editor"></form>
        <section data-testid="settings-sandboxes-delete"></section>
        <section data-testid="sandbox-page-action-result"></section>
        <section data-testid="settings-page-sandbox-restart-banner"></section>
      </main>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/sandbox",
      sandbox_action_result_visible: true,
      sandbox_delete_dialog_open: true,
      sandbox_editor_open: true,
      sandbox_profile_count: 2,
      sandbox_profile_names: ["browser-local-sandbox", "browser-blocked-sandbox"],
      sandbox_restart_banner_visible: true,
      sandbox_total_text: "2 profiles",
      sandbox_view_visible: true,
      sandbox_workspace_references_text: "1 workspace reference",
    });
  });

  it("captures Settings route section, vault, provider, and restart state", async () => {
    window.history.replaceState({}, "", "/settings/vault");
    document.title = "AGH";
    document.body.innerHTML = `
      <main data-testid="settings-shell">
        <nav data-testid="settings-section-nav">
          <a data-testid="settings-section-general"></a>
          <a data-testid="settings-section-vault"></a>
          <a data-testid="settings-section-network"></a>
        </nav>
        <section data-testid="settings-page-vault-action-result"></section>
        <form data-testid="settings-vault-editor"></form>
        <section data-testid="settings-vault-delete"></section>
        <section data-testid="settings-page-network-restart-banner"></section>
        <footer data-testid="settings-page-network-save-bar"></footer>
        <table data-testid="settings-page-vault-table">
          <tr data-testid="vault-secrets-row"></tr>
          <tr data-testid="vault-secrets-row"></tr>
        </table>
        <article data-testid="settings-page-providers-card-codex">
          <button data-testid="settings-page-providers-card-codex-edit"></button>
        </article>
        <article data-testid="settings-page-providers-card-claude">
          <button data-testid="settings-page-providers-card-claude-edit"></button>
        </article>
        <table>
          <tbody>
            <tr data-testid="settings-page-mcp-servers-row-filesystem">
              <td><button data-testid="settings-page-mcp-servers-row-filesystem-delete"></button></td>
            </tr>
          </tbody>
        </table>
      </main>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/settings/vault",
      settings_action_result_visible: true,
      settings_active_section: "vault",
      settings_mcp_server_count: 1,
      settings_provider_card_count: 2,
      settings_restart_banner_visible: true,
      settings_save_bar_visible: true,
      settings_section_count: 3,
      settings_vault_delete_dialog_open: true,
      settings_vault_editor_open: true,
      settings_vault_secret_count: 2,
      settings_view_visible: true,
    });
  });

  it("captures clean Settings section state without modal or restart affordances", async () => {
    window.history.replaceState({}, "", "/settings/general");
    document.title = "AGH";
    document.body.innerHTML = `
      <main data-testid="settings-shell">
        <nav data-testid="settings-section-nav">
          <a data-testid="settings-section-general"></a>
          <a data-testid="settings-section-vault"></a>
        </nav>
      </main>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/settings/general",
      settings_active_section: "general",
      settings_mcp_server_count: 0,
      settings_provider_card_count: 0,
      settings_restart_banner_visible: false,
      settings_save_bar_visible: false,
      settings_section_count: 2,
      settings_vault_delete_dialog_open: false,
      settings_vault_editor_open: false,
      settings_vault_secret_count: 0,
      settings_view_visible: true,
    });
    expect(routeState.settings_action_result_visible).toBe(false);
  });

  it("captures dashboard health and metric route context", async () => {
    window.history.replaceState({}, "", "/");
    document.title = "AGH";
    document.body.innerHTML = `
      <main data-testid="home-shell">
        <div data-testid="home-connection-indicator" data-status="connected"></div>
        <section data-testid="home-daemon-card" data-status="healthy"></section>
        <article data-testid="home-metric-active-sessions">
          <span data-slot="metric-value">1</span>
        </article>
        <article data-testid="home-metric-workspaces">
          <span data-slot="metric-value">2</span>
        </article>
        <article data-testid="home-metric-agents">
          <span data-slot="metric-value">3</span>
        </article>
        <article data-testid="home-metric-uptime">
          <span data-slot="metric-value">4m</span>
        </article>
      </main>
    `;

    const routeState = await captureRouteState({
      evaluate: async (callback: () => unknown) => callback(),
    });

    expect(routeState).toMatchObject({
      pathname: "/",
      home_view_visible: true,
      home_connection_status: "connected",
      home_daemon_status: "healthy",
      home_metric_count: 4,
      home_active_sessions_value: "1",
      home_workspaces_value: "2",
      home_agents_value: "3",
      home_uptime_value: "4m",
    });
  });
});
