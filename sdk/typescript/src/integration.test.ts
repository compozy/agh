import { execFileSync, spawn } from "node:child_process";
import { mkdtemp, readdir, rm, stat, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";
import readline from "node:readline";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

let tempDirs: string[] = [];
const sourceDir = __dirname;
const packageDir = resolve(sourceDir, "..");
const workspaceRoot = resolve(packageDir, "../..");
const tscBin = join(workspaceRoot, "node_modules/.bin/tsc");
const buildTimeoutMs = 120_000;
const integrationTimeoutMs = 30_000;

async function newestMtimeMs(path: string): Promise<number> {
  const entry = await stat(path);
  if (!entry.isDirectory()) {
    return entry.mtimeMs;
  }

  let newest = entry.mtimeMs;
  for (const child of await readdir(path, { withFileTypes: true })) {
    newest = Math.max(newest, await newestMtimeMs(join(path, child.name)));
  }

  return newest;
}

async function oldestMtimeMs(paths: string[]): Promise<number> {
  let oldest = Number.POSITIVE_INFINITY;

  for (const path of paths) {
    oldest = Math.min(oldest, (await stat(path)).mtimeMs);
  }

  return oldest;
}

async function sdkBuildIsCurrent(): Promise<boolean> {
  const outputs = [
    join(packageDir, "dist/esm/index.js"),
    join(packageDir, "dist/cjs/index.js"),
    join(packageDir, "dist/types/index.d.ts"),
    join(packageDir, "dist/esm/package.json"),
    join(packageDir, "dist/cjs/package.json"),
  ];

  try {
    const newestInput = Math.max(
      await newestMtimeMs(join(packageDir, "src")),
      await newestMtimeMs(join(packageDir, "package.json")),
      await newestMtimeMs(join(packageDir, "tsconfig.types.json")),
      await newestMtimeMs(join(packageDir, "tsconfig.esm.json")),
      await newestMtimeMs(join(packageDir, "tsconfig.cjs.json")),
      await newestMtimeMs(join(packageDir, "scripts/postbuild.mjs"))
    );
    const oldestOutput = await oldestMtimeMs(outputs);

    return oldestOutput >= newestInput;
  } catch {
    return false;
  }
}

async function buildSDK(): Promise<void> {
  execFileSync(tscBin, ["-p", join(packageDir, "tsconfig.types.json")], {
    cwd: packageDir,
    stdio: "pipe",
  });
  execFileSync(tscBin, ["-p", join(packageDir, "tsconfig.esm.json")], {
    cwd: packageDir,
    stdio: "pipe",
  });
  execFileSync(tscBin, ["-p", join(packageDir, "tsconfig.cjs.json")], {
    cwd: packageDir,
    stdio: "pipe",
  });
  execFileSync(process.execPath, [join(packageDir, "scripts/postbuild.mjs")], {
    cwd: packageDir,
    stdio: "pipe",
  });
}

async function ensureBuiltSDK(): Promise<void> {
  if (await sdkBuildIsCurrent()) {
    return;
  }

  await buildSDK();
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
  beforeAll(async () => {
    await ensureBuiltSDK();
  }, buildTimeoutMs);

  afterAll(async () => {
    await Promise.all(tempDirs.map(async dir => await rm(dir, { force: true, recursive: true })));
  });

  it(
    "builds an SDK-based extension and serves real JSON-RPC over stdio",
    async () => {
      const sdkEntry = resolve(packageDir, "dist/esm/index.js");
      const tempDir = await mkdtemp(join(tmpdir(), "agh-sdk-integration-"));
      tempDirs.push(tempDir);
      await writeFile(
        join(tempDir, "index.mjs"),
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

      const child = spawn(process.execPath, [join(tempDir, "index.mjs")], {
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
    },
    integrationTimeoutMs
  );

  it(
    "serves extension.tool descriptors and calls over real stdio",
    async () => {
      const sdkEntry = resolve(packageDir, "dist/esm/index.js");
      const tempDir = await mkdtemp(join(tmpdir(), "agh-sdk-tool-integration-"));
      tempDirs.push(tempDir);
      await writeFile(
        join(tempDir, "index.mjs"),
        `import { Extension } from ${JSON.stringify(sdkEntry)};
       const extension = new Extension({ name: "tool-ext", version: "0.1.0" });
       extension.tool("search", {
         readOnly: true,
         inputSchema: {
           type: "object",
           required: ["query"],
           properties: { query: { type: "string" } }
         }
       }, async ({ input }) => ({
         content: [{ type: "text", text: "result " + input.query }],
         truncated: false,
         bytes: 0,
         duration_ms: 0
       }));
       void extension.start();`
      );

      const child = spawn(process.execPath, [join(tempDir, "index.mjs")], {
        stdio: ["pipe", "pipe", "pipe"],
      });
      const stdout = readline.createInterface({ input: child.stdout });

      child.stdin.write(
        `${JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "initialize",
          params: {
            protocol_version: "1",
            supported_protocol_versions: ["1"],
            agh_version: "0.5.0",
            session_nonce: "tool-nonce",
            extension: { name: "tool-ext", version: "0.1.0", source_tier: "user" },
            capabilities: {
              provides: ["tool.provider"],
              granted_actions: [],
              granted_security: [],
              granted_resource_kinds: [],
              granted_resource_scopes: [],
            },
            methods: {
              daemon_requests: ["health_check", "shutdown"],
              extension_services: ["provide_tools", "tools/call"],
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

      await expect(nextMessage(stdout)).resolves.toMatchObject({
        id: 1,
        result: {
          accepted_capabilities: { provides: ["tool.provider"] },
          implemented_methods: expect.arrayContaining(["provide_tools", "tools/call"]),
        },
      });

      child.stdin.write(
        `${JSON.stringify({
          jsonrpc: "2.0",
          id: 2,
          method: "provide_tools",
          params: {},
        })}\n`
      );
      await expect(nextMessage(stdout)).resolves.toMatchObject({
        id: 2,
        result: {
          tools: [
            {
              id: "ext__tool_ext__search",
              handler: "search",
              read_only: true,
              risk: "read",
            },
          ],
        },
      });

      child.stdin.write(
        `${JSON.stringify({
          jsonrpc: "2.0",
          id: 3,
          method: "tools/call",
          params: {
            tool_id: "ext__tool_ext__search",
            handler: "search",
            session_id: "session-1",
            input: { query: "alpha" },
          },
        })}\n`
      );
      await expect(nextMessage(stdout)).resolves.toMatchObject({
        id: 3,
        result: {
          result: {
            content: [{ type: "text", text: "result alpha" }],
            truncated: false,
            bytes: 0,
            duration_ms: 0,
          },
        },
      });

      child.kill();
    },
    integrationTimeoutMs
  );
});
