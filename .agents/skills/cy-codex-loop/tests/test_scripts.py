#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from types import ModuleType


REPO_ROOT = Path(__file__).resolve().parents[4]
SKILL_ROOT = REPO_ROOT / ".agents" / "skills" / "cy-codex-loop"
SCRIPTS = SKILL_ROOT / "scripts"


def load_module(name: str, path: Path) -> ModuleType:
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load module from {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


coderabbit_to_rounds = load_module(
    "coderabbit_to_rounds",
    SCRIPTS / "coderabbit-to-rounds.py",
)
detect_phase = load_module("detect_phase", SCRIPTS / "detect-phase.py")
state_io = load_module("state_io", SCRIPTS / "_state_io.py")


class CyCodexLoopScriptTests(unittest.TestCase):
    def run_script(self, script: str, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPTS / script), *args],
            cwd=REPO_ROOT,
            check=False,
            text=True,
            capture_output=True,
        )

    def write_state(self, tasks_root: Path, slug: str, **overrides: object) -> Path:
        slug_dir = tasks_root / slug
        slug_dir.mkdir(parents=True, exist_ok=True)
        state = {
            "slug": slug,
            "created_at": "2026-05-05T00:00:00Z",
            "last_updated": "2026-05-05T00:00:00Z",
            "mode": "free",
            "iteration": 0,
            "goal_signature": "test goal",
            "tasks": {"total": 0, "completed": [], "current": None, "pending": []},
            "progress": {"deliverables_complete": True, "checklist": []},
            "qa": {"report_done": True, "execution_done": True},
            "coderabbit": {
                "rounds_completed": 0,
                "rounds_clean_streak": 0,
                "rounds_required": 3,
                "current_round_dir": None,
                "unresolved_critical": 0,
                "unresolved_high": 0,
            },
            "verify": {"last_run": "2026-05-05T00:00:00Z", "last_status": "PASS"},
            "iterations": [],
        }
        state.update(overrides)
        state_path = slug_dir / "state.yaml"
        state_io.dump(state, state_path)
        return state_path

    def test_norm_path_handles_symlinked_repo_root_alias(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            real_repo = base / "real" / "repo"
            real_repo.mkdir(parents=True)
            alias = base / "alias"
            alias.symlink_to(base / "real", target_is_directory=True)

            raw_path = alias / "repo" / "internal" / "worker.go"
            normalized = coderabbit_to_rounds._norm_path(str(raw_path), real_repo)

            self.assertEqual(normalized, "internal/worker.go")

    def test_find_findings_has_depth_guard(self) -> None:
        payload: object = {"findings": [{"title": "too deep"}]}
        for _ in range(32):
            payload = {"review": payload}

        findings = coderabbit_to_rounds._find_findings(payload)

        self.assertEqual(findings, [])

    def test_empty_round_creates_marker_and_reserves_round_number(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            review_json = base / "review.json"
            review_json.write_text(json.dumps({"findings": []}), encoding="utf-8")
            tasks_root = base / "tasks"
            slug = "empty-round"
            state_path = self.write_state(tasks_root, slug)
            out_dir = tasks_root / slug / "reviews-001"

            result = self.run_script(
                "coderabbit-to-rounds.py",
                str(review_json),
                str(out_dir),
                "--repo-root",
                str(REPO_ROOT),
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), "EMPTY")
            self.assertTrue((out_dir / ".empty").is_file())

            phase = self.run_script(
                "detect-phase.py",
                slug,
                "--tasks-root",
                str(tasks_root),
            )

            self.assertEqual(phase.returncode, 0, phase.stderr)
            self.assertEqual(phase.stdout.strip(), "phase=D action=coderabbit_round round=002")
            self.assertTrue(state_path.is_file())

    def test_jsonl_status_stream_with_complete_zero_findings_is_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            review_jsonl = base / "review.jsonl"
            review_jsonl.write_text(
                "\n".join(
                    [
                        json.dumps({"type": "review_context", "workingDirectory": str(REPO_ROOT)}),
                        json.dumps({"type": "status", "phase": "analyzing"}),
                        json.dumps({"type": "complete", "status": "review_completed", "findings": 0}),
                    ]
                )
                + "\n",
                encoding="utf-8",
            )
            out_dir = base / "reviews-001"

            result = self.run_script(
                "coderabbit-to-rounds.py",
                str(review_jsonl),
                str(out_dir),
                "--repo-root",
                str(REPO_ROOT),
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), "EMPTY")
            self.assertTrue((out_dir / ".empty").is_file())

    def test_jsonl_finding_events_are_converted_to_issues(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            review_jsonl = base / "review.jsonl"
            review_jsonl.write_text(
                "\n".join(
                    [
                        json.dumps({"type": "status", "phase": "reviewing"}),
                        json.dumps(
                            {
                                "type": "finding",
                                "severity": "high",
                                "file": "internal/memory/store.go",
                                "line": 42,
                                "title": "Store skips rollback on write failure",
                                "comment": "A failed write can leave a partial transaction open.",
                            }
                        ),
                        json.dumps({"type": "complete", "status": "review_completed", "findings": 1}),
                    ]
                )
                + "\n",
                encoding="utf-8",
            )
            out_dir = base / "reviews-001"

            result = self.run_script(
                "coderabbit-to-rounds.py",
                str(review_jsonl),
                str(out_dir),
                "--repo-root",
                str(REPO_ROOT),
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("wrote 1 issues", result.stdout)
            issue = (out_dir / "issue_001.md").read_text(encoding="utf-8")
            self.assertIn("severity: high", issue)
            self.assertIn("file: internal/memory/store.go", issue)
            self.assertIn("Store skips rollback on write failure", issue)

    def test_coderabbit_current_schema_maps_file_name_and_codegen_body(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            review_jsonl = base / "review.jsonl"
            review_jsonl.write_text(
                "\n".join(
                    [
                        json.dumps(
                            {
                                "type": "finding",
                                "severity": "major",
                                "fileName": "internal/memory/store.go",
                                "codegenInstructions": (
                                    "Verify each finding against current code. "
                                    "Fix only still-valid issues, skip the rest with a brief reason, "
                                    "keep changes minimal, and validate.\n\n"
                                    "In @internal/memory/store.go around lines 580 - 593, "
                                    "globalHomeFromMemoryDir returns the parent in both branches."
                                ),
                                "suggestions": [],
                            }
                        ),
                        json.dumps({"type": "complete", "status": "review_completed", "findings": 1}),
                    ]
                )
                + "\n",
                encoding="utf-8",
            )
            out_dir = base / "reviews-001"

            result = self.run_script(
                "coderabbit-to-rounds.py",
                str(review_jsonl),
                str(out_dir),
                "--repo-root",
                str(REPO_ROOT),
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            issue = (out_dir / "issue_001.md").read_text(encoding="utf-8")
            self.assertIn("severity: high", issue)
            self.assertIn("file: internal/memory/store.go", issue)
            self.assertIn("line: 580", issue)
            self.assertIn("# Issue 001: globalHomeFromMemoryDir returns the parent in both branches.", issue)
            self.assertIn("Verify each finding against current code", issue)

    def test_invalid_status_is_closed_for_round_cleanliness(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            round_dir = Path(tmp) / "reviews-001"
            round_dir.mkdir()
            (round_dir / "issue_001.md").write_text(
                "---\nstatus: invalid\nseverity: critical\n---\n\n# invalid\n",
                encoding="utf-8",
            )
            (round_dir / "issue_002.md").write_text(
                "---\nstatus: pending\nseverity: high\n---\n\n# pending\n",
                encoding="utf-8",
            )

            critical, high = detect_phase._round_has_unresolved(round_dir)
            result = self.run_script("check-rounds-clean.py", str(round_dir))

            self.assertEqual((critical, high), (0, 1))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("critical=0 high=1", result.stdout)

    def test_complete_progress_requires_exact_existing_text(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tasks_root = Path(tmp) / "tasks"
            slug = "progress"
            state_path = self.write_state(
                tasks_root,
                slug,
                progress={
                    "deliverables_complete": False,
                    "checklist": [
                        {"text": "Implement slice", "status": "in_progress", "iteration": 1}
                    ],
                },
            )

            result = self.run_script(
                "update-state.py",
                slug,
                "--tasks-root",
                str(tasks_root),
                "--complete-progress",
                " Implement slice ",
            )

            self.assertEqual(result.returncode, 1)
            self.assertIn("did not match", result.stderr)
            state = state_io.load(state_path)
            self.assertEqual(len(state["progress"]["checklist"]), 1)
            self.assertEqual(state["progress"]["checklist"][0]["status"], "in_progress")

            ok = self.run_script(
                "update-state.py",
                slug,
                "--tasks-root",
                str(tasks_root),
                "--phase",
                "B",
                "--complete-progress",
                "Implement slice",
            )

            self.assertEqual(ok.returncode, 0, ok.stderr)
            updated = state_io.load(state_path)
            self.assertEqual(updated["progress"]["checklist"][0]["status"], "completed")
            self.assertEqual(updated["iterations"][0]["phase"], "B")

    def test_update_state_drops_stale_top_level_current_phase(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tasks_root = Path(tmp) / "tasks"
            slug = "phase"
            state_path = self.write_state(tasks_root, slug)
            original = state_path.read_text(encoding="utf-8")
            state_path.write_text(
                original.replace("mode:", 'current_phase: "E"\nmode:', 1),
                encoding="utf-8",
            )

            result = self.run_script(
                "update-state.py",
                slug,
                "--tasks-root",
                str(tasks_root),
                "--phase",
                "D",
                "--action",
                "round closed",
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            text = state_path.read_text(encoding="utf-8")
            self.assertNotIn("current_phase:", text)
            updated = state_io.load(state_path)
            self.assertEqual(updated["iterations"][0]["phase"], "D")

    def test_update_state_reconciles_task_files_into_tasks_mode(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tasks_root = Path(tmp) / "tasks"
            slug = "reconcile"
            slug_dir = tasks_root / slug
            state_path = self.write_state(
                tasks_root,
                slug,
                mode="free",
                progress={"deliverables_complete": False, "checklist": []},
                qa={"report_done": False, "execution_done": False},
            )
            (slug_dir / "_tasks.md").write_text("# tasks\n", encoding="utf-8")
            (slug_dir / "task_01.md").write_text(
                "---\nstatus: completed\ntitle: one\ntype: backend\n---\n\n# one\n",
                encoding="utf-8",
            )
            (slug_dir / "task_02.md").write_text(
                "---\nstatus: in_progress\ntitle: two\ntype: backend\n---\n\n# two\n",
                encoding="utf-8",
            )
            (slug_dir / "task_03.md").write_text(
                "---\nstatus: pending\ntitle: three\ntype: test\n---\n\n# three\n",
                encoding="utf-8",
            )

            result = self.run_script(
                "update-state.py",
                slug,
                "--tasks-root",
                str(tasks_root),
                "--phase",
                "B",
                "--action",
                "reconcile task graph",
                "--outcome",
                "completed",
                "--reconcile-tasks",
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            updated = state_io.load(state_path)
            self.assertEqual(updated["mode"], "tasks")
            self.assertEqual(updated["tasks"]["total"], 3)
            self.assertEqual(updated["tasks"]["completed"], ["task_01"])
            self.assertEqual(updated["tasks"]["current"], "task_02")
            self.assertEqual(updated["tasks"]["pending"], ["task_02", "task_03"])

            phase = self.run_script(
                "detect-phase.py",
                slug,
                "--tasks-root",
                str(tasks_root),
            )

            self.assertEqual(phase.returncode, 0, phase.stderr)
            self.assertEqual(
                phase.stdout.strip(),
                "phase=B action=execute_task task=task_02",
            )


if __name__ == "__main__":
    unittest.main()
