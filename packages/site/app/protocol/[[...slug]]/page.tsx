import { protocolDocs } from "@/lib/source";
import { DocsBody, DocsPage } from "fumadocs-ui/page";
import { notFound, redirect } from "next/navigation";
import type { Metadata } from "next";
import { DocPageMasthead } from "@/components/docs/doc-page-masthead";
import { getMDXComponents } from "@/mdx-components";

interface PageProps {
  params: Promise<{ slug?: string[] }>;
}

export default async function Page(props: PageProps) {
  const params = await props.params;
  if (!params.slug || params.slug.length === 0) {
    redirect("/protocol/overview/");
  }
  const page = protocolDocs.getPage(params.slug);
  if (!page) notFound();

  const MDX = page.data.body;

  return (
    <DocsPage
      toc={page.data.toc}
      breadcrumb={{ enabled: false }}
      className="px-4 pt-8 pb-12 md:px-6 xl:layout:[--fd-toc-width:14rem] xl:pt-10"
    >
      <DocPageMasthead
        kind="protocol"
        slug={params.slug}
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
  return protocolDocs.generateParams();
}

export async function generateMetadata(props: PageProps): Promise<Metadata> {
  const params = await props.params;
  const page = protocolDocs.getPage(params.slug);
  if (!page) notFound();

  return {
    title: page.data.title,
    description: page.data.description,
  };
}
