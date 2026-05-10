import type { LucideIcon } from "lucide-react";

/**
 * Static topbar metadata declared by every TanStack Router route's `beforeLoad`.
 *
 * Plain fields carry static metadata; `getCount` is a function only because
 * counts may close over loader data. Eyebrow lives in `<PageShell>` content,
 * not the topbar, so the route context does not carry it.
 */
export interface TopbarRouteContext {
  title: string;
  icon?: LucideIcon;
  subtitle?: string;
  getCount?: () => number | string;
}
