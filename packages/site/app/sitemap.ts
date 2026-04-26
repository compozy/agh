import type { MetadataRoute } from "next";
import { protocolDocs, runtimeDocs } from "@/lib/source";
import { absoluteUrl, canonicalPath } from "@/lib/site-config";

export const dynamic = "force-static";

function pageEntry(path: string): MetadataRoute.Sitemap[number] {
  return {
    url: absoluteUrl(canonicalPath(path)),
    changeFrequency: "weekly",
    priority: path === "/" ? 1 : 0.7,
  };
}

export default function sitemap(): MetadataRoute.Sitemap {
  const docsPaths = [...runtimeDocs.getPages(), ...protocolDocs.getPages()].map(page => page.url);
  const paths = Array.from(new Set(["/", ...docsPaths])).sort();

  return paths.map(pageEntry);
}
