import { createMDX } from "fumadocs-mdx/next";

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  trailingSlash: true,
  images: {
    formats: ["image/avif", "image/webp"],
    qualities: [75, 90],
  },
};

export default withMDX(config);
