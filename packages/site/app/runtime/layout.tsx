import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { runtimeDocs } from "@/lib/source";
import type { ReactNode } from "react";

export default function RuntimeDocsLayout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      tree={runtimeDocs.pageTree}
      nav={{ enabled: false }}
      sidebar={{
        banner: (
          <p className="text-fd-muted-foreground text-xs font-medium uppercase tracking-widest">
            Runtime Docs
          </p>
        ),
      }}
    >
      {children}
    </DocsLayout>
  );
}
