"use client";

import * as React from "react";
import type { Components } from "streamdown";

import { cn } from "../../lib/utils";

type MdProps<T extends keyof React.JSX.IntrinsicElements> = React.ComponentPropsWithoutRef<T> & {
  node?: unknown;
};

const INLINE_CODE_CLASS =
  "rounded-xs bg-surface-glaze px-1 py-px font-mono text-form-input text-fg-strong";

function MarkdownAnchor({ className, ...props }: MdProps<"a">) {
  return (
    <a
      className={cn(
        "text-fg-strong underline underline-offset-2 decoration-line hover:decoration-fg-strong",
        className
      )}
      {...props}
    />
  );
}

function MarkdownStrong({ className, ...props }: MdProps<"strong">) {
  return <strong className={cn("font-medium text-fg-strong", className)} {...props} />;
}

function MarkdownBlockquote({ className, ...props }: MdProps<"blockquote">) {
  return (
    <blockquote
      className={cn("border-l-2 border-line-strong pl-3 text-muted", className)}
      {...props}
    />
  );
}

function MarkdownHr({ className, ...props }: MdProps<"hr">) {
  return <hr className={cn("my-4 border-line", className)} {...props} />;
}

function MarkdownParagraph({ className, ...props }: MdProps<"p">) {
  return <p className={cn(className)} {...props} />;
}

function MarkdownH1({ className, ...props }: MdProps<"h1">) {
  return (
    <h1 className={cn("text-item-title font-semibold text-fg-strong", className)} {...props} />
  );
}

function MarkdownH2({ className, ...props }: MdProps<"h2">) {
  return (
    <h2 className={cn("text-card-title font-semibold text-fg-strong", className)} {...props} />
  );
}

function MarkdownH3({ className, ...props }: MdProps<"h3">) {
  return (
    <h3 className={cn("text-small-body font-semibold text-fg-strong", className)} {...props} />
  );
}

function MarkdownH4({ className, ...props }: MdProps<"h4">) {
  return <h4 className={cn("text-small-body font-medium text-fg", className)} {...props} />;
}

function MarkdownH5({ className, ...props }: MdProps<"h5">) {
  return <h5 className={cn("text-small-body font-medium text-muted", className)} {...props} />;
}

function MarkdownH6({ className, ...props }: MdProps<"h6">) {
  return <h6 className={cn("text-small-body font-medium text-muted", className)} {...props} />;
}

function MarkdownUl({ className, ...props }: MdProps<"ul">) {
  return (
    <ul
      className={cn("list-disc pl-5 [&_ul]:list-[circle] [&_ul_ul]:list-[square]", className)}
      {...props}
    />
  );
}

function MarkdownOl({ className, ...props }: MdProps<"ol">) {
  return <ol className={cn("list-decimal pl-5", className)} {...props} />;
}

function MarkdownLi({ className, ...props }: MdProps<"li">) {
  return <li className={cn("my-0.5", className)} {...props} />;
}

function MarkdownTable({ className, ...props }: MdProps<"table">) {
  return (
    <div className="w-full overflow-x-auto">
      <table className={cn("w-full border-collapse", className)} {...props} />
    </div>
  );
}

function MarkdownTh({ className, ...props }: MdProps<"th">) {
  return (
    <th
      className={cn(
        "border-b border-line-strong px-2 py-1.5 text-left text-form-label font-medium text-fg-strong",
        className
      )}
      {...props}
    />
  );
}

function MarkdownTd({ className, ...props }: MdProps<"td">) {
  return (
    <td className={cn("border-b border-line px-2 py-1.5 text-form-input", className)} {...props} />
  );
}

type CodeProps = MdProps<"code"> & {
  inline?: boolean;
  "data-block"?: unknown;
};

function MarkdownInlineCode({
  className,
  children,
  inline: _inline,
  "data-block": dataBlock,
  node: _node,
  ...props
}: CodeProps) {
  const isBlock =
    dataBlock !== undefined ||
    (typeof className === "string" && className.includes("language-")) ||
    (typeof children === "string" && children.includes("\n"));
  if (isBlock) {
    return (
      <code
        className={className}
        {...(dataBlock !== undefined
          ? { "data-block": dataBlock === true ? true : dataBlock }
          : {})}
        {...props}
      >
        {children}
      </code>
    );
  }
  return (
    <code className={cn(INLINE_CODE_CLASS, className)} {...props}>
      {children}
    </code>
  );
}

/**
 * Explicit Streamdown component map for AGH markdown grammar. Merged on top of
 * STREAMDOWN_SAFE_CONFIG so security overrides (SafeImage) stay authoritative.
 */
const MARKDOWN_PROSE_COMPONENTS: Partial<Components> = {
  a: MarkdownAnchor,
  strong: MarkdownStrong,
  blockquote: MarkdownBlockquote,
  hr: MarkdownHr,
  p: MarkdownParagraph,
  h1: MarkdownH1,
  h2: MarkdownH2,
  h3: MarkdownH3,
  h4: MarkdownH4,
  h5: MarkdownH5,
  h6: MarkdownH6,
  ul: MarkdownUl,
  ol: MarkdownOl,
  li: MarkdownLi,
  table: MarkdownTable,
  th: MarkdownTh,
  td: MarkdownTd,
  code: MarkdownInlineCode,
  inlineCode: MarkdownInlineCode,
};

export { INLINE_CODE_CLASS, MARKDOWN_PROSE_COMPONENTS };
