"""Turn engine output plus /usr/bin/time -l logs into comparable numbers."""

import json
import os
import re
import sys
from collections import Counter


def timings(out_dir, label):
    """Median wall time in seconds and peak RSS in MB from three runs."""
    reals, peaks = [], []
    for i in (1, 2, 3):
        path = os.path.join(out_dir, f"{label}.time.{i}")
        if not os.path.exists(path):
            continue
        text = open(path, encoding="utf-8", errors="replace").read()
        m = re.search(r"([\d.]+)\s+real", text)
        if m:
            reals.append(float(m.group(1)))
        m = re.search(r"(\d+)\s+maximum resident set size", text)
        if m:
            peaks.append(int(m.group(1)) / 1024 / 1024)
    reals.sort()
    peaks.sort()
    return (reals[len(reals) // 2] if reals else None,
            peaks[len(peaks) // 2] if peaks else None)


def semgrep_findings(path):
    data = json.load(open(path, encoding="utf-8"))
    return [(r["check_id"].split(".")[-1], r["path"], r["start"]["line"])
            for r in data["results"]]


def main():
    out_dir, label = sys.argv[1], sys.argv[2]
    findings = semgrep_findings(os.path.join(out_dir, f"{label}.json"))
    per_rule = Counter(rule for rule, _, _ in findings)
    real, peak = timings(out_dir, label)

    print(f"== {label} ==")
    print(f"findings: {len(findings)}")
    for rule, n in sorted(per_rule.items()):
        print(f"  {n:3d}  {rule}")
    print(f"wall time (median of 3): {real:.1f}s" if real else "wall time: n/a")
    print(f"peak RSS: {peak:.0f} MB" if peak else "peak RSS: n/a")
    print()
    for rule, path, line in sorted(findings):
        print(f"  {rule}\t{path}:{line}")


if __name__ == "__main__":
    main()
