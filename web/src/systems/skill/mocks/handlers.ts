import { http, HttpResponse, type HttpHandler } from "msw";

import {
  skillActionFixture,
  skillContentFixtures,
  skillFixtures,
  skillMarketplaceDetailFixture,
  skillMarketplaceInstallFixture,
  skillMarketplaceListingBySlug,
  skillMarketplaceListingFixtures,
  skillMarketplaceRemoveFixture,
  skillMarketplaceUpdateFixtures,
} from "./fixtures";

const skillByName = new Map(skillFixtures.map(skill => [skill.name, skill]));

export const handlers: HttpHandler[] = [
  http.get("/api/skills", () => HttpResponse.json({ skills: skillFixtures })),
  http.get("/api/skills/marketplace/search", ({ request }) => {
    const url = new URL(request.url);
    const query = url.searchParams.get("query")?.trim() ?? "";
    if (query === "") {
      return HttpResponse.json({ error: "marketplace search query is required" }, { status: 400 });
    }

    const needle = query.toLowerCase();
    const matches = skillMarketplaceListingFixtures.filter(listing => {
      const haystack = [listing.name, listing.slug, listing.author, listing.description]
        .join(" ")
        .toLowerCase();
      return haystack.includes(needle);
    });

    return HttpResponse.json({ skills: matches });
  }),
  http.get("/api/skills/marketplace/info", ({ request }) => {
    const url = new URL(request.url);
    const slug = url.searchParams.get("slug")?.trim() ?? "";
    if (slug === "") {
      return HttpResponse.json({ error: "slug is required" }, { status: 400 });
    }

    const listing = skillMarketplaceListingBySlug.get(slug);
    if (!listing) {
      return HttpResponse.json({ error: `marketplace skill not found: ${slug}` }, { status: 404 });
    }

    return HttpResponse.json({
      skill: {
        ...skillMarketplaceDetailFixture,
        ...listing,
      },
    });
  }),
  http.post("/api/skills/marketplace/install", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      slug?: string;
      version?: string;
    };
    if (!body.slug) {
      return HttpResponse.json({ error: "slug is required" }, { status: 400 });
    }
    const listing = skillMarketplaceListingBySlug.get(body.slug);
    if (!listing) {
      return HttpResponse.json(
        { error: `marketplace skill not found: ${body.slug}` },
        { status: 404 }
      );
    }
    return HttpResponse.json({
      skill: {
        ...skillMarketplaceInstallFixture,
        name: listing.name,
        slug: listing.slug,
        version: body.version ?? listing.version ?? "0.0.0",
      },
    });
  }),
  http.post("/api/skills/marketplace/update", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      name?: string;
      all?: boolean;
      check_only?: boolean;
    };
    if (!body.name && !body.all) {
      return HttpResponse.json({ error: "name or all is required" }, { status: 400 });
    }
    return HttpResponse.json({ skills: skillMarketplaceUpdateFixtures });
  }),
  http.delete("/api/skills/marketplace/:name", ({ params }) => {
    const name = String(params.name);
    return HttpResponse.json({
      skill: { ...skillMarketplaceRemoveFixture, name },
    });
  }),
  http.get("/api/skills/:name", ({ params }) => {
    const name = String(params.name);
    const skill = skillByName.get(name);

    if (!skill) {
      return HttpResponse.json({ error: `Skill not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ skill });
  }),
  http.get("/api/skills/:name/content", ({ params }) => {
    const name = String(params.name);
    const content = skillContentFixtures[name];

    if (!content) {
      return HttpResponse.json({ error: `Skill not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ content });
  }),
  http.post("/api/skills/:name/enable", ({ params }) => {
    if (!skillByName.has(String(params.name))) {
      return HttpResponse.json(
        { error: `Skill not found: ${String(params.name)}` },
        { status: 404 }
      );
    }

    return HttpResponse.json(skillActionFixture);
  }),
  http.post("/api/skills/:name/disable", ({ params }) => {
    if (!skillByName.has(String(params.name))) {
      return HttpResponse.json(
        { error: `Skill not found: ${String(params.name)}` },
        { status: 404 }
      );
    }

    return HttpResponse.json(skillActionFixture);
  }),
];
