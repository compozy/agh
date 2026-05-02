"use client";

import { cn } from "@agh/ui";
import Link from "next/link";
import { useEffect, useState } from "react";
import { MonoEyebrow } from "./mono-eyebrow";
import type { TocItem } from "./toc-utils";

export type { TocItem } from "./toc-utils";

export interface TocRailProps {
  items: TocItem[];
}

export function TocRail({ items }: TocRailProps) {
  const [activeId, setActiveId] = useState<string | undefined>(items[0]?.url.replace(/^#/, ""));

  useEffect(() => {
    if (items.length === 0) return;
    if (typeof IntersectionObserver === "undefined") return;
    const ids = items.map(item => item.url.replace(/^#/, ""));
    const observer = new IntersectionObserver(
      entries => {
        const visible = entries
          .filter(entry => entry.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
        if (visible?.target.id) {
          setActiveId(visible.target.id);
        }
      },
      { rootMargin: "-30% 0px -55% 0px", threshold: [0, 1] }
    );
    ids.forEach(id => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });
    return () => observer.disconnect();
  }, [items]);

  if (items.length === 0) return null;

  return (
    <aside aria-label="Blog table of contents" className="sticky top-20 self-start">
      <MonoEyebrow tracking="wide">On this page</MonoEyebrow>
      <ul className="mt-4 flex flex-col gap-2.5">
        {items.map(item => {
          const id = item.url.replace(/^#/, "");
          const isActive = id === activeId;
          return (
            <li key={item.url}>
              <Link
                href={item.url}
                aria-current={isActive ? "location" : undefined}
                className={cn(
                  "block text-[13px] leading-[1.4] transition-colors",
                  isActive
                    ? "text-(--color-accent)"
                    : "text-(--color-text-secondary) hover:text-(--color-text-primary)",
                  item.depth >= 3 && "pl-3"
                )}
              >
                {item.title}
              </Link>
            </li>
          );
        })}
      </ul>
    </aside>
  );
}
