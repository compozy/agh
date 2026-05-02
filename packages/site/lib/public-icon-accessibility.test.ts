import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "./content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".ts", ".tsx"];

function listSourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const normalizedPath = fullPath.replaceAll("\\", "/");
    if (ignoredPathSegments.some(segment => normalizedPath.includes(segment))) {
      continue;
    }

    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listSourceFiles(fullPath));
      continue;
    }

    if (
      stat.isFile() &&
      sourceExtensions.some(extension => fullPath.endsWith(extension)) &&
      !/\.test\.[cm]?tsx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }

  return files.sort();
}

function importedIconNames(content: string): string[] {
  const imports = [
    ...content.matchAll(/import\s+\{([^}]*)\}\s+from\s+["'](?:lucide-react|@agh\/ui\/logos)["'];/g),
  ];

  const names = imports.flatMap(match =>
    (match[1] ?? "")
      .split(",")
      .map(specifier => specifier.trim())
      .filter(Boolean)
      .map(specifier => {
        const withoutType = specifier.replace(/^type\s+/, "").trim();
        const alias = withoutType.match(/\bas\s+([A-Za-z_$][\w$]*)$/)?.[1];
        return alias ?? withoutType.match(/^([A-Za-z_$][\w$]*)/)?.[1] ?? null;
      })
      .filter((name): name is string => Boolean(name))
  );

  return [...new Set(names)].sort();
}

function hasAttribute(tag: string, attribute: string): boolean {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`\\b${escapedAttribute}(?:\\s|=|>)`).test(tag);
}

function quotedAttribute(tag: string, attribute: string): string | null {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return (
    tag.match(new RegExp(`\\b${escapedAttribute}=["']([^"']*)["']`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{["']([^"']*)["']\\}`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{\\\`([^\\\`]*)\\\`\\}`))?.[1] ??
    null
  );
}

function iconElements(content: string, iconName: string): string[] {
  return [...content.matchAll(new RegExp(`<${iconName}\\b[^>]*(?:/>|>)`, "g"))].map(
    match => match[0] ?? ""
  );
}

function interactiveElementBodies(content: string): string[] {
  return [...content.matchAll(/<(a|Link|button|Button)\b[^>]*>[\s\S]*?<\/\1>/g)].map(
    match => match[0] ?? ""
  );
}

function isAccessibleIcon(tag: string): boolean {
  if (hasAttribute(tag, "aria-hidden")) {
    return true;
  }

  if (quotedAttribute(tag, "role") !== "img") {
    return false;
  }

  return hasAttribute(tag, "aria-label") || hasAttribute(tag, "aria-labelledby");
}

describe("public icon accessibility", () => {
  it("hides imported icons inside public links and buttons", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      const interactiveBodies = interactiveElementBodies(content);
      return importedIconNames(content).flatMap(iconName =>
        interactiveBodies.flatMap(body =>
          iconElements(body, iconName).flatMap(tag =>
            isAccessibleIcon(tag)
              ? []
              : [
                  `${relative(siteRoot, file)}: <${iconName}> inside an interactive control is missing aria-hidden or role="img" with a label`,
                ]
          )
        )
      );
    });

    expect(violations).toEqual([]);
  });
});
