# local-path-exporter

Prometheus exporter for disk usage of `rancher/local-path-provisioner` PVCs and
other directory-backed Kubernetes volumes.

It runs as a DaemonSet, scans the configured host storage path on each node,
caches the result, and serves metrics on `:9100/metrics`.

## Install

```bash
helm repo add local-path-exporter https://tmusial99.github.io/local-path-exporter
helm repo update
helm install local-path-exporter local-path-exporter/local-path-exporter \
  --namespace monitoring --create-namespace
```

Enable Prometheus Operator discovery:

```bash
helm upgrade --install local-path-exporter local-path-exporter/local-path-exporter \
  --namespace monitoring --create-namespace \
  --set serviceMonitor.enabled=true
```

The default storage path is `/var/lib/rancher/k3s/storage`. It must exist on
each target node unless you change `config.hostPathType`.

## Metrics

| Metric | Description | Labels |
|---|---|---|
| `local_path_pvc_usage_bytes` | Real on-disk PVC directory usage, like `du`. | from `config.metricTemplate` |
| `local_path_storage_capacity_bytes` | Total capacity of the scanned filesystem. | none |
| `local_path_storage_total_used_bytes` | Total used space on the scanned filesystem. | none |

## Values

| Value | Default | Notes |
|---|---:|---|
| `config.storagePath` | `/var/lib/rancher/k3s/storage` | Host path to scan. |
| `config.metricTemplate` | `pvc-*_{pvc_namespace}_{pvc_name}` | Directory-name parser and metric label source. |
| `config.refreshIntervalSeconds` | `60` | Background scan interval. |
| `config.hostPathType` | `Directory` | Kubernetes `hostPath` type. |
| `serviceMonitor.enabled` | `false` | Creates a `ServiceMonitor`. |

See [`values.yaml`](./values.yaml) for all values.

The DaemonSet mounts the host path read-only and runs as UID/GID `0` so it can
stat files owned by other users. The container drops Linux capabilities, uses a
read-only root filesystem, and does not mount a service account token.

## Signature

Chart packages are signed.

```bash
curl -fsSL https://tmusial99.github.io/local-path-exporter/helm-signing-key.asc | gpg --import
gpg --export > ~/.gnupg/pubring.gpg
helm pull local-path-exporter/local-path-exporter --verify --keyring ~/.gnupg/pubring.gpg
```

Signing key fingerprint:
`7FED455C2F76B3E0216E73F26F0F38A50C4D9D03`.

## Links

- Source: <https://github.com/tmusial99/local-path-exporter>
- Dashboard: <https://github.com/tmusial99/local-path-exporter/blob/master/grafana/dashboard.json>
- License: MIT
