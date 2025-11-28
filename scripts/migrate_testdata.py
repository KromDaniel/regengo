#!/usr/bin/env python3
"""
Migrate existing testdata.json to include labels.

This script reads the current testdata.json, analyzes each pattern using
regengo -analyze, and writes out the updated testdata.json with labels.
"""

import hashlib
import json
import os
import subprocess
import sys

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REGENGO_BIN = os.path.join(PROJECT_ROOT, "bin", "regengo")
TESTDATA_PATH = os.path.join(PROJECT_ROOT, "e2e", "testdata.json")


def md5_hash(s: str) -> str:
    """Return MD5 hash of a string for stable sorting."""
    return hashlib.md5(s.encode("utf-8")).hexdigest()


def analyze_pattern(pattern: str) -> dict:
    """Call regengo -analyze to get pattern labels."""
    result = subprocess.run(
        [REGENGO_BIN, "-analyze", "-pattern", pattern],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        print(f"Error analyzing pattern '{pattern}': {result.stderr}", file=sys.stderr)
        return None

    return json.loads(result.stdout)


def main():
    if not os.path.exists(REGENGO_BIN):
        print(f"Error: regengo binary not found at {REGENGO_BIN}", file=sys.stderr)
        print("Run 'go build -o bin/regengo ./cmd/regengo' first", file=sys.stderr)
        sys.exit(1)

    # Load existing testdata
    with open(TESTDATA_PATH, "r", encoding="utf-8") as f:
        testdata = json.load(f)

    print(f"Migrating {len(testdata)} test cases...")

    # Process each test case
    migrated = []
    failed = []
    for i, tc in enumerate(testdata):
        pattern = tc["pattern"]
        print(f"[{i+1}/{len(testdata)}] Analyzing: {pattern[:50]}{'...' if len(pattern) > 50 else ''}")

        analysis = analyze_pattern(pattern)
        if analysis is None:
            failed.append(pattern)
            continue

        # Create updated test case
        migrated_tc = {
            "pattern": pattern,
            "inputs": tc.get("inputs", ["example"]),
            "feature_labels": sorted(analysis["feature_labels"]),
            "engine_labels": sorted(analysis["engine_labels"]),
        }
        migrated.append(migrated_tc)

    if failed:
        print(f"\nFailed to analyze {len(failed)} patterns:", file=sys.stderr)
        for p in failed:
            print(f"  - {p}", file=sys.stderr)
        sys.exit(1)

    # Sort by md5 hash for stable ordering
    migrated.sort(key=lambda tc: md5_hash(tc["pattern"]))

    # Write back
    with open(TESTDATA_PATH, "w", encoding="utf-8") as f:
        json.dump(migrated, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print(f"\nMigration complete! Updated {len(migrated)} test cases.")
    print(f"Saved to: {TESTDATA_PATH}")

    # Print label summary
    feature_counts = {}
    engine_counts = {}
    for tc in migrated:
        for label in tc["feature_labels"]:
            feature_counts[label] = feature_counts.get(label, 0) + 1
        for label in tc["engine_labels"]:
            engine_counts[label] = engine_counts.get(label, 0) + 1

    print("\nFeature label distribution:")
    for label, count in sorted(feature_counts.items()):
        print(f"  {label}: {count}")

    print("\nEngine label distribution:")
    for label, count in sorted(engine_counts.items()):
        print(f"  {label}: {count}")


if __name__ == "__main__":
    main()
