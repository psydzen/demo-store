"""Run the Semgrep rules against the fixture corpus and score the result.

Usage: python3 scripts/check_semgrep.py [--pro] [--require-nonempty]

By default the run is strict: every `// ruleid:` case must be found and no
`// ok:` case may be flagged. That is the local contract, and only the Pro
engine meets it — the free engine cannot reach across files.

`--require-nonempty` is the contract for CI: each rule has to find at least one
of its own cases. It answers the question the job is named after — is the rule
still doing anything at all — without failing on the differences between the
two engines.
"""

import json
import subprocess
import sys

import fixtures


def report_nonempty(found, expected):
    """Check every rule still matches at least one of its own fixtures."""
    broken = []
    for rule in fixtures.RULES:
        want = {f for f in expected if f[0] == rule}
        got = {f for f in found if f[0] == rule}
        hit = len(want & got)
        status = "ok" if hit else "FINDS NOTHING"
        print(f"  {rule}: {hit}/{len(want)} own cases  {status}")
        if not hit:
            broken.append(rule)
    return broken


def main():
    pro = "--pro" in sys.argv
    nonempty = "--require-nonempty" in sys.argv
    cmd = ["semgrep", "--metrics=off", "--config", "rules/semgrep",
           "--json", "--quiet", fixtures.CORPUS]
    if pro:
        cmd.insert(1, "--pro")

    proc = subprocess.run(cmd, capture_output=True, text=True)
    if not proc.stdout.strip():
        print(proc.stderr[-2000:], file=sys.stderr)
        return 1
    data = json.loads(proc.stdout)
    errors = data.get("errors", [])
    for err in errors:
        print("semgrep error:", str(err)[:300], file=sys.stderr)
    if errors:
        # An engine that refused to start reports no findings, which would look
        # exactly like six rules that all stopped working.
        print("semgrep did not run cleanly; not scoring the corpus", file=sys.stderr)
        return 1

    found = {(r["check_id"].split(".")[-1], r["path"], r["start"]["line"])
             for r in data["results"]}

    markers = fixtures.load_markers()
    expected, forbidden = fixtures.expectations(markers)
    label = "semgrep" + (" --pro" if pro else " (oss)")

    if nonempty:
        print(f"{label}: every rule must match at least one of its own fixtures")
        broken = report_nonempty(found, expected)
        return 2 if broken else 0

    result = fixtures.score(found, expected, forbidden)
    fixtures.report(label, result, expected)
    return 0 if not result["missed"] and not result["on_ok"] else 2


if __name__ == "__main__":
    sys.exit(main())
