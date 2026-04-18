import { existsSync, statSync } from "node:fs";
import { join, resolve, sep } from "node:path";

const rootArg = process.argv[2] ?? ".tmp/storybook-static";
const portArg = Number(process.argv[3] ?? process.env.AGH_WEB_STORYBOOK_PORT ?? 6008);
const hostArg = process.argv[4] ?? "127.0.0.1";

const root = resolve(rootArg);
if (!existsSync(root)) {
  console.error(`[serve-storybook] missing Storybook bundle at ${root}`);
  console.error(`[serve-storybook] run 'bun run build:visual' before serving`);
  process.exit(1);
}

const mimeTypes = new Map<string, string>([
  [".css", "text/css; charset=utf-8"],
  [".html", "text/html; charset=utf-8"],
  [".ico", "image/x-icon"],
  [".js", "application/javascript; charset=utf-8"],
  [".mjs", "application/javascript; charset=utf-8"],
  [".json", "application/json; charset=utf-8"],
  [".map", "application/json; charset=utf-8"],
  [".png", "image/png"],
  [".svg", "image/svg+xml"],
  [".woff", "font/woff"],
  [".woff2", "font/woff2"],
]);

function guessContentType(path: string): string {
  const idx = path.lastIndexOf(".");
  const ext = idx >= 0 ? path.slice(idx).toLowerCase() : "";
  return mimeTypes.get(ext) ?? "application/octet-stream";
}

function resolveRequestPath(pathname: string): string | null {
  const safePath = pathname.replace(/\/+/g, "/").replace(/^\/+/, "");
  const abs = resolve(join(root, safePath));
  if (!abs.startsWith(root + sep) && abs !== root) {
    return null;
  }
  if (!existsSync(abs)) {
    return null;
  }
  if (statSync(abs).isDirectory()) {
    const index = join(abs, "index.html");
    return existsSync(index) ? index : null;
  }
  return abs;
}

const server = Bun.serve({
  hostname: hostArg,
  port: portArg,
  fetch(req) {
    const url = new URL(req.url);
    const resolved = resolveRequestPath(url.pathname === "/" ? "/index.html" : url.pathname);
    if (!resolved) {
      return new Response("Not Found", { status: 404 });
    }
    const file = Bun.file(resolved);
    return new Response(file, {
      headers: {
        "content-type": guessContentType(resolved),
        "cache-control": "no-store",
      },
    });
  },
});

process.on("SIGINT", () => {
  server.stop();
  process.exit(0);
});
process.on("SIGTERM", () => {
  server.stop();
  process.exit(0);
});

console.log(`[serve-storybook] serving ${root} at http://${hostArg}:${portArg}`);
