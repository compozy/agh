---
status: resolved
file: internal/procutil/process_started_at_unix.go
line: 54
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58jpQj,comment:PRRC_kwDOR5y4QM655WEZ
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/procutil/process_started_at_unix.go
```

Repository: compozy/agh

Length of output: 1575

---

🏁 Script executed:

```shell
# Search for tests related to this function
fd -t f "test" internal/procutil/
```

Repository: compozy/agh

Length of output: 130

---

🏁 Script executed:

```shell
# Check if there are any comments or TODOs about locale issues
rg -i "locale|lstart|ps.*start" internal/procutil/
```

Repository: compozy/agh

Length of output: 341

---

🏁 Script executed:

```shell
# Search for any usage of psStartedAtLayout or StartedAt function
rg "StartedAt|psStartedAtLayout" --type go
```

Repository: compozy/agh

Length of output: 26530

---

🏁 Script executed:

```shell
cat -n internal/procutil/procutil_test.go
```

Repository: compozy/agh

Length of output: 2685

---

🏁 Script executed:

```shell
cat -n internal/procutil/procutil_bench_test.go
```

Repository: compozy/agh

Length of output: 759

---

🏁 Script executed:

```shell
# Check how execabs.CommandContext is used elsewhere in the codebase
rg "CommandContext" --type go -A 3 | head -50
```

Repository: compozy/agh

Length of output: 2885

---

🌐 Web query:

`ps command lstart format locale dependent output`

💡 Result:

Yes, the ps command's lstart format specifier produces locale-dependent output. The lstart keyword displays the full time the command started, typically in a human-readable format like "Wed Nov 1 12:36:15 2017", where month and day names are affected by the current locale settings, particularly LC_TIME. Key evidence: - In the C locale: "Wed Nov 1 12:36:15 2017" (English abbreviations). - The format varies between implementations and locales; month names can differ in length across languages, potentially misaligning columns. - Environment variable LC_TIME specifically controls the date format for ps output. - To get consistent output, set LC_ALL=C or LC_TIME=C before running ps, e.g., LC_TIME=C ps -o lstart. This applies to procps-ng ps on Linux systems. Other keywords like bsdstart and start have similar locale influences on abbreviated month names, but lstart shows the full effect. Recent procps-ng documentation (as of man pages) confirms this behavior persists; use -D format or --date-format for fixed strftime-based formatting to avoid locale dependency.

Citations:

- 1: https://unix.stackexchange.com/questions/401785/ps-output-with-iso-date-format
- 2: https://ss64.com/bash/ps_keywords.html
- 3: https://man7.org/linux/man-pages/man1/ps.1p.html
- 4: https://stackoverflow.com/questions/44036947/process-start-timestamp-on-macos
- 5: https://man7.org/linux/man-pages/man1/ps.1.html
- 6: https://man.archlinux.org/man/ps.1.en.txt

---

🏁 Script executed:

```shell
# Let's also check if there's any existing locale handling in the codebase
rg "LC_ALL|LANG=" --type go
```

Repository: compozy/agh

Length of output: 197

---

🏁 Script executed:

```shell
# Check if there are any tests that might simulate non-English locales
rg -i "locale|utf|en_us" internal/procutil/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check how environment variables are typically handled when executing commands
rg "cmd.Env.*append.*Environ" --type go -B 2 -A 2 | head -40
```

Repository: compozy/agh

Length of output: 801

---

🏁 Script executed:

```shell
# Check what's already imported in the file
head -20 internal/procutil/process_started_at_unix.go
```

Repository: compozy/agh

Length of output: 453

---

🏁 Script executed:

```shell
# Verify if os.Environ() requires importing "os"
grep -n "import" internal/procutil/process_started_at_unix.go
```

Repository: compozy/agh

Length of output: 65

---



</details>

**Set LC_ALL=C locale when running `ps` to ensure consistent timestamp parsing.**

The `ps -o lstart` output format depends on system locale settings. On non-English systems, month and day names are localized, causing parse failures against the hardcoded English layout. This breaks process start-time matching on non-English hosts.

<details>
<summary>Required fix</summary>

```diff
 import (
 	"context"
 	"fmt"
+	"os"
 	"strconv"
 	"strings"
 	"time"
@@
-	output, err := execabs.CommandContext(
+	cmd := execabs.CommandContext(
 		ctx,
 		"ps",
 		"-o",
 		"lstart=",
 		"-p",
 		strconv.Itoa(pid),
-	).Output()
+	)
+	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
+	output, err := cmd.Output()
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	cmd := execabs.CommandContext(
		ctx,
		"ps",
		"-o",
		"lstart=",
		"-p",
		strconv.Itoa(pid),
	)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("procutil: read process %d start time: %w", pid, err)
	}

	startedAtText := strings.TrimSpace(string(output))
	if startedAtText == "" {
		return time.Time{}, fmt.Errorf("procutil: process %d start time is empty", pid)
	}

	startedAt, err := time.ParseInLocation(psStartedAtLayout, startedAtText, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"procutil: parse process %d start time %q: %w",
			pid,
			startedAtText,
			err,
		)
	}

	return startedAt.UTC(), nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/procutil/process_started_at_unix.go` around lines 26 - 54, The ps
output must be forced to the C locale before parsing; change the
execabs.CommandContext invocation that runs "ps -o lstart= -p strconv.Itoa(pid)"
so you create a cmd variable (using execabs.CommandContext), set cmd.Env =
append(os.Environ(), "LC_ALL=C") and then call cmd.Output(); keep existing error
handling and use psStartedAtLayout, pid and startedAtText as before so parsing
remains identical but now deterministic across locales.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the Unix implementation parses `ps -o lstart` with a hardcoded English layout but does not force the `ps` process locale, so non-`C` locales can produce localized month/day names that fail parsing.
- Fix plan: build the `ps` command explicitly, force `LC_ALL=C` before execution, and add regression coverage around the command environment/configuration path.
- Resolution: forced `LC_ALL=C` for the `ps` subprocess via a dedicated environment helper and added Unix-only regression coverage that verifies the locale override remains singular and deterministic.
- Verification: `go test ./internal/procutil` and `make verify` passed after the change.
