import { mkdtemp } from "node:fs/promises";
import path from "node:path";

import type { BrowserContext, ConsoleMessage, Page, Request, Response } from "@playwright/test";

import {
  type ArtifactManifest,
  type ArtifactCollector,
  type BrowserConsoleEntry,
  type BrowserNetworkEntry,
  type BrowserRouteState,
  mirrorBrowserScreenshotForQA,
  persistBrowserArtifacts,
} from "./artifacts";

export interface BrowserArtifactSessionOptions {
  collector: ArtifactCollector;
  context: BrowserContext;
  qaOutputRootDir?: string;
}

export class BrowserArtifactSession {
  private readonly collector: ArtifactCollector;
  private readonly context: BrowserContext;
  private readonly tempDirPromise: Promise<string>;
  private readonly consoleEntries: BrowserConsoleEntry[] = [];
  private readonly networkEntries: BrowserNetworkEntry[] = [];
  private readonly qaOutputRootDir?: string;
  private readonly screenshotPaths: string[] = [];
  private readonly pages = new Set<Page>();

  private persistedManifest: ArtifactManifest | null = null;

  private constructor(options: BrowserArtifactSessionOptions) {
    this.collector = options.collector;
    this.context = options.context;
    this.qaOutputRootDir = options.qaOutputRootDir?.trim() || undefined;
    this.tempDirPromise = mkdtemp(path.join(this.collector.rootDir, ".capture-"));
  }

  static async start(options: BrowserArtifactSessionOptions): Promise<BrowserArtifactSession> {
    const session = new BrowserArtifactSession(options);
    await session.context.tracing.start({ screenshots: true, snapshots: true });

    for (const page of session.context.pages()) {
      session.attachPage(page);
    }
    session.context.on("page", page => {
      session.attachPage(page);
    });

    return session;
  }

  async captureScreenshot(name = "final", page?: Page): Promise<string | null> {
    const targetPage = page ?? this.selectPage();
    if (targetPage === null) {
      return null;
    }

    const tempDir = await this.tempDirPromise;
    const filePath = path.join(tempDir, `${sanitizePathComponent(name)}.png`);
    await targetPage.screenshot({ fullPage: true, path: filePath });
    this.screenshotPaths.push(filePath);
    if (this.qaOutputRootDir) {
      await mirrorBrowserScreenshotForQA(filePath, this.qaOutputRootDir, name);
    }
    return filePath;
  }

  async persist(page?: Page): Promise<ArtifactManifest> {
    if (this.persistedManifest !== null) {
      return this.persistedManifest;
    }

    const targetPage = page ?? this.selectPage();

    if (this.screenshotPaths.length === 0) {
      await this.captureScreenshot("final", targetPage ?? undefined);
    }

    const tempDir = await this.tempDirPromise;
    const tracePath = path.join(tempDir, "trace.zip");
    await this.context.tracing.stop({ path: tracePath });

    const routeState = targetPage ? await captureRouteState(targetPage) : undefined;

    this.persistedManifest = await persistBrowserArtifacts(this.collector, {
      tracePath,
      screenshotPaths: this.screenshotPaths,
      consoleEntries: this.consoleEntries,
      networkEntries: this.networkEntries,
      routeState,
    });
    return this.persistedManifest;
  }

  private attachPage(page: Page): void {
    if (this.pages.has(page)) {
      return;
    }

    this.pages.add(page);
    page.on("console", message => {
      this.consoleEntries.push(consoleEntryFromMessage(message));
    });
    page.on("pageerror", error => {
      this.consoleEntries.push({
        type: "pageerror",
        text: error.message,
      });
    });
    page.on("response", response => {
      this.networkEntries.push(networkEntryFromResponse(response));
    });
    page.on("requestfailed", request => {
      this.networkEntries.push(networkEntryFromFailedRequest(request));
    });
  }

  private selectPage(): Page | null {
    for (const page of [...this.pages].reverse()) {
      if (!page.isClosed()) {
        return page;
      }
    }
    return null;
  }
}

function consoleEntryFromMessage(message: ConsoleMessage): BrowserConsoleEntry {
  const location = message.location();
  return {
    type: message.type(),
    text: message.text(),
    location:
      location.url === "" && location.lineNumber === 0 && location.columnNumber === 0
        ? undefined
        : {
            url: location.url || undefined,
            line_number: location.lineNumber || undefined,
            column_number: location.columnNumber || undefined,
          },
  };
}

function networkEntryFromResponse(response: Response): BrowserNetworkEntry {
  const request = response.request();
  return {
    event: "response",
    url: response.url(),
    method: request.method(),
    resource_type: request.resourceType(),
    status: response.status(),
    ok: response.ok(),
  };
}

function networkEntryFromFailedRequest(request: Request): BrowserNetworkEntry {
  const failure = request.failure();
  return {
    event: "requestfailed",
    url: request.url(),
    method: request.method(),
    resource_type: request.resourceType(),
    failure: failure?.errorText ?? "unknown request failure",
  };
}

function sanitizePathComponent(value: string): string {
  const trimmed = value.trim().toLowerCase();
  if (trimmed === "") {
    return "artifact";
  }
  return trimmed.replace(/[^a-z0-9._-]+/g, "-");
}

export async function captureRouteState(page: Pick<Page, "evaluate">): Promise<BrowserRouteState> {
  return await page.evaluate(() => {
    const readText = (testId: string) =>
      document.querySelector<HTMLElement>(`[data-testid="${testId}"]`)?.textContent?.trim() ||
      undefined;
    const readHeading = (testId: string) =>
      document.querySelector<HTMLElement>(`[data-testid="${testId}"] h2`)?.textContent?.trim() ||
      undefined;
    const countByPrefix = (prefix: string) =>
      document.querySelectorAll(`[data-testid^="${prefix}"]`).length;
    const countAutomationRunCards = () =>
      [...document.querySelectorAll<HTMLElement>("[data-testid]")]
        .map(element => element.dataset.testid || "")
        .filter(
          testId =>
            testId.startsWith("automation-run-") &&
            testId !== "automation-run-history" &&
            testId !== "automation-run-history-loading" &&
            testId !== "automation-run-history-error" &&
            testId !== "automation-run-history-empty" &&
            !testId.startsWith("automation-run-session-link-")
        ).length;
    const networkActiveTab = document.querySelector(
      '[data-testid="network-tab-channels"][aria-pressed="true"]'
    )
      ? "channels"
      : document.querySelector('[data-testid="network-tab-peers"][aria-pressed="true"]')
        ? "peers"
        : undefined;
    const automationActiveTab = document.querySelector(
      '[data-testid="automation-kind-jobs"][aria-pressed="true"]'
    )
      ? "jobs"
      : document.querySelector('[data-testid="automation-kind-triggers"][aria-pressed="true"]')
        ? "triggers"
        : undefined;
    const automationSelectedItem =
      document
        .querySelector<HTMLElement>('[data-testid="automation-detail-panel"] h2')
        ?.textContent?.trim() || undefined;
    const automationEditorKind = document.querySelector('[data-testid="automation-job-form"]')
      ? "job"
      : document.querySelector('[data-testid="automation-trigger-form"]')
        ? "trigger"
        : undefined;
    const bridgeScopeFilter = document.querySelector(
      '[data-testid="bridge-scope-all"][aria-pressed="true"]'
    )
      ? "all"
      : document.querySelector('[data-testid="bridge-scope-global"][aria-pressed="true"]')
        ? "global"
        : document.querySelector('[data-testid="bridge-scope-workspace"][aria-pressed="true"]')
          ? "workspace"
          : undefined;
    const bridgeSelectedItem =
      document
        .querySelector<HTMLElement>('[data-testid="bridge-detail-panel"] h2')
        ?.textContent?.trim() || undefined;

    return {
      url: window.location.href,
      pathname: window.location.pathname,
      title: document.title,
      automation_active_tab: automationActiveTab,
      automation_editor_kind: automationEditorKind,
      automation_item_count: countByPrefix("automation-item-"),
      automation_run_count: countAutomationRunCards(),
      automation_run_history_visible:
        document.querySelector('[data-testid="automation-run-history"]') !== null,
      automation_selected_item: automationSelectedItem,
      automation_session_link_count: countByPrefix("automation-run-session-link-"),
      automation_view_visible:
        document.querySelector('[data-testid="automation-kind-tabs"]') !== null,
      bridge_create_dialog_open:
        document.querySelector('[data-testid="bridge-create-dialog"]') !== null,
      bridge_detail_visible: document.querySelector('[data-testid="bridge-detail-panel"]') !== null,
      bridge_edit_dialog_open:
        document.querySelector('[data-testid="bridge-edit-dialog"]') !== null,
      bridge_item_count: countByPrefix("bridge-item-"),
      bridge_route_count: countByPrefix("bridge-route-"),
      bridge_scope_filter: bridgeScopeFilter,
      bridge_secret_binding_count: countByPrefix("bridge-secret-binding-"),
      bridge_selected_item: bridgeSelectedItem,
      bridge_test_delivery_open:
        document.querySelector('[data-testid="bridge-test-delivery-dialog"]') !== null,
      bridge_test_delivery_result_visible:
        document.querySelector('[data-testid="bridge-test-delivery-result"]') !== null,
      bridge_view_visible:
        document.querySelector('[data-testid="bridge-list-panel"]') !== null ||
        document.querySelector('[data-testid="bridges-empty-state"]') !== null ||
        document.querySelector('[data-testid="bridge-detail-panel"]') !== null,
      chat_view_visible: document.querySelector('[data-testid="chat-view"]') !== null,
      message_count: document.querySelectorAll(
        '[data-testid="message-bubble-user"], [data-testid="message-bubble-assistant"]'
      ).length,
      network_active_tab: networkActiveTab,
      network_channel_count: countByPrefix("network-channel-item-"),
      network_message_count: countByPrefix("network-channel-message-"),
      network_peer_count: countByPrefix("network-peer-item-"),
      network_selected_channel: readHeading("network-channel-detail-panel"),
      network_selected_peer: readHeading("network-peer-detail-panel"),
      network_view_visible: document.querySelector('[data-testid="network-tab-pills"]') !== null,
      permission_prompt_visible:
        document.querySelector('[data-testid="permission-prompt"]') !== null,
      processing_indicator_visible:
        document.querySelector('[data-testid="processing-indicator"]') !== null,
      resume_button_visible: document.querySelector('[data-testid="resume-button"]') !== null,
      session_name: readText("session-name"),
      stop_button_visible: document.querySelector('[data-testid="stop-button"]') !== null,
    };
  });
}
