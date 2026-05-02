"use client";

import type { Separator } from "fumadocs-core/page-tree";

export function SidebarSectionLabel({ item }: { item: Separator }) {
  return (
    <p className="mt-5 mb-1 px-2 text-[10px] font-semibold uppercase tracking-[0.08em] text-fd-muted-foreground/70 first:mt-3">
      {item.icon}
      {item.name}
    </p>
  );
}
