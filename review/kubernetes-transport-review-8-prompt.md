# Kubernetes Transport Implementation Plan - Review 8

## Review Request

Please review the implementation plan at `/home/vlad/Repos/mutagen/implementations/kubernetes-transport.md` for the Kubernetes transport layer in Mutagen.

## Review Focus Areas

### 1. Scope Alignment

**Original Task:** Implement Kubernetes support in Mutagen to enable file synchronization and network forwarding to containers running in Kubernetes clusters.

Please verify:
- Does the implementation plan fully address the original scope?
- Is the MVP scope appropriately sized for an initial release?

### 2. Previous Review Items (Review 7)

Verify the following issues from review-7 have been adequately addressed:

1. **Extended error classification in probeContainer** (Major): Does `probeContainer()` now correctly classify kubeconfig, context, namespace, and cluster connectivity errors instead of returning opaque "container probing failed" messages?

2. **Tar/read-only detection in Copy** (Major): Does the `Copy()` method now detect tar-not-found and read-only filesystem errors during `kubectl cp` with clear user-facing messages, rather than generic "unable to run kubectl cp" errors?

3. **Context override test** (Minor): Is `TestKubernetesContextOverride` now complete with actual assertions, helper functions, and test cases that validate CLI context-precedence and related error handling?

### 3. Implementation Completeness

- Are error classification paths consistent across `probeContainer()`, `Copy()`, and `ClassifyError()`?
- Are the new unit tests (`TestProbeErrorClassification`, `TestCopyErrorClassification`) adequate?
- Are the new integration tests (`TestKubernetesReadOnlyFilesystem`, `TestKubernetesTarNotFound`) properly structured?

### 4. Technical Correctness

- Is the error precedence order correct (cluster/context errors before namespace errors before pod/container errors)?
- Are the error fragment patterns comprehensive enough to catch kubectl variations?
- Do the helper functions for pod deployment (`deployReadOnlyPod`, `deployNoTarPod`) correctly create the test conditions?

## Decision Criteria

Please provide one of the following:
- **Go**: The implementation plan is ready for development
- **Go with minor notes**: Ready with small suggestions that don't block implementation
- **No-Go**: Significant issues must be addressed before implementation

## Previous Reviews

This is review #8. Previous reviews addressed:
- Review 1-3: Initial structure, environment handling, error patterns
- Review 4: Integration test helpers, error fragment robustness
- Review 5: crypto/rand, distroless pod images, HTTP retry/backoff
- Review 6: Quoting/escaping, extended error classification, lifecycle handling, MVP scope
- Review 7: Probe-time error classification, Copy error detection, context override test
