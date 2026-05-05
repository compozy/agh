import { CornerUpLeft, GitFork, MoreHorizontal, Pin, SmilePlus } from "lucide-react";

import { Button } from "@agh/ui";

import { cn } from "@/lib/utils";

export interface HoverToolbarHandlers {
  onReply?: () => void;
  onPin?: () => void;
  onFork?: () => void;
  onMore?: () => void;
}

export interface HoverToolbarProps extends HoverToolbarHandlers {
  className?: string;
  /** Stable test id suffix so multiple rows on screen remain addressable. */
  testIdSuffix: string;
}

interface ToolbarButtonProps {
  icon: typeof CornerUpLeft;
  label: string;
  testId: string;
  onClick?: () => void;
  disabled?: boolean;
  disabledTooltip?: string;
}

function ToolbarButton({
  icon: Icon,
  label,
  testId,
  onClick,
  disabled,
  disabledTooltip,
}: ToolbarButtonProps) {
  return (
    <Button
      aria-label={label}
      data-testid={testId}
      disabled={disabled}
      onClick={onClick}
      size="icon-sm"
      title={disabled ? disabledTooltip : label}
      type="button"
      variant="ghost"
    >
      <Icon aria-hidden="true" className="size-3.5" />
    </Button>
  );
}

export function HoverToolbar({
  className,
  testIdSuffix,
  onReply,
  onPin,
  onFork,
  onMore,
}: HoverToolbarProps) {
  return (
    <div
      aria-label="Message actions"
      className={cn(
        "absolute -top-3 right-4 z-10 flex items-center gap-0.5 rounded-[6px] border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-0.5 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100",
        className
      )}
      data-testid={`network-message-toolbar-${testIdSuffix}`}
      role="toolbar"
    >
      <ToolbarButton
        icon={CornerUpLeft}
        label="Reply in thread"
        onClick={onReply}
        testId={`network-message-toolbar-reply-${testIdSuffix}`}
      />
      <ToolbarButton
        disabled
        disabledTooltip="Reactions land post-MVP"
        icon={SmilePlus}
        label="Add reaction"
        testId={`network-message-toolbar-react-${testIdSuffix}`}
      />
      <ToolbarButton
        icon={Pin}
        label="Pin to capability"
        onClick={onPin}
        testId={`network-message-toolbar-pin-${testIdSuffix}`}
      />
      <ToolbarButton
        icon={GitFork}
        label="Fork thread"
        onClick={onFork}
        testId={`network-message-toolbar-fork-${testIdSuffix}`}
      />
      <ToolbarButton
        icon={MoreHorizontal}
        label="More actions"
        onClick={onMore}
        testId={`network-message-toolbar-more-${testIdSuffix}`}
      />
    </div>
  );
}
