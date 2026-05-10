import { clsx, type ClassValue } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

const customTwMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      "font-size": [
        "text-eyebrow",
        "text-badge",
        "text-micro",
        "text-small-body",
        "text-display-2xl",
        "text-site-lead",
        "text-item-title",
        "text-inline-code",
        "text-accent-glyph",
        "text-ui-title-lg",
        "text-pill-group-badge",
      ],
    },
  },
});

export function cn(...inputs: ClassValue[]) {
  return customTwMerge(clsx(inputs));
}
