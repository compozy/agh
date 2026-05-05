#!/usr/bin/env python3
"""Audit AGH real-scenario QA evidence against a machine-readable contract."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import sys
from typing import Any


MOCK_MARKERS = ("mock", "acpmock", "fake", "stub", "fixture")
CORE_CROSS_SURFACES = {"cli", "api", "web", "runtime"}


@dataclass
class Finding:
    check: str
    message: str
    evidence: str = ""


def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"{path} must contain a JSON object")
    return data


def load_json_compatible_yaml(path: Path) -> dict[str, Any]:
    text = path.read_text(encoding="utf-8").strip()
    if not text:
        raise ValueError(f"{path} is empty")
    try:
        data = json.loads(text)
    except json.JSONDecodeError as err:
        raise ValueError(
            f"{path} must be JSON-compatible YAML because PyYAML is not a project dependency: {err}"
        ) from err
    if not isinstance(data, dict):
        raise ValueError(f"{path} must contain an object")
    return data


def load_journey_log(path: Path) -> list[dict[str, Any]]:
    entries: list[dict[str, Any]] = []
    if not path.exists():
        return entries
    with path.open("r", encoding="utf-8") as handle:
        for lineno, line in enumerate(handle, start=1):
            stripped = line.strip()
            if not stripped:
                continue
            try:
                entry = json.loads(stripped)
            except json.JSONDecodeError as err:
                raise ValueError(f"{path}:{lineno} is invalid JSONL: {err}") from err
            if not isinstance(entry, dict):
                raise ValueError(f"{path}:{lineno} must be a JSON object")
            entries.append(entry)
    return entries


def count_value(value: Any) -> int:
    if value is None:
        return 0
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, int):
        return max(value, 0)
    if isinstance(value, list):
        return len(value)
    if isinstance(value, dict):
        return len(value)
    if isinstance(value, str):
        return 1 if value.strip() else 0
    return 0


def list_item_id(item: Any) -> str:
    if isinstance(item, str):
        return item.strip()
    if isinstance(item, dict):
        for key in ("id", "name", "channel", "task_id", "run_id"):
            value = item.get(key)
            if isinstance(value, str) and value.strip():
                return value.strip()
    return ""


def min_int(contract: dict[str, Any], key: str) -> int:
    minimums = contract.get("minimums", {})
    value = minimums.get(key, 0) if isinstance(minimums, dict) else 0
    return value if isinstance(value, int) else 0


def task_minimums(contract: dict[str, Any]) -> dict[str, int]:
    minimums = contract.get("minimums", {})
    tasks = minimums.get("tasks", {}) if isinstance(minimums, dict) else {}
    if not isinstance(tasks, dict):
        return {"roots": 0, "subtasks": 0, "dependencies": 0, "runs": 0}
    return {
        "roots": int(tasks.get("roots", 0) or 0),
        "subtasks": int(tasks.get("subtasks", 0) or 0),
        "dependencies": int(tasks.get("dependencies", 0) or 0),
        "runs": int(tasks.get("runs", 0) or 0),
    }


def surfaces_required(contract: dict[str, Any]) -> set[str]:
    minimums = contract.get("minimums", {})
    values = minimums.get("surfaces_required", []) if isinstance(minimums, dict) else []
    if not isinstance(values, list):
        return set()
    return {str(value).lower() for value in values if str(value).strip()}


def resolve_path(path_text: str, roots: list[Path]) -> Path | None:
    if not path_text:
        return None
    path = Path(path_text).expanduser()
    if path.is_absolute():
        return path
    for root in roots:
        candidate = (root / path).resolve()
        if candidate.exists():
            return candidate
    return (roots[0] / path).resolve() if roots else path.resolve()


def evidence_exists(path_text: str, roots: list[Path]) -> bool:
    path = resolve_path(path_text, roots)
    return bool(path and path.is_file() and path.stat().st_size > 0)


def actor_ids(charter: dict[str, Any], log_entries: list[dict[str, Any]]) -> set[str]:
    ids = {str(entry.get("actor", "")).strip() for entry in log_entries if str(entry.get("actor", "")).strip()}
    for item in charter.get("agents", []) if isinstance(charter.get("agents"), list) else []:
        item_id = list_item_id(item)
        if item_id:
            ids.add(item_id)
    return ids


def role_ids(charter: dict[str, Any]) -> set[str]:
    roles: set[str] = set()
    for item in charter.get("agents", []) if isinstance(charter.get("agents"), list) else []:
        if isinstance(item, dict):
            role = str(item.get("role", "")).strip().lower()
            if role:
                roles.add(role)
    return roles


def channel_ids(charter: dict[str, Any], log_entries: list[dict[str, Any]]) -> set[str]:
    ids: set[str] = set()
    for item in charter.get("channels", []) if isinstance(charter.get("channels"), list) else []:
        item_id = list_item_id(item)
        if item_id:
            ids.add(item_id)
    for entry in log_entries:
        for key in ("channel", "channel_id"):
            value = str(entry.get(key, "")).strip()
            if value:
                ids.add(value)
        target = str(entry.get("target", ""))
        match = re.search(r"(?:channel[:=/]|channels/)([A-Za-z0-9_.-]+)", target)
        if match:
            ids.add(match.group(1))
    return ids


def task_counts(charter: dict[str, Any], log_entries: list[dict[str, Any]]) -> dict[str, int]:
    tree = charter.get("task_tree", {})
    counts = {"roots": 0, "subtasks": 0, "dependencies": 0, "runs": 0}
    if isinstance(tree, dict):
        counts["roots"] = count_value(tree.get("roots"))
        counts["subtasks"] = count_value(tree.get("subtasks"))
        counts["dependencies"] = count_value(tree.get("dependencies"))
        counts["runs"] = count_value(tree.get("runs"))
    seen: dict[str, set[str]] = {key: set() for key in counts}
    kind_map = {
        "root": "roots",
        "subtask": "subtasks",
        "dependency": "dependencies",
        "run": "runs",
    }
    for entry in log_entries:
        kind = kind_map.get(str(entry.get("task_kind", "")).lower())
        if not kind:
            continue
        ids = entry.get("ids", [])
        if isinstance(ids, list):
            for value in ids:
                if str(value).strip():
                    seen[kind].add(str(value).strip())
    for key, values in seen.items():
        counts[key] = max(counts[key], len(values))
    return counts


def cross_surface_count(log_entries: list[dict[str, Any]]) -> tuple[int, dict[str, list[str]]]:
    by_id: dict[str, set[str]] = {}
    for entry in log_entries:
        surface = str(entry.get("surface", "")).strip().lower()
        if not surface:
            continue
        ids = entry.get("ids", [])
        if not isinstance(ids, list):
            continue
        for value in ids:
            text = str(value).strip()
            if not text:
                continue
            by_id.setdefault(text, set()).add(surface)
    matched = {
        object_id: sorted(surfaces)
        for object_id, surfaces in by_id.items()
        if CORE_CROSS_SURFACES.issubset(surfaces)
    }
    return len(matched), matched


def provider_live_count(provider_attempt: dict[str, Any]) -> int:
    live_ids = provider_attempt.get("live_proof_session_ids", [])
    decisions = provider_attempt.get("observed_agent_decisions", [])
    if not isinstance(decisions, list):
        decisions = []
    if not isinstance(live_ids, list):
        return 0
    probed = provider_attempt.get("providers_probed", [])
    marker_text = json.dumps(probed).lower() if isinstance(probed, list) else ""
    if any(marker in marker_text for marker in MOCK_MARKERS):
        return 0
    meaningful = [decision for decision in decisions if str(decision).strip()]
    if not meaningful:
        for item in probed if isinstance(probed, list) else []:
            if isinstance(item, dict):
                item_decisions = item.get("observed_agent_decisions", [])
                if isinstance(item_decisions, list):
                    meaningful.extend(decision for decision in item_decisions if str(decision).strip())
    return len([value for value in live_ids if str(value).strip()]) if meaningful else 0


def artifact_reuse_count(charter: dict[str, Any], log_entries: list[dict[str, Any]], roots: list[Path]) -> int:
    count = 0
    for item in charter.get("artifacts", []) if isinstance(charter.get("artifacts"), list) else []:
        if not isinstance(item, dict):
            continue
        path_text = str(item.get("path", "")).strip()
        used_later = item.get("used_later_by") or item.get("consumer_action")
        if path_text and used_later and evidence_exists(path_text, roots):
            count += 1
    used_paths = {
        str(entry.get("evidence_path", "")).strip()
        for entry in log_entries
        if str(entry.get("action", "")).lower() in {"artifact_used", "use_artifact", "consume_artifact"}
    }
    for path_text in used_paths:
        if evidence_exists(path_text, roots):
            count += 1
    return count


def disruption_count(log_entries: list[dict[str, Any]]) -> int:
    phases_by_probe: dict[str, set[str]] = {}
    for entry in log_entries:
        probe_id = str(entry.get("probe_id", "")).strip()
        if not probe_id:
            continue
        phase = str(entry.get("phase", "")).strip().lower()
        action = str(entry.get("action", "")).strip().lower()
        phases_by_probe.setdefault(probe_id, set()).add(phase or action)
    complete = 0
    for phases in phases_by_probe.values():
        if phases.intersection({"trigger", "start", "execute"}) and phases.intersection({"observed", "result", "verify"}):
            complete += 1
    return complete


def final_verify_paths(report_text: str) -> list[str]:
    candidates = re.findall(r"[\w./~:-]*(?:final-)?make-verify\.log", report_text)
    candidates.extend(re.findall(r"[\w./~:-]*verify(?:-[\w.-]+)?\.log", report_text))
    return sorted(set(candidates))


def audit(args: argparse.Namespace) -> tuple[list[Finding], list[Finding], dict[str, Any]]:
    qa_output_path = Path(args.qa_output_path).resolve()
    qa_root = qa_output_path / "qa" if (qa_output_path / "qa").is_dir() else qa_output_path
    roots = [qa_root, qa_output_path, Path.cwd()]

    blockers: list[Finding] = []
    warnings: list[Finding] = []
    metadata: dict[str, Any] = {"qa_root": str(qa_root)}

    try:
        contract = load_json(Path(args.scenario_contract).resolve())
    except Exception as err:  # noqa: BLE001 - CLI should report all loader failures uniformly.
        return [Finding("C2", f"scenario contract failed to load: {err}", args.scenario_contract)], [], metadata
    try:
        charter = load_json_compatible_yaml(Path(args.charter).resolve())
    except Exception as err:  # noqa: BLE001
        return [Finding("C1", f"behavioral charter failed to load: {err}", args.charter)], [], metadata
    try:
        log_entries = load_journey_log(Path(args.journey_log).resolve())
    except Exception as err:  # noqa: BLE001
        return [Finding("C4", f"journey log failed to load: {err}", args.journey_log)], [], metadata
    try:
        provider_attempt = load_json(Path(args.provider_attempt).resolve())
    except Exception as err:  # noqa: BLE001
        provider_attempt = {}
        blockers.append(Finding("C9", f"provider attempt failed to load: {err}", args.provider_attempt))

    metadata["contract"] = contract
    metadata["log_entries"] = len(log_entries)

    required_contract_keys = {"schema_version", "release_grade", "scope_slug", "minimums", "enforcement"}
    missing_contract = sorted(required_contract_keys - set(contract))
    if missing_contract:
        blockers.append(Finding("C2", f"scenario contract missing keys: {', '.join(missing_contract)}", args.scenario_contract))

    required_charter_keys = {
        "schema_version",
        "startup_situation",
        "operator_intent",
        "expected_business_outcome",
        "agents",
        "channels",
        "task_tree",
        "provider_plan",
        "cross_surface_targets",
        "disruption_probes",
        "artifacts",
    }
    missing_charter = sorted(required_charter_keys - set(charter))
    if missing_charter:
        blockers.append(Finding("C1", f"behavioral charter missing keys: {', '.join(missing_charter)}", args.charter))

    manifest_path = qa_root / "bootstrap-manifest.json"
    if not manifest_path.is_file():
        blockers.append(Finding("C3", "bootstrap manifest is missing", str(manifest_path)))
    else:
        try:
            manifest = load_json(manifest_path)
            status = manifest.get("status", {}) if isinstance(manifest.get("status"), dict) else {}
            health = str(status.get("health", "")).lower()
            if health not in {"fresh", "healthy"}:
                blockers.append(Finding("C3", f"bootstrap manifest health is not healthy/fresh: {health}", str(manifest_path)))
        except Exception as err:  # noqa: BLE001
            blockers.append(Finding("C3", f"bootstrap manifest failed to load: {err}", str(manifest_path)))

    actors = actor_ids(charter, log_entries)
    min_agents = min_int(contract, "agents")
    if len(actors) < min_agents:
        blockers.append(Finding("C4", f"distinct actors {len(actors)} < required {min_agents}", args.journey_log))

    roles = role_ids(charter)
    min_roles = min_int(contract, "differentiated_roles")
    if len(roles) < min_roles:
        blockers.append(Finding("C4", f"differentiated roles {len(roles)} < required {min_roles}", args.charter))

    channels = channel_ids(charter, log_entries)
    min_channels = min_int(contract, "channels")
    if len(channels) < min_channels:
        blockers.append(Finding("C5", f"channels {len(channels)} < required {min_channels}", args.journey_log))

    actual_tasks = task_counts(charter, log_entries)
    required_tasks = task_minimums(contract)
    for key, required in required_tasks.items():
        if actual_tasks.get(key, 0) < required:
            blockers.append(Finding("C6", f"task {key} {actual_tasks.get(key, 0)} < required {required}", args.journey_log))

    actual_surfaces = {str(entry.get("surface", "")).strip().lower() for entry in log_entries}
    missing_surfaces = sorted(surfaces_required(contract) - actual_surfaces)
    if missing_surfaces:
        blockers.append(Finding("C7", f"missing required surfaces in journey log: {', '.join(missing_surfaces)}", args.journey_log))

    cross_count, cross_objects = cross_surface_count(log_entries)
    metadata["cross_surface_objects"] = cross_objects
    min_cross = min_int(contract, "cross_surface_objects")
    if cross_count < min_cross:
        blockers.append(Finding("C8", f"cross-surface objects {cross_count} < required {min_cross}", args.journey_log))

    min_provider = min_int(contract, "provider_backed_sessions")
    live_count = provider_live_count(provider_attempt)
    boundary = provider_attempt.get("boundary") if isinstance(provider_attempt, dict) else None
    if live_count < min_provider:
        if boundary:
            blockers.append(
                Finding("C9", "live provider behavior is blocked; release-grade QA must be reported as BLOCKED, not PASS", args.provider_attempt)
            )
        else:
            blockers.append(Finding("C9", f"live provider-backed sessions {live_count} < required {min_provider}", args.provider_attempt))

    artifact_count = artifact_reuse_count(charter, log_entries, roots)
    min_artifacts = min_int(contract, "artifacts_used_later")
    if artifact_count < min_artifacts:
        blockers.append(Finding("C10", f"artifacts used later {artifact_count} < required {min_artifacts}", args.charter))

    actual_disruptions = disruption_count(log_entries)
    min_disruptions = min_int(contract, "disruption_probes")
    if actual_disruptions < min_disruptions:
        blockers.append(Finding("C11", f"completed disruption probes {actual_disruptions} < required {min_disruptions}", args.journey_log))

    final_report_path = Path(args.final_report).resolve()
    if not final_report_path.is_file() or final_report_path.stat().st_size == 0:
        blockers.append(Finding("C12", "final verification report is missing or empty", str(final_report_path)))
        report_text = ""
    else:
        report_text = final_report_path.read_text(encoding="utf-8")
        if "<" in report_text and ">" in report_text and re.search(r"<[^>\n]+>", report_text):
            blockers.append(Finding("C12", "final report still contains template placeholders", str(final_report_path)))
        if not re.search(r"\b(PASS|FAIL|BLOCKED)\b", report_text):
            blockers.append(Finding("C12", "final report has no explicit PASS/FAIL/BLOCKED verdict", str(final_report_path)))

    smoke_paths = set(re.findall(r"Smoke[^:\n]*:\s*([^\s]+)", report_text, flags=re.IGNORECASE))
    behavioral_paths = set(re.findall(r"Behavioral[^:\n]*:\s*([^\s]+)", report_text, flags=re.IGNORECASE))
    overlap = smoke_paths.intersection(behavioral_paths)
    if overlap:
        blockers.append(Finding("C13", "smoke evidence overlaps behavioral evidence", ", ".join(sorted(overlap))))

    verify_candidates = final_verify_paths(report_text)
    verify_existing = [path for path in verify_candidates if evidence_exists(path, roots)]
    if not verify_existing:
        for entry in log_entries:
            if str(entry.get("action", "")).lower() in {"verify_gate", "final_verify", "make_verify"}:
                path_text = str(entry.get("evidence_path", "")).strip()
                if evidence_exists(path_text, roots):
                    verify_existing.append(path_text)
        if not verify_existing:
            blockers.append(Finding("C14", "final make verify evidence is missing", str(final_report_path)))

    if args.api_base_url:
        warnings.append(Finding("C15", "API deep equality check is not implemented; rely on captured CLI/API/Web/runtime evidence", args.api_base_url))

    return blockers, warnings, metadata


def write_reports(qa_output_path: Path, blockers: list[Finding], warnings: list[Finding], metadata: dict[str, Any]) -> tuple[Path, Path]:
    qa_root = qa_output_path / "qa" if (qa_output_path / "qa").is_dir() else qa_output_path
    qa_root.mkdir(parents=True, exist_ok=True)
    verdict = "fail" if blockers else "warn" if warnings else "pass"
    json_path = qa_root / "qa-audit-report.json"
    md_path = qa_root / "qa-audit-report.md"
    payload = {
        "schema_version": 1,
        "verdict": verdict,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "blockers": [finding.__dict__ for finding in blockers],
        "warnings": [finding.__dict__ for finding in warnings],
        "metadata": metadata,
    }
    json_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    lines = ["# QA Audit Report", "", f"- Verdict: {verdict.upper()}", f"- Blockers: {len(blockers)}", f"- Warnings: {len(warnings)}"]
    if blockers:
        lines.extend(["", "## Blockers"])
        for finding in blockers:
            lines.append(f"- {finding.check}: {finding.message} ({finding.evidence})")
    if warnings:
        lines.extend(["", "## Warnings"])
        for finding in warnings:
            lines.append(f"- {finding.check}: {finding.message} ({finding.evidence})")
    md_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return json_path, md_path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--qa-output-path", required=True)
    parser.add_argument("--scenario-contract", default="")
    parser.add_argument("--charter", default="")
    parser.add_argument("--journey-log", default="")
    parser.add_argument("--provider-attempt", default="")
    parser.add_argument("--final-report", default="")
    parser.add_argument("--api-base-url", default="")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--warn-only", action="store_true")
    parser.add_argument("--explain", action="store_true")
    args = parser.parse_args()

    qa_output_path = Path(args.qa_output_path).resolve()
    qa_root = qa_output_path / "qa" if (qa_output_path / "qa").is_dir() else qa_output_path
    args.scenario_contract = args.scenario_contract or str(qa_root / "scenario-contract.json")
    args.charter = args.charter or str(qa_root / "behavioral-scenario-charter.yaml")
    args.journey_log = args.journey_log or str(qa_root / "journey-log.jsonl")
    args.provider_attempt = args.provider_attempt or str(qa_root / "provider-attempt.json")
    args.final_report = args.final_report or str(qa_root / "verification-report.md")

    blockers, warnings, metadata = audit(args)
    json_path, md_path = write_reports(qa_output_path, blockers, warnings, metadata)
    print(f"QA audit report: {json_path}")
    print(f"QA audit summary: {md_path}")
    if args.explain and blockers:
        for finding in blockers:
            print(f"{finding.check}: {finding.message}")
    if args.warn_only:
        return 0 if not warnings else 1
    if blockers:
        return 2
    if warnings:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
