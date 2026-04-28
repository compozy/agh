VERIFICATION REPORT
-------------------
Claim: <what is being claimed>
Command: `<full verification command>`
Executed: <timestamp or relative time>
Exit code: <0 or non-zero>
Output summary: <key pass/fail lines, counts, build result>
Warnings: <none or list>
Errors: <none or list>
Verdict: PASS or FAIL

BEHAVIORAL EVIDENCE (when behavior-first QA was executed)
---------------------------------------------------------
Operator journey: <what real operator goal was tested>
Business outcome: <achieved, failed, or blocked>
Live provider/LLM: <provider-backed evidence or exact blocked boundary>
Agent behavior:
  - <agent role>: <decision, artifact, message, handoff, or task/channel behavior>
Artifacts produced and used:
  - <artifact path/name>: <producer and downstream use>
Cross-surface truth checks:
  - <object/artifact>: <CLI/API/Web/runtime agreement>
Disruption probes:
  - <probe>: <expected vs actual>
Smoke/readiness checks only:
  - <check>: <result and why it is not final behavioral proof>

BROWSER EVIDENCE (when Web UI flows were tested)
-------------------------------------------------
Dev server: <start command and confirmed URL>
Flows tested: <number of flows exercised>
Flow details:
  - <flow name>: <entry URL> -> <final URL> | Verdict: PASS or FAIL
    Evidence: <screenshot path or inline observation>
  - <flow name>: <entry URL> -> <final URL> | Verdict: PASS or FAIL
    Evidence: <screenshot path or inline observation>
Viewports tested: <list of viewports or "default only">
Authentication: <method used or "not required">
Blocked flows: <none or list with reason>

TEST CASE COVERAGE (when qa-report artifacts exist)
----------------------------------------------------------
Test cases found: <number of TC-*.md files in qa-output-path/test-cases/>
Executed: <number exercised during this QA run>
Results:
  - <TC-SCEN-ID>: PASS or FAIL | Behavioral journey: <summary> | Bug: <BUG-ID or "none">
  - <TC-ID>: PASS or FAIL | Bug: <BUG-ID or "none">
  - <TC-ID>: PASS or FAIL | Bug: <BUG-ID or "none">
  - <TC-ID>: BLOCKED | Reason: <why>
Not executed: <list of TC-IDs skipped with reason, or "none">

ISSUES FILED
-------------
Total: <number of BUG-*.md files created in qa-output-path/issues/>
By severity:
  - Critical: <count>
  - High: <count>
  - Medium: <count>
  - Low: <count>
Details:
  - <BUG-ID>: <short-title> | Severity: <level> | Priority: <P0-P3> | Status: <Open or Fixed>
