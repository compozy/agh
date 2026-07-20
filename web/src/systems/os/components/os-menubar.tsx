import { Bell, ChevronsUpDown, Settings } from "lucide-react";

import { Icon, Logo } from "@agh/ui";

import { cn } from "@/lib/utils";

/**
 * The desktop menubar: AGH mark, workspace trigger, app menus, the approvals
 * bell, the ⌘K palette chip, and Settings. Glass shell chrome (the sanctioned
 * carve-out). Presentational — popovers, workspace switching, and palette
 * behavior are wired by the shell in Task 04; a control renders as a <button>
 * only when a real callback is supplied, otherwise as truthful presentation.
 *
 * The mark uses the official `@agh/ui` `Logo` `symbol` at menubar chrome size.
 */
export interface OsMenuBarProps extends React.ComponentProps<"header"> {
  /** Active workspace identity. */
  workspace: { name: string; monogram: string };
  /** App menu labels. */
  menus?: string[];
  /** Approvals count from the bell aggregator; 0/undefined renders no badge. */
  notifications?: number;
  /** Non-interactive system status rendered before the approvals bell. */
  status?: React.ReactNode;
  onLogoClick?: () => void;
  onWorkspaceClick?: () => void;
  onMenuClick?: (menu: string) => void;
  onNotificationsClick?: () => void;
  onCommandClick?: () => void;
  onSettingsClick?: () => void;
  /** Wraps the workspace trigger in a real popover/menu owner (shell wiring). */
  renderWorkspaceTrigger?: (trigger: React.ReactElement) => React.ReactNode;
  /** Wraps one app-menu button in its dropdown owner (shell wiring). */
  renderMenuTrigger?: (menu: string, trigger: React.ReactElement) => React.ReactNode;
  /** Wraps the bell in its popover owner (shell wiring). */
  renderBellTrigger?: (trigger: React.ReactElement) => React.ReactNode;
}

const INTERACTIVE =
  "transition-colors duration-base hover:bg-btn-default-fill hover:text-fg-strong focus-visible:shadow-focus-ring focus-visible:outline-none";

interface ControlProps extends Omit<React.ComponentProps<"button">, "onClick" | "children"> {
  onClick?: () => void;
  children: React.ReactNode;
  /** Hands the button to an overlay owner (popover/menu trigger `render`). */
  wrap?: (trigger: React.ReactElement) => React.ReactNode;
}

/**
 * Renders a <button> when a callback or overlay owner exists, else a
 * non-interactive span (truthful chrome, no dead buttons).
 */
function Control({ onClick, wrap, className, children, ...props }: ControlProps) {
  if (wrap) {
    return wrap(
      <button type="button" className={cn(INTERACTIVE, className)} {...props}>
        {children}
      </button>
    );
  }
  if (!onClick) {
    return (
      <span className={className} {...(props as React.ComponentProps<"span">)}>
        {children}
      </span>
    );
  }
  return (
    <button type="button" className={cn(INTERACTIVE, className)} onClick={onClick} {...props}>
      {children}
    </button>
  );
}

function NotificationBadge({ count }: { count: number }) {
  return (
    <span className="absolute top-0.5 right-0 grid h-3.5 min-w-3.5 place-items-center rounded-full bg-accent px-1 font-mono text-micro font-bold text-accent-ink">
      {count > 9 ? "9+" : count}
    </span>
  );
}

export function OsMenuBar({
  workspace,
  menus = ["Session", "View", "Help"],
  notifications,
  status,
  onLogoClick,
  onWorkspaceClick,
  onMenuClick,
  onNotificationsClick,
  onCommandClick,
  onSettingsClick,
  renderWorkspaceTrigger,
  renderMenuTrigger,
  renderBellTrigger,
  className,
  ...props
}: OsMenuBarProps) {
  return (
    <header
      data-slot="os-menubar"
      aria-label="System bar"
      className={cn(
        "flex h-menubar shrink-0 items-center justify-between border-b border-line bg-shell-glass px-2.5 backdrop-blur-shell",
        className
      )}
      {...props}
    >
      <div className="flex items-center gap-1">
        <Control
          data-slot="os-menubar-logo"
          aria-label="AGH"
          className="grid size-7 place-items-center rounded-menubar-control"
          onClick={onLogoClick}
        >
          <Logo variant="symbol" decorative className="size-menubar-logo" />
        </Control>
        <Control
          data-slot="os-menubar-workspace"
          aria-haspopup={onWorkspaceClick || renderWorkspaceTrigger ? "true" : undefined}
          className="flex h-7 items-center gap-menubar-workspace-gap rounded-md px-2"
          onClick={onWorkspaceClick}
          wrap={renderWorkspaceTrigger}
        >
          <span className="grid size-workspace-avatar place-items-center rounded-sm border border-line-strong bg-elevated font-mono text-badge font-semibold tracking-mono text-fg">
            {workspace.monogram}
          </span>
          <span className="text-small-body font-semibold text-fg-strong">{workspace.name}</span>
          <Icon as={ChevronsUpDown} size="sm" className="text-subtle" />
        </Control>
        <nav data-slot="os-menubar-menus" aria-label="Menus" className="ml-1.5 flex items-center">
          {menus.map(menu => (
            <Control
              key={menu}
              data-menu={menu.toLowerCase()}
              className="flex h-7 items-center rounded-md px-2.5 text-small-body text-muted"
              onClick={onMenuClick ? () => onMenuClick(menu) : undefined}
              wrap={renderMenuTrigger ? trigger => renderMenuTrigger(menu, trigger) : undefined}
            >
              {menu}
            </Control>
          ))}
        </nav>
      </div>

      <div className="flex items-center gap-2">
        {status}
        <Control
          data-slot="os-menubar-bell"
          aria-label="Approvals"
          aria-haspopup={onNotificationsClick || renderBellTrigger ? "true" : undefined}
          className="relative grid size-7 place-items-center rounded-md text-muted"
          onClick={onNotificationsClick}
          wrap={renderBellTrigger}
        >
          <Icon as={Bell} size="lg" />
          {notifications ? <NotificationBadge count={notifications} /> : null}
        </Control>
        <Control
          data-slot="os-menubar-command"
          title="Command palette"
          className="flex h-menubar-chip items-center rounded-md border border-line px-2.5 font-mono text-eyebrow text-muted"
          onClick={onCommandClick}
        >
          ⌘K
        </Control>
        <Control
          data-slot="os-menubar-settings"
          aria-label="Settings"
          title="Settings"
          className="grid size-7 place-items-center rounded-md text-muted"
          onClick={onSettingsClick}
        >
          <Icon as={Settings} size="lg" />
        </Control>
      </div>
    </header>
  );
}
