import { createElement, type ReactElement } from "react";
import { loader } from "fumadocs-core/source";
import {
  Activity,
  Book,
  FileCode,
  FileText,
  Layers,
  Rocket,
  Settings,
  Terminal,
  type LucideIcon,
} from "lucide-react";
import { runtime, protocol } from "@/.source/server";

const iconMap: Record<string, LucideIcon> = {
  Activity,
  Book,
  FileCode,
  FileText,
  Layers,
  Rocket,
  Settings,
  Terminal,
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
