#!/usr/bin/env python3
"""
Manage e2e test cases for regengo.

Usage:
  ./scripts/manage_e2e_test.py -p '(?P<name>\\w+)@\\w+' -i '["test@example.com", "user@domain"]'
  ./scripts/manage_e2e_test.py -p 'hello' -i 'hello world'
  ./scripts/manage_e2e_test.py -p 'hello'  # Update existing pattern (keeps inputs)
"""

import argparse
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


def parse_inputs(inputs_str: str | None) -> list[str] | None:
    """Parse inputs from command line argument."""
    if not inputs_str:
        return None

    # Try JSON array first
    try:
        parsed = json.loads(inputs_str)
        if isinstance(parsed, list):
            return [str(x) for x in parsed]
    except json.JSONDecodeError:
        pass

    # Treat as single input string
    return [inputs_str]


def analyze_pattern(pattern: str) -> dict:
    """Call regengo -analyze to get pattern labels."""
    if not os.path.exists(REGENGO_BIN):
        print(f"Error: regengo binary not found at {REGENGO_BIN}", file=sys.stderr)
        print("Run 'go build -o bin/regengo ./cmd/regengo' first", file=sys.stderr)
        sys.exit(1)

    result = subprocess.run(
        [REGENGO_BIN, "-analyze", "-pattern", pattern],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        print(f"Error analyzing pattern: {result.stderr}", file=sys.stderr)
        sys.exit(1)

    return json.loads(result.stdout)


def load_testdata() -> list[dict]:
    """Load existing test data from testdata.json."""
    if not os.path.exists(TESTDATA_PATH):
        return []

    with open(TESTDATA_PATH, "r", encoding="utf-8") as f:
        return json.load(f)


def save_testdata(testdata: list[dict]) -> None:
    """Save test data to testdata.json, sorted by md5(pattern)."""
    # Sort by md5 hash of pattern for stable ordering
    testdata.sort(key=lambda tc: md5_hash(tc["pattern"]))

    with open(TESTDATA_PATH, "w", encoding="utf-8") as f:
        json.dump(testdata, f, indent=2, ensure_ascii=False)
        f.write("\n")  # Trailing newline


def find_existing_pattern(testdata: list[dict], pattern: str) -> int | None:
    """Find index of existing test case with the same pattern."""
    for i, tc in enumerate(testdata):
        if tc["pattern"] == pattern:
            return i
    return None


def main():
    parser = argparse.ArgumentParser(
        description="Manage e2e test cases for regengo",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Add a new test case with inputs
  ./scripts/manage_e2e_test.py -p '(?P<name>\\w+)' -i '["hello", "world"]'

  # Add a test case with a single input
  ./scripts/manage_e2e_test.py -p 'hello' -i 'hello world'

  # Update an existing pattern (keeps existing inputs, updates labels)
  ./scripts/manage_e2e_test.py -p 'hello'
""",
    )
    parser.add_argument("-p", "--pattern", required=True, help="Regex pattern")
    parser.add_argument(
        "-i",
        "--inputs",
        help="Test inputs (JSON array string or single string)",
    )
    args = parser.parse_args()

    # 1. Analyze pattern to get labels
    print(f"Analyzing pattern: {args.pattern}")
    analysis = analyze_pattern(args.pattern)

    # 2. Parse inputs
    inputs = parse_inputs(args.inputs)

    # 3. Load existing testdata
    testdata = load_testdata()

    # 4. Find existing test case by pattern
    existing_idx = find_existing_pattern(testdata, args.pattern)

    # 5. Determine inputs to use
    if inputs is not None:
        final_inputs = inputs
    elif existing_idx is not None:
        final_inputs = testdata[existing_idx].get("inputs", ["example"])
        print(f"Using existing inputs: {final_inputs}")
    else:
        final_inputs = ["example"]
        print("Using default inputs: ['example']")

    # 6. Create test case
    test_case = {
        "pattern": args.pattern,
        "inputs": final_inputs,
        "feature_labels": sorted(analysis["feature_labels"]),
        "engine_labels": sorted(analysis["engine_labels"]),
    }

    # 7. Update or add test case
    if existing_idx is not None:
        testdata[existing_idx] = test_case
        print(f"Updated existing test case for pattern: {args.pattern}")
    else:
        testdata.append(test_case)
        print(f"Added new test case for pattern: {args.pattern}")

    # 8. Save testdata.json
    save_testdata(testdata)

    print(f"Feature labels: {test_case['feature_labels']}")
    print(f"Engine labels: {test_case['engine_labels']}")
    print(f"Saved to: {TESTDATA_PATH}")


if __name__ == "__main__":
    main()
