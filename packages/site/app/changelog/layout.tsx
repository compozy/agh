import { HomeLayout } from "fumadocs-ui/layouts/home";
import { baseOptions } from "@/lib/layout.shared";
import { HomeHeader } from "@/components/site/home-header";
import type { ReactNode } from "react";

export default function ChangelogLayout({ children }: { children: ReactNode }) {
  return (
    <HomeLayout
      {...baseOptions}
      slots={{
        ...baseOptions.slots,
        header: HomeHeader,
      }}
    >
      <main id="main-content" className="site-home min-h-full bg-(--color-canvas)">
        {children}
      </main>
    </HomeLayout>
  );
}
