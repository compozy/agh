import { notFound } from "next/navigation";
import { allPosts } from "@/lib/blog";
import { protocolDocs, runtimeDocs } from "@/lib/source";
import { generateOGImage } from "./og";

export const dynamic = "force-static";
export const contentType = "image/png";

interface RouteParams {
  slug: string[];
}

const TRAILING = "image.png" as const;

function withImageSuffix(slug: string[]): string[] {
  return [...slug, TRAILING];
}

export function generateStaticParams(): RouteParams[] {
  const runtime = runtimeDocs
    .generateParams()
    .map(p => ({ slug: withImageSuffix(["runtime", ...(p.slug ?? [])]) }));
  const protocol = protocolDocs
    .generateParams()
    .map(p => ({ slug: withImageSuffix(["protocol", ...(p.slug ?? [])]) }));
  const blog = allPosts().map(post => ({
    slug: withImageSuffix(["blog", post.slug.replace(/^posts\//, "")]),
  }));

  return [...runtime, ...protocol, ...blog];
}

interface ResolvedPage {
  eyebrow: string;
  title: string;
  description?: string;
}

function resolveDoc(tree: "runtime" | "protocol", rest: string[]): ResolvedPage | null {
  const loader = tree === "runtime" ? runtimeDocs : protocolDocs;
  const page = loader.getPage(rest);
  if (!page) return null;
  return {
    eyebrow: tree === "runtime" ? "AGH Runtime" : "AGH Network Protocol",
    title: page.data.title,
    description: page.data.description,
  };
}

function resolveBlog(rest: string[]): ResolvedPage | null {
  const slug = rest.join("/");
  const post = allPosts().find(p => p.slug === `posts/${slug}` || p.slug === slug);
  if (!post) return null;
  return {
    eyebrow: "AGH Blog",
    title: post.title,
    description: post.description,
  };
}

export async function GET(_req: Request, { params }: { params: Promise<RouteParams> }) {
  const { slug } = await params;
  if (slug.length < 2 || slug[slug.length - 1] !== TRAILING) notFound();

  const [tree, ...rest] = slug.slice(0, -1);
  let page: ResolvedPage | null = null;
  if (tree === "runtime" || tree === "protocol") {
    page = resolveDoc(tree, rest);
  } else if (tree === "blog") {
    page = resolveBlog(rest);
  }
  if (!page) notFound();

  return generateOGImage({
    eyebrow: page.eyebrow,
    title: page.title,
    description: page.description,
  });
}
