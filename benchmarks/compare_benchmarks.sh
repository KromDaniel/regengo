#!/bin/bash
# compare_benchmarks.sh - Compare benchmark results between two files

set -e

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
    echo "Usage: $0 <new_results_file> [old_results_file]"
    echo ""
    echo "Compare benchmark results between files."
    echo "If old_results_file is not provided, uses BENCHMARK_RESULTS.txt"
    echo ""
    echo "Examples:"
    echo "  $0 /tmp/new_results.txt"
    echo "  $0 /tmp/new_results.txt BENCHMARK_RESULTS.txt"
    exit 1
fi

NEW_FILE="$1"
OLD_FILE="${2:-BENCHMARK_RESULTS.txt}"

if [ ! -f "$NEW_FILE" ]; then
    echo "Error: New results file not found: $NEW_FILE"
    exit 1
fi

if [ ! -f "$OLD_FILE" ]; then
    echo "Error: Old results file not found: $OLD_FILE"
    exit 1
fi

echo "=========================================="
echo "  BENCHMARK COMPARISON"
echo "=========================================="
echo ""
echo "Old: $OLD_FILE"
echo "New: $NEW_FILE"
echo ""

# Extract and compare overall summary
echo "=========================================="
echo "  OVERALL SUMMARY"
echo "=========================================="
echo ""
echo "--- OLD ---"
grep "OVERALL SUMMARY" -A 4 "$OLD_FILE" | tail -4
echo ""
echo "--- NEW ---"
grep "OVERALL SUMMARY" -A 4 "$NEW_FILE" | tail -4
echo ""

# Compare by category
for cat in "simple" "complex" "very_complex"; do
    echo "=========================================="
    echo "  CATEGORY: $cat"
    echo "=========================================="
    echo ""
    echo "--- OLD ---"
    grep "Category: $cat" -A 4 "$OLD_FILE" | head -5
    echo ""
    echo "--- NEW ---"
    grep "Category: $cat" -A 4 "$NEW_FILE" | head -5
    echo ""
done

# Extract key numbers for easy comparison
echo "=========================================="
echo "  KEY METRICS COMPARISON"
echo "=========================================="
echo ""

extract_metric() {
    local file=$1
    local pattern=$2
    grep "$pattern" "$file" | head -1
}

echo "Overall Performance:"
echo "  OLD: $(extract_metric "$OLD_FILE" "Avg time:.*overall")"
echo "  NEW: $(extract_metric "$NEW_FILE" "Avg time:.*overall")"
echo ""

echo "Win Rate:"
OLD_WINS=$(grep "Regengo faster:" "$OLD_FILE" | grep "OVERALL" -B 1 | tail -1 | awk '{print $3}')
NEW_WINS=$(grep "Regengo faster:" "$NEW_FILE" | grep "OVERALL" -B 1 | tail -1 | awk '{print $3}')
OLD_TOTAL=$(grep "Regengo faster:" "$OLD_FILE" | grep "OVERALL" -B 1 | tail -1 | awk '{sum=$3+$7} END {print sum}')
NEW_TOTAL=$(grep "Regengo faster:" "$NEW_FILE" | grep "OVERALL" -B 1 | tail -1 | awk '{sum=$3+$7} END {print sum}')

echo "  OLD: $OLD_WINS wins"
echo "  NEW: $NEW_WINS wins"
echo ""

# Calculate change
if [ -n "$OLD_WINS" ] && [ -n "$NEW_WINS" ]; then
    DIFF=$((NEW_WINS - OLD_WINS))
    if [ $DIFF -gt 0 ]; then
        echo "  ✅ Performance improved: +$DIFF patterns faster"
    elif [ $DIFF -lt 0 ]; then
        echo "  ⚠️  Performance regression: $DIFF patterns faster"
    else
        echo "  ⚪ No change in win rate"
    fi
fi

echo ""
echo "=========================================="
echo "  END OF COMPARISON"
echo "=========================================="
