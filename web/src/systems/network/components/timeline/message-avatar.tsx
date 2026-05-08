import { cn } from "@/lib/utils";

import { getIdentityInitial, pickIdentityPaletteColors } from "../../lib/palette";

export interface MessageAvatarProps {
  seed: string;
  initialFrom: string;
  /**
   * 36 in the channel timeline (`_design.md` §5.2.1), 32 inside the thread
   * overlay (§3.2), 20 in the channel rail Direct Rooms section.
   */
  sizePx: 36 | 32 | 20;
  className?: string;
}

export function MessageAvatar({ seed, initialFrom, sizePx, className }: MessageAvatarProps) {
  const [background, foreground] = pickIdentityPaletteColors(seed);
  const initial = getIdentityInitial(initialFrom);

  return (
    <div
      aria-hidden="true"
      className={cn(
        "flex shrink-0 select-none items-center justify-center rounded-chip font-mono text-xs font-semibold uppercase tracking-mono",
        className
      )}
      data-testid="network-message-avatar"
      style={{
        backgroundColor: background,
        color: foreground,
        height: sizePx,
        width: sizePx,
      }}
    >
      {initial}
    </div>
  );
}
