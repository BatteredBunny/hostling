#!/usr/bin/env python3
"""Format changed screenshots into a single, replaceable PR comment."""
import json
import sys
from collections import defaultdict
from pathlib import Path

report_dir = Path(sys.argv[1])
before_commit, after_commit = sys.argv[2:4]
urls = dict(line.split("\t", 1) for line in Path(sys.argv[4]).read_text().splitlines())
report = json.loads((report_dir / "changes.json").read_text())
counts = report["counts"]
changes = report["shots"]

print(f"## Screenshots\n\nCompared `{before_commit}` → `{after_commit}`\n")
if not changes:
    print("No visual changes detected.")
    sys.exit(0)

print(" · ".join(f"{counts[key]} {key}" for key in ["changed", "added", "removed", "unchanged"]) + "\n")
groups = defaultdict(list)
for shot in changes:
    groups[shot["category"]].append(shot)
for category, shots in groups.items():
    print(f"<details>\n<summary>{category.capitalize()} ({len(shots)})</summary>\n")
    for shot in shots:
        print(f"**{shot['title']} — {shot['status']}**\n")
        columns = [key for key in ["before", "after", "diff"] if shot.get(key)]
        print("| " + " | ".join(key.capitalize() for key in columns) + " |")
        print("| " + " | ".join(":-:" for _ in columns) + " |")
        print("| " + " | ".join(f"![{shot['name']} {key}]({urls[shot[key]]})" for key in columns) + " |\n")
    print("</details>\n")
print("<sub>Images expire after 3 hours. Re-run the workflow to regenerate.</sub>")
