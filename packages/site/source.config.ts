import { defineDocs, defineConfig } from "fumadocs-mdx/config";

export const runtime = defineDocs({
  dir: "content/runtime",
});

export const protocol = defineDocs({
  dir: "content/protocol",
});

export default defineConfig({
  mdxOptions: {
    rehypeCodeOptions: {
      themes: {
        light: "vitesse-light",
        dark: "vitesse-dark",
      },
    },
  },
});
