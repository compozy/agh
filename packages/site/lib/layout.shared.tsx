import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { Logo } from "@agh/ui";

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: <Logo variant="logo" decorative className="h-8 w-auto" />,
    url: "/",
  },
  githubUrl: "https://github.com/compozy/agh",
  themeSwitch: { enabled: false },
  links: [
    { text: "Home", url: "/", active: "url" },
    { text: "Runtime", url: "/runtime", active: "nested-url" },
    { text: "AGH Network", url: "/protocol", active: "nested-url" },
  ],
};
