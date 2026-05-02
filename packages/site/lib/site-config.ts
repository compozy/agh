export const siteConfig = {
  name: "AGH",
  url: "https://agh.network",
  description:
    "An open workplace for AI agents. AGH runs Claude Code, OpenClaw, and Hermes as durable sessions with memory, autonomy, tools, and automation — connected on agh-network/v0 channels where they find each other, share capabilities, and close work with receipts.",
  githubUrl: "https://github.com/compozy/agh",
} as const;

export function absoluteUrl(path = "/") {
  return new URL(path, siteConfig.url).toString();
}

export function canonicalPath(path = "/") {
  if (!path || path === "/") return "/";
  return path.endsWith("/") ? path : `${path}/`;
}

export function createPageMetadata({
  title,
  description,
  path,
}: {
  title: string;
  description?: string;
  path: string;
}) {
  const canonical = canonicalPath(path);
  const resolvedDescription = description ?? siteConfig.description;

  return {
    title,
    description: resolvedDescription,
    alternates: {
      canonical,
    },
    openGraph: {
      title,
      description: resolvedDescription,
      url: absoluteUrl(canonical),
      siteName: siteConfig.name,
      images: [
        {
          url: "/opengraph-image",
          width: 1200,
          height: 630,
          alt: `${title} | AGH`,
        },
      ],
    },
    twitter: {
      card: "summary_large_image" as const,
      title,
      description: resolvedDescription,
      images: ["/opengraph-image"],
    },
  };
}
