import { runtimeDocs, protocolDocs } from "@/lib/source";
import { createSearchAPI } from "fumadocs-core/search/server";

export const revalidate = false;

const server = createSearchAPI("advanced", {
  indexes: [
    ...runtimeDocs.getPages().map(page => ({
      title: page.data.title,
      description: page.data.description,
      structuredData: page.data.structuredData,
      id: page.url,
      url: page.url,
      tag: "Runtime",
    })),
    ...protocolDocs.getPages().map(page => ({
      title: page.data.title,
      description: page.data.description,
      structuredData: page.data.structuredData,
      id: page.url,
      url: page.url,
      tag: "Protocol",
    })),
  ],
});

export const { staticGET: GET } = server;
