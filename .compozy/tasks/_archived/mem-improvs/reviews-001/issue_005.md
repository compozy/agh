---
status: resolved
file: internal/memory/catalog.go
line: 535
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745ax,comment:PRRC_kwDOR5y4QM65BAP-
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In SQLite FTS5 MATCH syntax, are AND/OR/NOT/NEAR reserved operators, and what is the recommended way to search for those words literally?`

💡 Result:

Yes, AND, OR, and NOT are reserved boolean operators in SQLite FTS5 MATCH syntax (case-sensitive). NEAR is a reserved keyword for proximity groups. To search for these words literally as terms (e.g., rows containing the token "AND"), enclose them in double quotes: "AND", "OR", "NOT", "NEAR". Within quoted strings, escape embedded double quotes by doubling them (SQL-style). Barewords (unquoted tokens) that exactly match "AND", "OR", or "NOT" are not treated as strings but as operators, so quoting is required for literal search. NEAR as a bareword initiates a NEAR group, so quote it for literal matching. Examples: - MATCH '"AND"' -- matches rows containing token "AND" - MATCH '"one two"' -- phrase search - MATCH 'NEAR("AND" "OR")' -- NEAR group with literals "AND" and "OR"

Citations:

- 1: https://www.sqlite.org/fts5.html
- 2: https://sqlite.org/fts5.html

---



</details>

**Quote MATCH terms before concatenating them to prevent reserved operator interpretation.**

`tokenizeSearchQuery()` emits operator keywords like `and`, `or`, `not`, and `near`. Joining them unquoted (e.g., `not found` or `auth or sessions`) causes FTS5 to interpret these as operators rather than literal search terms, producing incorrect results or syntax errors. Double-quote each token and join with explicit `AND`.

<details>
<summary>Suggested fix</summary>

```diff
 func buildCatalogMatchQuery(query string) (string, error) {
 	terms := tokenizeSearchQuery(query)
 	if len(terms) == 0 {
 		return "", wrapValidationError("search query", query, errors.New("query is required"))
 	}
-	return strings.Join(terms, " "), nil
+	quoted := make([]string, 0, len(terms))
+	for _, term := range terms {
+		quoted = append(quoted, fmt.Sprintf("%q", term))
+	}
+	return strings.Join(quoted, " AND "), nil
 }
```
</details>

Also applies to: 537-554

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/catalog.go` around lines 529 - 535, The current
buildCatalogMatchQuery uses tokenizeSearchQuery terms unquoted, letting FTS5
treat words like "and/or/not/near" as operators; update buildCatalogMatchQuery
(and the similar logic at the other block around lines 537-554) to wrap each
token in double quotes and join them with explicit AND (e.g., "\"token\" AND
\"token\"") so every token is treated as a literal phrase; reference the
tokenizeSearchQuery call to get tokens, map each token to a quoted form, and
return strings.Join(quotedTokens, " AND "), preserving existing error handling.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `buildCatalogMatchQuery` currently concatenates raw search tokens into an FTS5 `MATCH` expression.
  - Tokens such as `and`, `or`, `not`, and `near` are parsed as operators instead of literal terms, which can change results or produce syntax errors for otherwise normal user queries.
  - I will quote each token, join them with explicit `AND`, and add regression coverage through the store search path.

## Resolution

- Quoted every catalog search token and joined terms with explicit `AND` so FTS5 reserved keywords are treated as literals.
- Added a store search regression test proving queries like `not` work through the catalog path instead of failing as operator syntax.
