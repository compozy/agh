import path from "node:path";
import { mkdir, mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";

import { afterEach, describe, expect, it } from "vitest";

import { parseArgs, renderHelp, scaffoldExtension } from "./index.js";

const tempDirs: string[] = [];

describe("@agh/create-extension", () => {
  afterEach(async () => {
    await Promise.all(tempDirs.map(async dir => await rm(dir, { recursive: true, force: true })));
    tempDirs.length = 0;
  });

  it("parses CLI arguments", () => {
    expect(
      parseArgs([
        "my-ext",
        "--template",
        "memory-backend",
        "--dir",
        "./tmp/ext",
        "--sdk-spec",
        "file:../sdk",
      ])
    ).toEqual({
      name: "my-ext",
      template: "memory-backend",
      directory: "./tmp/ext",
      sdkSpec: "file:../sdk",
      help: false,
    });
  });

  it("renders help text", () => {
    expect(renderHelp()).toContain("create-extension <name>");
  });

  it("scaffolds a memory backend template with replacements", async () => {
    const baseDir = await mkdtemp(path.join(tmpdir(), "agh-create-extension-"));
    tempDirs.push(baseDir);

    const projectDir = path.join(baseDir, "my-memory");
    await scaffoldExtension({
      name: "My Memory",
      template: "memory-backend",
      directory: projectDir,
      sdkSpec: "file:../sdk/typescript",
    });

    const packageJSON = await readFile(path.join(projectDir, "package.json"), "utf8");
    const extensionManifest = await readFile(path.join(projectDir, "extension.toml"), "utf8");
    const source = await readFile(path.join(projectDir, "src/index.ts"), "utf8");

    expect(packageJSON).toContain('"name": "my-memory"');
    expect(packageJSON).toContain('"@agh/extension-sdk": "file:../sdk/typescript"');
    expect(extensionManifest).toContain('name = "my-memory"');
    expect(source).toContain('name: "my-memory"');
  });

  it("rejects non-empty target directories", async () => {
    const baseDir = await mkdtemp(path.join(tmpdir(), "agh-create-extension-full-"));
    tempDirs.push(baseDir);
    await mkdir(path.join(baseDir, "existing"), { recursive: true });

    await expect(
      scaffoldExtension({
        name: "existing",
        template: "hook-subprocess",
        directory: baseDir,
      })
    ).rejects.toThrow("target directory is not empty");
  });
});
