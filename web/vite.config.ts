import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";

export default defineConfig({
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          minSize: 20_000,
          maxSize: 250_000,
          groups: [
            {
              name: "markdown-vendor",
              test: /node_modules[\\/](react-markdown|remark-gfm|react-syntax-highlighter)[\\/]/,
              priority: 30,
            },
            {
              name: "ai-vendor",
              test: /node_modules[\\/](@ai-sdk[\\/]react|ai)[\\/]/,
              priority: 25,
            },
            {
              name: "vendor",
              test: /node_modules[\\/]/,
              priority: 10,
            },
          ],
        },
      },
    },
  },
  plugins: [
    devtools(),
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    viteReact(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:2123",
        changeOrigin: true,
      },
    },
  },
});
