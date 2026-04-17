import { copyFile, mkdir, mkdtemp, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

const MANIFEST_VERSION = 1;
const TEMP_DIR_PREFIX = "agh-playwright-artifacts-";
const VITE_DEV_MARKERS = ["/@vite/client", "/src/main.tsx", "vite.svg"];

export const browserArtifactSpecs = {
  browser_trace: { relativePath: "browser_trace.zip", isDir: false },
  browser_screenshots: { relativePath: "browser_screenshots", isDir: true },
  browser_console: { relativePath: "browser_console.json", isDir: false },
  browser_network: { relativePath: "browser_network.json", isDir: false },
  browser_route_state: { relativePath: "browser_route_state.json", isDir: false },
} as const;

export type BrowserArtifactKind = keyof typeof browserArtifactSpecs;

export interface ArtifactEntry {
  kind: BrowserArtifactKind;
  path: string;
  media_type?: string;
}

export interface ArtifactManifest {
  version: number;
  artifacts: ArtifactEntry[];
}

export interface BrowserConsoleEntry {
  type: string;
  text: string;
  location?: {
    url?: string;
    line_number?: number;
    column_number?: number;
  };
}

export interface BrowserNetworkEntry {
  event: "response" | "requestfailed";
  url: string;
  method: string;
  resource_type: string;
  status?: number;
  ok?: boolean;
  failure?: string;
}

export interface BrowserArtifactBundle {
  tracePath: string;
  screenshotPaths: string[];
  consoleEntries: BrowserConsoleEntry[];
  networkEntries: BrowserNetworkEntry[];
  routeState?: BrowserRouteState;
}

export interface BrowserRouteState {
  url: string;
  pathname: string;
  title: string;
  automation_active_tab?: "jobs" | "triggers";
  automation_editor_kind?: "job" | "trigger";
  automation_item_count?: number;
  automation_run_count?: number;
  automation_run_history_visible?: boolean;
  automation_selected_item?: string;
  automation_session_link_count?: number;
  automation_view_visible?: boolean;
  bridge_create_dialog_open?: boolean;
  bridge_detail_visible?: boolean;
  bridge_edit_dialog_open?: boolean;
  bridge_item_count?: number;
  bridge_route_count?: number;
  bridge_scope_filter?: "all" | "global" | "workspace";
  bridge_secret_binding_count?: number;
  bridge_selected_item?: string;
  bridge_test_delivery_open?: boolean;
  bridge_test_delivery_result_visible?: boolean;
  bridge_view_visible?: boolean;
  chat_view_visible: boolean;
  session_name?: string;
  permission_prompt_visible: boolean;
  processing_indicator_visible: boolean;
  stop_button_visible: boolean;
  resume_button_visible: boolean;
  message_count: number;
  network_view_visible: boolean;
  network_active_tab?: "channels" | "peers";
  network_channel_count: number;
  network_peer_count: number;
  network_message_count: number;
  network_selected_channel?: string;
  network_selected_peer?: string;
}

export function isLikelyViteDevHTML(html: string): boolean {
  return VITE_DEV_MARKERS.some(marker => html.includes(marker));
}

export function resolveBrowserArtifactPath(rootDir: string, relativePath: string): string {
  const targetPath = path.resolve(rootDir, relativePath);
  const relativeTarget = path.relative(rootDir, targetPath);
  if (relativeTarget.startsWith("..") || path.isAbsolute(relativeTarget)) {
    throw new Error(`artifact path ${relativePath} escapes root ${rootDir}`);
  }
  return targetPath;
}

export class ArtifactCollector {
  readonly rootDir: string;
  readonly manifestPath: string;

  private readonly entries = new Map<BrowserArtifactKind, ArtifactEntry>();

  private constructor(rootDir: string) {
    this.rootDir = rootDir;
    this.manifestPath = path.join(rootDir, "manifest.json");
  }

  static async create(rootDir?: string): Promise<ArtifactCollector> {
    const resolvedRoot =
      rootDir === undefined || rootDir.trim() === ""
        ? await mkdtemp(path.join(os.tmpdir(), TEMP_DIR_PREFIX))
        : rootDir;
    await mkdir(resolvedRoot, { recursive: true });
    return new ArtifactCollector(resolvedRoot);
  }

  artifactPath(kind: BrowserArtifactKind): string {
    const spec = browserArtifactSpecs[kind];
    return resolveBrowserArtifactPath(this.rootDir, spec.relativePath);
  }

  manifest(): ArtifactManifest {
    return {
      version: MANIFEST_VERSION,
      artifacts: [...this.entries.values()].sort((left, right) =>
        left.path.localeCompare(right.path)
      ),
    };
  }

  async writeManifest(): Promise<ArtifactManifest> {
    const manifest = this.manifest();
    await writeFile(this.manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, "utf8");
    return manifest;
  }

  async captureJSON(kind: BrowserArtifactKind, value: unknown): Promise<void> {
    const payload = `${JSON.stringify(value, null, 2)}\n`;
    await this.captureBytes(kind, Buffer.from(payload, "utf8"), "application/json");
  }

  async captureText(kind: BrowserArtifactKind, value: string): Promise<void> {
    await this.captureBytes(kind, Buffer.from(value, "utf8"), "text/plain");
  }

  async captureFile(
    kind: BrowserArtifactKind,
    sourcePath: string,
    mediaType: string
  ): Promise<void> {
    if (browserArtifactSpecs[kind].isDir) {
      throw new Error(`artifact ${kind} requires multiple files`);
    }

    const targetPath = this.artifactPath(kind);
    await mkdir(path.dirname(targetPath), { recursive: true });
    await copyFile(sourcePath, targetPath);
    this.entries.set(kind, {
      kind,
      path: browserArtifactSpecs[kind].relativePath,
      media_type: mediaType.trim(),
    });
  }

  async captureFiles(
    kind: BrowserArtifactKind,
    sourcePaths: string[],
    mediaType: string
  ): Promise<void> {
    if (!browserArtifactSpecs[kind].isDir) {
      throw new Error(`artifact ${kind} does not accept multiple files`);
    }

    const targetDir = this.artifactPath(kind);
    await mkdir(targetDir, { recursive: true });
    for (const sourcePath of sourcePaths) {
      await copyFile(sourcePath, path.join(targetDir, path.basename(sourcePath)));
    }

    this.entries.set(kind, {
      kind,
      path: browserArtifactSpecs[kind].relativePath,
      media_type: mediaType.trim(),
    });
  }

  private async captureBytes(
    kind: BrowserArtifactKind,
    payload: Uint8Array,
    mediaType: string
  ): Promise<void> {
    if (browserArtifactSpecs[kind].isDir) {
      throw new Error(`artifact ${kind} requires file collection`);
    }

    const targetPath = this.artifactPath(kind);
    await mkdir(path.dirname(targetPath), { recursive: true });
    await writeFile(targetPath, payload);
    this.entries.set(kind, {
      kind,
      path: browserArtifactSpecs[kind].relativePath,
      media_type: mediaType.trim(),
    });
  }
}

export async function persistBrowserArtifacts(
  collector: ArtifactCollector,
  bundle: BrowserArtifactBundle
): Promise<ArtifactManifest> {
  await collector.captureFile("browser_trace", bundle.tracePath, "application/zip");
  if (bundle.screenshotPaths.length > 0) {
    await collector.captureFiles("browser_screenshots", bundle.screenshotPaths, "image/png");
  }
  await collector.captureJSON("browser_console", bundle.consoleEntries);
  await collector.captureJSON("browser_network", bundle.networkEntries);
  if (bundle.routeState) {
    await collector.captureJSON("browser_route_state", bundle.routeState);
  }
  return collector.writeManifest();
}
