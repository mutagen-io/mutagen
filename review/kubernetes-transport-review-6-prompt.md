# Review Prompt: Kubernetes Transport Implementation Plan (Final Review)

## Context for Reviewer

You are performing the final review of the Kubernetes transport implementation plan for Mutagen before implementation begins. This plan has been through 5 review passes addressing compile errors, test robustness, and edge cases.

**File to review:** `/home/vlad/Repos/mutagen/implementations/kubernetes-transport.md`

## Review Focus

This final review should focus on:

1. **Edge Case Coverage** - Are all realistic edge cases handled?
2. **Implementation Realism** - Can this actually be built as specified?
3. **Estimate Accuracy** - Is 34-48 hours realistic for the scope?

## Changes Since Last Review

Fixed 3 remaining issues:
- Removed unused `regexp` import (compile error)
- Fixed `resp.Body` leak in retry loop (drain + close each response)
- Corrected misleading backoff comment (now accurately describes 8s cap)

## Edge Cases to Verify

Please verify each edge case is properly handled:

### Container/Pod Edge Cases
- [ ] Pod doesn't exist
- [ ] Pod exists but not running (Pending, ContainerCreating, CrashLoop)
- [ ] Pod has multiple containers without container specified
- [ ] Specified container doesn't exist in pod
- [ ] Container restarts during sync/forward session
- [ ] Pod is deleted while session is active
- [ ] Pod on Windows node (rare but possible)
- [ ] Pod with HOME and USERPROFILE both set (Git Bash scenario)
- [ ] Distroless/scratch container with no shell
- [ ] Container without tar (kubectl cp limitation)
- [ ] Container running as non-root user
- [ ] Container with read-only filesystem

### Kubernetes Connection Edge Cases
- [ ] kubectl not installed or not in PATH
- [ ] KUBECONFIG points to non-existent file
- [ ] Context doesn't exist
- [ ] Namespace doesn't exist
- [ ] RBAC denies exec/cp to pod
- [ ] Network timeout to API server
- [ ] Cluster unreachable (VPN down, etc.)
- [ ] In-cluster service account auth
- [ ] Multiple kubeconfig files (KUBECONFIG with colons)

### URL/Parameter Edge Cases
- [ ] Namespace with special characters
- [ ] Pod name at max length (253 chars)
- [ ] Container name with hyphens/underscores
- [ ] Path with spaces or special characters
- [ ] Home-relative path (`~/data`)
- [ ] Windows path in URL (`C:\Users\...`)
- [ ] CLI flags override URL components
- [ ] Empty/nil parameters map

### Session Lifecycle Edge Cases
- [ ] Session creation fails mid-way (cleanup?)
- [ ] Agent binary already exists in container
- [ ] Agent binary corrupted or wrong architecture
- [ ] Concurrent sessions to same pod
- [ ] Session resume after container restart

### Test Infrastructure Edge Cases
- [ ] Parallel test execution (unique names)
- [ ] Test cleanup on failure
- [ ] Test timeout handling
- [ ] CI without Kubernetes cluster (skip gracefully)
- [ ] Windows nodes in CI cluster

## Realism Check

### Does the design match existing patterns?
- [ ] Transport interface matches Docker/SSH transports
- [ ] Error handling follows Mutagen conventions
- [ ] URL parsing is consistent with other protocols
- [ ] CLI flags follow existing naming patterns

### Are there hidden complexities?
- [ ] `kubectl exec` stdin/stdout handling for agent protocol
- [ ] `kubectl cp` tar extraction in container
- [ ] Working directory emulation without `--workdir`
- [ ] Shell detection and fallback logic
- [ ] Windows path separator handling

### Are dependencies realistic?
- [ ] kubectl must be installed (documented?)
- [ ] tar must exist in container for `kubectl cp`
- [ ] Shell required for working directory support
- [ ] Network connectivity to Kubernetes API

### Are estimates realistic for each phase?

| Phase | Estimated | Realistic? |
|-------|-----------|------------|
| Proto & URL Parsing | 3-4h | ? |
| Kubectl Package | 2-3h | ? |
| Transport Layer | 6-8h | ? |
| Protocol Handlers | 2-3h | ? |
| CLI Integration | 3-4h | ? |
| Testing | 8-12h | ? |
| Documentation | 2-3h | ? |
| Validation & Polish | 4-6h | ? |
| **Total** | **34-48h** | ? |

## Questions to Answer

1. **Are there edge cases missing?** What scenarios could break the implementation that aren't covered?

2. **Is the error handling comprehensive?** Will users get clear, actionable error messages in all failure modes?

3. **Is the test coverage sufficient?** Are there gaps that would let bugs slip through?

4. **Are there any blockers not identified?** Dependencies, Kubernetes API limitations, or Mutagen architecture constraints?

5. **Is the implementation order optimal?** Would a different sequence reduce risk or enable earlier testing?

6. **What's the MVP vs nice-to-have?** If time runs short, what can be deferred to a follow-up?

## How to Review

```bash
# Read the full implementation plan
cat /home/vlad/Repos/mutagen/implementations/kubernetes-transport.md

# Compare with Docker transport for pattern parity
diff -u <(grep -E "func.*Transport|func.*Copy|func.*Command|func.*ClassifyError" \
    /home/vlad/Repos/mutagen/pkg/agent/transport/docker/transport.go) \
    <(grep -E "func.*Transport|func.*Copy|func.*Command|func.*ClassifyError" \
    /home/vlad/Repos/mutagen/implementations/kubernetes-transport.md) || true

# Check for TODO/FIXME comments that might indicate incomplete areas
grep -n "TODO\|FIXME\|XXX\|HACK" /home/vlad/Repos/mutagen/implementations/kubernetes-transport.md

# Verify error fragment coverage
grep -A2 "Fragments\|fragments" /home/vlad/Repos/mutagen/implementations/kubernetes-transport.md
```

## Expected Output

Please provide:
1. **Edge cases missing** - Any scenarios not covered
2. **Realism concerns** - Anything that seems impractical
3. **Estimate adjustments** - If 34-48h seems off
4. **Blockers** - Anything that would prevent implementation
5. **Recommendations** - Suggestions for improvement
6. **Go/No-Go** - Is this plan ready for implementation?
