# Kubernetes Transport Implementation - Review 10 Request

## Document Under Review

`/home/vlad/Repos/mutagen/implementations/kubernetes-transport.md`

## Scope of Review

This is a final verification review following review-9 fixes. Please confirm that:

1. **All review-9 issues are resolved:**
   - ClassifyError precedence now matches Copy/probeContainer (tar/RO before forbidden, lifecycle after forbidden)
   - classifyCopyError test helper is an exact replica of Copy's classification with identical messages
   - Error messages are normalized across all three methods (probeContainer, Copy, ClassifyError)

2. **Test assertions use exact matching:**
   - TestProbeErrorClassification uses `err.Error() != tc.expectedError` (exact match)
   - TestCopyErrorClassification uses `err.Error() != tc.expectedError` (exact match)
   - Expected error strings match the full user-facing messages including all guidance text

3. **Precedence is consistent across all methods:**
   - Order: context/cluster → kubeconfig → namespace → pod → container → tar/RO → forbidden → lifecycle
   - All methods check in this order

4. **Error message consistency:**
   - Context: `"Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG"`
   - Kubeconfig: `"kubeconfig file not found or invalid; check --kubernetes-kubeconfig or KUBECONFIG environment variable"`
   - Cluster: `"unable to connect to Kubernetes cluster; check cluster connectivity and credentials"`
   - Pod running: `"pod is not running"` (not "pod not running")
   - Multi-container: `"pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)"`
   - Tar: `"tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)"`
   - Read-only: `"container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)"`
   - Forbidden: `"access forbidden; check RBAC permissions for pod/exec"`

## Expected Outcome

Reply with **"Go"** if all review-9 fixes are correctly implemented and the document is ready.

If any issues remain, provide specific line numbers and the required corrections.

## Review Focus

- Correctness of all error message strings
- Test assertion methodology (exact vs contains matching)
- Precedence order alignment between probeContainer, Copy, ClassifyError, and classifyCopyError
- Test expected values matching actual error message strings
