"use client";

import * as React from "react";
import { Dialog as SheetPrimitive } from "@base-ui/react/dialog";
import { AnimatePresence, m, type Variants } from "motion/react";
import { XIcon } from "lucide-react";

import { cn } from "../lib/utils";
import { Button } from "./button";
import { useInitialState } from "./use-initial-state";

type SheetSide = "top" | "right" | "bottom" | "left";

type SheetActionsRef = React.RefObject<SheetPrimitive.Root.Actions | null>;

interface SheetMotionContextValue {
  actionsRef: SheetActionsRef;
  open: boolean;
}

const SheetMotionContext = React.createContext<SheetMotionContextValue | null>(null);

function useSheetMotion(): SheetMotionContextValue {
  const ctx = React.use(SheetMotionContext);
  if (!ctx) {
    throw new Error("Sheet.* components must be used inside <Sheet>.");
  }
  return ctx;
}

type SheetRootProps = SheetPrimitive.Root.Props;

function Sheet({
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  children,
  ...props
}: SheetRootProps) {
  const actionsRef = React.useRef<SheetPrimitive.Root.Actions | null>(null);
  const [uncontrolledOpen, setUncontrolledOpen] = useInitialState(defaultOpen);
  const isControlled = controlledOpen !== undefined;
  const open = isControlled ? Boolean(controlledOpen) : uncontrolledOpen;

  const handleOpenChange: NonNullable<SheetRootProps["onOpenChange"]> = (next, details) => {
    if (!isControlled) setUncontrolledOpen(next);
    onOpenChange?.(next, details);
  };

  const value = React.useMemo<SheetMotionContextValue>(() => ({ actionsRef, open }), [open]);

  return (
    <SheetPrimitive.Root
      data-slot="sheet"
      actionsRef={actionsRef}
      open={open}
      defaultOpen={defaultOpen}
      onOpenChange={handleOpenChange}
      {...props}
    >
      <SheetMotionContext.Provider value={value}>
        {children as React.ReactNode}
      </SheetMotionContext.Provider>
    </SheetPrimitive.Root>
  );
}

function SheetTrigger({ ...props }: SheetPrimitive.Trigger.Props) {
  return <SheetPrimitive.Trigger data-slot="sheet-trigger" {...props} />;
}

function SheetClose({ ...props }: SheetPrimitive.Close.Props) {
  return <SheetPrimitive.Close data-slot="sheet-close" {...props} />;
}

function SheetPortal({ ...props }: SheetPrimitive.Portal.Props) {
  return <SheetPrimitive.Portal data-slot="sheet-portal" {...props} />;
}

function SheetOverlay({ className, ...props }: SheetPrimitive.Backdrop.Props) {
  return (
    <SheetPrimitive.Backdrop
      data-slot="sheet-overlay"
      render={
        <m.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2, ease: [0.2, 0, 0, 1] }}
        />
      }
      className={cn("fixed inset-0 z-50 bg-(--overlay-scrim)", className)}
      {...props}
    />
  );
}

const SIDE_VARIANTS: Record<SheetSide, Variants> = {
  top: {
    hidden: { opacity: 0, y: "-2.5rem" },
    visible: { opacity: 1, y: 0 },
  },
  bottom: {
    hidden: { opacity: 0, y: "2.5rem" },
    visible: { opacity: 1, y: 0 },
  },
  left: {
    hidden: { opacity: 0, x: "-2.5rem" },
    visible: { opacity: 1, x: 0 },
  },
  right: {
    hidden: { opacity: 0, x: "2.5rem" },
    visible: { opacity: 1, x: 0 },
  },
};

const SIDE_CLASSES: Record<SheetSide, string> = {
  top: "inset-x-0 top-0 h-auto rounded-b-xl",
  bottom: "inset-x-0 bottom-0 h-auto rounded-t-xl",
  left: "inset-y-0 left-0 h-full w-3/4 rounded-r-xl sm:max-w-sm",
  right: "inset-y-0 right-0 h-full w-3/4 rounded-l-xl sm:max-w-sm",
};

interface SheetContentProps extends SheetPrimitive.Popup.Props {
  side?: SheetSide;
  showCloseButton?: boolean;
}

function SheetContent({
  className,
  children,
  side = "right",
  showCloseButton = true,
  ...props
}: SheetContentProps) {
  const { actionsRef, open } = useSheetMotion();

  const handleExitComplete = React.useCallback(() => {
    actionsRef.current?.unmount();
  }, [actionsRef]);

  return (
    <AnimatePresence onExitComplete={handleExitComplete}>
      {open ? (
        <SheetPortal key="sheet-portal" keepMounted>
          <SheetOverlay />
          <SheetPrimitive.Popup
            data-slot="sheet-content"
            data-side={side}
            render={
              <m.div
                variants={SIDE_VARIANTS[side]}
                initial="hidden"
                animate="visible"
                exit="hidden"
                transition={{ duration: 0.2, ease: [0.4, 0, 0.2, 1] }}
              />
            }
            className={cn(
              "fixed z-50 flex flex-col gap-4 bg-(--canvas-soft) bg-clip-padding text-[13px] text-(--fg) shadow-[var(--shadow-overlay)] outline-none",
              SIDE_CLASSES[side],
              className
            )}
            {...props}
          >
            {children}
            {showCloseButton ? (
              <SheetPrimitive.Close
                data-slot="sheet-close"
                render={
                  <Button variant="ghost" className="absolute top-3 right-3" size="icon-sm" />
                }
              >
                <XIcon />
                <span className="sr-only">Close</span>
              </SheetPrimitive.Close>
            ) : null}
          </SheetPrimitive.Popup>
        </SheetPortal>
      ) : null}
    </AnimatePresence>
  );
}

function SheetHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-header"
      className={cn("flex flex-col gap-0.5 p-4", className)}
      {...props}
    />
  );
}

function SheetFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-footer"
      className={cn("mt-auto flex flex-col gap-2 p-4", className)}
      {...props}
    />
  );
}

function SheetTitle({ className, ...props }: SheetPrimitive.Title.Props) {
  return (
    <SheetPrimitive.Title
      data-slot="sheet-title"
      className={cn("text-[15px] font-[510] tracking-[-0.014em] text-(--fg-strong)", className)}
      {...props}
    />
  );
}

function SheetDescription({ className, ...props }: SheetPrimitive.Description.Props) {
  return (
    <SheetPrimitive.Description
      data-slot="sheet-description"
      className={cn("text-[13px] text-(--muted)", className)}
      {...props}
    />
  );
}

export {
  Sheet,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
};
