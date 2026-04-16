import defaultMdxComponents from "fumadocs-ui/mdx";
import { OperatorNote, RouteList, RouteRow } from "@/components/docs/mdx-blocks";

export function getMDXComponents() {
  return {
    ...defaultMdxComponents,
    OperatorNote,
    RouteList,
    RouteRow,
  };
}
