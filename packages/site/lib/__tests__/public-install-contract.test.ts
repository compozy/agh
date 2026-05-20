import { mkdtempSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { relative, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { describe, expect, it } from "vitest";
import { siteRoot } from "../content-test-utils";

const publicRoot = resolve(siteRoot, "public");
const installScriptPath = resolve(publicRoot, "install.sh");
const headersPath = resolve(publicRoot, "_headers");
const installPagePath = resolve(siteRoot, "content/runtime/core/getting-started/installation.mdx");
const launchPostPath = resolve(
  siteRoot,
  "content/blog/posts/introducing-agh-the-first-agent-network-protocol.mdx"
);
const landingInstallPath = resolve(siteRoot, "components/landing/install-section.tsx");

const primaryInstallCommand = "curl -fsSL https://agh.network/install.sh | sh";
const packageInstallCommand = "brew install compozy/compozy/agh";
const sourceInstallCommand = "go build -o ./bin/agh ./cmd/agh";
const retiredPackageInstallCommands = [
  "brew install --cask pedronauck/agh/agh",
  "pedronauck/agh/agh",
  "homebrew-agh",
];
const installOptions = ["--version", "--dir", "--skip-bootstrap", "--dry-run", "--help"];
const installEnvVars = ["AGH_VERSION", "AGH_INSTALL_DIR", "AGH_SKIP_BOOTSTRAP"];
const installerReleaseGuaranteeSnippets = [
  "curl -fsSL https://agh.network/install.sh | sh",
  "Requires:",
  "curl, tar, cosign, and sha256sum or shasum.",
  'command -v cosign >/dev/null 2>&1 || fail "cosign is required to verify release provenance"',
  'BUNDLE_URL="${BASE_URL}/checksums.txt.sigstore.json"',
  'log "verifying checksum provenance"',
  'cosign verify-blob "$CHECKSUM_PATH"',
  '--bundle "$BUNDLE_PATH"',
  '--certificate-identity-regexp "$COSIGN_CERT_IDENTITY_REGEXP"',
  '--certificate-oidc-issuer "$COSIGN_CERT_OIDC_ISSUER"',
  'CHECKSUM_CMD="sha256sum"',
  'CHECKSUM_CMD="shasum"',
  "shasum -a 256 -c - >/dev/null",
];
const installerCriticalErrorSnippets = [
  "failed to resolve latest release",
  "latest release resolved to unexpected ref:",
  "unsupported operating system:",
  "unsupported architecture:",
  "curl is required",
  "tar is required",
  "cosign is required to verify release provenance",
  "sha256sum or shasum is required to verify the download",
  "checksums.txt does not include",
  "archive did not contain an agh binary",
  "warning: ${INSTALL_DIR} is not on PATH",
  "add it to PATH or run ${TARGET} directly",
  "no interactive terminal detected; run this next:",
  "agh install",
];
const ttyPermissionProbePattern =
  /\[\s*-[rwe]\s+["']?\/dev\/tty["']?\s*\]|\btest\s+-[rwe]\s+["']?\/dev\/tty["']?/;
const ttyOpenProbePattern =
  /^\s*(?:if|elif)\s+!?\s*[^\n]*<\s*\/dev\/tty[^\n]*>\s*\/dev\/tty[^\n]*;\s*then/m;
const credentialEnvPattern =
  /(?:API_?KEY|ACCESS_KEY|PRIVATE_KEY|TOKEN|SECRET|PASSWORD|PASSWD|CREDENTIAL|AUTH|COOKIE|SESSION)|_KEY$/i;
type InstallEnv = Record<string, string | undefined>;

function readSiteFile(path: string): string {
  return readFileSync(path, "utf8");
}

function runInstallScript(args: string[]) {
  const installDir = mkdtempSync(resolve(tmpdir(), "agh-install-contract-"));
  return spawnSync("sh", [installScriptPath, ...args, "--dir", installDir], {
    cwd: siteRoot,
    encoding: "utf8",
    env: hermeticInstallEnv(),
  });
}

function hermeticInstallEnv(source: InstallEnv = process.env): NodeJS.ProcessEnv {
  const env: InstallEnv = {};
  for (const [key, value] of Object.entries(source)) {
    if (value === undefined || blocksHermeticInstallEnv(key)) {
      continue;
    }
    env[key] = value;
  }
  env.TZ = "UTC";
  env.LANG = "C.UTF-8";
  env.LC_ALL = "C.UTF-8";
  env.LC_CTYPE = "C.UTF-8";
  env.NODE_ENV ??= "test";
  env.AGH_SKIP_BOOTSTRAP = "";
  return env as NodeJS.ProcessEnv;
}

function blocksHermeticInstallEnv(key: string): boolean {
  const normalized = key.trim().toUpperCase();
  return (
    normalized.startsWith("AGH_") ||
    credentialEnvPattern.test(normalized) ||
    [
      "CLAUDE_CONFIG_DIR",
      "CODEX_HOME",
      "OPENCODE_CONFIG_DIR",
      "PI_CODING_AGENT_DIR",
      "PROVIDER_CODEX_HOME",
      "PROVIDER_HOME",
    ].includes(normalized)
  );
}

describe("public install contract", () => {
  it("keeps the install script safe for the public curl entrypoint", () => {
    const script = readSiteFile(installScriptPath);
    const downloadIndex = script.indexOf('curl -fsSL "$ARCHIVE_URL" -o "$ARCHIVE_PATH"');
    const checksumDownloadIndex = script.indexOf('curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH"');
    const bundleDownloadIndex = script.indexOf('curl -fsSL "$BUNDLE_URL" -o "$BUNDLE_PATH"');
    const provenanceVerifyIndex = script.indexOf("verifying checksum provenance");
    const checksumVerifyIndex = script.indexOf('log "verifying checksum"');
    const extractIndex = script.indexOf('tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"');

    expect(script.startsWith("#!/bin/sh\nset -eu\n")).toBe(true);
    expect(script).toContain('RELEASE_REPO="compozy/agh"');
    expect(script).not.toContain("AGH_RELEASE_REPO");
    expect(script).toContain('BASE_URL="https://github.com/${RELEASE_REPO}/releases');
    expect(script).not.toContain("http://");
    expect(script).toContain('command -v curl >/dev/null 2>&1 || fail "curl is required"');
    expect(script).toContain('command -v tar >/dev/null 2>&1 || fail "tar is required"');
    expect(script).toContain(
      'command -v cosign >/dev/null 2>&1 || fail "cosign is required to verify release provenance"'
    );
    expect(script).toContain('BUNDLE_URL="${BASE_URL}/checksums.txt.sigstore.json"');
    expect(script).toContain(
      "COSIGN_CERT_IDENTITY_REGEXP='^https://github\\.com/compozy/agh/\\.github/workflows/release\\.yml@refs/tags/v[0-9][A-Za-z0-9._-]*$'"
    );
    expect(script).toContain("v[0-9][A-Za-z0-9._-]*)");
    expect(script).toContain(
      'COSIGN_CERT_OIDC_ISSUER="https://token.actions.githubusercontent.com"'
    );
    expect(script).toContain("resolve_latest_release_tag()");
    expect(script).toContain('VERSION="$(resolve_latest_release_tag)"');
    expect(script).toContain('cosign verify-blob "$CHECKSUM_PATH"');
    expect(script).toContain('--bundle "$BUNDLE_PATH"');
    expect(script).toContain('--certificate-identity-regexp "$COSIGN_CERT_IDENTITY_REGEXP"');
    expect(script).toContain('--certificate-oidc-issuer "$COSIGN_CERT_OIDC_ISSUER"');
    expect(script).toContain('CHECKSUM_CMD="sha256sum"');
    expect(script).toContain('CHECKSUM_CMD="shasum"');
    expect(script).toContain("shasum -a 256 -c - >/dev/null");
    expect(script).toContain("trap cleanup EXIT INT TERM");
    expect(script).toContain('TMP_TARGET="${INSTALL_DIR}/.agh.tmp.$$"');
    expect(script).toContain('chmod 0755 "$TMP_TARGET"');
    expect(script).toContain('mv "$TMP_TARGET" "$TARGET"');
    expect(script).toContain('"$TARGET" version >/dev/null');
    expect(script).toContain('"$TARGET" install </dev/tty >/dev/tty');
    expect(downloadIndex).toBeGreaterThan(-1);
    expect(checksumDownloadIndex).toBeGreaterThan(downloadIndex);
    expect(bundleDownloadIndex).toBeGreaterThan(checksumDownloadIndex);
    expect(provenanceVerifyIndex).toBeGreaterThan(bundleDownloadIndex);
    expect(checksumVerifyIndex).toBeGreaterThan(provenanceVerifyIndex);
    expect(extractIndex).toBeGreaterThan(checksumVerifyIndex);
  });

  it("keeps documented installer options executable in dry-run mode", () => {
    const help = runInstallScript(["--help"]);
    const dryRun = runInstallScript(["--dry-run", "--skip-bootstrap", "--version", "v0.1.0"]);
    const badOption = runInstallScript(["--not-a-real-option"]);

    expect(help.status).toBe(0);
    expect(help.stdout).toContain(primaryInstallCommand);
    for (const option of installOptions) {
      expect(help.stdout, option).toContain(option);
    }
    for (const envVar of installEnvVars) {
      expect(help.stdout, envVar).toContain(envVar);
    }

    expect(dryRun.status).toBe(0);
    expect(dryRun.stdout).toContain("AGH installer");
    expect(dryRun.stdout).toContain("release: compozy/agh v0.1.0");
    expect(dryRun.stdout).toContain(
      "archive: https://github.com/compozy/agh/releases/download/v0.1.0/"
    );
    expect(dryRun.stdout).toContain("bootstrap: skipped");
    expect(dryRun.stdout).toContain("dry run complete");
    expect(dryRun.stderr).toBe("");

    expect(badOption.status).not.toBe(0);
    expect(badOption.stderr).toContain("unknown option: --not-a-real-option");
  });

  it("keeps public installer release guarantees and recovery text in source", () => {
    const script = readSiteFile(installScriptPath);

    for (const snippet of installerReleaseGuaranteeSnippets) {
      expect(script, snippet).toContain(snippet);
    }
    for (const snippet of installerCriticalErrorSnippets) {
      expect(script, snippet).toContain(snippet);
    }
    expect(script).toContain("printf 'agh installer: %s\\n' \"$*\" >&2");
    expect(script).toContain('curl -fsSL "$ARCHIVE_URL" -o "$ARCHIVE_PATH"');
    expect(script).toContain('curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH"');
    expect(script).toContain('curl -fsSL "$BUNDLE_URL" -o "$BUNDLE_PATH"');
    expect(script).toContain(
      'printf \'%s\\n\' "$CHECKSUM_LINE" | (cd "$TMP_DIR" && sha256sum -c - >/dev/null)'
    );
    expect(script).toContain(
      'printf \'%s\\n\' "$CHECKSUM_LINE" | (cd "$TMP_DIR" && shasum -a 256 -c - >/dev/null)'
    );
    expect(script).toContain('log "next: agh install"');
  });

  it("runs install contract checks with a hermetic release environment", () => {
    const env = hermeticInstallEnv({
      AGH_VERSION: "v9.9.9",
      AGH_INSTALL_DIR: "/operator/bin",
      HOME: "/Users/operator",
      OPENAI_API_KEY: "sk-operator",
      PATH: "/usr/bin",
      PROVIDER_HOME: "/Users/operator/.provider",
      TZ: "America/Sao_Paulo",
    });

    expect(env.HOME).toBe("/Users/operator");
    expect(env.PATH).toBe("/usr/bin");
    expect(env.AGH_VERSION).toBeUndefined();
    expect(env.AGH_INSTALL_DIR).toBeUndefined();
    expect(env.OPENAI_API_KEY).toBeUndefined();
    expect(env.PROVIDER_HOME).toBeUndefined();
    expect(env.AGH_SKIP_BOOTSTRAP).toBe("");
    expect(env.TZ).toBe("UTC");
    expect(env.LANG).toBe("C.UTF-8");
    expect(env.LC_ALL).toBe("C.UTF-8");
  });

  it("opens the tty before starting interactive bootstrap", () => {
    const script = readSiteFile(installScriptPath);

    expect(script).not.toMatch(ttyPermissionProbePattern);
    expect(script).toMatch(ttyOpenProbePattern);
  });

  it("serves install.sh with script-safe headers", () => {
    const headers = readSiteFile(headersPath);
    const installHeaderBlock = headers.match(/\/install\.sh\n([\s\S]*?)(?:\n\n|$)/)?.[1] ?? "";

    expect(installHeaderBlock).toContain("Content-Type: text/plain; charset=utf-8");
    expect(installHeaderBlock).toContain("Cache-Control: public, max-age=300, must-revalidate");
    expect(headers).toContain("X-Content-Type-Options: nosniff");
    expect(headers).toContain("Content-Security-Policy:");
  });

  it("keeps landing, docs, and launch post aligned on install commands", () => {
    const checkedFiles = [landingInstallPath, installPagePath, launchPostPath];
    const missingPrimaryCommand = checkedFiles
      .filter(path => !readSiteFile(path).includes(primaryInstallCommand))
      .map(path => relative(siteRoot, path));
    const installPage = readSiteFile(installPagePath);
    const landingInstall = readSiteFile(landingInstallPath);

    expect(missingPrimaryCommand).toEqual([]);
    expect(landingInstall).toContain(packageInstallCommand);
    expect(landingInstall).toContain(sourceInstallCommand);
    expect(installPage).toContain(packageInstallCommand);
    expect(installPage).toContain(sourceInstallCommand);
    for (const retiredCommand of retiredPackageInstallCommands) {
      expect(landingInstall).not.toContain(retiredCommand);
      expect(installPage).not.toContain(retiredCommand);
    }
    for (const option of installOptions.slice(0, -1)) {
      expect(installPage, option).toContain(option);
    }
    for (const envVar of ["AGH_VERSION", "AGH_INSTALL_DIR", "AGH_SKIP_BOOTSTRAP"]) {
      expect(installPage, envVar).toContain(envVar);
    }
    expect(installPage).toContain("cosign");
    expect(installPage).toContain("checksums.txt.sigstore.json");
  });
});
