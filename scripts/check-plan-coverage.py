#!/usr/bin/env python3
"""Check FR ownership from the PRD through the phase plan frontmatter."""

from collections import Counter, defaultdict
from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
FR = re.compile(r"FR-\d+")
SOURCE_PATH = re.compile(
    r"(?<![\w./-])@?((?:\.planning|docs|internal|cmd|apps|deploy|scripts|"
    r"protocol|test|sdk|spikes)/[\w./-]+\.[\w]+)"
)


def table_ids(path: Path) -> Counter[str]:
    return Counter(
        match.group(1)
        for line in path.read_text().splitlines()
        if (match := re.match(r"^\| (FR-\d+) \|", line))
    )


def main() -> int:
    errors: list[str] = []
    prd = table_ids(ROOT / "docs/PRD.md")
    requirements = table_ids(ROOT / ".planning/REQUIREMENTS.md")
    if prd != requirements:
        errors.append(f"PRD and requirements IDs differ: {prd - requirements}, {requirements - prd}")
    for label, ids in (("PRD", prd), ("requirements", requirements)):
        errors.extend(f"duplicate {label} ID: {fr}" for fr, count in ids.items() if count != 1)

    ownership: dict[str, str] = {}
    for line in (ROOT / ".planning/REQUIREMENTS.md").read_text().splitlines():
        match = re.match(r"^\| (FR-\d+) \|.*\| Phase (\d\d) \|", line)
        if match:
            ownership[match.group(1)] = match.group(2)
    if set(ownership) != set(requirements):
        errors.append(f"requirements missing phase owner: {sorted(set(requirements) - set(ownership))}")

    roadmap = (ROOT / ".planning/ROADMAP.md").read_text()
    roadmap_owners: dict[str, str] = {}
    for phase, details in re.findall(
        r"^### Phase (\d\d):[^\n]*\n(.*?)(?=^### Phase |\Z)", roadmap, re.M | re.S
    ):
        match = re.search(r"^\*\*Requirements\*\*: (.*)$", details, re.M)
        if match:
            for fr in FR.findall(match.group(1)):
                if fr in roadmap_owners:
                    errors.append(f"roadmap assigns {fr} twice")
                roadmap_owners[fr] = phase
    if roadmap_owners != ownership:
        errors.append("roadmap phase ownership differs from requirements register")

    plans: dict[str, set[str]] = defaultdict(set)
    produced: dict[str, set[str]] = defaultdict(set)
    read_first: list[tuple[str, str]] = []
    plan_count = 0
    for path in (ROOT / ".planning/phases").rglob("*-PLAN.md"):
        phase = path.name[:2]
        if not ("03" <= phase <= "17"):
            continue
        plan_count += 1
        parts = path.read_text().split("---", 2)
        if len(parts) != 3:
            errors.append(f"missing frontmatter: {path.relative_to(ROOT)}")
            continue
        match = re.search(r"^requirements: (.*)$", parts[1], re.M)
        if not match:
            errors.append(f"missing requirements: {path.relative_to(ROOT)}")
            continue
        for fr in FR.findall(match.group(1)):
            plans[fr].add(phase)
            if ownership.get(fr) != phase:
                errors.append(f"{path.relative_to(ROOT)} assigns {fr} to wrong phase")
        for source in SOURCE_PATH.findall(parts[1]):
            produced[source].add(path.name[:5])
        for files in re.findall(r"<files>(.*?)</files>", parts[2], re.S):
            for source in SOURCE_PATH.findall(files):
                produced[source].add(path.name[:5])
        for reading in re.findall(r"<read_first>(.*?)</read_first>", parts[2], re.S):
            read_first.extend((path.name[:5], source) for source in SOURCE_PATH.findall(reading))
    for fr, phase in sorted(ownership.items()):
        if "03" <= phase <= "17" and phase not in plans[fr]:
            errors.append(f"{fr} has no Phase {phase} plan")
    for consumer, source in read_first:
        if (ROOT / source).exists():
            continue
        producers = produced.get(source, set())
        if not producers:
            errors.append(f"{consumer} reads missing unowned source {source}")
        elif min(producers) > consumer:
            errors.append(f"{consumer} reads {source} before producer {min(producers)}")

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(
        f"PASS: {len(prd)} FRs, {len(ownership)} phase owners, "
        f"{plan_count} Phase 03–17 plans, {len(read_first)} source pointers"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
