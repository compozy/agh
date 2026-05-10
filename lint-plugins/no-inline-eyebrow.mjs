/**
 * OXC/ESLint Plugin: No inline eyebrow tuples.
 *
 * Forbids inlining the AGH mono-uppercase eyebrow style as a className tuple.
 *
 * Triggers on a JSX `className` attribute whose value (string literal or
 * template literal text segments) contains BOTH `font-mono` AND `uppercase`,
 * OR contains an arbitrary `text-[Npx]` / `tracking-[Nem]` value combined
 * with `uppercase` / `font-mono`. The fix is to render through `<Eyebrow>`
 * from `@agh/ui`, which already encodes the `--text-eyebrow|badge|micro`
 * size tokens, the `--tracking-mono` tracking, and the canonical tone map.
 *
 * Exemptions:
 *  - Any file whose path includes `__tests__`, `.stories.`, or `.test.`.
 *  - The `@agh/ui` package itself (`packages/ui/src/components/...`) where
 *    structural primitives intentionally apply eyebrow typography to a
 *    non-span element (`<dt>`, `<th>`, `<label>`, sidebar/table head, wire
 *    card head, etc.). Consumers in `web/`, `packages/site/`, etc. always
 *    go through `<Eyebrow>`.
 *
 * Doc: `DESIGN.md` §3 + `docs/_memory/lessons/L-022-eyebrow-canonical-source.md`.
 */

const ALLOW_PATH_SEGMENTS = ["__tests__", "/.storybook/", "/storybook-static/"];
const ALLOW_FILE_SUFFIXES = [".test.tsx", ".test.ts", ".stories.tsx", ".stories.ts"];
const ALLOW_PACKAGE_DIRS = ["/packages/ui/src/"];
// Files whose entire purpose is to BE a typographic primitive (avatars whose
// rendered glyph happens to use mono uppercase, custom site eyebrows, etc.).
// New consumers go through `<Eyebrow>` from `@agh/ui`; these declare it.
const ALLOW_FILE_PATHS = [
  "/web/src/systems/network/components/timeline/message-avatar.tsx",
  "/packages/site/components/blog/mono-eyebrow.tsx",
  "/packages/site/components/blog/date-stamp.tsx",
  "/packages/site/components/blog/kind-chip.tsx",
];

// Structural HTML elements that legitimately receive eyebrow typography directly
// (because they are the load-bearing semantic node for their context — `<dt>` in a
// definition list, `<label>` for form controls, `<th>` table heads, breadcrumb
// wrappers, etc.). Non-element callers (custom components like `<TableHead>` from
// shadcn) are also exempt because their typography is owned by the primitive.
const STRUCTURAL_ELEMENTS = new Set([
  "label",
  "dt",
  "dd",
  "th",
  "thead",
  "tr",
  "summary",
  "figcaption",
  "legend",
]);

const ARBITRARY_TEXT_RE = /text-\[[^\]]+\]/;
const ARBITRARY_TRACKING_RE = /tracking-\[[^\]]+\]/;

function isExemptPath(filename) {
  if (!filename) return true;
  for (const seg of ALLOW_PATH_SEGMENTS) {
    if (filename.includes(seg)) return true;
  }
  for (const suf of ALLOW_FILE_SUFFIXES) {
    if (filename.endsWith(suf)) return true;
  }
  for (const dir of ALLOW_PACKAGE_DIRS) {
    if (filename.includes(dir)) return true;
  }
  for (const path of ALLOW_FILE_PATHS) {
    if (filename.endsWith(path) || filename.includes(path)) return true;
  }
  return false;
}

function isViolation(value) {
  if (!value || typeof value !== "string") return false;
  const tokens = value.split(/\s+/);
  const hasMono = tokens.includes("font-mono");
  const hasUpper = tokens.includes("uppercase");
  if (hasMono && hasUpper) return true;
  if (hasUpper && (ARBITRARY_TEXT_RE.test(value) || ARBITRARY_TRACKING_RE.test(value))) {
    return true;
  }
  if (hasMono && (ARBITRARY_TEXT_RE.test(value) || ARBITRARY_TRACKING_RE.test(value))) {
    // Allow plain mono code/badge styling without uppercase — only flag when
    // arbitrary values are present alongside mono, since those bypass tokens.
    return ARBITRARY_TEXT_RE.test(value) && ARBITRARY_TRACKING_RE.test(value);
  }
  return false;
}

function extractStringValues(node) {
  if (!node) return [];
  if (node.type === "Literal" && typeof node.value === "string") {
    return [node.value];
  }
  if (node.type === "TemplateLiteral") {
    return node.quasis.map(q => (q.value && q.value.cooked) || "");
  }
  if (node.type === "JSXExpressionContainer") {
    return extractStringValues(node.expression);
  }
  if (node.type === "CallExpression") {
    // cn(...) / clsx(...) / cva(...) — collect string args.
    const out = [];
    for (const arg of node.arguments) {
      out.push(...extractStringValues(arg));
    }
    return out;
  }
  if (node.type === "ConditionalExpression") {
    return [...extractStringValues(node.consequent), ...extractStringValues(node.alternate)];
  }
  if (node.type === "LogicalExpression") {
    return [...extractStringValues(node.left), ...extractStringValues(node.right)];
  }
  return [];
}

const noInlineEyebrow = {
  meta: {
    type: "problem",
    docs: {
      description:
        "Forbid inlining the AGH mono-uppercase eyebrow tuple in JSX className. Use <Eyebrow> from @agh/ui instead.",
      recommended: true,
    },
    messages: {
      inlineEyebrow:
        'Inlined eyebrow tuple in className. Use <Eyebrow> from @agh/ui (case="upper", size="eyebrow|badge|micro", tone) instead. See DESIGN.md §3 / L-022.',
    },
    schema: [],
  },
  create(context) {
    const filename = context.filename || "";
    if (isExemptPath(filename)) {
      return {};
    }

    return {
      JSXAttribute(node) {
        if (!node.name || node.name.name !== "className") return;
        // Skip when the enclosing element is a structural HTML primitive
        // (label, dt, dd, th, etc.) that legitimately owns eyebrow typography.
        const opening = node.parent;
        if (opening && opening.type === "JSXOpeningElement" && opening.name) {
          const elementName = opening.name.type === "JSXIdentifier" ? opening.name.name : null;
          if (elementName && STRUCTURAL_ELEMENTS.has(elementName)) {
            return;
          }
          // Custom-component className passes typography down (e.g. <TableHead>,
          // <TreeItemLabel>, <Item>). Treat any PascalCase tag as exempt — those
          // are component surfaces, not direct DOM nodes.
          if (elementName && /^[A-Z]/.test(elementName)) {
            return;
          }
        }
        const values = extractStringValues(node.value);
        for (const value of values) {
          if (isViolation(value)) {
            context.report({
              node,
              messageId: "inlineEyebrow",
            });
            return;
          }
        }
      },
    };
  },
};

const plugin = {
  meta: {
    name: "compozy-design-system",
  },
  rules: {
    "no-inline-eyebrow": noInlineEyebrow,
  },
};

export default plugin;
