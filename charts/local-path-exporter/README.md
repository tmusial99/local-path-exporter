# local-path-exporter

A Prometheus exporter that reports the disk usage of
[`rancher/local-path-provisioner`](https://github.com/rancher/local-path-provisioner)
(and other directory/HostPath) Persistent Volumes in Kubernetes.

Standard kubelet metrics do not report per-PVC usage for directory-backed
storage; this exporter fills that gap. It runs as a **DaemonSet** on every node,
periodically scans the provisioner's storage directory, computes each PVC
directory's real on-disk usage, and exposes Prometheus metrics on port `9100` at
`/metrics`.

## Install

```bash
helm repo add local-path-exporter https://tmusial99.github.io/local-path-exporter
helm repo update

helm install local-path-exporter local-path-exporter/local-path-exporter \
  --namespace monitoring --create-namespace
```

Enable a `ServiceMonitor` (Prometheus Operator):

```bash
helm install local-path-exporter local-path-exporter/local-path-exporter \
  --namespace monitoring --create-namespace \
  --set serviceMonitor.enabled=true
```

> The DaemonSet mounts the host storage path read-only and runs as `root`
> (required to stat files owned by other users on the host). With the default
> `config.hostPathType: Directory`, the storage path must already exist on every
> targeted node or the pod will fail to mount.

## Metrics

| Metric | Type | Description | Labels |
|---|---|---|---|
| `local_path_pvc_usage_bytes` | Gauge | Actual on-disk usage (block-based, `du`-equivalent) of the PVC directory in bytes. | `pvc_namespace`, `pvc_name` |
| `local_path_storage_capacity_bytes` | Gauge | Total capacity of the underlying storage filesystem. | – |
| `local_path_storage_total_used_bytes` | Gauge | Total used space on the underlying filesystem (OS + PVCs). | – |

## Key configuration

| Parameter | Description | Default |
|---|---|---|
| `config.storagePath` | Host path the provisioner stores data in. | `/var/lib/rancher/k3s/storage` |
| `config.metricTemplate` | Pattern to extract labels from directory names. | `pvc-*_{pvc_namespace}_{pvc_name}` |
| `config.refreshIntervalSeconds` | How often directory sizes are recalculated. | `60` |
| `config.hostPathType` | `hostPath` volume type (`Directory` / `DirectoryOrCreate`). | `Directory` |
| `containerSecurityContext` | Hardened container security context (read-only rootfs, dropped caps, seccomp). | see `values.yaml` |
| `livenessProbe` / `readinessProbe` | HTTP probes on `/metrics`. | enabled |
| `serviceMonitor.enabled` | Create a `ServiceMonitor` for the Prometheus Operator. | `false` |

See [`values.yaml`](./values.yaml) for the full list.

## Grafana dashboard

A ready-made dashboard is available at
[`grafana/dashboard.json`](https://github.com/tmusial99/local-path-exporter/blob/master/grafana/dashboard.json).

## Source & license

- Source: <https://github.com/tmusial99/local-path-exporter>
- License: MIT
