import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { Logo } from "@agh/ui";
import { siteConfig } from "./site-config";

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <>
        <span className="sr-only">AGH home</span>
        <Logo variant="logo" decorative className="h-8 w-auto" />
      </>
    ),
    url: "/",
  },
  githubUrl: siteConfig.githubUrl,
  themeSwitch: { enabled: false },
  links: [
    { text: "Home", url: "/", active: "url" },
    { text: "Runtime", url: "/runtime", active: "nested-url" },
    { text: "AGH Network", url: "/protocol", active: "nested-url" },
    { text: "Blog", url: "/blog", active: "nested-url" },
    { text: "Changelog", url: "/changelog", active: "nested-url" },
  ],
};
