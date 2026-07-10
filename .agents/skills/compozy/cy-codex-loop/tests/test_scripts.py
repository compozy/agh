#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import os
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


detect_phase = load_module("detect_phase", SCRIPTS / "detect-phase.py")
state_io = load_module("state_io", SCRIPTS / "_state_io.py")


def _git(cwd: Path, *args: str, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=cwd,
        check=False,
        text=True,
        capture_output=True,
        env=env,
    )


def _git_env() -> dict[str, str]:
    env = os.environ.copy()
    env.update(
        {
            "GIT_AUTHOR_NAME": "test-runner",
            "GIT_AUTHOR_EMAIL": "test@example.com",
            "GIT_COMMITTER_NAME": "test-runner",
            "GIT_COMMITTER_EMAIL": "test@example.com",
            "GIT_CONFIG_GLOBAL": "/dev/null",
            "GIT_CONFIG_SYSTEM": "/dev/null",
        }
    )
    return env


class CyCodexLoopScriptTests(unittest.TestCase):
    def run_script(self, script: str, *args: str, cwd: Path | None = None) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPTS / script), *args],
            cwd=str(cwd) if cwd else REPO_ROOT,
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
            "verify": {"last_run": "2026-05-05T00:00:00Z", "last_status": "PASS"},
            "iterations": [],
        }
        state.update(overrides)
        state_path = slug_dir / "state.yaml"
        state_io.dump(state, state_path)
        return state_path

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
                "B",
                "--action",
                "round closed",
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            text = state_path.read_text(encoding="utf-8")
            self.assertNotIn("current_phase:", text)
            updated = state_io.load(state_path)
            self.assertEqual(updated["iterations"][0]["phase"], "B")

    def test_update_state_rejects_phase_d_choice(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tasks_root = Path(tmp) / "tasks"
            slug = "phase-d-rejected"
            self.write_state(tasks_root, slug)

            result = self.run_script(
                "update-state.py",
                slug,
                "--tasks-root",
                str(tasks_root),
                "--phase",
                "D",
                "--action",
                "should fail",
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("--phase", result.stderr)

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

    def test_detect_phase_emits_done_when_qa_complete_and_verify_pass(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tasks_root = Path(tmp) / "tasks"
            slug = "done-ready"
            self.write_state(tasks_root, slug)  # default has qa flags + verify PASS

            result = self.run_script(
                "detect-phase.py",
                slug,
                "--tasks-root",
                str(tasks_root),
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), "phase=E action=done")

    def test_detect_phase_reenters_qa_execution_when_verify_not_pass(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tasks_root = Path(tmp) / "tasks"
            slug = "verify-failed"
            self.write_state(
                tasks_root,
                slug,
                verify={"last_run": "2026-05-05T00:00:00Z", "last_status": "FAIL"},
            )

            result = self.run_script(
                "detect-phase.py",
                slug,
                "--tasks-root",
                str(tasks_root),
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), "phase=C action=qa_execution")

    # ---- commit-checkpoint.py ------------------------------------------------

    def _init_git_repo(self, root: Path) -> None:
        env = _git_env()
        init = _git(root, "init", "-q", "-b", "main", env=env)
        self.assertEqual(init.returncode, 0, init.stderr)
        # one anchoring commit so HEAD exists and rev-parse works
        (root / ".gitkeep").write_text("", encoding="utf-8")
        add = _git(root, "add", ".gitkeep", env=env)
        self.assertEqual(add.returncode, 0, add.stderr)
        first = _git(root, "commit", "-q", "-m", "anchor", env=env)
        self.assertEqual(first.returncode, 0, first.stderr)

    def _setup_checkpoint_repo(self, root: Path, slug: str) -> Path:
        self._init_git_repo(root)
        tasks_root = root / ".compozy" / "tasks"
        self.write_state(tasks_root, slug, mode="tasks", iteration=4)
        # Track the state.yaml so subsequent checkpoint runs see a clean tree
        env = _git_env()
        _git(root, "add", "-A", env=env)
        _git(root, "commit", "-q", "-m", "state", env=env)
        return tasks_root

    def test_commit_checkpoint_skips_when_no_changes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            slug = "noop"
            tasks_root = self._setup_checkpoint_repo(root, slug)

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "commit-checkpoint.py"),
                    slug,
                    "--task",
                    "task_07",
                    "--tasks-root",
                    str(tasks_root.relative_to(root)),
                ],
                cwd=root,
                env=_git_env(),
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), "SKIP: no changes")

    def test_commit_checkpoint_tasks_mode_builds_message_from_frontmatter(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            slug = "with-task"
            tasks_root = self._setup_checkpoint_repo(root, slug)
            slug_dir = tasks_root / slug
            (slug_dir / "task_07.md").write_text(
                "---\nstatus: pending\ntitle: implement backend tests\n---\n\nbody\n",
                encoding="utf-8",
            )
            (root / "feature.txt").write_text("change", encoding="utf-8")

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "commit-checkpoint.py"),
                    slug,
                    "--task",
                    "task_07",
                    "--tasks-root",
                    str(tasks_root.relative_to(root)),
                ],
                cwd=root,
                env=_git_env(),
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            sha = result.stdout.strip()
            self.assertEqual(len(sha), 40)

            log = _git(root, "log", "-1", "--pretty=%B", env=_git_env())
            self.assertEqual(log.returncode, 0, log.stderr)
            self.assertIn("feat: implement backend tests #07", log.stdout)
            self.assertIn("Checkpoint via cy-codex-loop (iteration 4, phase B mode=tasks).", log.stdout)
            self.assertIn(
                "Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>",
                log.stdout,
            )

    def test_commit_checkpoint_free_mode_truncates_long_slice(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            slug = "long-slice"
            tasks_root = self._setup_checkpoint_repo(root, slug)
            (root / "change.txt").write_text("x", encoding="utf-8")

            long_slice = "implement " + ("very " * 60) + "long slice"

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "commit-checkpoint.py"),
                    slug,
                    "--slice",
                    long_slice,
                    "--tasks-root",
                    str(tasks_root.relative_to(root)),
                ],
                cwd=root,
                env=_git_env(),
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)

            header = _git(root, "log", "-1", "--pretty=%s", env=_git_env())
            self.assertEqual(header.returncode, 0, header.stderr)
            subject = header.stdout.strip()
            self.assertLessEqual(len(subject), 72)
            self.assertTrue(subject.startswith("feat: implement "))

    def test_commit_checkpoint_rejects_missing_flags(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            slug = "no-flags"
            tasks_root = self._setup_checkpoint_repo(root, slug)
            (root / "change.txt").write_text("x", encoding="utf-8")

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "commit-checkpoint.py"),
                    slug,
                    "--tasks-root",
                    str(tasks_root.relative_to(root)),
                ],
                cwd=root,
                env=_git_env(),
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 2)
            self.assertIn("--task", result.stderr)

    def test_commit_checkpoint_includes_state_iteration_in_body(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            slug = "iteration"
            tasks_root = self._setup_checkpoint_repo(root, slug)
            slug_dir = tasks_root / slug
            (slug_dir / "task_03.md").write_text(
                "---\nstatus: pending\ntitle: bump iteration\n---\n",
                encoding="utf-8",
            )

            # Bump iteration via update-state so state.yaml stays canonical
            bump = self.run_script(
                "update-state.py",
                slug,
                "--tasks-root",
                str(tasks_root),
                "--phase",
                "B",
                "--action",
                "advance",
            )
            self.assertEqual(bump.returncode, 0, bump.stderr)
            # Commit the state mutation as anchor so the checkpoint diff is the new file
            env = _git_env()
            _git(root, "add", "-A", env=env)
            _git(root, "commit", "-q", "-m", "anchor-state", env=env)

            (root / "delta.txt").write_text("y", encoding="utf-8")

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "commit-checkpoint.py"),
                    slug,
                    "--task",
                    "task_03",
                    "--tasks-root",
                    str(tasks_root.relative_to(root)),
                ],
                cwd=root,
                env=env,
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)

            state = state_io.load(tasks_root / slug / "state.yaml")
            current_iter = int(state["iteration"])

            log = _git(root, "log", "-1", "--pretty=%B", env=env)
            self.assertIn(f"iteration {current_iter}", log.stdout)
            self.assertIn("phase B mode=tasks", log.stdout)

    def test_commit_checkpoint_rejects_state_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._init_git_repo(root)
            (root / "change.txt").write_text("x", encoding="utf-8")

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "commit-checkpoint.py"),
                    "missing-slug",
                    "--task",
                    "task_01",
                ],
                cwd=root,
                env=_git_env(),
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 2)
            self.assertIn("state.yaml", result.stderr)


if __name__ == "__main__":
    unittest.main()
