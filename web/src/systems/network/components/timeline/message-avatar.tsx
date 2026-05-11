import { Eyebrow } from "@agh/ui";

import { cn } from "@/lib/utils";

import { getIdentityInitial, pickIdentityPaletteColors } from "../../lib/palette";

export type MessageAvatarRole = "agent" | "human" | "system";

const ROLE_LABEL: Record<MessageAvatarRole, string> = {
  agent: "Agent",
  human: "Human",
  system: "System",
};

export interface MessageAvatarProps {
  seed: string;
  initialFrom: string;
  /**
   * 36 in the channel timeline (`_design.md` §5.2.1), 32 inside the thread
   * overlay (§3.2), 20 in the channel rail Direct Rooms section.
   */
  sizePx: 36 | 32 | 20;
  /**
   * Owner role drives the `role="img"` aria-label per ADR-013 §2. When
   * provided, the avatar announces `{Role} {Name}` so screen readers retain
   * the signal that previously came from the role pill on message rows.
   */
  role?: MessageAvatarRole;
  /** Human-readable name for the aria-label (defaults to `initialFrom`). */
  name?: string;
  className?: string;
}

export function MessageAvatar({
  seed,
  initialFrom,
  sizePx,
  role,
  name,
  className,
}: MessageAvatarProps) {
  const [background, foreground] = pickIdentityPaletteColors(seed);
  const initial = getIdentityInitial(initialFrom);
  const labeled = role !== undefined;
  const announcedName = name?.trim() || initialFrom;
  const ariaLabel = labeled ? `${ROLE_LABEL[role]} ${announcedName}` : undefined;

  return (
    <div
      aria-hidden={labeled ? undefined : true}
      aria-label={ariaLabel}
      className={cn(
        "flex shrink-0 select-none items-center justify-center rounded-chip",
        className
      )}
      data-owner-role={role}
      data-testid="network-message-avatar"
      role={labeled ? "img" : undefined}
      style={{
        backgroundColor: background,
        color: foreground,
        height: sizePx,
        width: sizePx,
      }}
    >
      <Eyebrow aria-hidden="true" className="text-current">
        {initial}
      </Eyebrow>
    </div>
  );
}
