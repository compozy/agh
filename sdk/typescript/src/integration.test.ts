import { execFileSync, spawn } from "node:child_process";
import { mkdtemp, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { tmpdir } from "node:os";
import readline from "node:readline";

import { afterAll, describe, expect, it } from "vitest";

let tempDirs: string[] = [];

async function buildSDK(): Promise<void> {
  const packageDir = process.cwd();
  const workspaceRoot = path.resolve(packageDir, "../..");
  const tscBin = path.join(workspaceRoot, "node_modules/.bin/tsc");

  execFileSync(tscBin, ["-p", path.join(packageDir, "tsconfig.types.json")], {
    cwd: packageDir,
    stdio: "pipe",
  });
  execFileSync(tscBin, ["-p", path.join(packageDir, "tsconfig.esm.json")], {
    cwd: packageDir,
    stdio: "pipe",
  });
  execFileSync(tscBin, ["-p", path.join(packageDir, "tsconfig.cjs.json")], {
    cwd: packageDir,
    stdio: "pipe",
  });
  execFileSync(process.execPath, [path.join(packageDir, "scripts/postbuild.mjs")], {
    cwd: packageDir,
    stdio: "pipe",
  });
}

async function nextMessage(rl: readline.Interface): Promise<Record<string, unknown>> {
  return await new Promise((resolve, reject) => {
    rl.once("line", line => {
      try {
        resolve(JSON.parse(line) as Record<string, unknown>);
      } catch (error) {
        reject(error);
      }
    });
  });
}

describe("SDK integration", () => {
  afterAll(async () => {
    await Promise.all(tempDirs.map(async dir => await rm(dir, { force: true, recursive: true })));
  });

  it("builds an SDK-based extension and serves real JSON-RPC over stdio", async () => {
    await buildSDK();

    const packageDir = process.cwd();
    const sdkEntry = path.resolve(packageDir, "dist/esm/index.js");
    const tempDir = await mkdtemp(path.join(tmpdir(), "agh-sdk-integration-"));
    tempDirs.push(tempDir);
    await writeFile(
      path.join(tempDir, "index.mjs"),
      `import { Extension } from ${JSON.stringify(sdkEntry)};
       const extension = new Extension(
         {
           name: "integration-ext",
           version: "0.1.0",
           capabilities: { provides: ["memory.backend"] },
           actions: { requires: ["sessions/list"] },
           security: { capabilities: ["memory.read", "memory.write", "session.read"] }
         }
       );
       extension.handle("memory/store", async (_ctx, params) => ({ stored: params.key }));
       extension.handle("memory/recall", async () => ({ entries: [] }));
       extension.handle("memory/forget", async () => ({}));
       extension.handle("health_check", async () => ({ healthy: true, message: "", details: {} }));
       extension.onReady(async (host) => {
         const sessions = await host.sessions.list();
         console.error("ready sessions", sessions.length);
       });
       void extension.start();`
    );

    const child = spawn(process.execPath, [path.join(tempDir, "index.mjs")], {
      stdio: ["pipe", "pipe", "pipe"],
    });
    const stdout = readline.createInterface({ input: child.stdout });
    const stderr = readline.createInterface({ input: child.stderr });
    const stderrLines: string[] = [];
    stderr.on("line", line => {
      stderrLines.push(line);
    });

    child.stdin.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {
          protocol_version: "1",
          supported_protocol_versions: ["1"],
          agh_version: "0.5.0",
          session_nonce: "integration-nonce",
          extension: { name: "integration-ext", version: "0.1.0", source_tier: "user" },
          capabilities: {
            provides: ["memory.backend"],
            granted_actions: ["sessions/list"],
            granted_security: ["memory.read", "memory.write", "session.read"],
            granted_resource_kinds: [],
            granted_resource_scopes: [],
          },
          methods: {
            daemon_requests: ["health_check", "shutdown"],
            extension_services: ["memory/store", "memory/recall", "memory/forget"],
          },
          runtime: {
            health_check_interval_ms: 30000,
            health_check_timeout_ms: 5000,
            shutdown_timeout_ms: 10000,
            default_hook_timeout_ms: 5000,
          },
        },
      })}\n`
    );

    const initializeResponse = await nextMessage(stdout);
    expect(initializeResponse).toMatchObject({
      id: 1,
      result: expect.objectContaining({
        protocol_version: "1",
      }),
    });

    const sessionsListRequest = await nextMessage(stdout);
    expect(sessionsListRequest).toMatchObject({
      method: "sessions/list",
    });
    child.stdin.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: sessionsListRequest.id,
        result: [
          {
            id: "sess-1",
            agent: "claude",
            state: "active",
            created_at: "2026-04-10T12:00:00.000Z",
          },
        ],
      })}\n`
    );

    child.stdin.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: 2,
        method: "memory/store",
        params: {
          key: "alpha",
          content: "remember this",
        },
      })}\n`
    );
    const storeResponse = await nextMessage(stdout);
    expect(storeResponse).toMatchObject({
      id: 2,
      result: { stored: "alpha" },
    });

    child.stdin.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: 3,
        method: "health_check",
        params: {},
      })}\n`
    );
    const healthResponse = await nextMessage(stdout);
    expect(healthResponse).toMatchObject({
      id: 3,
      result: { healthy: true },
    });

    await new Promise(resolve => setTimeout(resolve, 25));
    expect(stderrLines.join("\n")).toMatch(/ready sessions.*1/);

    child.kill();
  }, 20_000);
});
