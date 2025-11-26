# Kubernetes Transport Plan Review (current draft)

Overall: The draft is much closer to workable, but there are still a few blockers and realism gaps. The largest issues are a compile-time bug in environment parsing, Windows/container detection that can misclassify and break execution, and reliance on shells that may not exist in target images.

## Blockers
- Environment parsing won’t compile: `environment.ParseBlock` returns `[]string`, but the transport code treats it as `map[string]string` (`envBlock := environment.ParseBlock(env); envBlock["HOME"]`). The current plan would fail to build; use `environment.ParseBlock` + `environment.ToMap` or the Docker transport’s `findEnviromentVariable`.
- Windows misclassification: The probe assigns `home` from POSIX `env` output unconditionally; many Windows containers set `HOME`, so the Windows probe never runs and `containerIsWindows` stays false. Subsequent commands wrap in `sh -c` and POSIX paths, which will fail on Windows containers (no `/bin/sh`, wrong separators). A platform indicator is needed (e.g., always attempt a Windows probe if HOME is set but USERPROFILE exists, or probe `OS`/kernel).
- Workdir requires shell: `command` wraps execution with `sh -c` or `cmd /c` to emulate `--workdir`, but `kubectl exec` targets (distroless, busybox with no sh, Windows Nano without cmd) will fail outright. Without a fallback (e.g., detect shell availability or run without workdir) the transport can’t support shell-less images.

## Risks / gaps to address
- `kubectl cp` still assumes `tar` in the container. This is a known kubectl limitation; document it and plan a fallback (streamed copy via exec) or accept the constraint.
- Error-fragment matching may be brittle: messages for multi-container ambiguity or forbidden access vary across kubectl versions; consider checking exit codes plus structured `kubectl` errors if possible.
- CLI flag application helpers need imports (`urlpkg`) and should guard against nil `Parameters` to avoid panics; the plan hints at this but should specify.
- Context exclusion from URLs is intentional, but validation around conflicting namespace/container values (URL vs parameters vs flags) is still “silent overwrite”; confirm that aligns with Mutagen’s UX expectations.

## Realism / coverage
- Expand the probe logic parity with Docker: ensure UTF-8 checks for usernames/groups, and that Windows detection runs even if HOME is set. Add a unit test for Windows probe with HOME present.
- Add a test that exercises workdir wrapping when no shell exists to define expected behavior (fail fast with clear error or skip wrapping).
- Integration plan should explicitly note the shell/tar requirements and add a Windows pod case to validate platform detection and path handling.

If these blockers are resolved (compile issue, Windows detection, and shell dependency), the rest of the plan looks executable and the estimates remain reasonable.***
