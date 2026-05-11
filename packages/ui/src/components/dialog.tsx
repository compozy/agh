"use client";

import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import * as React from "react";

import { cn } from "../lib/utils";
import { Button } from "./button";
import { useInitialState } from "./use-initial-state";

type DialogActionsRef = React.RefObject<DialogPrimitive.Root.Actions | null>;

interface DialogMotionContextValue {
  actionsRef: DialogActionsRef;
  open: boolean;
}

const DialogMotionContext = React.createContext<DialogMotionContextValue | null>(null);

function useDialogMotion(): DialogMotionContextValue {
  const ctx = React.use(DialogMotionContext);
  if (!ctx) {
    throw new Error("Dialog.* components must be used inside <Dialog>.");
  }
  return ctx;
}

type DialogRootProps = DialogPrimitive.Root.Props;

function Dialog({
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  children,
  ...props
}: DialogRootProps) {
  const actionsRef = React.useRef<DialogPrimitive.Root.Actions | null>(null);
  const [uncontrolledOpen, setUncontrolledOpen] = useInitialState(defaultOpen);
  const isControlled = controlledOpen !== undefined;
  const open = isControlled ? Boolean(controlledOpen) : uncontrolledOpen;

  const handleOpenChange: NonNullable<DialogRootProps["onOpenChange"]> = (next, details) => {
    if (!isControlled) setUncontrolledOpen(next);
    onOpenChange?.(next, details);
  };

  const value = React.useMemo<DialogMotionContextValue>(() => ({ actionsRef, open }), [open]);

  return (
    <DialogPrimitive.Root
      data-slot="dialog"
      actionsRef={actionsRef}
      open={open}
      defaultOpen={defaultOpen}
      onOpenChange={handleOpenChange}
      {...props}
    >
      <DialogMotionContext.Provider value={value}>
        {children as React.ReactNode}
      </DialogMotionContext.Provider>
    </DialogPrimitive.Root>
  );
}

function DialogTrigger({ ...props }: DialogPrimitive.Trigger.Props) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}

function DialogPortal({ ...props }: DialogPrimitive.Portal.Props) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />;
}

function DialogClose({ ...props }: DialogPrimitive.Close.Props) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}

function DialogOverlay({ className, style, ...props }: DialogPrimitive.Backdrop.Props) {
  const overlayRender = React.useMemo(
    () => (
      <m.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2, ease: [0.2, 0, 0, 1] }}
      />
    ),
    []
  );

  return (
    <DialogPrimitive.Backdrop
      data-slot="dialog-overlay"
      render={overlayRender}
      className={cn("fixed inset-0 isolate z-50 bg-(--overlay-scrim)", className)}
      style={{ backdropFilter: "blur(var(--overlay-blur))", ...style }}
      {...props}
    />
  );
}

type DialogChromeVariant = "default" | "ruled";

const DIALOG_CONTENT_BASE =
  "fixed top-1/2 left-1/2 z-50 grid w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 rounded-lg bg-(--canvas-soft) text-[13px] text-(--fg) shadow-[var(--shadow-overlay)] outline-none sm:max-w-sm";
const DIALOG_CONTENT_FRAMED = "gap-4 p-4";
const DIALOG_CONTENT_UNFRAMED = "gap-0 p-0";

const DIALOG_HEADER_DEFAULT = "flex flex-col gap-2";
const DIALOG_HEADER_RULED =
  "flex flex-col gap-2 border-b border-(--line) bg-(--canvas-soft) px-5 py-4";

const DIALOG_FOOTER_DEFAULT =
  "-mx-4 -mb-4 flex flex-col-reverse gap-2 rounded-b-lg border-t border-(--line) bg-(--canvas-tint) p-4 sm:flex-row sm:justify-end";
const DIALOG_FOOTER_RULED =
  "flex flex-col-reverse gap-2 border-t border-(--line) bg-(--canvas-tint) px-5 py-3 sm:flex-row sm:justify-end";

interface DialogContentProps extends DialogPrimitive.Popup.Props {
  showCloseButton?: boolean;
  /**
   * When `true`, drops the default `gap-4 p-4` chrome so callers can compose
   * flush headers, bodies, and footers (typically alongside `DialogHeader`/
   * `DialogFooter` `variant="ruled"`).
   */
  unframed?: boolean;
}

function DialogContent({
  className,
  children,
  showCloseButton = true,
  unframed = false,
  ...props
}: DialogContentProps) {
  const { actionsRef, open } = useDialogMotion();

  const handleExitComplete = React.useCallback(() => {
    actionsRef.current?.unmount();
  }, [actionsRef]);
  const popupRender = React.useMemo(
    () => (
      <m.div
        initial={{ opacity: 0, scale: 0.97 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.97 }}
        transition={{ duration: 0.2, ease: [0.2, 0, 0, 1] }}
      />
    ),
    []
  );

  return (
    <AnimatePresence onExitComplete={handleExitComplete}>
      {open ? (
        <DialogPortal key="dialog-portal" keepMounted>
          <DialogOverlay />
          <DialogPrimitive.Popup
            data-slot="dialog-content"
            data-frame={unframed ? "unframed" : "framed"}
            render={popupRender}
            className={cn(
              DIALOG_CONTENT_BASE,
              unframed ? DIALOG_CONTENT_UNFRAMED : DIALOG_CONTENT_FRAMED,
              unframed && "overflow-hidden",
              className
            )}
            {...props}
          >
            {children}
            {showCloseButton ? (
              <DialogPrimitive.Close
                data-slot="dialog-close"
                render={
                  <Button variant="ghost" className="absolute top-2 right-2" size="icon-sm" />
                }
              >
                <XIcon />
                <span className="sr-only">Close</span>
              </DialogPrimitive.Close>
            ) : null}
          </DialogPrimitive.Popup>
        </DialogPortal>
      ) : null}
    </AnimatePresence>
  );
}

interface DialogHeaderProps extends React.ComponentProps<"div"> {
  variant?: DialogChromeVariant;
}

function DialogHeader({ className, variant = "default", ...props }: DialogHeaderProps) {
  return (
    <div
      data-slot="dialog-header"
      data-variant={variant}
      className={cn(variant === "ruled" ? DIALOG_HEADER_RULED : DIALOG_HEADER_DEFAULT, className)}
      {...props}
    />
  );
}

interface DialogFooterProps extends React.ComponentProps<"div"> {
  showCloseButton?: boolean;
  variant?: DialogChromeVariant;
}

function DialogFooter({
  className,
  showCloseButton = false,
  variant = "default",
  children,
  ...props
}: DialogFooterProps) {
  return (
    <div
      data-slot="dialog-footer"
      data-variant={variant}
      className={cn(variant === "ruled" ? DIALOG_FOOTER_RULED : DIALOG_FOOTER_DEFAULT, className)}
      {...props}
    >
      {children}
      {showCloseButton ? (
        <DialogClose render={<Button variant="outline" />}>Close</DialogClose>
      ) : null}
    </div>
  );
}

function DialogTitle({ className, ...props }: DialogPrimitive.Title.Props) {
  return (
    <DialogPrimitive.Title
      data-slot="dialog-title"
      className={cn(
        "text-[15px] leading-none font-medium tracking-[-0.014em] text-(--fg-strong)",
        className
      )}
      {...props}
    />
  );
}

function DialogDescription({ className, ...props }: DialogPrimitive.Description.Props) {
  return (
    <DialogPrimitive.Description
      data-slot="dialog-description"
      className={cn(
        "text-[13px] text-(--muted) *:[a]:underline *:[a]:underline-offset-3 *:[a]:hover:text-(--fg-strong)",
        className
      )}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
};
export type { DialogChromeVariant, DialogContentProps, DialogFooterProps, DialogHeaderProps };
