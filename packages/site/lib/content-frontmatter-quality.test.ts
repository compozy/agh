import { describe, expect, it } from "vitest";
import { listManualDocs } from "./content-test-utils";

function frontmatter(content: string): string | null {
  return content.match(/^---\n([\s\S]*?)\n---/)?.[1] ?? null;
}

function fieldValue(frontmatterText: string, field: string): string | null {
  const escapedField = field.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = frontmatterText.match(new RegExp(`^${escapedField}:\\s*(.+)$`, "m"));
  return match?.[1]?.replace(/^["']|["']$/g, "").trim() ?? null;
}

describe("manual content frontmatter quality", () => {
  it("gives every manual runtime, protocol, and blog page a title and useful description", () => {
    const violations = listManualDocs().flatMap(doc => {
      const frontmatterText = frontmatter(doc.content);
      if (!frontmatterText) {
        return [`${doc.path}: missing frontmatter`];
      }

      const title = fieldValue(frontmatterText, "title");
      const description = fieldValue(frontmatterText, "description");
      const issues: string[] = [];
      if (!title) {
        issues.push("missing title");
      }
      if (!description) {
        issues.push("missing description");
      } else if (description.length < 30) {
        issues.push("description too short");
      } else if (description.length > 160) {
        issues.push(`description too long (${description.length} characters)`);
      }
      return issues.map(issue => `${doc.path}: ${issue}`);
    });

    expect(violations).toEqual([]);
  });

  it("keeps manual page titles and descriptions unique for navigation and social cards", () => {
    const documents = listManualDocs().map(doc => {
      const frontmatterText = frontmatter(doc.content);
      return {
        path: doc.path,
        title: frontmatterText ? fieldValue(frontmatterText, "title") : null,
        description: frontmatterText ? fieldValue(frontmatterText, "description") : null,
      };
    });

    const duplicates = ["title", "description"].flatMap(field => {
      const groups = new Map<string, string[]>();
      for (const document of documents) {
        const value = document[field as "title" | "description"];
        if (!value) {
          continue;
        }
        groups.set(value, [...(groups.get(value) ?? []), document.path]);
      }
      return [...groups.entries()]
        .filter(([, paths]) => paths.length > 1)
        .map(([value, paths]) => `${field} "${value}" reused by ${paths.join(", ")}`);
    });

    expect(duplicates).toEqual([]);
  });
});
