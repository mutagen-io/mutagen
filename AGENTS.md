# Mutagen Agent Guide

This file contains repository-specific guidance for coding agents.
Keep it concise and stable. When this file and the code disagree,
follow the code and update this file.

## Source Of Truth

- `README.md` provides the public project overview.
- `BUILDING.md` describes the build flow and protobuf regeneration.
- `CONTRIBUTING.md` defines coding style, testing expectations, DCO
  requirements, and commit message guidelines.
- `SECURITY.md` defines the security reporting process.
- `scripts/build.go` is the source of truth for build modes and the
  main platform matrix.
- `.github/workflows/ci.yml` is the source of truth for CI jobs and
  tested Go versions.
- `images/sidecar/linux/Dockerfile` is the source of truth for
  sidecar image targets.
- `pkg/mutagen/version.go` defines versioning and development-mode
  behavior.
- `pkg/filesystem/mutagen.go` defines Mutagen data directory
  selection.
- `pkg/agent/bundle.go` defines agent bundle lookup and extraction.
- `pkg/generate.go` defines protobuf generation commands and the
  ghost imports required for that workflow.

## Repository Map

- `cmd/mutagen` is the main CLI. Its command groups live under
  `cmd/mutagen/sync`, `cmd/mutagen/forward`,
  `cmd/mutagen/project`, and `cmd/mutagen/daemon`.
- `cmd/mutagen/common` contains shared CLI support code.
- `cmd/mutagen-agent` is the remote agent binary deployed to
  endpoints.
- `cmd/mutagen-sidecar` is the sidecar entry point. It is built
  separately from the main `scripts/build.go` flow.
- `cmd/external` is the exported Go API entry point for
  programmatic session management.
- `cmd/profile` contains profiling helpers and the profiling
  binary entry point.
- `pkg/api/models` contains API-facing request and response models.
- `pkg/configuration` and `pkg/project` handle global,
  directory-level, and project-level configuration loading.
- `pkg/synchronization` contains the synchronization engine and its
  endpoints. `pkg/synchronization/core` contains the core scan,
  reconcile, stage, and transition logic, while `rsync`,
  `compression`, and `hashing` provide transfer primitives.
- `pkg/forwarding` contains the forwarding engine and its endpoints.
- `pkg/forwarding/protocols` and `pkg/synchronization/protocols`
  register transport-specific endpoint handlers.
- `pkg/agent` contains agent installation, transport, and bundle
  handling. `pkg/agent/transport` contains transport-specific agent
  dialing code.
- `pkg/daemon` contains daemon lifecycle, locking, and IPC code.
- `pkg/filesystem` contains filesystem primitives and watching code.
  `pkg/filesystem/watching` is the key entry point for watcher
  behavior.
- `pkg/docker`, `pkg/ssh`, and `pkg/url` implement the Docker, SSH,
  and URL layers used by remote endpoints.
- `pkg/multiplexing`, `pkg/stream`, and `pkg/grpcutil` contain the
  lower-level transport and RPC helpers used across the daemon,
  agent, and services.
- `pkg/service` contains protobuf and gRPC service definitions and
  handlers. Its subpackages mirror the daemon's public services.
- `pkg/integration` contains end-to-end test infrastructure and
  fixtures.
- `pkg/sidecar` contains sidecar environment detection and helpers.
- `scripts/` contains the main build orchestration and CI helper
  scripts.
- `images/sidecar/linux` contains the sidecar image build.
- `tools/` contains standalone developer utilities such as
  `scan_bench` and `watch_demo`.
- `sspl/` contains optional SSPL-licensed code paths.

## Build And Test

- Use `go run scripts/build.go --mode=local` for a local CLI and
  agent build.
- Use `go run scripts/build.go` for the default slim build.
- `release` builds require macOS. On other platforms, `slim` and
  `release-slim` builds continue without cgo-enabled macOS agents.
- The agent bundle is a separate `mutagen-agents.tar.gz` artifact.
  It ships alongside the CLI or in `libexec`; it is not embedded in
  the CLI.
- Sidecar binaries and images are built via `cmd/mutagen-sidecar`,
  `scripts/ci/build.sh`, and `images/sidecar/linux/Dockerfile`.
- Run package tests sequentially with `go test -p 1 ./pkg/...`.
- In sandboxes or other restricted environments, set
  `MUTAGEN_DATA_DIRECTORY` to a writable absolute path before
  running tests. Development builds default to `~/.mutagen-dev`;
  release builds use `~/.mutagen`.
- Run `go test -v -tags mutagensspl ./sspl/...` when touching
  SSPL code.

## Generated Code And Dependencies

- Generated `.pb.go` and `_grpc.pb.go` files are checked in.
- Regenerate protobufs with `go generate ./pkg/...` after changing
  `.proto` files.
- Keep `pkg/generate.go` intact. Its `//go:build generate`
  constraint and ghost imports are part of the generation workflow.
- The `k8s.io/apimachinery` replace directive in `go.mod` is
  intentional.

## Change Guidance

- Follow `CONTRIBUTING.md` for code style.
- Use [Conventional Commits](https://www.conventionalcommits.org/)
  for commit messages. Common types: `feat`, `fix`, `build`, `ci`,
  `docs`, `deps`, `refactor`, `test`, `perf`, `chore`. Append `!`
  for breaking changes.
- Run `gofmt` on touched Go files.
- Match the surrounding comment-heavy style, especially in core
  packages.
- Agent and CLI versions must match exactly. Be careful with changes
  under `pkg/mutagen/version.go`.
- When changing Go versions, update the relevant build, CI, and
  version files together rather than changing only one of them.
- Treat `scripts/build.go` and the sidecar Dockerfile as the source
  of truth for supported platforms and architectures instead of
  restating those lists here.
- Fanotify support is Linux-only, SSPL-only, and currently
  sidecar-specific.
- External pull requests are not accepted for code under `sspl/`.
- Public docs in this repository should not describe local scratch
  files, untracked helper scripts, or internal-only workflow
  details.

## Maintaining This File

- Prefer pointers to tracked source files over exhaustive prose.
- Do not turn this file into an exhaustive package inventory or an
  architecture dump.
- Avoid exact counts, full platform lists, and other details that
  drift easily.
- Keep statements public-safe and repository-wide. If a detail only
  applies to a local checkout, it does not belong here.
