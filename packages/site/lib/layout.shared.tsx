import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { Logo } from "@/components/logo";

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: <Logo />,
    url: "/",
  },
  githubUrl: "https://github.com/pedronauck/agh",
  themeSwitch: { enabled: false },
  links: [
    { text: "Home", url: "/", active: "url" },
    { text: "Runtime", url: "/runtime", active: "nested-url" },
    { text: "AGH Network", url: "/protocol", active: "nested-url" },
  ],
};
