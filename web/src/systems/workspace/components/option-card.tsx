import * as React from "react";

import { cn, Section } from "@agh/ui";

import { useOptionCardSlot } from "../hooks/use-option-card-slot";
import {
  OptionCardContext,
  type OptionCardContextValue,
  type OptionCardDensity,
} from "./option-card-context";

type OptionCardTone = "neutral" | "accent";

interface OptionCardHeaderProps {
  eyebrow?: React.ReactNode;
  right?: React.ReactNode;
}

function OptionCardHeaderSentinel(_: OptionCardHeaderProps): null {
  useOptionCardSlot("Header");
  return null;
}

function isHeaderElement(node: React.ReactNode): node is React.ReactElement<OptionCardHeaderProps> {
  return React.isValidElement(node) && node.type === OptionCardHeaderSentinel;
}

interface OptionCardProps extends React.ComponentProps<typeof Section> {
  density?: OptionCardDensity;
}

function OptionCardRoot({
  className,
  density = "comfortable",
  children,
  ...props
}: OptionCardProps) {
  const ctx = React.useMemo<OptionCardContextValue>(() => ({ density }), [density]);

  let headerEyebrow: React.ReactNode;
  let headerRight: React.ReactNode;
  const body: React.ReactNode[] = [];

  React.Children.forEach(children, child => {
    if (isHeaderElement(child)) {
      headerEyebrow = child.props.eyebrow;
      headerRight = child.props.right;
      return;
    }
    body.push(child);
  });

  return (
    <OptionCardContext.Provider value={ctx}>
      <Section
        {...props}
        data-slot="option-card"
        data-density={density}
        label={headerEyebrow}
        right={headerRight}
        className={cn(
          "rounded-2xl border border-[color:var(--line)] bg-[color:var(--canvas-soft)]",
          density === "comfortable" ? "p-5" : "p-4",
          className
        )}
      >
        {body}
      </Section>
    </OptionCardContext.Provider>
  );
}

function OptionCardBody({ className, children, ...props }: React.ComponentProps<"div">) {
  useOptionCardSlot("Body");
  return (
    <div
      data-slot="option-card-body"
      className={cn("flex items-start gap-3", className)}
      {...props}
    >
      {children}
    </div>
  );
}

interface OptionCardIconProps extends React.ComponentProps<"span"> {
  tone?: OptionCardTone;
}

function OptionCardIcon({ className, tone = "neutral", children, ...props }: OptionCardIconProps) {
  useOptionCardSlot("Icon");
  return (
    <span
      data-slot="option-card-icon"
      data-tone={tone}
      aria-hidden="true"
      className={cn(
        "inline-flex size-10 shrink-0 items-center justify-center rounded-2xl border border-[color:var(--line)] bg-[color:var(--canvas-soft)]",
        tone === "accent" ? "text-[color:var(--accent)]" : "text-[color:var(--fg)]",
        className
      )}
      {...props}
    >
      {children}
    </span>
  );
}

function OptionCardContent({ className, children, ...props }: React.ComponentProps<"div">) {
  useOptionCardSlot("Content");
  return (
    <div data-slot="option-card-content" className={cn("min-w-0 flex-1", className)} {...props}>
      {children}
    </div>
  );
}

function OptionCardTitle({ className, children, ...props }: React.ComponentProps<"p">) {
  useOptionCardSlot("Title");
  return (
    <p
      data-slot="option-card-title"
      className={cn("text-sm font-semibold text-[color:var(--fg)]", className)}
      {...props}
    >
      {children}
    </p>
  );
}

function OptionCardDescription({ className, children, ...props }: React.ComponentProps<"p">) {
  useOptionCardSlot("Description");
  return (
    <p
      data-slot="option-card-description"
      className={cn("mt-1 text-sm leading-6 text-[color:var(--muted)]", className)}
      {...props}
    >
      {children}
    </p>
  );
}

function OptionCardMeta({ className, children, ...props }: React.ComponentProps<"p">) {
  useOptionCardSlot("Meta");
  return (
    <p
      data-slot="option-card-meta"
      className={cn("mt-3 truncate font-mono text-eyebrow text-[color:var(--subtle)]", className)}
      {...props}
    >
      {children}
    </p>
  );
}

function OptionCardAction({ className, children, ...props }: React.ComponentProps<"div">) {
  useOptionCardSlot("Action");
  return (
    <div data-slot="option-card-action" className={className} {...props}>
      {children}
    </div>
  );
}

const OptionCard = Object.assign(OptionCardRoot, {
  Header: OptionCardHeaderSentinel,
  Body: OptionCardBody,
  Icon: OptionCardIcon,
  Content: OptionCardContent,
  Title: OptionCardTitle,
  Description: OptionCardDescription,
  Meta: OptionCardMeta,
  Action: OptionCardAction,
});

export { OptionCard };
export type {
  OptionCardDensity,
  OptionCardHeaderProps,
  OptionCardIconProps,
  OptionCardProps,
  OptionCardTone,
};
