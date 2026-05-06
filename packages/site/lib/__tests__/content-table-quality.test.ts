import { describe, expect, it } from "vitest";
import { listManualDocs, stripFencedCode } from "../content-test-utils";

type MarkdownTable = {
  line: number;
  header: string[];
  separator: string[];
  rows: Array<{
    line: number;
    cells: string[];
    isSeparator: boolean;
  }>;
};

function splitTableRow(line: string): string[] | null {
  const trimmed = line.trim();
  if (!trimmed.startsWith("|") || !trimmed.endsWith("|")) {
    return null;
  }
  return trimmed
    .slice(1, -1)
    .split("|")
    .map(cell => cell.trim());
}

function isSeparatorRow(line: string): boolean {
  const cells = splitTableRow(line);
  return cells !== null && cells.every(cell => /^:?-{3,}:?$/.test(cell));
}

function markdownTables(content: string): MarkdownTable[] {
  const lines = stripFencedCode(content).split("\n");
  const tables: MarkdownTable[] = [];

  for (let index = 0; index < lines.length - 1; index += 1) {
    const header = splitTableRow(lines[index] ?? "");
    const separatorLine = lines[index + 1] ?? "";
    if (!header || !isSeparatorRow(separatorLine)) {
      continue;
    }

    const separator = splitTableRow(separatorLine) ?? [];
    const rows: MarkdownTable["rows"] = [];
    let rowIndex = index + 2;
    while (rowIndex < lines.length) {
      const cells = splitTableRow(lines[rowIndex] ?? "");
      if (!cells) {
        break;
      }
      rows.push({
        line: rowIndex + 1,
        cells,
        isSeparator: isSeparatorRow(lines[rowIndex] ?? ""),
      });
      rowIndex += 1;
    }

    tables.push({
      line: index + 1,
      header,
      separator,
      rows,
    });
    index = rowIndex;
  }

  return tables;
}

describe("manual content table quality", () => {
  it("keeps Markdown table column counts consistent", () => {
    const violations = listManualDocs().flatMap(doc =>
      markdownTables(doc.content).flatMap(table => {
        const expected = table.header.length;
        const tableViolations: string[] = [];

        if (table.separator.length !== expected) {
          tableViolations.push(
            `${doc.path}:${table.line}: separator has ${table.separator.length} columns, expected ${expected}`
          );
        }

        for (const row of table.rows) {
          if (row.cells.length !== expected) {
            tableViolations.push(
              `${doc.path}:${row.line}: row has ${row.cells.length} columns, expected ${expected}`
            );
          }
        }

        return tableViolations;
      })
    );

    expect(violations).toEqual([]);
  });

  it("does not publish empty table headers", () => {
    const violations = listManualDocs().flatMap(doc =>
      markdownTables(doc.content).flatMap(table =>
        table.header
          .map((cell, index) => ({ cell, index }))
          .filter(header => header.cell.length === 0)
          .map(
            header => `${doc.path}:${table.line}: empty table header at column ${header.index + 1}`
          )
      )
    );

    expect(violations).toEqual([]);
  });

  it("does not repeat Markdown table separators inside table bodies", () => {
    const violations = listManualDocs().flatMap(doc =>
      markdownTables(doc.content).flatMap(table =>
        table.rows
          .filter(row => row.isSeparator)
          .map(row => `${doc.path}:${row.line}: repeated table separator`)
      )
    );

    expect(violations).toEqual([]);
  });
});
