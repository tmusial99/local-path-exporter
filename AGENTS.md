# local-path-exporter Agent Manual

This file is the source of truth for AI agents working in this repository.
Humans should keep it aligned with the real project behavior.

`CLAUDE.md` must contain only `@AGENTS.md`. Do not duplicate these instructions
there.

## Repo context

`local-path-exporter` is a **Prometheus exporter** written in **Go** that reports
disk usage of `rancher/local-path-provisioner` (and other directory/HostPath)
Persistent Volumes in Kubernetes. Standard kubelet metrics do not report
per-PVC usage for directory-backed storage; this exporter fills that gap.

It runs as a **DaemonSet** on every node, periodically scans the provisioner's
storage directory (default `/var/lib/rancher/k3s/storage`), computes each PVC
directory's size with a recursive walk, parses Namespace/PVC labels from the
directory name via a configurable template, and exposes Prometheus metrics on
port `9100` at `/metrics`.

The container is a **scratch** image with a static binary; it runs as **root**
(`runAsUser: 0`) with a **read-only hostPath** mount because it must stat files
owned by other users on the host.

### Distribution (this is the product surface)

- **Helm chart on Artifact Hub.** The chart in `charts/local-path-exporter/` is
  published to a Helm repository hosted on **GitHub Pages**
  (`https://tmusial99.github.io/local-path-exporter`). The served repo index and
  packaged chart archive live under `docs/` (`docs/index.yaml`,
  `docs/local-path-exporter-*.tgz`, `docs/artifacthub-repo.yml`). Artifact Hub
  reads `artifacthub-repo.yml` (ownership) and the chart's `artifacthub.io/*`
  annotations. **Treat the chart, its annotations, and the published repo index
  as a public contract.**
- **Container image on GHCR** (`ghcr.io/tmusial99/local-path-exporter`), built and
  pushed by `.github/workflows/docker-publish.yaml` on a semver git tag.

## Language

- Reply to the operator in Polish by default.
- Use English for code, identifiers, comments, commit messages, branch names,
  logs, documentation files, chart values, and metric/label names.

## Layout

```text
app/                      Go module (module path: local-path-exporter)
  main.go                 entrypoint: env config -> parser -> collector -> /metrics
  parser/parser.go        template ("pvc-*_{ns}_{name}") -> regex -> ordered labels
  collector/collector.go  prometheus.Collector: background scanner, mutex cache,
                          Statfs capacity/used, recursive per-dir size
  Dockerfile              multi-stage -> static binary on scratch
  go.mod / go.sum
charts/local-path-exporter/   Helm chart (DaemonSet, Service, ServiceMonitor, helpers)
docs/                     Published Helm repo (index.yaml + .tgz) + artifacthub-repo.yml
grafana/                  dashboard.json + screenshot
.github/workflows/        ci.yaml (PR/push checks), docker-publish.yaml (image on semver tag), chart-release.yaml (signed chart on chart-* tag)
README.md
```

Go module path is `local-path-exporter` (not a VCS URL). The Go code lives under
`app/`; run Go commands from `app/`.

## Commands

```bash
# Go (run from app/)
cd app
go build ./...            # build all packages
go test ./...             # unit tests (parser + collector)
go vet ./...
golangci-lint run         # v2; config in app/.golangci.yml

# Helm (run from repo root)
helm lint charts/local-path-exporter
helm template t charts/local-path-exporter        # render with defaults
helm template t charts/local-path-exporter --set serviceMonitor.enabled=true

# Container
docker build -t local-path-exporter ./app
```

Go toolchain: **Go 1.26.x** (`app/go.mod`; Dockerfile pins `golang:1.26.4-alpine`).
The collector uses `syscall.Statfs`, which is **Unix-only** (Linux + macOS/BSD,
not Windows). The `Statfs_t` field types differ per platform, but the code
converts them explicitly, so all packages build and `go test` runs natively on
both macOS (darwin/arm64) and Linux. CI runs on `ubuntu-latest`.

## Architecture notes

- **Config** (`main.go`): all settings come from required env vars
  (`STORAGE_PATH`, `METRIC_TEMPLATE`, `LISTEN_ADDR`, `REFRESH_INTERVAL_SECONDS`).
  Missing/invalid config is a fatal `log.Fatalf` so a misconfigured pod crashes
  loudly. The Helm DaemonSet template wires these from `values.yaml`.
- **Parser** (`parser/`): turns a human template into a regex. `{label}` becomes a
  named capture and `*` becomes a non-greedy wildcard. `LabelNames` order is the
  Prometheus label order — it must stay stable for a given template, and the
  same `DirParser` instance is shared with the collector's metric `Desc`.
- **Collector** (`collector/`): implements `prometheus.Collector`. A background
  goroutine (`StartBackgroundScanner`) rescans on a ticker and atomically swaps a
  cached slice of data points under a `sync.RWMutex`; `Collect` reads the cache
  under `RLock`. Scrapes therefore never block on disk I/O. `Statfs` provides
  filesystem capacity/used; per-PVC size is a `filepath.WalkDir` sum.
- **Cardinality**: labels are derived from on-disk directory names. Only
  directories matching the template are emitted, and the cache is fully replaced
  each scan (so deleted PVCs drop out). The template is the cardinality control —
  a loose template can match unintended directories.

## Versioning & release contract

There are **two independent release axes**, each with its own tag namespace:

- **Image release** — bare semver tag `X.Y.Z` (e.g. `1.1.0`). Triggers
  `docker-publish.yaml` → image `ghcr.io/tmusial99/local-path-exporter:X.Y.Z`.
  The chart's `image.tag` defaults to `.Chart.AppVersion`, so `appVersion` must
  equal the pushed image tag.
- **Chart release** — tag `chart-X.Y.Z` (e.g. `chart-0.2.2`), where `X.Y.Z` is
  the **chart** `version` in `Chart.yaml`. Triggers `chart-release.yaml`, which
  packages + **signs** the chart and publishes it to `docs/` (the Pages Helm
  repo). A chart-only change (no app change) keeps `appVersion` the same and is
  shipped with a `chart-*` tag alone — no image rebuild.

Consistency rules:

- `appVersion` must equal the image tag that exists on GHCR.
- A `chart-X.Y.Z` tag must match `Chart.yaml`'s `version` (CI enforces this).
- The published `docs/index.yaml`, the `docs/*.tgz` (+ `.tgz.prov`), and the
  README version/badge must all agree. Drift is a release bug (e.g. `index.yaml`
  advertising a chart version the `.tgz` does not contain).
- The chart is signed (Helm provenance). The public key lives at
  `docs/helm-signing-key.asc`; the private key is **not** in this repo (operator
  secret: GitHub Actions secret `HELM_GPG_PRIVATE_KEY`, also kept in the
  operator's SOPS store). Signing key fingerprint
  `7FED455C2F76B3E0216E73F26F0F38A50C4D9D03`.

## Secrets and safety

- No application secrets. CI credentials: the built-in `GITHUB_TOKEN`
  (packages: write for GHCR; contents: write for the chart-release commit) and
  the `HELM_GPG_PRIVATE_KEY` secret (base64 of the chart-signing private key,
  used only by `chart-release.yaml`).
- Do not read, print, modify, or commit real secret values. There are no `.env`
  files to read here.
- The exporter runs as **root** with a **hostPath** mount; treat the security
  context, the read-only mount, and the scanned path as security-relevant — do
  not widen them without reason.

## CI / release pipeline

- `.github/workflows/ci.yaml`: runs on pull requests and pushes to `master`. Two
  jobs — **go** (`go vet`, `go test -race`, `golangci-lint`, `govulncheck`) and
  **helm** (`helm lint`, `helm template` with default and
  `serviceMonitor.enabled=true`). This is the PR/push quality gate.
- `.github/workflows/docker-publish.yaml`: triggered on a **bare semver** tag
  (`[0-9]+.[0-9]+.[0-9]+`). Logs in to GHCR, derives tags via
  `docker/metadata-action` (semver), and builds/pushes a **multi-arch** image
  (`linux/amd64,linux/arm64`) from `./app` with GHA build cache, attaching
  **provenance + SBOM** and a **keyless Cosign signature** (Sigstore OIDC).
- `.github/workflows/chart-release.yaml`: triggered on a `chart-X.Y.Z` tag.
  Checks out `master`, imports the GPG key from the `HELM_GPG_PRIVATE_KEY`
  secret, verifies `Chart.yaml` `version` == tag, runs `helm package --sign`
  into `docs/`, regenerates `docs/index.yaml` (`--merge`), and commits the
  packaged `.tgz` + `.tgz.prov` + index back to `master`. GitHub Pages serves
  `docs/` from `master`. To cut a chart release: bump `Chart.yaml` `version`,
  commit, then push `chart-<version>`.
- The Pages source is **master `/docs`**; the repo intentionally publishes from
  the default branch, not a `gh-pages` branch.

## Git

Read `git status --short --branch` before edits. The default branch is
**`master`**. Do not create commits, branches, tags, PRs, or pushes unless the
operator explicitly asks. A semver tag push **publishes a public image**, so
never tag/push without explicit approval. When suggesting a commit, stage exact
paths; avoid `git add .`.

## Project skills

Canonical skills live under `.agents/skills/`; `.claude/skills/*/SKILL.md` are
symlinks to them. They form one audit -> tasks -> execute pipeline:

- `deep-repo-audit`: single-model, autonomous, read-only audit of the whole repo.
  Focus: exporter/metric correctness, collector concurrency, parser/label
  cardinality, filesystem-scan safety/performance, Go quality & tests, the Helm
  chart, Artifact Hub packaging + version consistency, Docker/scratch image,
  GitHub Actions reliability, dependency freshness, docs, and DRY/KISS/YAGNI.
- `audit-to-tasks`: Q&A skill that turns a finalized audit into ordered, English
  executor task files with explicit parallelism groups under
  `audit/audit-to-tasks/tasks/`.
- `execute-audit-tasks`: orchestrator that executes the task queue with one
  operator approval gate, per-task verification (Go + Helm), commits, and
  reports.

`.claude/skills/*/SKILL.md` must be symlinks to the matching canonical
`.agents/skills/*/SKILL.md`. When updating a skill, edit `.agents/skills/...`
first and refresh the symlink only if it is broken.

Audit/task working artefacts under `audit/` are local working state, not source.
Commit them only when the operator explicitly wants a durable record.

## Verification

Smallest relevant check first:

```bash
cd app && go vet ./... && go test ./...
helm lint charts/local-path-exporter && helm template t charts/local-path-exporter >/dev/null
```

Chart changes should be validated by rendering (`helm template`) with both
default and `serviceMonitor.enabled=true` values. Go (including the collector)
builds and tests natively on macOS and Linux; only Windows is unsupported
(`syscall.Statfs`).
