import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { protocolDocs } from "@/lib/source";
import type { ReactNode } from "react";

export default function ProtocolDocsLayout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      tree={protocolDocs.pageTree}
      nav={{ enabled: false }}
      sidebar={{
        banner: (
          <p className="text-fd-muted-foreground text-xs font-medium uppercase tracking-widest">
            Protocol Spec
          </p>
        ),
      }}
    >
      {children}
    </DocsLayout>
  );
}
