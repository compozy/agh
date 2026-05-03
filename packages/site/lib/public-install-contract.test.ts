import { mkdtempSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { relative, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { describe, expect, it } from "vitest";
import { siteRoot } from "./content-test-utils";

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
const packageInstallCommand = "brew install --cask pedronauck/agh/agh";
const sourceInstallCommand = "go build -o ./bin/agh ./cmd/agh";
const installOptions = ["--version", "--dir", "--skip-bootstrap", "--dry-run", "--help"];
const installEnvVars = ["AGH_VERSION", "AGH_INSTALL_DIR", "AGH_SKIP_BOOTSTRAP"];

function readSiteFile(path: string): string {
  return readFileSync(path, "utf8");
}

function runInstallScript(args: string[]) {
  const installDir = mkdtempSync(resolve(tmpdir(), "agh-install-contract-"));
  return spawnSync("sh", [installScriptPath, ...args, "--dir", installDir], {
    cwd: siteRoot,
    encoding: "utf8",
    env: {
      ...process.env,
      AGH_SKIP_BOOTSTRAP: "",
    },
  });
}

describe("public install contract", () => {
  it("keeps the install script safe for the public curl entrypoint", () => {
    const script = readSiteFile(installScriptPath);
    const downloadIndex = script.indexOf('curl -fsSL "$ARCHIVE_URL" -o "$ARCHIVE_PATH"');
    const checksumDownloadIndex = script.indexOf('curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH"');
    const signatureDownloadIndex = script.indexOf(
      'curl -fsSL "$SIGNATURE_URL" -o "$SIGNATURE_PATH"'
    );
    const certificateDownloadIndex = script.indexOf(
      'curl -fsSL "$CERTIFICATE_URL" -o "$CERTIFICATE_PATH"'
    );
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
    expect(script).toContain('SIGNATURE_URL="${BASE_URL}/checksums.txt.sig"');
    expect(script).toContain('CERTIFICATE_URL="${BASE_URL}/checksums.txt.pem"');
    expect(script).toContain(
      "COSIGN_CERT_IDENTITY_REGEXP='^https://github\\.com/compozy/agh/\\.github/workflows/release\\.yml@refs/tags/v[0-9][A-Za-z0-9._-]*$'"
    );
    expect(script).toContain(
      'COSIGN_CERT_OIDC_ISSUER="https://token.actions.githubusercontent.com"'
    );
    expect(script).toContain("resolve_latest_release_tag()");
    expect(script).toContain('VERSION="$(resolve_latest_release_tag)"');
    expect(script).toContain("COSIGN_EXPERIMENTAL=1 cosign verify-blob");
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
    expect(signatureDownloadIndex).toBeGreaterThan(checksumDownloadIndex);
    expect(certificateDownloadIndex).toBeGreaterThan(signatureDownloadIndex);
    expect(provenanceVerifyIndex).toBeGreaterThan(certificateDownloadIndex);
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
    for (const option of installOptions.slice(0, -1)) {
      expect(installPage, option).toContain(option);
    }
    for (const envVar of ["AGH_VERSION", "AGH_INSTALL_DIR", "AGH_SKIP_BOOTSTRAP"]) {
      expect(installPage, envVar).toContain(envVar);
    }
    expect(installPage).toContain("cosign");
  });
});
