# Performance Analysis Report - regengo v0.3.0

## Executive Summary

After the Thompson NFA and TDFA merge, regengo shows strong performance improvements for most patterns but has notable regressions in specific cases. This report identifies the root causes.

## Benchmark Results Overview

### Where regengo WINS (2-14x faster)

| Pattern | stdlib | regengo | regengo_reuse | Speedup |
|---------|--------|---------|---------------|---------|
| DateCapture | 99ns | 19ns | 7ns | **5x / 14x** |
| EmailCapture | 245ns | 102ns | 91ns | **2.4x / 2.7x** |
| EmailMatch | 1528ns | 490ns | - | **3x** |
| Greedy | 733ns | 461ns | - | **1.6x** |
| Lazy | 1237ns | 462ns | - | **2.7x** |
| MultiDate | 690ns | 146ns | 71ns | **4.7x / 9.6x** |
| TDFALogParser | 406ns | 120ns | 108ns | **3.4x / 3.8x** |
| TDFAComplexURL | 479ns | 259ns | 232ns | **1.8x / 2x** |
| TDFANestedWord_0 | 298ns | 153ns | 145ns | **2x** |
| TDFAPathological_0 | 210ns | 112ns | 103ns | **1.9x / 2x** |

### Where regengo LOSES

| Pattern | stdlib | regengo | regengo_reuse | Factor |
|---------|--------|---------|---------------|--------|
| **URLCapture** | 192ns | 1357ns | 1338ns | **7x slower** |
| **TDFASemVer** | 128ns | 764ns | 816ns | **6x slower** |
| TDFAPathological_3 (no match) | 1111ns | 3065ns | 3043ns | **3x slower** |
| TDFANestedWord (reuse allocs) | - | - | 48-192 B/op | **should be 0** |

---

## Root Cause Analysis

### 1. URLCapture - 7x Slower

**Pattern:** `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`

**Engine Used:** TDFA for FindString (13 states)

**Root Causes:**

#### RCA-1: Massive Static Data Tables Re-initialization
The TDFA implementation generates enormous lookup tables as **local variables**:
- `transitions [13][128]int`
- `tagActionCount [13][128]int`
- `tagActionTags [13][128][2]int`
- `tagActionOffsets [13][128][2]int`

**Total: ~80KB of stack/heap data initialized per function call**

Because these are defined inside the function scope (e.g., `transitions := [...]int{...}`), Go re-initializes them on every call. For short inputs, this `memcpy` or element-wise initialization dominates execution time.

#### RCA-2: Unanchored Pattern Outer Loop
```go
for searchStart := 0; searchStart <= l; searchStart++ {
    // Try matching from each position
}
```
For an 18-character URL that matches at position 0, we still enter the outer loop once. But the overhead of resetting state and checking `isAccept[state]` at each character adds up.

#### RCA-3: Tag Action Processing Per Character
```go
actionCount := tagActionCount[state][c]
for i := 0; i < actionCount; i++ {
    tag := tagActionTags[state][c][i]
    offset := tagActionOffsets[state][c][i]
    tags[tag] = pos + offset
}
```
This loop runs for every character, even when `actionCount` is 0. The array lookups add overhead.

#### Why stdlib is faster:
- stdlib uses optimized prefix/literal searching (Boyer-Moore-like) for patterns starting with literals like `"http"`
- stdlib's VM-based interpreter has much smaller memory footprint
- For simple patterns matching at position 0, stdlib's backtracking is actually efficient

---

### 2. TDFASemVer - 6x Slower

**Pattern:** `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?`

**Engine Used:** Thompson NFA for MatchString (uses `map[int]uint64`), TDFA for FindString

**Root Causes:**

#### RCA-1: Map Lookup in Hot Path (Thompson NFA)
```go
epsilonClosures := map[int]uint64{2: ..., 5: ..., 7: ..., ...}
// In inner loop:
next |= epsilonClosures[2]  // Map lookup on every state transition!
```

Go maps have hash computation overhead. For 9 epsilon closure lookups per character processed, this becomes a significant bottleneck.

**Fix:** Replace `map[int]uint64` with `[N]uint64` array for O(1) indexed access.

#### RCA-2: Short Input Amplifies Overhead
Test input: `"1.0.0"` (5 characters)

For a 5-character input that matches at position 0:
- Setup cost (table initialization) dominates
- Per-character overhead is multiplied by the small character count
- stdlib's simple backtracking is highly optimized for these trivial cases

#### RCA-3: Same TDFA Table Overhead as URLCapture
- `transitions [10][128]int`
- `tagActionCount [10][128]int`
- `tagActionTags [10][128][1]int`
- `tagActionOffsets [10][128][1]int`

~50KB of static data for a pattern matching `"1.0.0"`.

---

### 3. TDFAPathological_3 (No Match Case) - 3x Slower

**Test Input:** A string that doesn't match the pattern

**Root Cause:**

For non-matching inputs, we must try every starting position:
```go
for searchStart := 0; searchStart <= l; searchStart++ {
    // Try, fail, retry from next position
}
```

This is O(n) iterations of the outer loop, each doing O(1) work until failure. stdlib can often reject non-matches faster using prefix optimizations or fail-fast heuristics.

---

### 4. TDFANestedWord Reuse Allocations

**Expected:** 0 B/op in reuse mode
**Actual:** 48-192 B/op

**Root Cause:**

The `visited` bit-vector for memoization is allocated inside the function:
```go
visitedSize := 14 * (l + 1)
visited := make([]uint32, (visitedSize+31)/32)  // ALLOCATION!
```

This array size depends on input length, so it can't be pooled or pre-allocated.

**Calculation:**
- For input length 85: `14 × 86 = 1204 bits → 38 uint32s → 152 bytes` (matches ~192 B/op with overhead)
- For input length 24: `14 × 25 = 350 bits → 11 uint32s → 44 bytes` (matches ~48 B/op)

---

## Improvement Recommendations

### High Priority (Robust Fixes)

1. **Move Static Tables to Package Scope** (✅ DONE)
   - **Problem:** Tables are currently local variables, causing re-initialization on every call.
   - **Solution:** Generate tables as `var table_PatternName = [...]` at package level.
   - **Result:** URLCapture improved from ~1300ns to ~70ns (18x speedup). TDFASemVer improved from ~764ns to ~130ns (5.8x speedup).
   - **Status:** Implemented in PR #17.

2. **Replace Map with Array for Epsilon Closures**
   - **Problem:** `map[int]uint64` is slow in hot paths.
   - **Solution:** Use `[maxStates]uint64` array.
   - **Expected:** 2-3x speedup for Thompson NFA patterns.
   - **Effort:** Low.

3. **Add Prefix/Literal Optimization**
   - **Problem:** Unanchored patterns try every position.
   - **Solution:** If pattern starts with literal (e.g., `"http"`), use `strings.Index` to jump to candidates.
   - **Expected:** Major speedup for URLCapture-like patterns.
   - **Effort:** Medium.

### Medium Priority

4. **Sync.Pool for Visited Arrays**
   - **Problem:** `visited` array depends on input length.
   - **Solution:** Use a `sync.Pool` of `[]uint32` buffers. Resize/grow as needed.
   - **Expected:** Zero allocations for all inputs.
   - **Effort:** Medium.

5. **Compress TDFA Tables**
   - **Problem:** Sparse tables waste cache.
   - **Solution:** Use sparse representation (e.g., CSR or list of transitions) for mostly-empty tables.
   - **Expected:** Reduced memory pressure.
   - **Effort:** High.

### Low Priority

6. **Specialized Code Generation**
   - Generate specialized code for anchored patterns (no outer loop).
   - Generate specialized code for patterns starting with literals.

---

## Conclusion

The v0.3.0 release prioritized correctness and catastrophic backtracking protection over raw speed for simple patterns. The main regressions come from:

1. **Re-initialization of large tables** - TDFA tables are local variables, causing massive copy overhead on every call.
2. **Map lookups in hot paths** - Thompson NFA uses maps instead of arrays.
3. **No prefix optimization** - We don't skip ahead for literal prefixes like stdlib does.
4. **Input-dependent allocations** - Memoization bit-vectors need pooling.

For patterns with catastrophic backtracking risk (like `(a+)+b`), regengo is infinitely faster than stdlib (which would hang). For simple patterns on short inputs, stdlib's hand-optimized VM wins.

The recommended path forward is to **move tables to package scope**, add prefix optimization, and fix the map-vs-array issue. These changes should close most of the gap without sacrificing the safety guarantees.

## v0.3.1 Optimization Results

### Step 1: Move Static Tables to Package Scope

**Implementation:**
We modified the TDFA compiler to generate static lookup tables (`transitions`, `tagActionCount`, `tagActionTags`, `tagActionOffsets`) as package-level variables instead of local variables inside the `FindString` function. This eliminates the massive initialization overhead (memcpy) that was occurring on every function call.

**Benchmark Comparison (MacBook Pro M3 Max):**

| Pattern | v0.3.0 (Local Tables) | v0.3.1 (Package Tables) | Speedup |
|---------|-----------------------|-------------------------|---------|
| **URLCapture** | 1357 ns/op | **73 ns/op** | **18.5x** |
| **TDFASemVer** | 764 ns/op | **131 ns/op** | **5.8x** |
| DateCapture | 19 ns/op | 19 ns/op | 1x (No change) |
| EmailCapture | 102 ns/op | 68 ns/op | 1.5x |
| MultiDate | 146 ns/op | 146 ns/op | 1x |

**Analysis:**
- **URLCapture** is now faster than the standard library (which was ~192ns).
- **TDFASemVer** is now comparable to the standard library (~128ns).
- The overhead of table initialization was indeed the primary bottleneck for short inputs.
- Zero-allocation performance is maintained.

