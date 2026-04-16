import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import { protocolDocs } from "@/lib/source";
import { baseOptions } from "@/lib/layout.shared";
import type { ReactNode } from "react";

export default function ProtocolDocsLayout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      {...baseOptions}
      nav={{
        ...baseOptions.nav,
        mode: "auto",
      }}
      tree={protocolDocs.pageTree}
      tabMode="navbar"
    >
      {children}
    </DocsLayout>
  );
}
