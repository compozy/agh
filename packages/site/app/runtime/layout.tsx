import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import { RootProvider } from "fumadocs-ui/provider/next";
import { runtimeLayoutTree, runtimeTabs } from "@/lib/source";
import { baseOptions } from "@/lib/layout.shared";
import { DocsHeader } from "@/components/site/docs-header";
import type { ReactNode } from "react";

export default function RuntimeDocsLayout({ children }: { children: ReactNode }) {
  return (
    <RootProvider
      theme={{
        defaultTheme: "dark",
        forcedTheme: "dark",
        enabled: false,
      }}
    >
      <DocsLayout
        {...baseOptions}
        nav={{
          ...baseOptions.nav,
          mode: "auto",
        }}
        slots={{ header: DocsHeader }}
        tree={runtimeLayoutTree}
        tabs={runtimeTabs}
        tabMode="navbar"
      >
        {children}
      </DocsLayout>
    </RootProvider>
  );
}
