# cy-codex-loop — Iteration {{ iteration }} summary

- **Slug:** {{ slug }}
- **Phase entered:** {{ phase_in }} → **Phase exiting:** {{ phase_out }}
- **Mode:** {{ mode }}
- **Action taken:** {{ action }}
- **Outcome:** {{ outcome }}     <!-- completed | partial | blocked -->
- **Memory written:** {{ memory_paths_csv }}
- **State updated:** `.compozy/tasks/{{ slug }}/state.yaml`
- **Verify:** {{ verify_status }} ({{ verify_evidence }})
- **Checkpoint commit:** {{ commit_sha_or_skip_or_none }}     <!-- short SHA, "SKIP: no changes", or "n/a (phase != B)" -->
- **Blockers (if any):** {{ blockers_or_none }}
- **Next phase per detect-phase.py:** {{ next_phase }}

<!--
This is a human-filled substitution template. No bundled helper renders
the {{ placeholders }} automatically.

This block is required at the END of every iteration's last assistant
message. The codex-loop goal-check confirmation prompt scans for it as
evidence of progress. When phase_out == E, the agent ALSO emits the
content of assets/done-signature.txt on a line of its own.
-->
