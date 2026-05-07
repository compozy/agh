---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/hardcut_residue_test.go
line: 119
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaqH,comment:PRRC_kwDOR5y4QM6-7HYj
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
python - <<'PY'
import os

targets = ["cmd", "internal", "web", "packages/site", "openapi", "config.toml"]
root = "."
limit = 1024 * 1024
count = 0

for target in targets:
    p = os.path.join(root, target)
    if not os.path.exists(p):
        continue

    files = [p] if os.path.isfile(p) else [
        os.path.join(dp, f)
        for dp, _, fs in os.walk(p)
        for f in fs
    ]

    for fp in files:
        try:
            size = os.path.getsize(fp)
            if size <= limit:
                continue
            with open(fp, "rb") as fh:
                head = fh.read(8192)
                fh.seek(0)
                first_mib = fh.read(limit + 1)
            has_nul = b"\x00" in head
            has_newline = b"\n" in first_mib
            if has_nul or not has_newline:
                count += 1
                print(f"{fp}\tsize={size}\thas_nul={has_nul}\tnewline_in_first_1MiB={has_newline}")
        except Exception:
            pass

print(f"potentially problematic files: {count}")
PY
```

Repository: compozy/agh

Length of output: 3487

---

🏁 Script executed:

```shell
cat -n internal/modelcatalog/hardcut_residue_test.go
```

Repository: compozy/agh

Length of output: 5250

---



</details>

**Binary image files in scan targets are not filtered and could trigger scanner errors on unrelated asset changes**

The test scans `packages/site/public/images/` but `skipResidueGuardFile` (line 132) does not filter image files. The verification found 27 PNG files (>1 MiB each with binary null bytes) that will be opened and scanned. If a PNG's binary content lacks newlines for >1 MiB spans, `bufio.Scanner` will fail with `ErrTooLong`, failing the test despite no residue issue.

Add a binary-file detector or extend file-type filtering to skip images and other non-text assets:

<details>
<summary>Suggested fix</summary>

```diff
 import (
+	"bytes"
 	"bufio"
 	"fmt"
 	"io/fs"
 	"os"
@@
 	if skipResidueGuardFile(rel) {
 		return residues
 	}
+	if !isLikelyTextFile(t, path) {
+		return residues
+	}
 	file, err := os.Open(path)
@@
 	scanner := bufio.NewScanner(file)
-	scanner.Buffer(make([]byte, 1024), 1024*1024)
+	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
@@
+
+func isLikelyTextFile(t *testing.T, path string) bool {
+	t.Helper()
+	f, err := os.Open(path)
+	if err != nil {
+		t.Fatalf("os.Open(%q) error = %v", path, err)
+	}
+	defer func() {
+		if closeErr := f.Close(); closeErr != nil {
+			t.Errorf("Close(%q) error = %v", path, closeErr)
+		}
+	}()
+	buf := make([]byte, 8192)
+	n, err := f.Read(buf)
+	if err != nil && err.Error() != "EOF" {
+		t.Fatalf("Read(%q) error = %v", path, err)
+	}
+	return !bytes.Contains(buf[:n], []byte{0})
+}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/hardcut_residue_test.go` around lines 101 - 119, The
test iterates with bufio.NewScanner (scanner) over files under
packages/site/public/images/ and currently skipResidueGuardFile is not excluding
binary assets, which causes ErrTooLong on PNGs; update skipResidueGuardFile (or
add a new isBinaryOrLargeFile check invoked before creating scanner) to skip
common non-text files (e.g., .png/.jpg/.gif/.webp/.ico/.svg? if treated binary)
and/or files larger than a threshold, or else perform a quick binary-detection
(read first N bytes and skip if NULs or high non-printable ratio) before calling
scanner; ensure the test still runs allowedProviderModelResidue and fields
checks only on text files and append to residues unchanged for scanned files.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/modelcatalog/hardcut_residue_test.go` scans every file under `packages/site`, including many `packages/site/public/images/*.png` assets larger than 1 MiB.
  - The residue guard is intended for text surfaces, but today it opens binary assets and relies on `bufio.Scanner` line-token behavior on arbitrary image bytes.
  - Fix plan: skip obvious binary asset extensions before scanning so the guard remains focused on text residue and does not depend on incidental newline placement inside PNGs.
  - Fixed in `internal/modelcatalog/hardcut_residue_test.go` and verified with focused package tests plus `make verify`.
