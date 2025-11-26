# Kubernetes Transport Implementation Plan - Review 9

## Review Request

Please review the implementation plan at `/home/vlad/Repos/mutagen/implementations/kubernetes-transport.md` for the Kubernetes transport layer in Mutagen.

## Review Focus Areas

### 1. Previous Review Items (Review 8)

Verify the following issues from review-8 have been adequately addressed:

1. **Duplicate `getAvailableContexts` function** (Major): Confirm only one definition exists and the integration test file will compile.

2. **Copy error classification alignment** (Major): Verify that `Copy()` now:
   - Checks errors in the same precedence order as `probeContainer()` and `ClassifyError()`: context/cluster → kubeconfig → namespace → pod state → container state → tar/read-only → forbidden
   - Includes detection for `podNotRunningFragments`, `containerNotFoundFragments`, and `multiContainerFragments`
   - No longer falls through to generic "unable to run kubectl cp" for these cases

3. **Context classification patterns** (Minor): Confirm `isContextOrClusterError()` now matches:
   - "current-context is not set"
   - "context ... is not set"
   - "context ... not configured"
   - "context ... invalid"

4. **Strengthened unit tests** (Minor): Verify that:
   - Tests now execute actual classification logic (via `classifyCopyError` helper or similar)
   - Tests validate user-facing error messages, not just fragment matching
   - `TestErrorClassificationPrecedence` exists and validates the precedence order

### 2. Scope Alignment

**Original Task:** Implement Kubernetes support in Mutagen to enable file synchronization and network forwarding to containers running in Kubernetes clusters.

- Does the MVP scope (Linux pods with shell+tar, Windows experimental) align with the original task?
- Is the 50-60 hour estimate reasonable?

### 3. Code Consistency

- Are error messages consistent across `probeContainer()`, `Copy()`, and `ClassifyError()`?
- Is the `classifyCopyError` test helper an accurate mirror of the actual `Copy()` classification logic?

## Response Format

Please structure your response as:

```
## Findings

[List each finding with severity (Major/Minor) and specific line references]

## Decision

[One of: Go | Go with minor notes | No-Go]

## Next Steps (if No-Go)

[Specific items to address]
```

## Previous Reviews Summary

| Review | Key Issues Addressed |
|--------|---------------------|
| 1-3 | Initial structure, environment handling, error patterns |
| 4 | Integration test helpers, error fragment robustness |
| 5 | crypto/rand, distroless pod images, HTTP retry/backoff |
| 6 | Quoting/escaping, extended error classification, lifecycle handling, MVP scope |
| 7 | Probe-time error classification, Copy error detection, context override test |
| 8 | Duplicate helper, Copy classification divergence, context patterns, weak unit tests |
