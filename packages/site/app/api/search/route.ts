import { runtimeDocs, protocolDocs } from "@/lib/source";
import { createSearchAPI } from "fumadocs-core/search/server";

export const revalidate = false;

type SearchPage = {
  url: string;
};

function byURL(left: SearchPage, right: SearchPage): number {
  return left.url.localeCompare(right.url);
}

function sortedByURL<Page extends SearchPage>(pages: Page[]): Page[] {
  return [...pages].sort(byURL);
}

const server = createSearchAPI("advanced", {
  indexes: [
    ...sortedByURL(runtimeDocs.getPages()).map(page => ({
      title: page.data.title,
      description: page.data.description,
      structuredData: page.data.structuredData,
      id: page.url,
      url: page.url,
      tag: "Runtime",
    })),
    ...sortedByURL(protocolDocs.getPages()).map(page => ({
      title: page.data.title,
      description: page.data.description,
      structuredData: page.data.structuredData,
      id: page.url,
      url: page.url,
      tag: "AGH Network",
    })),
  ],
});

export const { staticGET: GET } = server;
