import { execFile } from "node:child_process";
import { readdir, readFile, writeFile, mkdir } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const maxGitCliffOutputBytes = 32 * 1024 * 1024;
const changelogDirectory = "packages/site/content/blog/changelog";
const releaseNotesDirectory = ".release-notes";
const summaryMaxLength = 280;

export type ReleaseStatus = "stable" | "beta" | "alpha" | "breaking";
type ReleaseNoteType = "breaking" | "feature" | "fix" | "highlight";

export interface GitCliffCommit {
  message: string;
  group: string;
  breaking: boolean;
  breakingDescription?: string;
  scope?: string;
  rawMessage?: string;
}

export interface GitCliffRelease {
  version?: string;
  timestamp?: number;
  commits: GitCliffCommit[];
  previousVersion?: string;
}

export interface ReleaseNoteInput {
  title: string;
  type: ReleaseNoteType;
  body: string;
  summary?: string;
  sourcePath: string;
}

export interface ChangelogReleaseEntry {
  version: string;
  date: string;
  status: ReleaseStatus;
  summary: string;
  added: string[];
  changed: string[];
  fixed: string[];
  breaking: string[];
  compareUrl?: string;
  body: string;
}

export interface BuildChangelogReleaseInput {
  version: string;
  generatedAt?: string;
  previousTag?: string;
  githubOwner?: string;
  githubRepo?: string;
  context: GitCliffRelease[];
  releaseNotes?: ReleaseNoteInput[];
}

export function parseGitCliffContext(raw: string): GitCliffRelease[] {
  const parsed: unknown = JSON.parse(raw);
  if (!Array.isArray(parsed)) {
    throw new Error("git-cliff context must be a JSON array");
  }
  return parsed.map(parseGitCliffRelease);
}

export function parseReleaseNoteMarkdown(
  sourcePath: string,
  content: string
): ReleaseNoteInput | null {
  const normalized = content.replaceAll("\r\n", "\n");
  if (!normalized.startsWith("---\n")) {
    return null;
  }
  const rest = normalized.slice("---\n".length);
  const footerIndex = rest.indexOf("\n---\n");
  if (footerIndex === -1) {
    return null;
  }
  const metadata = parseSimpleFrontmatter(rest.slice(0, footerIndex));
  const title = metadata.get("title")?.trim();
  const type = parseReleaseNoteType(metadata.get("type"));
  const body = rest.slice(footerIndex + "\n---\n".length).trim();
  if (!title || !type || !body) {
    return null;
  }
  const summary = metadata.get("summary")?.trim();
  return {
    title,
    type,
    body,
    summary: summary === "" ? undefined : summary,
    sourcePath,
  };
}

export function buildChangelogRelease(input: BuildChangelogReleaseInput): ChangelogReleaseEntry {
  const version = normalizeVersionTag(input.version);
  const release = selectRelease(input.context, version);
  const releaseNotes = [...(input.releaseNotes ?? [])].sort((left, right) =>
    left.sourcePath.localeCompare(right.sourcePath)
  );
  const buckets = {
    added: [] as string[],
    changed: [] as string[],
    fixed: [] as string[],
    breaking: [] as string[],
  };
  for (const note of releaseNotes) {
    addReleaseNoteToBuckets(buckets, note);
  }
  for (const commit of release.commits) {
    const item = formatCommitItem(commit);
    if (item === "") {
      continue;
    }
    if (commit.breaking) {
      addUnique(buckets.breaking, commit.breakingDescription ?? item);
      continue;
    }
    const bucket = classifyCommit(commit);
    if (bucket !== undefined) {
      addUnique(buckets[bucket], item);
    }
  }
  const status = releaseStatus(version, buckets.breaking);
  return {
    version,
    date: releaseDate(input.generatedAt, release.timestamp),
    status,
    summary: releaseSummary(version, buckets, releaseNotes),
    added: buckets.added,
    changed: buckets.changed,
    fixed: buckets.fixed,
    breaking: buckets.breaking,
    compareUrl: compareUrl(input, release.previousVersion, version),
    body: releaseNotesBody(version, releaseNotes),
  };
}

export function renderReleaseMdx(entry: ChangelogReleaseEntry): string {
  const frontmatter = [
    "---",
    `version: ${yamlString(entry.version)}`,
    `date: ${yamlString(entry.date)}`,
    `status: ${yamlString(entry.status)}`,
    `summary: ${yamlString(entry.summary)}`,
    yamlStringArray("added", entry.added),
    yamlStringArray("changed", entry.changed),
    yamlStringArray("fixed", entry.fixed),
    yamlStringArray("breaking", entry.breaking),
    entry.compareUrl === undefined ? undefined : `compareUrl: ${yamlString(entry.compareUrl)}`,
    "---",
  ].filter((line): line is string => line !== undefined);
  return `${frontmatter.join("\n")}\n\n${entry.body.trim()}\n`;
}

export async function generateChangelogRelease(
  cwd: string,
  env: NodeJS.ProcessEnv
): Promise<string> {
  const version = normalizeVersionTag(requiredEnv(env, "PR_RELEASE_VERSION"));
  const rawContext = await runGitCliffContext(cwd, version);
  const context = parseGitCliffContext(rawContext);
  const releaseNotes = await readReleaseNotes(cwd, version);
  const repository = repositoryFromEnv(env);
  const entry = buildChangelogRelease({
    version,
    generatedAt: env.PR_RELEASE_DATE,
    previousTag: emptyToUndefined(env.PR_RELEASE_PREVIOUS_TAG),
    githubOwner: repository.owner,
    githubRepo: repository.repo,
    context,
    releaseNotes,
  });
  const outputDirectory = path.join(cwd, changelogDirectory);
  await mkdir(outputDirectory, { recursive: true });
  const outputPath = path.join(outputDirectory, `${version}.mdx`);
  await writeFile(outputPath, renderReleaseMdx(entry), "utf8");
  return outputPath;
}

async function runGitCliffContext(cwd: string, version: string): Promise<string> {
  const { stdout } = await execFileAsync(
    "git-cliff",
    ["--context", "--unreleased", "--tag", version, "--strip", "all"],
    {
      cwd,
      maxBuffer: maxGitCliffOutputBytes,
    }
  );
  return stdout;
}

async function readReleaseNotes(cwd: string, version: string): Promise<ReleaseNoteInput[]> {
  const directories = [
    path.join(cwd, releaseNotesDirectory),
    path.join(cwd, releaseNotesDirectory, "archive", version),
  ];
  const notes: ReleaseNoteInput[] = [];
  for (const directory of directories) {
    const entries = await readDirectoryEntries(directory);
    for (const entry of entries) {
      if (!entry.isFile() || path.extname(entry.name) !== ".md") {
        continue;
      }
      const sourcePath = path.relative(cwd, path.join(directory, entry.name));
      const content = await readFile(path.join(directory, entry.name), "utf8");
      const note = parseReleaseNoteMarkdown(sourcePath, content);
      if (note !== null) {
        notes.push(note);
      }
    }
  }
  return notes.sort((left, right) => left.sourcePath.localeCompare(right.sourcePath));
}

async function readDirectoryEntries(directory: string) {
  try {
    return await readdir(directory, { withFileTypes: true });
  } catch (error) {
    if (isNodeError(error) && error.code === "ENOENT") {
      return [];
    }
    throw error;
  }
}

function parseGitCliffRelease(value: unknown): GitCliffRelease {
  if (!isRecord(value)) {
    throw new Error("git-cliff release entry must be an object");
  }
  const commitsValue = value.commits;
  if (!Array.isArray(commitsValue)) {
    throw new Error("git-cliff release entry must include commits");
  }
  return {
    version: readString(value, "version"),
    timestamp: readNumber(value, "timestamp"),
    commits: commitsValue.map(parseGitCliffCommit),
    previousVersion: parsePreviousVersion(value.previous),
  };
}

function parseGitCliffCommit(value: unknown): GitCliffCommit {
  if (!isRecord(value)) {
    throw new Error("git-cliff commit entry must be an object");
  }
  return {
    message: readString(value, "message") ?? "",
    group: readString(value, "group") ?? "",
    breaking: readBoolean(value, "breaking") ?? false,
    breakingDescription: emptyToUndefined(readString(value, "breaking_description")),
    scope: emptyToUndefined(readString(value, "scope")),
    rawMessage: emptyToUndefined(readString(value, "raw_message")),
  };
}

function parsePreviousVersion(value: unknown): string | undefined {
  if (!isRecord(value)) {
    return undefined;
  }
  return emptyToUndefined(readString(value, "version"));
}

function selectRelease(context: GitCliffRelease[], version: string): GitCliffRelease {
  const selected =
    context.find(release => normalizeVersionTag(release.version ?? "") === version) ?? context[0];
  if (selected === undefined) {
    return { version, commits: [] };
  }
  return selected;
}

function addReleaseNoteToBuckets(
  buckets: Pick<ChangelogReleaseEntry, "added" | "changed" | "fixed" | "breaking">,
  note: ReleaseNoteInput
): void {
  switch (note.type) {
    case "breaking":
      addUnique(buckets.breaking, note.title);
      return;
    case "feature":
    case "highlight":
      addUnique(buckets.added, note.title);
      return;
    case "fix":
      addUnique(buckets.fixed, note.title);
  }
}

function classifyCommit(commit: GitCliffCommit): "added" | "changed" | "fixed" | undefined {
  const kind = conventionalCommitKind(commit.rawMessage);
  switch (kind) {
    case "test":
    case "ci":
    case "style":
    case "chore":
      return undefined;
    case "feat":
    case "feature":
      return "added";
    case "fix":
    case "bugfix":
    case "security":
      return "fixed";
    case "docs":
    case "perf":
    case "refactor":
    case "deps":
    case "build":
      return "changed";
  }
  const group = normalizeWhitespace(commit.group).toLowerCase();
  if (includesAny(group, ["feature", "features"])) {
    return "added";
  }
  if (includesAny(group, ["bug fix", "bug fixes", "security"])) {
    return "fixed";
  }
  if (includesAny(group, ["documentation", "performance", "refactor", "dependenc", "build"])) {
    return "changed";
  }
  return undefined;
}

function formatCommitItem(commit: GitCliffCommit): string {
  const message = normalizeWhitespace(commit.message);
  if (message === "") {
    return "";
  }
  const summary = uppercaseFirst(message);
  return commit.scope === undefined ? summary : `${commit.scope}: ${summary}`;
}

function conventionalCommitKind(rawMessage: string | undefined): string | undefined {
  if (rawMessage === undefined) {
    return undefined;
  }
  const match = /^([a-z]+)(?:\([^)]*\))?!?:/.exec(rawMessage.trim().toLowerCase());
  return match?.[1];
}

function includesAny(value: string, needles: string[]): boolean {
  return needles.some(needle => value.includes(needle));
}

function releaseStatus(version: string, breaking: string[]): ReleaseStatus {
  if (breaking.length > 0) {
    return "breaking";
  }
  const versionNumber = version.replace(/^v/, "");
  const prerelease = versionNumber.split("-", 2)[1];
  if (prerelease?.startsWith("beta")) {
    return "beta";
  }
  if (prerelease !== undefined) {
    return "alpha";
  }
  const major = Number.parseInt(versionNumber.split(".", 1)[0] ?? "0", 10);
  return major >= 1 ? "stable" : "alpha";
}

function releaseDate(generatedAt: string | undefined, timestamp: number | undefined): string {
  const generatedDate = parseDate(generatedAt);
  if (generatedDate !== undefined) {
    return generatedDate.toISOString();
  }
  if (timestamp !== undefined) {
    return new Date(timestamp * 1000).toISOString();
  }
  return new Date().toISOString();
}

function releaseSummary(
  version: string,
  buckets: Pick<ChangelogReleaseEntry, "added" | "changed" | "fixed" | "breaking">,
  releaseNotes: ReleaseNoteInput[]
): string {
  const noteSummary = releaseNotes.find(note => note.summary !== undefined)?.summary;
  const noteTitle = releaseNotes[0]?.title;
  const candidate =
    noteSummary ??
    noteTitle ??
    buckets.breaking[0] ??
    buckets.added[0] ??
    buckets.fixed[0] ??
    buckets.changed[0] ??
    `Release ${version}`;
  return clampSummary(candidate);
}

function compareUrl(
  input: BuildChangelogReleaseInput,
  previousVersion: string | undefined,
  version: string
): string | undefined {
  if (input.githubOwner === undefined || input.githubRepo === undefined) {
    return undefined;
  }
  const previous = input.previousTag ?? previousVersion;
  const base = `https://github.com/${input.githubOwner}/${input.githubRepo}`;
  if (previous === undefined) {
    return `${base}/releases/tag/${version}`;
  }
  return `${base}/compare/${previous}...${version}`;
}

function releaseNotesBody(version: string, releaseNotes: ReleaseNoteInput[]): string {
  const posture = releaseVerificationPostureBody();
  if (releaseNotes.length === 0) {
    return `Generated from release artifacts for ${version}.\n\n${posture}`;
  }
  const sections = releaseNotes.map(note => `## ${note.title}\n\n${note.body.trim()}`);
  return `${sections.join("\n\n")}\n\n${posture}`;
}

function releaseVerificationPostureBody(): string {
  return [
    "## Verification posture",
    "",
    "This generated release entry names the release gates and artifact guarantees that the AGH release workflow owns:",
    "",
    "- Repository gate: `make verify` covers codegen drift, Bun lint/typecheck/test/build, Go fmt/lint/test/build, and import boundaries.",
    "- Release PR dry-run: `pr-release dry-run`, `make test-e2e-nightly`, and `make test-integration` run before the release commit is merged.",
    "- Production release: generated release assets are validated before `goreleaser release --clean` publishes the release.",
    "- Artifact provenance: GoReleaser signs `checksums.txt` with cosign, publishes the Sigstore bundle `checksums.txt.sigstore.json`, and generates Syft SBOMs for archives, packages, and source.",
    "",
    "Known limitation: this generated changelog does not claim a manual post-release install smoke or live-provider QA run unless a release note in this entry names that evidence.",
  ].join("\n");
}

function parseSimpleFrontmatter(frontmatter: string): Map<string, string> {
  const metadata = new Map<string, string>();
  for (const line of frontmatter.split("\n")) {
    const match = /^([A-Za-z_][A-Za-z0-9_-]*):\s*(.*)$/.exec(line.trim());
    if (match === null) {
      continue;
    }
    const key = match[1];
    const value = match[2];
    if (key !== undefined && value !== undefined) {
      metadata.set(key, unquoteFrontmatterValue(value));
    }
  }
  return metadata;
}

function parseReleaseNoteType(value: string | undefined): ReleaseNoteType | undefined {
  const normalized = value?.trim().toLowerCase();
  switch (normalized) {
    case "breaking":
    case "feature":
    case "fix":
    case "highlight":
      return normalized;
    default:
      return undefined;
  }
}

function unquoteFrontmatterValue(value: string): string {
  const trimmed = value.trim();
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}

function repositoryFromEnv(env: NodeJS.ProcessEnv): { owner?: string; repo?: string } {
  const owner = emptyToUndefined(env.PR_RELEASE_GITHUB_OWNER ?? env.GITHUB_REPOSITORY_OWNER);
  const repo = emptyToUndefined(env.PR_RELEASE_GITHUB_REPO);
  if (owner !== undefined && repo !== undefined) {
    return { owner, repo };
  }
  const slug = emptyToUndefined(env.GITHUB_REPOSITORY);
  if (slug === undefined || !slug.includes("/")) {
    return { owner, repo };
  }
  const [slugOwner, slugRepo] = slug.split("/", 2);
  return {
    owner: owner ?? emptyToUndefined(slugOwner),
    repo: repo ?? emptyToUndefined(slugRepo),
  };
}

function yamlStringArray(key: string, values: string[]): string {
  if (values.length === 0) {
    return `${key}: []`;
  }
  return [`${key}:`, ...values.map(value => `  - ${yamlString(value)}`)].join("\n");
}

function yamlString(value: string): string {
  return JSON.stringify(value);
}

function normalizeVersionTag(version: string): string {
  const trimmed = version.trim();
  if (trimmed === "") {
    throw new Error("release version cannot be empty");
  }
  return trimmed.startsWith("v") ? trimmed : `v${trimmed}`;
}

function normalizeWhitespace(value: string): string {
  return value.replaceAll(/\s+/g, " ").trim();
}

function uppercaseFirst(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function addUnique(values: string[], value: string): void {
  const normalized = normalizeWhitespace(value);
  if (normalized !== "" && !values.includes(normalized)) {
    values.push(normalized);
  }
}

function clampSummary(value: string): string {
  const normalized = normalizeWhitespace(value);
  if (normalized.length <= summaryMaxLength) {
    return normalized;
  }
  return `${normalized.slice(0, summaryMaxLength - 3).trimEnd()}...`;
}

function parseDate(value: string | undefined): Date | undefined {
  if (value === undefined) {
    return undefined;
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

function requiredEnv(env: NodeJS.ProcessEnv, key: string): string {
  const value = emptyToUndefined(env[key]);
  if (value === undefined) {
    throw new Error(`${key} is required`);
  }
  return value;
}

function emptyToUndefined(value: string | undefined | null): string | undefined {
  const trimmed = value?.trim();
  return trimmed === undefined || trimmed === "" ? undefined : trimmed;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readString(record: Record<string, unknown>, key: string): string | undefined {
  const value = record[key];
  return typeof value === "string" ? value : undefined;
}

function readNumber(record: Record<string, unknown>, key: string): number | undefined {
  const value = record[key];
  return typeof value === "number" ? value : undefined;
}

function readBoolean(record: Record<string, unknown>, key: string): boolean | undefined {
  const value = record[key];
  return typeof value === "boolean" ? value : undefined;
}

function isNodeError(error: unknown): error is Error & { code?: string } {
  return error instanceof Error && "code" in error;
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) {
  const outputPath = await generateChangelogRelease(process.cwd(), process.env);
  process.stdout.write(`Generated ${path.relative(process.cwd(), outputPath)}\n`);
}
