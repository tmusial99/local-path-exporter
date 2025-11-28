package collector

import (
	"io/fs"
	"local-path-exporter/parser"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type dataPoint struct {
	sizeBytes float64
	labels    []string
}

type PVCCollector struct {
	path   string
	parser *parser.DirParser

	descPVC      *prometheus.Desc
	descCapacity *prometheus.Desc
	descUsed     *prometheus.Desc

	// Cache with Mutex
	mu           sync.RWMutex
	cachePVCs    []dataPoint
	fsCapacity   float64
	fsDeviceUsed float64
}

func NewPVCCollector(storagePath string, p *parser.DirParser) *PVCCollector {
	return &PVCCollector{
		path:   storagePath,
		parser: p,
		descPVC: prometheus.NewDesc(
			"local_path_pvc_usage_bytes",
			"Actual disk usage of the specific local-path PVC directory",
			p.LabelNames,
			nil,
		),
		descCapacity: prometheus.NewDesc(
			"local_path_storage_capacity_bytes",
			"Total capacity of the underlying storage filesystem",
			nil, nil,
		),
		descUsed: prometheus.NewDesc(
			"local_path_storage_total_used_bytes",
			"Total used space on the underlying storage filesystem (includes non-PVC data)",
			nil, nil,
		),
		cachePVCs: make([]dataPoint, 0),
	}
}

func (c *PVCCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.descPVC
	ch <- c.descCapacity
	ch <- c.descUsed
}

func (c *PVCCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, dp := range c.cachePVCs {
		ch <- prometheus.MustNewConstMetric(
			c.descPVC,
			prometheus.GaugeValue,
			dp.sizeBytes,
			dp.labels...,
		)
	}

	if c.fsCapacity > 0 {
		ch <- prometheus.MustNewConstMetric(c.descCapacity, prometheus.GaugeValue, c.fsCapacity)
		ch <- prometheus.MustNewConstMetric(c.descUsed, prometheus.GaugeValue, c.fsDeviceUsed)
	}
}

// StartBackgroundScanner performs periodic scanning of PVC directories
func (c *PVCCollector) StartBackgroundScanner(interval time.Duration) {
	c.scan()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.scan()
		}
	}()
}

// scan performs disk usage scanning and updates cache
func (c *PVCCollector) scan() {
	start := time.Now()

	var stat syscall.Statfs_t
	var capacity, used float64

	if err := syscall.Statfs(c.path, &stat); err != nil {
		log.Printf("Error getting filesystem stats for %s: %v", c.path, err)
	} else {
		bsize := uint64(stat.Bsize)
		totalBlocks := uint64(stat.Blocks)
		freeBlocks := uint64(stat.Bfree)

		capacity = float64(totalBlocks * bsize)
		used = float64((totalBlocks - freeBlocks) * bsize)
	}

	entries, err := os.ReadDir(c.path)
	if err != nil {
		log.Printf("Error reading storage root %s: %v", c.path, err)
		return
	}

	var newPVCs []dataPoint

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		labelValues, matched := c.parser.Parse(dirName)
		if !matched {
			continue
		}

		fullPath := filepath.Join(c.path, dirName)
		size, err := getDirSize(fullPath)
		if err != nil {
			log.Printf("Error scanning %s: %v", dirName, err)
			continue
		}

		newPVCs = append(newPVCs, dataPoint{
			sizeBytes: size,
			labels:    labelValues,
		})
	}

	c.mu.Lock()
	c.cachePVCs = newPVCs
	c.fsCapacity = capacity
	c.fsDeviceUsed = used
	c.mu.Unlock()

	if os.Getenv("DEBUG") == "true" {
		log.Printf("Scan finished in %v. Found %d PVCs. Disk Usage: %.2f GB / %.2f GB",
			time.Since(start), len(newPVCs), used/1024/1024/1024, capacity/1024/1024/1024)
	}
}

func getDirSize(path string) (float64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return float64(size), err
}
