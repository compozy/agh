import { cn } from "@agh/ui";
import { isValidElement, type ComponentProps, type ReactNode } from "react";

function textFromNode(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(textFromNode).join("");
  if (isValidElement<{ children?: ReactNode }>(node)) return textFromNode(node.props.children);
  return "";
}

function slugifyHeading(heading: string): string {
  return heading
    .replace(/<[^>]+>/g, "")
    .replace(/`([^`]+)`/g, "$1")
    .toLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .replace(/\s+/g, "-");
}

function resolveHeadingID(id: string | undefined, children: ReactNode): string | undefined {
  if (id && id !== "$undefined") return id;
  const generated = slugifyHeading(textFromNode(children));
  return generated.length > 0 ? generated : undefined;
}

export function ProseH2({ id, children, className, ...props }: ComponentProps<"h2">) {
  return (
    <h2
      id={resolveHeadingID(id, children)}
      {...props}
      className={cn(
        "mt-16 border-t border-(--color-divider) pt-4 font-sans text-site-doc-heading font-semibold leading-tight tracking-tight text-(--color-text-primary)",
        className
      )}
    >
      {children}
    </h2>
  );
}

export function ProseH3({ id, children, className, ...props }: ComponentProps<"h3">) {
  return (
    <h3
      id={resolveHeadingID(id, children)}
      {...props}
      className={cn(
        "mt-10 font-sans text-site-subheading font-semibold leading-tight tracking-tight text-(--color-text-primary)",
        className
      )}
    >
      {children}
    </h3>
  );
}

export function ProseParagraph({ children, className, ...props }: ComponentProps<"p">) {
  return (
    <p
      {...props}
      className={cn(
        "mt-5 max-w-[72ch] font-sans text-base leading-doc-body text-(--color-text-secondary)",
        className
      )}
    >
      {children}
    </p>
  );
}

export function ProseList({ children, className, ...props }: ComponentProps<"ul">) {
  return (
    <ul
      {...props}
      className={cn(
        "mt-5 ml-5 max-w-[72ch] list-disc text-base leading-7 text-(--color-text-secondary) marker:text-(--color-text-tertiary) [&>li+li]:mt-2",
        className
      )}
    >
      {children}
    </ul>
  );
}

export function ProseOrderedList({ children, className, ...props }: ComponentProps<"ol">) {
  return (
    <ol
      {...props}
      className={cn(
        "mt-5 ml-5 max-w-[72ch] list-decimal text-base leading-7 text-(--color-text-secondary) marker:text-(--color-text-tertiary) [&>li+li]:mt-2",
        className
      )}
    >
      {children}
    </ol>
  );
}

export function PullQuote({ children, className, ...props }: ComponentProps<"blockquote">) {
  return (
    <blockquote
      {...props}
      className={cn(
        "mt-9 mb-3 max-w-[40ch] border-l-2 border-accent pl-6 font-display text-site-quote font-normal leading-tight tracking-tight text-(--color-text-primary)",
        className
      )}
    >
      {children}
    </blockquote>
  );
}

export interface MonoProps extends ComponentProps<"code"> {
  "data-language"?: string;
  "data-theme"?: string;
}

export function Mono({ children, className, ...props }: MonoProps) {
  const isHighlightedBlock = Boolean(props["data-language"] || props["data-theme"]);

  if (isHighlightedBlock) {
    return (
      <code {...props} className={cn("font-mono text-inherit", className)}>
        {children}
      </code>
    );
  }

  return (
    <code
      {...props}
      className={cn(
        "rounded-md border border-(--color-divider) bg-(--color-surface-elevated) px-1.5 py-0.5 font-mono text-inline-code text-(--color-text-primary)",
        className
      )}
    >
      {children}
    </code>
  );
}

export interface CalloutProps {
  tone?: "accent" | "success" | "danger" | "warning" | "info";
  eyebrow?: string;
  children?: ReactNode;
  className?: string;
}

const calloutBorderClass: Record<NonNullable<CalloutProps["tone"]>, string> = {
  accent: "border-l-accent",
  success: "border-l-success",
  danger: "border-l-(--color-danger)",
  warning: "border-l-(--color-warning)",
  info: "border-l-(--color-info)",
};

const calloutEyebrowToneClass: Record<NonNullable<CalloutProps["tone"]>, string> = {
  accent: "text-accent",
  success: "text-success",
  danger: "text-(--color-danger)",
  warning: "text-(--color-warning)",
  info: "text-(--color-info)",
};

export function Callout({ tone = "accent", eyebrow, children, className }: CalloutProps) {
  return (
    <aside
      role="note"
      className={cn(
        "mt-7 rounded-xl border border-(--color-divider) border-l-4 bg-(--color-surface) p-5",
        calloutBorderClass[tone],
        className
      )}
    >
      {eyebrow && (
        <p
          className={cn(
            "font-mono text-eyebrow font-semibold uppercase tracking-badge",
            calloutEyebrowToneClass[tone]
          )}
        >
          {eyebrow}
        </p>
      )}
      <div className="mt-3 font-sans text-item-title leading-7 text-(--color-text-primary)">
        {children}
      </div>
    </aside>
  );
}

export interface WireCardRow {
  label: string;
  value: string;
  tone?: "neutral" | "accent" | "success" | "danger" | "warning" | "info";
}

const wireValueToneClass: Record<NonNullable<WireCardRow["tone"]>, string> = {
  neutral: "text-(--color-text-primary)",
  accent: "text-accent",
  success: "text-success",
  danger: "text-(--color-danger)",
  warning: "text-(--color-warning)",
  info: "text-(--color-info)",
};

export interface WireCardProps {
  kind: string;
  rows: WireCardRow[];
  protocol?: string;
}

export function WireCard({ kind, rows, protocol = "v0" }: WireCardProps) {
  return (
    <div className="mt-7 max-w-[520px] overflow-hidden rounded-md border border-(--color-divider) bg-(--color-surface)">
      <div className="border-b border-(--color-divider) bg-(--color-canvas-deep) px-3 py-1.5 font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
        kind={kind} · {protocol}
      </div>
      <div className="grid grid-cols-[80px_1fr] gap-x-3 gap-y-1 px-3 py-3 font-mono text-eyebrow leading-7">
        {rows.map(row => (
          <div key={row.label} className="contents">
            <span className="text-(--color-text-tertiary)">{row.label}</span>
            <span className={wireValueToneClass[row.tone ?? "neutral"]}>{row.value}</span>
          </div>
        ))}
      </div>
      <div className="flex items-center gap-3 border-t border-(--color-divider) bg-(--color-canvas-deep) px-3 py-1.5">
        <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
          Inspect →
        </span>
        <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
          Replay
        </span>
      </div>
    </div>
  );
}
