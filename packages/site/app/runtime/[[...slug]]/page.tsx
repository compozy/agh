import { runtimeDocs } from "@/lib/source";
import { DocsBody, DocsPage } from "fumadocs-ui/page";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { DocPageMasthead } from "@/components/docs/doc-page-masthead";
import { getMDXComponents } from "@/mdx-components";
import { createPageMetadata } from "@/lib/site-config";

interface PageProps {
  params: Promise<{ slug?: string[] }>;
}

export default async function Page(props: PageProps) {
  const params = await props.params;
  const slug = params.slug ?? [];
  const page = runtimeDocs.getPage(slug);
  if (!page) notFound();

  const MDX = page.data.body;

  return (
    <DocsPage
      id="main-content"
      toc={page.data.toc}
      breadcrumb={{ enabled: true }}
      className="px-4 pt-8 pb-12 md:px-6 xl:layout:[--fd-toc-width:14rem] xl:pt-10"
    >
      <DocPageMasthead
        kind="runtime"
        slug={slug}
        title={page.data.title}
        description={page.data.description}
      />
      <DocsBody className="site-doc-body mt-8 max-w-none">
        <MDX components={getMDXComponents()} />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return runtimeDocs.generateParams();
}

export async function generateMetadata(props: PageProps): Promise<Metadata> {
  const params = await props.params;
  const page = runtimeDocs.getPage(params.slug ?? []);
  if (!page) notFound();

  return {
    ...createPageMetadata({
      title: page.data.title,
      description: page.data.description,
      path: page.url,
    }),
  };
}
