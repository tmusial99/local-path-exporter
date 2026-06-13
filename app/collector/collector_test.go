package collector

import (
	"local-path-exporter/parser"
	"os"
	"path/filepath"
	"testing"
)

const testTemplate = "pvc-*_{namespace}_{name}"

func newTestCollector(t *testing.T, storagePath string) *PVCCollector {
	t.Helper()
	p, err := parser.NewDirParser(testTemplate)
	if err != nil {
		t.Fatalf("failed to build parser: %v", err)
	}
	return NewPVCCollector(storagePath, p)
}

// findByLabels returns the cached dataPoint whose labels match all given values.
func findByLabels(c *PVCCollector, labels ...string) (dataPoint, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, dp := range c.cachePVCs {
		if len(dp.labels) != len(labels) {
			continue
		}
		match := true
		for i := range labels {
			if dp.labels[i] != labels[i] {
				match = false
				break
			}
		}
		if match {
			return dp, true
		}
	}
	return dataPoint{}, false
}

func TestScanReportsBlockRoundedSize(t *testing.T) {
	root := t.TempDir()

	pvcDir := filepath.Join(root, "pvc-abc123_default_my-claim")
	if err := os.Mkdir(pvcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A small file: its block usage should be rounded up to at least one block.
	if err := os.WriteFile(filepath.Join(pvcDir, "data"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := newTestCollector(t, root)
	c.scan()

	dp, ok := findByLabels(c, "default", "my-claim")
	if !ok {
		t.Fatalf("expected matched PVC directory in cache, got %+v", c.cachePVCs)
	}

	size := int64(dp.sizeBytes)
	if size%512 != 0 {
		t.Errorf("expected block-rounded size (multiple of 512), got %d", size)
	}
	if size < 512 {
		t.Errorf("expected at least one block of usage, got %d", size)
	}
	// Block usage must exceed the 5-byte apparent file length (it counts the
	// directory block plus the file's allocated block).
	if size <= 5 {
		t.Errorf("expected block usage to exceed apparent file length, got %d", size)
	}
}

func TestScanExcludesNonMatchingDir(t *testing.T) {
	root := t.TempDir()

	if err := os.Mkdir(filepath.Join(root, "not-a-pvc"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "pvc-x_team-a_claim-a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	c := newTestCollector(t, root)
	c.scan()

	if len(c.cachePVCs) != 1 {
		t.Fatalf("expected exactly 1 matched PVC, got %d: %+v", len(c.cachePVCs), c.cachePVCs)
	}
	if _, ok := findByLabels(c, "team-a", "claim-a"); !ok {
		t.Errorf("expected matched dir to be present in cache")
	}
}

func TestScanDropsDeletedDir(t *testing.T) {
	root := t.TempDir()

	dirA := filepath.Join(root, "pvc-1_ns-a_claim-a")
	dirB := filepath.Join(root, "pvc-2_ns-b_claim-b")
	for _, d := range []string{dirA, dirB} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	c := newTestCollector(t, root)
	c.scan()
	if len(c.cachePVCs) != 2 {
		t.Fatalf("expected 2 matched PVCs after first scan, got %d", len(c.cachePVCs))
	}

	if err := os.RemoveAll(dirB); err != nil {
		t.Fatalf("remove dir: %v", err)
	}

	c.scan()
	if len(c.cachePVCs) != 1 {
		t.Fatalf("expected 1 matched PVC after deletion, got %d: %+v", len(c.cachePVCs), c.cachePVCs)
	}
	if _, ok := findByLabels(c, "ns-b", "claim-b"); ok {
		t.Errorf("expected deleted PVC to drop out of cache")
	}
	if _, ok := findByLabels(c, "ns-a", "claim-a"); !ok {
		t.Errorf("expected surviving PVC to remain in cache")
	}
}
