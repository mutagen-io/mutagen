# Kubernetes Transport Plan Review (pass 3)

This draft fixes many earlier issues, but a few blockers and realism gaps remain before implementation is safe.

## Blockers
- Environment parsing still won’t compile: `environment.ParseBlock` returns `[]string`, but the plan treats it like a map (`envBlock["HOME"]`). Convert with `environment.ToMap` or reuse the Docker helper before reading vars.
- Windows detection can misclassify: if a Windows container exposes `HOME`, the POSIX probe succeeds and the Windows probe never runs, leaving `containerIsWindows` false. Subsequent commands wrap with `sh -c` and POSIX paths, which will fail on Windows images. Add a reliable platform probe (always check for `USERPROFILE`/`cmd /c set` even when HOME is present) or another indicator.
- Workdir emulation assumes shells exist: wrapping commands in `sh -c`/`cmd /c` will fail on shell-less images (distroless, Windows Nano without cmd). Need a fallback (skip workdir with clear error, or detect shell presence) to avoid hard failures.

## Additional risks/gaps
- `kubectl cp` still relies on `tar` in the container; document this dependency or plan a streamed copy fallback.
- Error matching relies on fragments that vary across kubectl versions (multi-container, forbidden). Consider exit codes or a broader match set to reduce false negatives.
- CLI helper `ApplyKubernetesParametersToURL` needs the `urlpkg` import and should ensure `Parameters` is non-nil; otherwise it won’t compile.
- Integration scaffolding references helper functions (`randomString`, `createSyncSession`, etc.) that aren’t defined; call out the need to implement or reuse existing helpers to keep the estimate realistic.

## Realism notes
- Add unit coverage for the Windows probe when `HOME` is present and `USERPROFILE` differs, and for workdir behavior when no shell is available.
- Confirm the silent-precedence rule (CLI > parameters > URL) is the desired UX; otherwise emit warnings on conflicts.

Resolve the compile-time env parsing bug, strengthen platform detection, and define a shell-less fallback; with those in place, the plan becomes executable. Estimates may need a small bump to cover the new probing/tests.***
