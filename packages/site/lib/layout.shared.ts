import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";

export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: "AGH",
    },
    links: [
      {
        text: "Runtime",
        url: "/runtime",
        active: "nested-url",
      },
      {
        text: "Protocol",
        url: "/protocol",
        active: "nested-url",
      },
    ],
    githubUrl: "https://github.com/compozy/agh",
  };
}
