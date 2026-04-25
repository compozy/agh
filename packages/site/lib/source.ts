import { createElement, type ReactElement } from "react";
import { loader } from "fumadocs-core/source";
import { getLayoutTabs } from "fumadocs-ui/layouts/shared";
import {
  Activity,
  Brain,
  Book,
  FileCode,
  FileText,
  FolderTree,
  Layers,
  Network,
  Plug,
  Rocket,
  Settings,
  Terminal,
  Waypoints,
  type LucideIcon,
} from "lucide-react";
import { runtime, protocol } from "@/.source/server";
import { createRuntimeLayoutTree } from "./runtime-navigation";

const iconMap: Record<string, LucideIcon> = {
  Activity,
  Brain,
  Book,
  FileCode,
  FileText,
  FolderTree,
  Layers,
  Network,
  Plug,
  Rocket,
  Settings,
  Terminal,
  Waypoints,
};

function iconResolver(icon?: string): ReactElement | undefined {
  if (!icon) return undefined;
  const Icon = iconMap[icon];
  return Icon ? createElement(Icon) : undefined;
}

export const runtimeDocs = loader({
  source: runtime.toFumadocsSource(),
  baseUrl: "/runtime",
  icon: iconResolver,
});

export const protocolDocs = loader({
  source: protocol.toFumadocsSource(),
  baseUrl: "/protocol",
  icon: iconResolver,
});

export const runtimeLayoutTree = createRuntimeLayoutTree(runtimeDocs.pageTree);

export const runtimeTabs = getLayoutTabs(runtimeDocs.pageTree);
