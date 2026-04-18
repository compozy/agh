import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [viteReact()],
  test: {
    name: "ui",
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    exclude: ["**/node_modules/**", "**/dist/**"],
    setupFiles: ["./src/test-setup.ts"],
    passWithNoTests: true,
  },
});
