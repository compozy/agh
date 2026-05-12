import { AGH_CODE_THEMES } from "@agh/ui/lib/code-theme";
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
    themes: { light: AGH_CODE_THEMES.light, dark: AGH_CODE_THEMES.dark },
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
