import { useEffect, useState } from "react";

import { formatDuration } from "@agh/ui";

/**
 * Live-ticking elapsed label for an active run. Recomputes once per second
 * while `active`; renders a stable formatted duration otherwise. Interval sync
 * with wall-clock time is an external-system concern, so the effect is the
 * right tool here.
 */
export function useLiveElapsed(startedAt?: string | null, active = false): string | undefined {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [active]);

  if (!startedAt) return undefined;
  const startedMs = Date.parse(startedAt);
  if (Number.isNaN(startedMs)) return undefined;
  return formatDuration(Math.max(0, now - startedMs));
}
