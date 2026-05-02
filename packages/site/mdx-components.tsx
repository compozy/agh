import defaultMdxComponents from "fumadocs-ui/mdx";
import { createAPIPage } from "fumadocs-openapi/ui";
import { defaultShikiFactory } from "fumadocs-core/highlight/shiki/full";
import { openapi } from "@/lib/openapi";
import {
  GuideCard,
  GuideGrid,
  OperatorNote,
  RouteList,
  RouteRow,
  Workflow,
  WorkflowStep,
} from "@/components/docs/mdx-blocks";
import { Mermaid } from "@/components/docs/mermaid";

const APIPage = createAPIPage(openapi, {
  shiki: defaultShikiFactory,
  shikiOptions: {
    themes: { light: "vitesse-light", dark: "vitesse-dark" },
  },
  playground: { enabled: false },
});

export function getMDXComponents() {
  return {
    ...defaultMdxComponents,
    APIPage,
    Mermaid,
    GuideCard,
    GuideGrid,
    OperatorNote,
    RouteList,
    RouteRow,
    Workflow,
    WorkflowStep,
  };
}
