import { CodeBlock, normalizeAghCodeLanguage } from "@agh/ui";
import { isValidElement, memo, type ReactElement, type ReactNode } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { cn } from "@/lib/utils";

function extractCodeLanguage(className?: string): string {
  const match = /language-([-\w]+)/.exec(className ?? "");
  return match?.[1]?.toLowerCase() ?? "";
}

type MarkdownCodeElement = ReactElement<
  {
    children?: ReactNode;
    className?: string;
  },
  "code"
>;

function getSingleChild(children: ReactNode): ReactNode {
  return Array.isArray(children) && children.length === 1 ? children[0] : children;
}

function isCodeElement(child: ReactNode): child is MarkdownCodeElement {
  return (
    isValidElement<{ children?: ReactNode; className?: string }>(child) && child.type === "code"
  );
}

function toCodeString(children: ReactNode): string {
  return String(children).replace(/\n$/, "");
}

function isFencedCode(children: ReactNode): boolean {
  const code = String(children);
  return code.endsWith("\n") || code.includes("\n");
}

function renderCodeBlock(code: string, rawLanguage?: string) {
  const normalizedLanguage = normalizeAghCodeLanguage(rawLanguage);
  return (
    <CodeBlock
      code={code}
      language={rawLanguage}
      caption={rawLanguage ? (normalizedLanguage ?? rawLanguage) : undefined}
      showPrompt={false}
      copyable
      className="my-2"
    />
  );
}

export interface MessageMarkdownProps {
  content: string;
}

export const MessageMarkdown = memo(
  function MessageMarkdown({ content }: MessageMarkdownProps) {
    return (
      <Markdown
        remarkPlugins={[remarkGfm]}
        components={{
          code({ className, children, ...props }) {
            const rawLanguage = extractCodeLanguage(className);

            if (rawLanguage || isFencedCode(children)) {
              return renderCodeBlock(toCodeString(children), rawLanguage || undefined);
            }

            return (
              <code
                className={cn(
                  "rounded-md bg-elevated px-1.5 py-0.5",
                  "text-small-body text-fg",
                  className
                )}
                {...props}
              >
                {children}
              </code>
            );
          },
          a({ children, href, ...props }) {
            return (
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent underline underline-offset-2 hover:opacity-80"
                {...props}
              >
                {children}
              </a>
            );
          },
          pre({ children }) {
            const child = getSingleChild(children);
            if (isCodeElement(child)) {
              const rawLanguage = extractCodeLanguage(child.props.className);
              return renderCodeBlock(toCodeString(child.props.children), rawLanguage || undefined);
            }

            return <>{children}</>;
          },
        }}
      >
        {content}
      </Markdown>
    );
  },
  (prev, next) => prev.content === next.content
);
