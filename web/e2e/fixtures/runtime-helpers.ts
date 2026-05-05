import process from "node:process";

import { isLikelyViteDevHTML } from "./artifacts";

export interface RuntimeModeAttach {
  kind: "attach";
  baseURL: string;
}

export interface RuntimeModeLaunch {
  kind: "launch";
}

export type RuntimeMode = RuntimeModeAttach | RuntimeModeLaunch;

export interface RuntimeConfigInput {
  host: string;
  networkEnabled?: boolean;
  port: number;
  socketPath: string;
}

export interface WorkspacePayload {
  id: string;
  root_dir: string;
  name: string;
}

export function resolveRuntimeMode(env: NodeJS.ProcessEnv = process.env): RuntimeMode {
  const rawBaseURL = env.AGH_E2E_BASE_URL?.trim();
  if (rawBaseURL === undefined || rawBaseURL === "") {
    return { kind: "launch" };
  }

  return {
    kind: "attach",
    baseURL: normalizeBaseURL(rawBaseURL),
  };
}

export function normalizeBaseURL(rawValue: string): string {
  const baseURL = new URL(rawValue);
  baseURL.hash = "";
  baseURL.search = "";

  if (baseURL.pathname !== "/" && baseURL.pathname !== "") {
    throw new Error(
      `AGH_E2E_BASE_URL must point at the daemon root, received path ${baseURL.pathname}`
    );
  }

  return baseURL.toString().replace(/\/$/, "");
}

export function renderRuntimeConfig(input: RuntimeConfigInput): string {
  return [
    "[daemon]",
    `socket = ${tomlString(input.socketPath)}`,
    "",
    "[http]",
    `host = ${tomlString(input.host)}`,
    `port = ${input.port}`,
    "",
    ...(input.networkEnabled ? ["[network]", "enabled = true", ""] : []),
  ].join("\n");
}

export function requiresHTTPAPIReadinessProbe(host: string | undefined): boolean {
  const normalized = host?.trim().replace(/^\[/, "").replace(/\]$/, "").toLowerCase() ?? "";
  if (normalized === "" || normalized === "localhost" || normalized === "::1") {
    return true;
  }
  return /^127(?:\.|$)/.test(normalized);
}

export function runtimeURL(baseURL: string, pathname = "/"): string {
  const url = new URL(ensureLeadingSlash(pathname), `${baseURL.replace(/\/$/, "")}/`);
  return url.toString();
}

export function buildResolveWorkspaceRequest(path: string): { path: string } {
  return { path };
}

export function assertDaemonServedHTML(html: string, baseURL: string): void {
  if (isLikelyViteDevHTML(html)) {
    throw new Error(`expected daemon-served embedded assets at ${baseURL}, received Vite dev HTML`);
  }
}

export function ensureLeadingSlash(value: string): string {
  return value.startsWith("/") ? value : `/${value}`;
}

export function prependPath(prefix: string, currentPath: string | undefined): string {
  if (currentPath === undefined || currentPath.trim() === "") {
    return prefix;
  }
  return `${prefix}${process.platform === "win32" ? ";" : ":"}${currentPath}`;
}

function tomlString(value: string): string {
  return JSON.stringify(value);
}
