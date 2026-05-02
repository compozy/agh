import type { ComponentType } from "react";
import { CodeBlock } from "./code-block";
import { KindChip } from "./kind-chip";
import { MonoBadge } from "./mono-badge";
import {
  Callout,
  Mono,
  ProseH2,
  ProseH3,
  ProseList,
  ProseOrderedList,
  ProseParagraph,
  PullQuote,
  WireCard,
} from "./prose";

export type MdxComponents = Record<string, ComponentType<Record<string, unknown>>>;

export const mdxComponents = {
  h2: ProseH2,
  h3: ProseH3,
  p: ProseParagraph,
  ul: ProseList,
  ol: ProseOrderedList,
  blockquote: PullQuote,
  code: Mono,
  pre: CodeBlock,
  Callout,
  WireCard,
  KindChip,
  MonoBadge,
} as unknown as MdxComponents;
