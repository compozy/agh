---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridges/resource_projection.go
line: 355
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsT,comment:PRRC_kwDOR5y4QM6-67EZ
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go's encoding/json package, when Decoder.UseNumber is enabled, does json.Number preserve the original numeric literal, and would json.Number("1").String() differ from json.Number("1.0").String()?`

💡 Result:

Yes, when Decoder.UseNumber is enabled, json.Number preserves the original numeric literal from the JSON input as a string. The Decoder tokenizes the JSON number and passes the exact literal string s to convertNumber, which returns Number(s) [1][2]. The json.Number type is defined as type Number string, and its String method simply returns string(n), which is the preserved literal [1][3]. Consequently, json.Number("1").String returns "1", while json.Number("1.0").String returns "1.0", so they differ [1].

Citations:

- 1: https://github.com/golang/go/blob/master/src%2Fencoding%2Fjson%2Fdecode.go
- 2: https://pkg.go.dev/encoding/json
- 3: https://pkg.go.dev/encoding/json@go1.25.6

---

🏁 Script executed:

```shell
find . -type f -name "resource_projection.go"
```

Repository: compozy/agh

Length of output: 184

---

🏁 Script executed:

```shell
cd internal/bridges && wc -l resource_projection.go
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
cd internal/bridges && sed -n '340,370p' resource_projection.go
```

Repository: compozy/agh

Length of output: 768

---

🏁 Script executed:

```shell
cd internal/bridges && sed -n '300,355p' resource_projection.go
```

Repository: compozy/agh

Length of output: 1521

---

🏁 Script executed:

```shell
cd internal/bridges && grep -n "func semanticJSONValuesEqual" resource_projection.go -A 2
```

Repository: compozy/agh

Length of output: 181

---

🏁 Script executed:

```shell
cd internal/bridges && sed -n '1,50p' resource_projection.go | head -30
```

Repository: compozy/agh

Length of output: 794

---

🏁 Script executed:

```shell
cd internal/bridges && grep -n "useNumber\|UseNumber\|json.Number" resource_projection.go
```

Repository: compozy/agh

Length of output: 147

---

🏁 Script executed:

```shell
cd internal/bridges && grep -n "semanticJSONValuesEqual" resource_projection.go
```

Repository: compozy/agh

Length of output: 304

---

🏁 Script executed:

```shell
cd internal/bridges && sed -n '280,320p' resource_projection.go
```

Repository: compozy/agh

Length of output: 1028

---

🏁 Script executed:

```shell
cd internal/bridges && grep -n "semanticJSONEqual\|json.Number" resource_projection.go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 472

---

🏁 Script executed:

```shell
cd internal/bridges && grep -rn "semanticJSONValuesEqual\|semanticJSONEqual" . --include="*.go" | head -20
```

Repository: compozy/agh

Length of output: 720

---

🏁 Script executed:

```shell
cd internal/bridges && cat json_equal_bench_test.go
```

Repository: compozy/agh

Length of output: 761

---



</details>

**Compare `json.Number` values numerically, not lexically.**

With `UseNumber`, equal numbers like `1`, `1.0`, and `1e0` preserve different spellings as strings. The current `.String()` comparison treats mathematically identical values as different, causing false deltas in bridge replacements and triggering unnecessary resource updates.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridges/resource_projection.go` around lines 353 - 355, The
json.Number branch currently compares numeric values lexically via
leftValue.String() == rightValue.String(), causing semantically equal numbers
like "1" and "1.0" to be treated as different; change the comparison to compare
numerically: parse leftValue.String() and right.(json.Number).String() into a
high-precision numeric type (e.g., big.Rat via new(big.Rat).SetString) and
return true if the resulting big.Rat Cmp == 0; if parsing into big.Rat fails for
either side, fall back to a safe numeric parse (e.g., strconv.ParseFloat) or
finally to the string comparison as a last resort. Ensure this change is applied
in the json.Number case handling (using the existing leftValue and right
variables) so mathematically equal numbers are treated as equal.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `semanticJSONValuesEqual` compares `json.Number.String()` verbatim, so semantically equal JSON numbers like `1` and `1.0` compare unequal. The numeric branch should compare parsed numeric values instead of lexical formatting.
