#!/usr/bin/env python3
"""
Generate a performance comparison chart from benchmark results.

This script reads Go benchmark output and generates a PNG chart comparing
regengo performance against stdlib regexp.

Usage:
    go test -bench=. -benchmem ./benchmarks/curated/ 2>&1 | python scripts/curated/chart.py

Output:
    assets/curated_benchmark_chart.png

Requirements:
    pip install matplotlib
"""

import re
import sys
from collections import defaultdict

try:
    import matplotlib.pyplot as plt
    import matplotlib.patches as mpatches
except ImportError:
    print("Error: matplotlib is required. Install with: pip install matplotlib", file=sys.stderr)
    sys.exit(1)


def parse_benchmark_output(lines: list[str]) -> dict:
    """Parse benchmark output and extract timing data.

    New nested format:
    - BenchmarkPattern/Category/Input[i]/variant-CPU
    - BenchmarkPattern/Replace/Template[j]/Input[i]/variant-CPU

    Categories: Match, FindFirst, FindAll, Replace
    Variants: stdlib, regengo, regengo_reuse, regengo_append, regengo_runtime
    """
    # Pattern for nested benchmark format
    # Example: BenchmarkDateCapture/Match/Input[0]/stdlib-12    16418577    72.75 ns/op
    pattern = re.compile(
        r"Benchmark(\w+)/(Match|FindFirst|FindAll|Replace)"
        r"(?:/Template\[\d+\])?/Input\[\d+\]/(\w+)-\d+\s+\d+\s+([\d.]+)\s+ns/op"
    )

    # Group by pattern+category -> variant -> list of ns/op values
    results = defaultdict(lambda: defaultdict(list))

    for line in lines:
        match = pattern.search(line)
        if match:
            name, category, variant, ns_op = match.groups()
            ns_op = float(ns_op)

            # Create key combining pattern and category for more granular comparison
            key = f"{name}/{category}"

            # Only collect stdlib and regengo (skip reuse/append variants for chart simplicity)
            if variant in ("stdlib", "regengo"):
                results[key][variant].append(ns_op)

    # Average the results for each benchmark
    averaged = {}
    for key, data in results.items():
        if data["stdlib"] and data["regengo"]:
            averaged[key] = {
                "stdlib": sum(data["stdlib"]) / len(data["stdlib"]),
                "regengo": sum(data["regengo"]) / len(data["regengo"]),
            }

    return averaged


def generate_chart(results: dict, output_path: str = "assets/curated_benchmark_chart.png"):
    """Generate a bar chart comparing stdlib vs regengo performance."""
    # Filter to benchmarks that have both stdlib and regengo results
    valid_benchmarks = {
        name: data for name, data in results.items()
        if "stdlib" in data and "regengo" in data
    }

    if not valid_benchmarks:
        print("No valid benchmark pairs found", file=sys.stderr)
        return False

    # Sort by speedup (largest first)
    sorted_benchmarks = sorted(
        valid_benchmarks.items(),
        key=lambda x: x[1]["stdlib"] / x[1]["regengo"],
        reverse=True
    )

    # Limit to top 8 for readability
    sorted_benchmarks = sorted_benchmarks[:8]

    names = [name for name, _ in sorted_benchmarks]
    stdlib_times = [data["stdlib"] for _, data in sorted_benchmarks]
    regengo_times = [data["regengo"] for _, data in sorted_benchmarks]
    speedups = [s / r for s, r in zip(stdlib_times, regengo_times)]

    # Create figure
    fig, ax = plt.subplots(figsize=(12, 6))

    x = range(len(names))
    width = 0.35

    # Plot bars
    bars1 = ax.bar([i - width/2 for i in x], stdlib_times, width,
                   label='stdlib regexp', color='#e74c3c', alpha=0.8)
    bars2 = ax.bar([i + width/2 for i in x], regengo_times, width,
                   label='regengo', color='#27ae60', alpha=0.8)

    # Add speedup annotations
    for i, (speedup, stdlib_time) in enumerate(zip(speedups, stdlib_times)):
        ax.annotate(f'{speedup:.1f}x',
                   xy=(i, stdlib_time),
                   xytext=(0, 5),
                   textcoords='offset points',
                   ha='center',
                   fontsize=9,
                   fontweight='bold',
                   color='#2c3e50')

    # Formatting
    ax.set_ylabel('Time (ns/op)', fontsize=11)
    ax.set_title('Regengo vs stdlib regexp Performance', fontsize=14, fontweight='bold')
    ax.set_xticks(x)
    ax.set_xticklabels(names, rotation=45, ha='right', fontsize=9)
    ax.legend(loc='upper right')
    ax.set_yscale('log')

    # Add grid
    ax.yaxis.grid(True, alpha=0.3)
    ax.set_axisbelow(True)

    # Tight layout
    plt.tight_layout()

    # Save
    plt.savefig(output_path, dpi=150, bbox_inches='tight')
    print(f"Chart saved to {output_path}")
    return True


def main():
    # Read benchmark output from stdin
    lines = sys.stdin.readlines()

    if not lines:
        print("Usage: go test -bench=. ./benchmarks/... | python scripts/benchmark_chart.py",
              file=sys.stderr)
        sys.exit(1)

    results = parse_benchmark_output(lines)

    if not results:
        print("No benchmark results found in input", file=sys.stderr)
        sys.exit(1)

    print(f"Found {len(results)} benchmark patterns")

    success = generate_chart(results)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
