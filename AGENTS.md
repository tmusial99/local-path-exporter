# local-path-exporter Agent Notes

Keep this file short and aligned with the real project.
`CLAUDE.md` must contain only `@AGENTS.md`.

## Project

`local-path-exporter` is a Go Prometheus exporter for Kubernetes
directory-backed PVC storage, especially `rancher/local-path-provisioner`.

It runs as a DaemonSet, scans the host storage path, parses PVC labels from
directory names, caches scan results, and serves metrics on `:9100/metrics`.

Public release surfaces:

- Helm chart: `charts/local-path-exporter/`
- Published Helm repo: `docs/`
- Container image: `ghcr.io/tmusial99/local-path-exporter`
- Root README and chart README, including Artifact Hub content

## Layout

```text
app/                         Go module and Dockerfile
app/parser/                  directory-name template parser
app/collector/               Prometheus collector and disk scanner
charts/local-path-exporter/  Helm chart
docs/                        GitHub Pages Helm repository
grafana/                     dashboard JSON and screenshot
.github/workflows/           CI, image release, chart release
```

## Commands

Run Go commands from `app/`:

```bash
go build ./...
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
```

Run Helm commands from the repo root:

```bash
helm lint charts/local-path-exporter
helm template t charts/local-path-exporter
helm template t charts/local-path-exporter --set serviceMonitor.enabled=true
```

Ask before running cloud, cluster, infrastructure, Docker publish, Helm publish,
tag, push, or other release commands.

## Rules

- Read `git status --short --branch` before edits.
- Read any applicable `AGENTS.md` before editing files in another subtree.
- Keep repository files and generated docs in English.
- Keep changes small and tied to the request.
- Do not read, print, edit, or commit secrets or real environment files.
- Do not commit, tag, push, or create releases unless explicitly asked.
- Do not change hostPath, root access, or security context behavior casually.
- Keep chart docs, root docs, `Chart.yaml`, `values.yaml`, and metrics aligned.
- Do not edit packaged chart files in `docs/` by hand except during a chart
  release.

## Release Facts

- App/image releases use bare semver tags like `1.1.0`.
- Chart releases use tags like `chart-0.2.3`.
- `Chart.yaml` `appVersion` should match an existing image tag.
- `Chart.yaml` `version` should match the `chart-*` tag.
- Chart packages are signed with the public key in `docs/helm-signing-key.asc`.

## Notes

The scanner uses Unix filesystem APIs and is not Windows-compatible. Scrapes
serve the last cached scan result, so Prometheus scrapes should not block on
disk walks.
