import { loader } from "fumadocs-core/source";
import { runtime, protocol } from "@/.source/server";

export const runtimeDocs = loader({
  source: runtime.toFumadocsSource(),
  baseUrl: "/runtime",
});

export const protocolDocs = loader({
  source: protocol.toFumadocsSource(),
  baseUrl: "/protocol",
});
