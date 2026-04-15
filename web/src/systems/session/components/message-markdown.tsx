import { memo } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { PrismAsyncLight as SyntaxHighlighter } from "react-syntax-highlighter";
import bash from "react-syntax-highlighter/dist/esm/languages/prism/bash";
import diff from "react-syntax-highlighter/dist/esm/languages/prism/diff";
import go from "react-syntax-highlighter/dist/esm/languages/prism/go";
import javascript from "react-syntax-highlighter/dist/esm/languages/prism/javascript";
import json from "react-syntax-highlighter/dist/esm/languages/prism/json";
import jsx from "react-syntax-highlighter/dist/esm/languages/prism/jsx";
import python from "react-syntax-highlighter/dist/esm/languages/prism/python";
import sql from "react-syntax-highlighter/dist/esm/languages/prism/sql";
import tsx from "react-syntax-highlighter/dist/esm/languages/prism/tsx";
import typescript from "react-syntax-highlighter/dist/esm/languages/prism/typescript";
import yaml from "react-syntax-highlighter/dist/esm/languages/prism/yaml";
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism";

import { cn } from "@/lib/utils";
import { CopyButton } from "./copy-button";

SyntaxHighlighter.registerLanguage("bash", bash);
SyntaxHighlighter.registerLanguage("diff", diff);
SyntaxHighlighter.registerLanguage("go", go);
SyntaxHighlighter.registerLanguage("javascript", javascript);
SyntaxHighlighter.registerLanguage("json", json);
SyntaxHighlighter.registerLanguage("jsx", jsx);
SyntaxHighlighter.registerLanguage("python", python);
SyntaxHighlighter.registerLanguage("sql", sql);
SyntaxHighlighter.registerLanguage("tsx", tsx);
SyntaxHighlighter.registerLanguage("typescript", typescript);
SyntaxHighlighter.registerLanguage("yaml", yaml);

const CODE_LANGUAGE_ALIASES: Record<string, string> = {
  js: "javascript",
  shell: "bash",
  sh: "bash",
  text: "",
  ts: "typescript",
  yml: "yaml",
};

const SUPPORTED_CODE_LANGUAGES = new Set([
  "bash",
  "diff",
  "go",
  "javascript",
  "json",
  "jsx",
  "python",
  "sql",
  "tsx",
  "typescript",
  "yaml",
]);

function normalizeCodeLanguage(className?: string): string {
  const match = /language-([-\w]+)/.exec(className ?? "");
  if (!match) return "";

  const rawLanguage = match[1].toLowerCase();
  const normalizedLanguage = CODE_LANGUAGE_ALIASES[rawLanguage] ?? rawLanguage;

  return SUPPORTED_CODE_LANGUAGES.has(normalizedLanguage) ? normalizedLanguage : "";
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
            const language = normalizeCodeLanguage(className);
            const codeString = String(children).replace(/\n$/, "");

            if (language) {
              return (
                <div className="group/codeblock relative">
                  <SyntaxHighlighter
                    style={oneDark}
                    language={language}
                    PreTag="div"
                    customStyle={{
                      margin: 0,
                      borderRadius: "0.5rem",
                      fontSize: "0.8125rem",
                    }}
                  >
                    {codeString}
                  </SyntaxHighlighter>
                  <CopyButton
                    text={codeString}
                    ariaLabel="Copy code"
                    className={cn(
                      "absolute top-2 right-2 rounded-md p-1.5",
                      "border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)]",
                      "opacity-0 transition-opacity duration-200 group-hover/codeblock:opacity-100",
                      "text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-primary)]"
                    )}
                  />
                </div>
              );
            }

            return (
              <code
                className={cn(
                  "rounded-md bg-[color:var(--color-surface-elevated)] px-1.5 py-0.5",
                  "text-[0.8125rem] text-[color:var(--color-text-primary)]",
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
                className="text-[color:var(--color-accent)] underline underline-offset-2 hover:opacity-80"
                {...props}
              >
                {children}
              </a>
            );
          },
          pre({ children }) {
            return <div className="my-2 overflow-x-auto">{children}</div>;
          },
        }}
      >
        {content}
      </Markdown>
    );
  },
  (prev, next) => prev.content === next.content
);
