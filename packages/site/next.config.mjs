import { createMDX } from "fumadocs-mdx/next";
import { STATIC_EXPORT_BUILD_ID } from "./lib/static-export-build-id.mjs";

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  output: "export",
  reactStrictMode: true,
  generateBuildId: async () => STATIC_EXPORT_BUILD_ID,
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
};

export default withMDX(config);
