# Local Path Provisioner Exporter

![Version](https://img.shields.io/badge/version-1.1.0-blue.svg)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/local-path-exporter)](https://artifacthub.io/packages/search?repo=local-path-exporter)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=flat&logo=kubernetes&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green.svg)

A specialized Prometheus Exporter for monitoring **local-path** PVC usage in Kubernetes.

## 🚀 The Problem

When using `rancher/local-path-provisioner` (or any directory-based storage like HostPath), standard Kubernetes metrics often fail to report the actual usage of specific Persistent Volumes.

## 💡 The Solution

This exporter runs as a **DaemonSet** on every node. It scans the directory structure used by the provisioner (e.g., `/var/lib/rancher/k3s/storage`), calculates the size of each directory (PVC) using a fast recursive traversal, and exposes metrics to Prometheus.

It automatically parses directory names to extract **Namespace** and **PVC Name** labels based on a configurable template.

## 📦 Installation via Helm

### 1. Add the Repository
```bash
helm repo add local-path-exporter https://tmusial99.github.io/local-path-exporter
helm repo update
```

### 2. Install the Chart
```bash
helm install local-path-exporter local-path-exporter/local-path-exporter \
  --namespace monitoring \
  --create-namespace
```

## 📊 Metrics

The exporter exposes the following metrics on port `9100`:

| Metric Name                           | Type  | Description                                                | Labels                      |
|---------------------------------------|-------|------------------------------------------------------------|-----------------------------|
| `local_path_pvc_usage_bytes`          | Gauge | Actual on-disk usage (block-based, `du`-equivalent) of the PVC directory in bytes. | `pvc_namespace`, `pvc_name` |
| `local_path_storage_capacity_bytes`   | Gauge | Total capacity of the underlying filesystem.               | -                           |
| `local_path_storage_total_used_bytes` | Gauge | Total used space on the underlying filesystem (OS + PVCs). | -                           |

## 📈 Grafana Dashboard

A pre-built Grafana dashboard is included to visualize PVC usage and storage metrics in real-time. The dashboard provides an intuitive view of your local-path provisioner's storage consumption.

![Grafana Dashboard](./grafana/dashboard.png)

To import the dashboard into your Grafana instance, use the configuration file available at [`grafana/dashboard.json`](./grafana/dashboard.json).

## ⚙️ Configuration

You can customize the installation using `values.yaml`.

| Parameter                        | Description                                                  | Default                            |
|-----------------------------------|--------------------------------------------------------------|-------------------------------------|
| `config.storagePath`              | Absolute path to the local-path storage on the node.        | `/var/lib/rancher/k3s/storage`     |
| `config.refreshIntervalSeconds`   | How often to recalculate directory sizes.                    | `60`                               |
| `config.metricTemplate`           | Pattern to extract labels from directory names.              | `pvc-*_{pvc_namespace}_{pvc_name}` |
| `config.hostPathType`             | Type of the `hostPath` volume backing `config.storagePath` (see [Kubernetes hostPath volume types](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath)). | `Directory` |
| `containerSecurityContext`        | Container-level security context (hardened by default: no privilege escalation, read-only root filesystem, all capabilities dropped, `RuntimeDefault` seccomp profile). | see `values.yaml` |
| `livenessProbe`                   | Liveness probe configuration (HTTP GET on `/metrics`).       | see `values.yaml`                  |
| `readinessProbe`                  | Readiness probe configuration (HTTP GET on `/metrics`).      | see `values.yaml`                  |
| `serviceMonitor.enabled`          | Enable ServiceMonitor for Prometheus Operator.                | `false`                            |

> **Note:** With the default `config.hostPathType: Directory`, the path set in
> `config.storagePath` must already exist on every node targeted by this
> DaemonSet, or the pod will fail to start on that node.

## 🛠️ Architecture

- **Language**: Go
- **Deployment**: DaemonSet (runs on every node)
- **Base Image**: Scratch (Static binary, ~5MB image size)
- **Privileges**: Requires `root` (`securityContext.runAsUser: 0`) to read other users' files on the host system.

## ⚡ Performance

This exporter is designed to be lightweight, but the scan cost depends on your
data:
- Each scan walks every PVC directory and sums real on-disk block usage
  (`du`-equivalent), so cost scales with the **number of files and inodes**,
  not the total size in bytes. Directories with very large file counts will
  take longer to scan than a single large file of the same size.
- Scrapes never block on disk I/O — the background scanner runs on its own
  schedule and `/metrics` always serves the last cached result.
- `config.refreshIntervalSeconds` is the main throttle: increase it on nodes
  with many files/PVCs to reduce scan frequency and I/O load.
- **Memory Usage**: Typically in the 10-20MB RAM range per instance, but this
  can grow with the number of PVCs and labels tracked.

## 📜 License

MIT