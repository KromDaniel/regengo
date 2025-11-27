package compiler

// TNFA (Tagged NFA) generator is a placeholder for future implementation.
// Currently, capture functions use backtracking with memoization which
// provides polynomial time guarantees for patterns with catastrophic
// backtracking risk.
//
// Full TNFA implementation would track capture positions along with NFA
// states using a thread-based simulation, similar to RE2's approach.
//
// The current approach (Thompson NFA for Match + memoized backtracking
// for captures) provides:
// - O(n*m) for MatchString/MatchBytes (Thompson NFA)
// - O(n*m*k) for FindString with captures (memoized backtracking)
//
// where n = input length, m = NFA states, k = captures
