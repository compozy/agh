import defaultMdxComponents from "fumadocs-ui/mdx";
import { OperatorNote, RouteList, RouteRow } from "@/components/docs/mdx-blocks";
import { Mermaid } from "@/components/docs/mermaid";

export function getMDXComponents() {
  return {
    ...defaultMdxComponents,
    Mermaid,
    OperatorNote,
    RouteList,
    RouteRow,
  };
}
