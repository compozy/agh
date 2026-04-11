import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    name: "extension-sdk",
    environment: "node",
    include: ["src/**/*.test.ts"],
    exclude: ["dist/**", "**/node_modules/**"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      exclude: ["dist/**", "**/*.d.ts", "src/index.ts"],
    },
  },
});
