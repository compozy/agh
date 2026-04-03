---
status: resolved
file: web/vite.config.ts
line: 10
severity: medium
author: claude-reviewer
---

# Issue 158: Vite build output directory not configured to web/dist/ as required



## Review Comment

The task requirement states: "MUST produce static build output in web/dist/ for go:embed". However, the Vite config does not explicitly set the `build.outDir` option:

```typescript
export default defineConfig({
    plugins: [tailwindcss(), svelte(), svelteTesting()],
    resolve: { ... },
    server: { ... },
    test: { ... }
});
```

Vite's default output directory is `dist` relative to the project root, which would be `web/dist/` when building from the `web/` directory. So this likely works correctly by default. However, relying on the default is fragile -- if the build is invoked from a different working directory or the Vite defaults change, the output location could change.

**Suggested fix**: Explicitly set `build.outDir` to ensure the output goes to the expected location:

```typescript
build: {
    outDir: 'dist',
    emptyOutDir: true
}
```

## Triage

- Decision: `invalid`
- Notes:
  - Vite’s default `build.outDir` is `dist` relative to the Vite project root, and this config lives in `web/`, so the produced output is already `web/dist/` as required.
  - I found no code path in this repository that invokes the frontend build from a different Vite root or overrides the output directory.
  - Making the default explicit is harmless, but the review comment does not identify a current defect or failing behavior.
  - Resolution: closed as a non-issue; existing build behavior already satisfies the requirement.
