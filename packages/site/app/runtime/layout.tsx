import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import { runtimeDocs } from "@/lib/source";
import { baseOptions } from "@/lib/layout.shared";
import type { ReactNode } from "react";

export default function RuntimeDocsLayout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      {...baseOptions}
      nav={{
        ...baseOptions.nav,
        mode: "auto",
      }}
      tree={runtimeDocs.pageTree}
      tabMode="navbar"
    >
      {children}
    </DocsLayout>
  );
}
