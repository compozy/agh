import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const runtimeRoot = resolve(siteRoot, "content/runtime");

function readRuntimeDoc(...parts: string[]): string {
  return readFileSync(resolve(runtimeRoot, ...parts), "utf8");
}

function readJSON<T>(...parts: string[]): T {
  return JSON.parse(readRuntimeDoc(...parts)) as T;
}

function expectIncludesAll(content: string, values: string[]): void {
  for (const value of values) {
    expect(content).toContain(value);
  }
}

describe("runtime autonomy docs", () => {
  it("documents the MVP execution boundary and manual operator control", () => {
    const overview = readRuntimeDoc("core/autonomy/index.mdx");
    const coordinator = readRuntimeDoc("core/autonomy/coordinator.mdx");
    const config = readRuntimeDoc("core/configuration/config-toml.mdx");

    expectIncludesAll(overview, [
      "Creating a task records intent only",
      "does not enqueue claimable work",
      "publish",
      "start",
      "approves",
      "execution boundary",
    ]);
    expectIncludesAll(coordinator, [
      "Task creation alone does not start a coordinator",
      "Manual control stays explicit",
      "Global-scope runs do not auto-spawn a coordinator in the MVP",
    ]);
    expectIncludesAll(config, [
      "[autonomy.coordinator]",
      "Task creation is not executable work",
      "Publish, start, approval",
      "Workspace `.agh/config.toml` override",
      "Global `$AGH_HOME/config.toml`",
      "bundled/default coordinator agent definition",
    ]);
  });

  it("documents task leases and channel authority without exposing raw tokens in read paths", () => {
    const leases = readRuntimeDoc("core/autonomy/task-runs-and-leases.mdx");
    const channels = readRuntimeDoc("core/autonomy/coordination-channels.mdx");

    expectIncludesAll(leases, [
      "raw `claim_token`",
      "shown only in that synchronous claim response",
      "`claim_token_hash`",
      "One active lease per session",
      "Stale holders fail",
      "Never send raw claim tokens through `agh ch send`",
    ]);
    expectIncludesAll(channels, [
      "boundary",
      "Task creation alone does not create claimable work",
      "Channels are conversation, not ownership",
      "Channel messages never own task status",
      "`coordination_channel_id`",
      "`correlation_id`",
      "Raw `claim_token` fields are rejected",
    ]);
  });

  it("exposes autonomy docs in runtime navigation without a marketing redesign", () => {
    const coreMeta = readJSON<{ pages: string[] }>("core/meta.json");
    const autonomyMeta = readJSON<Record<string, unknown>>("core/autonomy/meta.json");

    expect(coreMeta.pages).toContain("autonomy");
    expect(autonomyMeta).toMatchObject({
      title: "Autonomy",
      pages: [
        "index",
        "coordinator",
        "task-runs-and-leases",
        "coordination-channels",
        "safe-spawn",
      ],
    });
  });
});

describe("generated autonomy CLI references", () => {
  const requiredPages = [
    "cli-reference/me/index.mdx",
    "cli-reference/me/context.mdx",
    "cli-reference/ch/index.mdx",
    "cli-reference/ch/list.mdx",
    "cli-reference/ch/recv.mdx",
    "cli-reference/ch/send.mdx",
    "cli-reference/ch/reply.mdx",
    "cli-reference/spawn.mdx",
    "cli-reference/task/next.mdx",
    "cli-reference/task/heartbeat.mdx",
    "cli-reference/task/complete.mdx",
    "cli-reference/task/fail.mdx",
    "cli-reference/task/release.mdx",
  ];

  it("keeps regenerated command pages present for agent-facing autonomy commands", () => {
    for (const page of requiredPages) {
      expect(existsSync(resolve(runtimeRoot, page))).toBe(true);
    }
  });

  it("lists exact implemented flags for task, channel, and spawn examples", () => {
    const taskNext = readRuntimeDoc("cli-reference/task/next.mdx");
    const heartbeat = readRuntimeDoc("cli-reference/task/heartbeat.mdx");
    const complete = readRuntimeDoc("cli-reference/task/complete.mdx");
    const fail = readRuntimeDoc("cli-reference/task/fail.mdx");
    const release = readRuntimeDoc("cli-reference/task/release.mdx");
    const send = readRuntimeDoc("cli-reference/ch/send.mdx");
    const reply = readRuntimeDoc("cli-reference/ch/reply.mdx");
    const spawn = readRuntimeDoc("cli-reference/spawn.mdx");

    expectIncludesAll(taskNext, ["--wait", "--lease-seconds", "--capability", "--priority-min"]);
    expectIncludesAll(heartbeat, ["--claim-token", "--lease-seconds"]);
    expectIncludesAll(complete, ["--claim-token", "--result"]);
    expectIncludesAll(fail, ["--claim-token", "--error", "--metadata"]);
    expectIncludesAll(release, ["--claim-token", "--reason"]);
    expectIncludesAll(send, [
      "--body",
      "--task-id",
      "--run-id",
      "--kind",
      "--correlation-id",
      "--coordination-channel-id",
    ]);
    expectIncludesAll(reply, ["--to-message", "--body", "--task-id", "--run-id"]);
    expectIncludesAll(spawn, [
      "--agent",
      "--ttl-seconds",
      "--provider",
      "--model",
      "--role",
      "--tool",
      "--skill",
      "--mcp-server",
      "--workspace-path",
      "--channel",
      "--sandbox-profile",
    ]);
  });
});
