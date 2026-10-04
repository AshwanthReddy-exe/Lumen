#!/usr/bin/env python3
"""Validate task outcome contracts and exact file ownership in GSD plans."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLAN_RE = re.compile(r"\d{2}-\d{2}-PLAN\.md$")
TASK_RE = re.compile(r"<task\b[^>]*>.*?</task>", re.DOTALL)


def normalize(value: str) -> str:
    return " ".join(value.split()).casefold()


def files_modified(plan: str, path: Path) -> set[str]:
    parts = plan.split("---", 2)
    if len(parts) < 3:
        raise ValueError("missing YAML frontmatter")
    lines = parts[1].splitlines()
    result: set[str] = set()
    active = False
    for line in lines:
        if re.match(r"^files_modified:", line):
            active = True
            value = line.split(":", 1)[1].strip()
            if value.startswith("["):
                result.update(
                    item.strip().strip("'\"")
                    for item in value.strip("[]").split(",")
                    if item.strip()
                )
            elif value.startswith("- "):
                result.add(value[2:].strip().strip("'\""))
            elif value:
                raise ValueError(f"unsupported files_modified value: {value}")
            continue
        if active:
            if line.strip() and not line.startswith((" ", "\t")):
                break
            match = re.match(r"^\s+-\s+([^\n]+)", line)
            if match:
                result.add(match.group(1).strip().strip("'\""))
    if not result:
        raise ValueError("files_modified is empty or unsupported")
    return result


def plan_files(plan: str) -> set[str]:
    return {
        item.strip()
        for block in TASK_RE.findall(plan)
        for field in re.findall(r"<files>(.*?)</files>", block, re.DOTALL)
        for item in field.split(",")
        if item.strip()
    }


def is_superseded(plan: str) -> bool:
    parts = plan.split("---", 2)
    return len(parts) >= 3 and bool(
        re.search(r"^status:\s*superseded\s*$", parts[1], re.MULTILINE)
    )


def task_text(block: str, tag: str) -> str:
    match = re.search(rf"<{tag}>(.*?)</{tag}>", block, re.DOTALL)
    return match.group(1).strip() if match else ""


def main() -> int:
    plans = sorted(
        path
        for path in (ROOT / ".planning/phases").glob("*/*-PLAN.md")
        if PLAN_RE.fullmatch(path.name)
    )
    errors: list[str] = []
    phase_counts: dict[str, int] = {}
    executable_count = 0

    for path in plans:
        text = path.read_text(encoding="utf-8")
        try:
            declared = files_modified(text, path)
        except ValueError as error:
            errors.append(f"{path.relative_to(ROOT)}: {error}")
            continue
        if not is_superseded(text):
            owned = plan_files(text)
            if owned != declared:
                missing = sorted(owned - declared)
                orphaned = sorted(declared - owned)
                errors.append(
                    f"{path.relative_to(ROOT)}: file ownership mismatch; "
                    f"undeclared={missing}; no task owner={orphaned}"
                )

        for block in TASK_RE.findall(text):
            if re.search(r'<task\b[^>]*type="checkpoint:', block):
                continue
            executable_count += 1
            phase = path.name[:2]
            phase_counts[phase] = phase_counts.get(phase, 0) + 1
            name = task_text(block, "name") or "unnamed task"
            label = f"{path.relative_to(ROOT)} ({name})"
            failure = task_text(block, "fails_when")
            outcome = task_text(block, "result_contract")
            behavior = normalize(task_text(block, "behavior"))
            done = normalize(task_text(block, "done"))
            if not failure:
                errors.append(f"{label}: missing fails_when")
            elif normalize(failure) in {behavior, done}:
                errors.append(f"{label}: fails_when merely repeats acceptance")
            if not outcome:
                errors.append(f"{label}: missing result_contract")
            else:
                upper = outcome.upper()
                if "PASS" not in upper or "FAIL" not in upper:
                    errors.append(f"{label}: result_contract must define PASS and FAIL")
                if "BLOCKED" not in upper and "UNSUPPORTED" not in upper:
                    errors.append(
                        f"{label}: result_contract must distinguish a blocked or unsupported gate"
                    )
            verify = task_text(block, "verify")
            automated = task_text(verify, "automated")
            if not automated:
                errors.append(f"{label}: missing runnable automated check")

    if errors:
        for error in errors:
            print(f"FAIL: {error}", file=sys.stderr)
        print(f"FAILED: {len(errors)} issue(s) across {len(plans)} plans", file=sys.stderr)
        return 1

    breakdown = ", ".join(
        f"{phase}={count}" for phase, count in sorted(phase_counts.items())
    )
    print(
        f"PASS: {executable_count} executable tasks across {len(plans)} plans; "
        f"task contracts, automated checks, and file ownership are complete ({breakdown})"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
