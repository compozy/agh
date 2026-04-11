import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    name: "create-extension",
    environment: "node",
    include: ["src/**/*.test.ts"],
    exclude: ["dist/**", "**/node_modules/**"],
  },
});
