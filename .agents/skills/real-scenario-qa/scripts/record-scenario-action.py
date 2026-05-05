#!/usr/bin/env python3
"""Append one structured QA scenario action to journey-log.jsonl."""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import sys


def parse_json_arg(value: str, label: str) -> object:
    if not value:
        return []
    try:
        return json.loads(value)
    except json.JSONDecodeError as err:
        raise ValueError(f"{label} must be valid JSON: {err}") from err


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--log", required=True, help="Path to qa/journey-log.jsonl")
    parser.add_argument("--surface", required=True, help="cli, api, web, runtime, provider, or other surface")
    parser.add_argument("--actor", required=True, help="Operator, agent, or system actor id")
    parser.add_argument("--action", required=True, help="Action performed")
    parser.add_argument("--target", required=True, help="Target object or route")
    parser.add_argument("--ids", default="[]", help="JSON array of relevant persisted IDs")
    parser.add_argument("--evidence-path", default="", help="Path to command output, screenshot, transcript, or artifact")
    parser.add_argument("--channel", default="", help="Channel id when applicable")
    parser.add_argument("--task-kind", default="", choices=["", "root", "subtask", "dependency", "run"])
    parser.add_argument("--probe-id", default="", help="Disruption probe id when applicable")
    parser.add_argument("--phase", default="", help="trigger, observed, or result for disruption probes")
    args = parser.parse_args()

    try:
        ids = parse_json_arg(args.ids, "--ids")
    except ValueError as err:
        print(err, file=sys.stderr)
        return 2
    if not isinstance(ids, list):
        print("--ids must decode to a JSON array", file=sys.stderr)
        return 2

    log_path = Path(args.log)
    log_path.parent.mkdir(parents=True, exist_ok=True)
    entry = {
        "ts": datetime.now(timezone.utc).isoformat(),
        "surface": args.surface.strip().lower(),
        "actor": args.actor.strip(),
        "action": args.action.strip(),
        "target": args.target.strip(),
        "ids": ids,
        "evidence_path": args.evidence_path.strip(),
    }
    if args.channel:
        entry["channel"] = args.channel.strip()
    if args.task_kind:
        entry["task_kind"] = args.task_kind
    if args.probe_id:
        entry["probe_id"] = args.probe_id.strip()
    if args.phase:
        entry["phase"] = args.phase.strip().lower()

    with log_path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(entry, sort_keys=True) + "\n")
    print(str(log_path))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
