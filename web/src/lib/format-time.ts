/**
 * Time formatters consumed by Tasks/Bridges/Knowledge/Settings runtime surfaces.
 *
 * The canonical implementation lives in `@agh/ui` (`packages/ui/src/lib/format-time.ts`)
 * because the `<Time>` primitive must consume them without crossing the
 * `@agh/ui` → `web/` package boundary. This module is a thin re-export so
 * runtime callsites can keep their `@/lib/format-time` import path.
 */
export {
  FORMAT_TIME_FALLBACK,
  formatAbsoluteTime,
  formatDuration,
  formatRelativeTime,
} from "@agh/ui";
